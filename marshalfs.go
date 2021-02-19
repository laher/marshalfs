package marshalfs

import (
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

type MarshalFunc func(i interface{}) ([]byte, error)
type FileOption func(*FileCommon)

// An FS is a simple read-only filesystem backed by objects and some serialization function.
type FS struct {
	files            map[string]FileSpec
	lock             sync.RWMutex
	defaultMarshaler MarshalFunc
}

type FilePaths map[string]FileSpec

func New(defaultMarshaler MarshalFunc, files FilePaths) *FS {
	mfs := &FS{defaultMarshaler: defaultMarshaler, files: files}
	return mfs
}

type FileSpec interface {
	Common() FileCommon
	readDirFunc() // this is in prepration for planned 'dynamic files'
}

// objectBackedFileSpec describes a file in a MarshalFS.
type objectBackedFileSpec struct {
	value interface{}
	FileCommon
}

func (f *objectBackedFileSpec) readDirFunc() {}

func (f *objectBackedFileSpec) Common() FileCommon {
	return f.FileCommon
}

type FileCommon struct {
	Mode            fs.FileMode // FileInfo.Mode
	ModTime         time.Time   // FileInfo.ModTime
	Sys             interface{} // FileInfo.Sys
	customMarshaler MarshalFunc
}

// NewFile creates a new File
func NewFile(value interface{}, opts ...FileOption) FileSpec {
	f := &objectBackedFileSpec{value: value}
	for _, opt := range opts {
		opt(&f.FileCommon)
	}
	return f
}

func WithMode(mode fs.FileMode) FileOption {
	return func(f *FileCommon) {
		f.Mode = mode
	}
}

func WithModTime(t time.Time) FileOption {
	return func(f *FileCommon) {
		f.ModTime = t
	}
}

func WithMarshaler(mf MarshalFunc) FileOption {
	return func(f *FileCommon) {
		f.customMarshaler = mf
	}
}

// ensure that they implement the right interfaces
var _ fs.FS = &FS{}
var _ fs.File = (*openMarshalFile)(nil)

// Open opens the named file.
func (mfs *FS) Open(name string) (fs.File, error) {
	mfs.lock.RLock()
	defer mfs.lock.RUnlock()
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	var file FileSpec
	for fpath, f := range mfs.files {
		switch ft := f.(type) {
		case *objectBackedFileSpec:
			if name == fpath {
				file = ft
				break
			}
		default:
			// TODO - handling for dynamic files
		}
	}
	if file != nil && file.Common().Mode&fs.ModeDir == 0 {
		marshaler := mfs.defaultMarshaler
		if file.Common().customMarshaler != nil {
			marshaler = file.Common().customMarshaler
		}

		var value interface{}
		switch ft := file.(type) {
		case *objectBackedFileSpec:
			value = ft.value
		default:
			// TODO - handling for dynamic files
		}

		data, err := marshaler(value)
		if err != nil {
			return nil, &fs.PathError{Op: "open", Path: name, Err: err}
		}
		// Ordinary file
		return &openMarshalFile{
			path: name,
			data: data,
			marshalFileInfo: marshalFileInfo{
				name: path.Base(name),
				f:    file.Common(),
				size: sizeNoCache(value, marshaler),
			},
			offset: 0,
		}, nil
	}

	// Directory, possibly synthesized.
	// Note that file can be nil here: the map need not contain explicit parent directories for all its files.
	// But file can also be non-nil, in case the user wants to set metadata for the directory explicitly.
	// Either way, we need to construct the list of children of this directory.
	var list []marshalFileInfo
	var elem string
	var need = make(map[string]bool)
	if name == "." {
		elem = "."
		for fpath, f := range mfs.files {
			switch ft := f.(type) {
			case *objectBackedFileSpec:
				i := strings.Index(fpath, "/")
				if i < 0 {
					list = append(list, marshalFileInfo{fpath, ft.FileCommon, sizeNoCache(ft.value, mfs.defaultMarshaler)})
				} else {
					need[fpath[:i]] = true
				}
			default:
				// TODO - directory handling for dynamic files
			}
		}
	} else {
		elem = name[strings.LastIndex(name, "/")+1:]
		prefix := name + "/"
		for fpath, f := range mfs.files {
			switch ft := f.(type) {
			case *objectBackedFileSpec:
				if strings.HasPrefix(fpath, prefix) {
					felem := fpath[len(prefix):]
					i := strings.Index(felem, "/")
					if i < 0 {
						list = append(list, marshalFileInfo{felem, ft.FileCommon, sizeNoCache(ft.value, mfs.defaultMarshaler)})
					} else {
						need[fpath[len(prefix):len(prefix)+i]] = true
					}
				}
			default:
				// TODO - directory handling for dynamic files
			}
		}
		// If the directory name is not in the map,
		// and there are no children of the name in the map,
		// then the directory is treated as not existing.
		if file == nil && list == nil && len(need) == 0 {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
	}
	for _, fi := range list {
		delete(need, fi.name)
	}
	for name := range need {
		list = append(list, marshalFileInfo{name, FileCommon{Mode: fs.ModeDir}, 0})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].name < list[j].name
	})

	if file == nil {
		file = &objectBackedFileSpec{FileCommon: FileCommon{Mode: fs.ModeDir}}
	}
	return &marshalDir{name, marshalFileInfo{elem, file.Common(), 0}, list, 0}, nil
}

