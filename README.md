# MarshalFS

Simulate a readonly filesystem, by serializing objects and/or functions, linked to file paths.

marshalfs only works with Go 1.16+ (Go 1.16 is in Beta right now).

[![Go Reference](https://pkg.go.dev/badge/github.com/laher/marshalfs.svg)](https://pkg.go.dev/github.com/laher/marshalfs)

## Why for?

Testing, mostly.

I can think of a bunch of uses for a read-only filesystem.

Testing config parsing. Injecting config into tests.

Reading 'files' from some other data source ...

## Config

Test your config parsing without actually storing heaps of files on the filesystem ...

```mfs := marshalfs.FS{Marshal: json.Marshal, Files: map[string]*marshalfs.File{"config.json": marshalfs.NewFile(input)}}```

 * e.g.: testing config files ... See [Example_forConfig()](./example_config_test.go) for a demonstration
 * e.g.: injecting config data without writing to the filesystem

## Database

 * e.g. read from a database ... See [Example_forDB()](./example_db_test.go) for a demonstration

## Caveats

 * This implementation is NOT computationally efficient. It will repeatedly marshal your objects to bytes, any time any Read or Seek operation is called. It's much like `testing/fstest`, but worse becuase of the marshalling step.
 * `fs.FS` is a read-only API.

## Incomplete plans

 * Caching layer to follow. I want to make a caching layer which will be cleared reasonably well, so I'll take a bit more time over it._
 * Options for creating files and filesystems
 * Maybe - generators which can be updated on read
