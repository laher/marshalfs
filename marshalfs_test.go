package marshalfs

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"testing"
	"testing/fstest"
)

func TestMarshalFS(t *testing.T) {
	const f0 = "your/file"
	const f1 = "my/file"
	m := &FS{
		defaultMarshaler: json.Marshal,
		files: map[string]FileSpec{
			f0: &objectBackedFileSpec{
				value: struct {
					Thingy []byte
					Number int
				}{Thingy: []byte("hello, world\n"), Number: 10},
			},
			f1: &objectBackedFileSpec{
				value: struct{ Info string }{"Some interesting info.\n"},
			},
		},
	}
	for _, fname := range []string{f0, f1} {
		fn := fname
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

func TestConflict(t *testing.T) {
	t.Run("NoConflict", func(t *testing.T) {
		files := FileSpecs{
			"dir/a": &objectBackedFileSpec{
				value: struct {
					Thingy []byte
					Number int
				}{Thingy: []byte("hello, world\n"), Number: 10},
			},
			"dir/b": &objectBackedFileSpec{
				value: struct{ Info string }{"Some interesting info.\n"},
			},
		}
		err := files.validate()
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Conflict", func(t *testing.T) {
		files := FileSpecs{
			"dir/a": &objectBackedFileSpec{
				value: struct {
					Thingy []byte
					Number int
				}{Thingy: []byte("hello, world\n"), Number: 10},
			},
			"dir/b": &objectBackedFileSpec{
				value: struct{ Info string }{"Some interesting info.\n"},
			},
			"dir/b/c": &objectBackedFileSpec{
				value: struct{ Info string }{"Some interesting info.\n"},
			},
		}
		err := files.validate()
		if !errors.Is(err, ErrPathConflict) {
			t.FailNow()
		}
	})
}
