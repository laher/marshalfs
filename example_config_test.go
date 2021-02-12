package marshalfs

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"log"
	"reflect"
)

func ExampleForConfig() {
	type myconfig struct {
		I int
		S string
	}

	// code under test
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

	// production code would use os.DirFS
	// config := loadConfig(os.DirFS("./config"))

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
