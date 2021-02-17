# MarshalFS [a go package]

Simulate a readonly filesystem, backed by serializable objects and/or functions.

With `marshalfs`, you can back your 'filesystem' with in-memory objects, or you can alternatively plug it in to external data sources (see 'Caveats', below).

Each time you open a file, it will re-marshal the backing object (or re-run the backing function).

`marshalfs` only works with Go 1.16+. It can be thought of as a tool to simplify building a custom `fs.FS`.

[![Go Reference](https://pkg.go.dev/badge/github.com/laher/marshalfs.svg)](https://pkg.go.dev/github.com/laher/marshalfs)

## Why for?

Haven't you heard, [everything is a file](https://en.wikipedia.org/wiki/Everything_is_a_file)?

Mainly, for testing, and for accessing any data source as though it were a file.

I can think of a bunch of uses for a read-only filesystem:

 * Testing:
  * Testing config parsing.
  * Injecting config into tests.
  * Simulate file changes over time.
  * Imitate a serial interface or some other filesystem-based resource.
 * Reading a completely different data source, as though it were a file.
 * Optionally, overlay this filesystem over a real `os.DirFS` filesystem, or any other `fs.FS`, using [mergefs](https://github.com/laher/mergefs).

Last but not least, if you just want to implement an exotic `fs.FS` filesystem, then marshalfs does some of the harder stuff for you.

### For testing Config

Test your config parsing without actually storing 'fixture' files on the filesystem. ...

 * e.g.: testing config files ... See [Example_forConfig()](./example_config_test.go) for a demonstration
 * e.g.2: injecting config data without writing to the filesystem:

```go
  mfs := marshalfs.New(json.Marshal, marshalfs.FileMap{
      "config.json": marshalfs.NewFile(&myconfig{Env: "production", I: 3}),
      "config-staging.json": marshalfs.NewFile(&myconfig{Env: "staging", I: 2}),
      "config.yaml": marshalfs.NewFile(&myconfig{S: "production", I: 3}, marshalfs.WithCustomMarshaler(yaml.Marshal)),
    })
```

### For representing a data source as files

To use an external, dynamic data source, you'll need to write a `marshalfs.Generator`

 * e.g. read from a 'database' ... See [Example_forDB()](./example_db_test.go) for a demonstration with a dummy database.
 * e.g.2: see [marshalfs-examples](https://github.com/laher/marshalfs-examples) for a [postgres-backed filesystem](https://github.com/laher/marshalfs-examples/blob/619720c38c44a4513032f7034d256e58ef789d0c/sqlx_test.go#L52-L77).

```go
	myfs := marshalfs.New(json.Marshal,
		marshalfs.NewFileGenerator("*.json", func(filename string) (interface{}, error) {
			base := filepath.Base(filename)
			id := base[:len(base)-5]
			v, err := queryByID(id)
			if err != nil {
				return nil, err
			}
			return v, nil
		}))
	b, err := myfs.Open("b.json")
```

## Marshalers

Known usages or examples of use. ...

_Please contribute by sending a PR with a link to an example._

| Marshaler | Verified | Notes |
|-----------|----------|-------|
| [json](https://godoc.org/encoding/json) | [[x]](./example_config_test.go) | |
| [yaml](https://godoc.org/gopkg.in/yaml.v2) | [x] | |
| [xml](https://godoc.org/encoding/xml) | [ ] | |
| [asn1](https://godoc.org/encoding/asn1) | [ ] | |
| [toml](https://pkg.go.dev/github.com/pelletier/go-toml) | [ ] | |
| [toml](https://github.com/BurntSushi/toml) | [ ] | |
| [ini](https://github.com/go-ini/ini) | [ ] | |
| [csv](https://pkg.go.dev/github.com/jszwec/csvutil) | [ ] | |

## Generators

You can use any data source to back a MarshalFS. Here are some data sources with (eventually) links to usage in conjunction with MarshalFS ...

_Please contribute by sending a PR with a link to an example._

| Marshaler | Verified | Notes |
|-----------|----------|-------|
| [db (sqlx)](https://github.com/jmoiron/sqlx) | [[x]](https://github.com/laher/marshalfs-examples) | |
| [dynamodb](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/dynamodbattribute/#Marshal) | [ ] | |
| [bigquery](https://godoc.org/cloud.google.com/go/bigquery) | [ ] | |
| [reform](https://godoc.org/gopkg.in/reform.v1) | [ ] | |
| [datastore](https://godoc.org/cloud.google.com/go/datastore) | [ ] | |
| [spanner](https://godoc.org/cloud.google.com/go/spanner) | [ ] | |
| [mongodb](https://godoc.org/labix.org/v2/mgo/bson) | [ ] | |
| [mongodb](https://godoc.org/go.mongodb.org/mongo-driver/bson/bsoncodec) | [ ] | |
| [gorm](https://godoc.org/github.com/jinzhu/gorm) | [ ] | |
| [validate](https://github.com/go-playground/validator) | [ ] | |
| [mapstructure](https://godoc.org/github.com/mitchellh/mapstructure) | [ ] | |
| [protobuf](https://github.com/golang/protobuf) | [ ] | |
| [s3](https://pkg.go.dev/github.com/aws/aws-sdk-go/service/s3) | [ ] | See S3FS, below |

## Caveats

 * This implementation is NOT computationally efficient. It keeps entire 'file's in RAM.
 * `fs.FS` is a read-only API.

## Incomplete plans

 * Support 'listable vs non-listable' FS.
   * Alongside `FileGen`, implement a `FileGenListable`, which takes a glob "dir/*", and knows how to list directories using some function.
   * Implement ReadDirFS, which only accepts `ListableFile` items, not just `FileDef` items.
 * Implement StatFS/ReadFileFS
 * ~Caching layer to follow. I want to make a caching layer which will be cleared reasonably well, so I'll take a bit more time over it._~
 * ~Options for New*() for files and filesystems~ - done in principle
 * Maybe copy mergefs into here?
 * Maybe - generators which can be updated on read

## Related Works

 * Standard Library:
   * [dirfs](https://tip.golang.org/pkg/os/) contains `os.DirFS` - this 'default' implementation is backed by an actual filesystem.
   * [testfs](https://tip.golang.org/pkg/testing/fstest/) contains a memory-map implementation and a testing tool. The standard library contains a few other fs.FS implementations (like 'zip')
   * [embedfs](https://tip.golang.org/pkg/embed/) provides access to files embedded in the running Go program.
 * An earlier work, [afero](https://github.com/spf13/afero) is a filesystem abstraction for Go, which has been the standard for filesystem abstractions up until go1.15. It's read-write, and it's a mature project. The interfaces look very different (big with lots of methods), so it's not really compatible.
 * [s3fs](https://github.com/jszwec/s3fs) is a fs.FS backed by the AWS S3 client
 * [mergefs](https://github.com/laher/mergefs) merge `fs.FS` filesystems together so that your FS can easily read from multiple sources.
 * [hashfs](https://pkg.go.dev/github.com/benbjohnson/hashfs) appends SHA256 hashes to filenames to allow for aggressive HTTP caching.
