package marshalfs

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"
	"testing"
	"testing/fstest"
)

func TestMarshalFS(t *testing.T) {
	const f0 = "your/file"
	const f1 = "my/file"
	const glob2 = "their/*"
	const f2 = "their/file"
	m := &FS{
		defaultMarshaler: json.Marshal,
		files: []FileDef{
			&ObjectFile{
				path: f0,
				value: struct {
					Thingy []byte
					Number int
				}{Thingy: []byte("hello, world\n"), Number: 10},
			},
			&ObjectFile{
				path:  f1,
				value: struct{ Info string }{"Some interesting info.\n"},
			},
			//glob2: {Value: struct{ Info string }{"Some globbed info.\n"}},
			&FileGenListable{
				FileGen: &FileGen{
					path: glob2,
					generator: func(name string) (interface{}, error) {
						return struct{ Info string }{"Some globbed info.\n"}, nil
					},
				},
				readDir: func(dirname string) ([]fs.FileInfo, error) {
					if dirname == "." {
						x := &marshalFileInfo{
							name: "their",
							f: FileCommon{
								Mode: fs.ModeDir,
							},
							size: 0,
						}
						return []fs.FileInfo{x}, nil
					} else if dirname == "their" {
						x := &marshalFileInfo{
							name: "file",
							f:    FileCommon{},
							size: 10,
						}
						return []fs.FileInfo{x}, nil
					}
					return nil, os.ErrNotExist
				},
			},
		},
	}
	for _, fn := range []string{f0, f1, f2} {
		t.Run("open "+fn, func(t *testing.T) {
			f, err := m.Open(fn)
			if err != nil {
				t.Fatal(err)
			}
			b, err := ioutil.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("file: %s", fn)
			t.Logf("file contents: %s", string(b))
		})
	}

	t.Run("fstest.TestFS", func(t *testing.T) {
		if err := fstest.TestFS(m, f0, f1, f2); err != nil {
			t.Fatal(err)
		}
	})

}
