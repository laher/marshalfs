package marshalfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func ExampleForDB() {
	table := map[string]string{"b": "bananas"}

	// production code would use os.DirFS
	// config := loadConfig(os.DirFS("./config"))

	// test code
	myfs := MarshalFS{
		Marshal: json.Marshal,
		Patterns: map[string]PatternGenerator{
			"*.json": func(filename string) (*MarshalFile, error) {
				base := filepath.Base(filename)
				id := base[:len(base)-5]
				v, ok := table[id]
				if !ok {
					return nil, os.ErrNotExist
				}
				return &MarshalFile{Value: v}, nil
			},
		},
	}

	_, err := myfs.Open("a.json")
	if !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}
	c, err := myfs.Open("b.json")
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(c)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output: "bananas"
}
