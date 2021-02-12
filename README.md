# MarshalFS

Simulate a readonly filesystem, by serializing objects and/or functions, linked to file paths

## Why for?

Testing, mostly.

I can think of a bunch of uses for a read-only filesystem.

Testing config parsing. Injecting config into tests.

Reading 'files' from some other data source ...

## Config

Test your config parsing without actually storing heaps of files on the filesystem ...

### Example 1: testing config files

```go
	// production code
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
```

### Example 2: injecting config data without writing to the filesystem

```go
```
## Database

### Example 3: read from a database ...

```go
marshalfs.MarshalFS{Marshal: json.Marshal, Patterns: {
  "*.json": func(filename string) (*MarshalFile, error) {
				return &MarshalFile{Value: struct{ Info string }{"Some globbed info.\n"}}, nil
  },
}

```

## Caveats

This implementation is NOT computationally efficient.

It will repeatedly marshal your objects to bytes, any time any Read or Seek operation is called.

It's much like `testing/fstest`, but worse becuase of the marshalling

_Caching layer to follow. I want to make a caching layer which will be cleared reasonably well, so I'll take a bit more time over it._
