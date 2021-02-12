package marshalfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Example_forDB() {
	// Given a database ...
	query := func(id string) (string, error) {
		v, ok := map[string]string{
			"a": "apples",
			"b": "bananas",
		}[id]
		if !ok {
			return "", os.ErrNotExist
		}
		return v, nil
	}

	// Configure a MarshalFS to query it ...
	myfs := MarshalFS{
		Marshal: json.Marshal,
		Patterns: map[string]Generator{
			"*.json": func(filename string) (*MarshalFile, error) {
				base := filepath.Base(filename)
				id := base[:len(base)-5]
				v, err := query(id)
				if err != nil {
					return nil, err
				}
				return &MarshalFile{value: v}, nil
			},
		},
	}

	// Verify that one file doesn't exist
	_, err := myfs.Open("z.json")
	if !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}

	// Verify the contents of a file which does ...
	c, err := myfs.Open("b.json")
	if err != nil {
		panic(err)
	}
	b, err := io.ReadAll(c)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(b))
	// Output: "bananas"
}
