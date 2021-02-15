package marshalfs

import (
	"encoding/json"
	"io/ioutil"
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
		files: []*File{
			{
				s: &static{
					path: f0,
					value: struct {
						Thingy []byte
						Number int
					}{Thingy: []byte("hello, world\n"), Number: 10},
				},
			},
			{
				s: &static{
					path:  f1,
					value: struct{ Info string }{"Some interesting info.\n"},
				},
			},
			//glob2: {Value: struct{ Info string }{"Some globbed info.\n"}},
			{
				g: &generating{
					path: glob2,
					generator: func(name string) (interface{}, error) {
						return struct{ Info string }{"Some globbed info.\n"}, nil
					},
				},
			},
		},
	}
	for _, fn := range []string{f0, f1, f2} {
		t.Run(fn, func(t *testing.T) {
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

	t.Run("TestFS", func(t *testing.T) {
		if err := fstest.TestFS(m, f0, f1); err != nil {
			t.Fatal(err)
		}
	})

}
