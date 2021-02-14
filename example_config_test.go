package marshalfs_test

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/laher/marshalfs"
)

func Example_forConfig() {
	// Given a config which is usually loaded from a file, ...
	type myconfig struct {
		I int    `json:"i"`
		S string `json:"s"`
	}

	// Here is the code under test
	// NOTE: production code would invoke it with os.DirFS
	// `config := loadConfig(os.DirFS("./config"))`
	var loadMyconfig = func(myfs fs.FS) (*myconfig, error) {
		f, err := myfs.Open("config.json")
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		c := &myconfig{}
		json.Unmarshal(b, c)
		return c, nil
	}

	// Set up ...
	input := &myconfig{S: "string", I: 3}
	mfs := marshalfs.New(json.Marshal, marshalfs.FileMap{"config.json": marshalfs.NewFile(input)})

	// Run the code
	output, err := loadMyconfig(mfs)
	// Verify file is loaded OK and content matches ...
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(input, output) {
		log.Fatal("loadConfig did not parse files as expected")
	}
	// Output:
}
