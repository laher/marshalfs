package marshalfs

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"log"
	"reflect"
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
		c, err := myfs.Open("config.json")
		if err != nil {
			return nil, err
		}
		ret := &myconfig{}

		b, err := ioutil.ReadAll(c)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(b, ret)
		return ret, nil
	}

	// test code
	input := &myconfig{S: "string", I: 3}
	mfs := MarshalFS{Marshal: json.Marshal, Files: map[string]*MarshalFile{"config.json": &MarshalFile{Value: input}}}
	output, err := loadMyconfig(mfs)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(input, output) {
		log.Fatal("loadConfig did not parse files as expected")
	}
	// Output:
}