func sizeNoCache(value interface{}, marshaller MarshalFunc) int64 {
	b, _ := marshaller(value)
	return int64(len(b))
}

// WriteFile is similar to os.WriteFile, except it takes a FileSpec instead of `[]byte, mode`
func (mfs *FS) WriteFile(filename string, item FileSpec) {
	mfs.lock.Lock()
	defer mfs.lock.Unlock()
	mfs.files[filename] = item
}

func (mfs *FS) Del(filename string) {
	mfs.lock.Lock()
	defer mfs.lock.Unlock()
	delete(mfs.files, filename)
}

func (mfs *FS) ReplaceAll(files map[string]FileSpec) {
	mfs.lock.Lock()
	defer mfs.lock.Unlock()
	mfs.files = files
}

// fsOnly is a wrapper that hides all but the fs.FS methods,
// to avoid an infinite recursion when implementing special
// methods in terms of helpers that would use them.
// (In general, implementing these methods using the package fs helpers
// is redundant and unnecessary, but having the methods may make
// MarshalFS exercise more code paths when used in tests.)
type fsOnly struct{ fs.FS }

func (mfs *FS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(fsOnly{mfs}, name)
}

func (mfs *FS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(fsOnly{mfs}, name)
}

func (mfs *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(fsOnly{mfs}, name)
}

func (mfs *FS) Glob(pattern string) ([]string, error) {
	return fs.Glob(fsOnly{mfs}, pattern)
}

type noSub struct {
	*FS
}

func (noSub) Sub() {} // not the fs.SubFS signature

func (mfs *FS) Sub(dir string) (fs.FS, error) {
	return fs.Sub(noSub{mfs}, dir)
}

// A marshalFileInfo implements fs.FileInfo and fs.DirEntry for a given map file.
type marshalFileInfo struct {
	name string
	f    FileCommon
	size int64
}

func (i *marshalFileInfo) Name() string       { return i.name }
func (i *marshalFileInfo) Mode() fs.FileMode  { return i.f.Mode }
func (i *marshalFileInfo) Type() fs.FileMode  { return i.f.Mode.Type() }
func (i *marshalFileInfo) ModTime() time.Time { return i.f.ModTime }
func (i *marshalFileInfo) IsDir() bool        { return i.f.Mode&fs.ModeDir != 0 }
func (i *marshalFileInfo) Sys() interface{}   { return i.f.Sys }

func (i *marshalFileInfo) Size() int64                { return i.size }
func (i *marshalFileInfo) Info() (fs.FileInfo, error) { return i, nil }

// An openMarshalFile is a regular (non-directory) fs.File open for reading.
type openMarshalFile struct {
	path string
	data []byte
	marshalFileInfo
	offset int64
}

func (f *openMarshalFile) Stat() (fs.FileInfo, error) { return &f.marshalFileInfo, nil }

func (f *openMarshalFile) Close() error { return nil }

func (f *openMarshalFile) Read(dst []byte) (int, error) {
	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}
	if f.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.path, Err: fs.ErrInvalid}
	}
	n := copy(dst, f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *openMarshalFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		// offset += 0
	case 1:
		offset += f.offset
	case 2:
		offset += int64(len(f.data))
	}
	if offset < 0 || offset > int64(len(f.data)) {
		return 0, &fs.PathError{Op: "seek", Path: f.path, Err: fs.ErrInvalid}
	}
	f.offset = offset
	return offset, nil
}

func (f *openMarshalFile) ReadAt(dest []byte, offset int64) (int, error) {
	if offset < 0 || offset > int64(len(f.data)) {
		return 0, &fs.PathError{Op: "read", Path: f.path, Err: fs.ErrInvalid}
	}
	n := copy(dest, f.data[offset:])
	if n < len(dest) {
		return n, io.EOF
	}
	return n, nil
}

// A marshalDir is a directory fs.File (so also an fs.ReadDirFile) open for reading.
type marshalDir struct {
	path string
	marshalFileInfo
	entry  []marshalFileInfo
	offset int
}

func (d *marshalDir) Stat() (fs.FileInfo, error) { return &d.marshalFileInfo, nil }
func (d *marshalDir) Close() error               { return nil }
func (d *marshalDir) Read(b []byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.path, Err: fs.ErrInvalid}
}

func (d *marshalDir) ReadDir(count int) ([]fs.DirEntry, error) {
	n := len(d.entry) - d.offset
	if count > 0 && n > count {
		n = count
	}
	if n == 0 && count > 0 {
		return nil, io.EOF
	}
	list := make([]fs.DirEntry, n)
	for i := range list {
		list[i] = &d.entry[d.offset+i]
	}
	d.offset += n
	return list, nil
}
