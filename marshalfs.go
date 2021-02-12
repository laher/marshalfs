package marshalfs

import (
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Generator func(name string) (*File, error)
type MarshalFunc func(i interface{}) ([]byte, error)

// An FS is a simple read-only filesystem backed by objects and some serialization function.
type FS struct {
	Files    map[string]*File
	Patterns map[string]Generator
	Marshal  MarshalFunc
}

func NewFile(value interface{}) *File {
	return &File{value: value}
}

// A MarshalFile describes a single file in a MarshalFS.
type File struct {
	value   interface{}
	Mode    fs.FileMode // FileInfo.Mode
	ModTime time.Time   // FileInfo.ModTime
	Sys     interface{} // FileInfo.Sys
}

var _ fs.FS = FS{}
var _ fs.File = (*openMarshalFile)(nil)

// Open opens the named file.
func (mfs FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	file, ok := mfs.Files[name]
	if !ok {
		// globbable?
		for k, v := range mfs.Patterns {
			if ok, err := filepath.Match(k, name); ok && err == nil {
				file, err = v(name)
				if err != nil {
					return nil, &fs.PathError{Op: "open", Path: name, Err: err}
				}
				break
			}
		}
	}
	if file != nil && file.Mode&fs.ModeDir == 0 {
		// Ordinary file
		return &openMarshalFile{name, marshalFileInfo{path.Base(name), file, sizeNoCache(file, mfs.Marshal)}, 0, mfs.Marshal}, nil
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
		for fname, f := range mfs.Files {
			i := strings.Index(fname, "/")
			if i < 0 {
				list = append(list, marshalFileInfo{fname, f, sizeNoCache(f, mfs.Marshal)})
			} else {
				need[fname[:i]] = true
			}
		}
	} else {
		elem = name[strings.LastIndex(name, "/")+1:]
		prefix := name + "/"
		for fname, f := range mfs.Files {
			if strings.HasPrefix(fname, prefix) {
				felem := fname[len(prefix):]
				i := strings.Index(felem, "/")
				if i < 0 {
					list = append(list, marshalFileInfo{felem, f, sizeNoCache(f, mfs.Marshal)})
				} else {
					need[fname[len(prefix):len(prefix)+i]] = true
				}
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
		list = append(list, marshalFileInfo{name, &File{Mode: fs.ModeDir}, zeroSize})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].name < list[j].name
	})

	if file == nil {
		file = &File{Mode: fs.ModeDir}
	}
	return &marshalDir{name, marshalFileInfo{elem, file, zeroSize}, list, 0}, nil
}

func marshalNoCache(f *File, marshaller MarshalFunc) ([]byte, error) {
	return marshaller(f.value)
}

func sizeNoCache(f *File, marshaller MarshalFunc) func() int64 {
	return func() int64 {
		b, _ := marshaller(f.value)
		return int64(len(b))
	}
}

func zeroSize() int64 {
	return 0
}

// fsOnly is a wrapper that hides all but the fs.FS methods,
// to avoid an infinite recursion when implementing special
// methods in terms of helpers that would use them.
// (In general, implementing these methods using the package fs helpers
// is redundant and unnecessary, but having the methods may make
// MarshalFS exercise more code paths when used in tests.)
type fsOnly struct{ fs.FS }

func (mfs FS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(fsOnly{mfs}, name)
}

func (mfs FS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(fsOnly{mfs}, name)
}

func (mfs FS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(fsOnly{mfs}, name)
}

func (mfs FS) Glob(pattern string) ([]string, error) {
	return fs.Glob(fsOnly{mfs}, pattern)
}

type noSub struct {
	FS
}

func (noSub) Sub() {} // not the fs.SubFS signature

func (mfs FS) Sub(dir string) (fs.FS, error) {
	return fs.Sub(noSub{mfs}, dir)
}

// A marshalFileInfo implements fs.FileInfo and fs.DirEntry for a given map file.
type marshalFileInfo struct {
	name string
	f    *File
	size func() int64
}

func (i *marshalFileInfo) Name() string       { return i.name }
func (i *marshalFileInfo) Mode() fs.FileMode  { return i.f.Mode }
func (i *marshalFileInfo) Type() fs.FileMode  { return i.f.Mode.Type() }
func (i *marshalFileInfo) ModTime() time.Time { return i.f.ModTime }
func (i *marshalFileInfo) IsDir() bool        { return i.f.Mode&fs.ModeDir != 0 }
func (i *marshalFileInfo) Sys() interface{}   { return i.f.Sys }

func (i *marshalFileInfo) Size() int64                { return i.size() }
func (i *marshalFileInfo) Info() (fs.FileInfo, error) { return i, nil }

// An openMarshalFile is a regular (non-directory) fs.File open for reading.
type openMarshalFile struct {
	path string
	marshalFileInfo
	offset     int64
	marshaller func(i interface{}) ([]byte, error)
}

// TODO cache bytes?
func (f *openMarshalFile) Marshal() ([]byte, error) {
	if f.marshalFileInfo.f.value == nil {
		return nil, nil
	}
	return f.marshaller(f.marshalFileInfo.f.value)
}

func (f *openMarshalFile) Stat() (fs.FileInfo, error) { return &f.marshalFileInfo, nil }

func (f *openMarshalFile) Close() error { return nil }

func (f *openMarshalFile) Read(dst []byte) (int, error) {
	data, err := f.Marshal()
	if err != nil {
		return 0, err
	}
	if f.offset >= int64(len(data)) {
		return 0, io.EOF
	}
	if f.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.path, Err: fs.ErrInvalid}
	}
	n := copy(dst, data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *openMarshalFile) Seek(offset int64, whence int) (int64, error) {
	data, err := f.Marshal()
	if err != nil {
		return 0, err
	}
	switch whence {
	case 0:
		// offset += 0
	case 1:
		offset += f.offset
	case 2:
		offset += int64(len(data))
	}
	if offset < 0 || offset > int64(len(data)) {
		return 0, &fs.PathError{Op: "seek", Path: f.path, Err: fs.ErrInvalid}
	}
	f.offset = offset
	return offset, nil
}

func (f *openMarshalFile) ReadAt(dest []byte, offset int64) (int, error) {
	data, err := f.Marshal()
	if err != nil {
		return 0, err
	}
	if offset < 0 || offset > int64(len(data)) {
		return 0, &fs.PathError{Op: "read", Path: f.path, Err: fs.ErrInvalid}
	}
	n := copy(dest, data[offset:])
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
