# MarshalFS [a go package]

Simulate a 'readonly' filesystem, backed by serializable objects. Supply a marshaler(s) so that calling code can read your files via a standard `io.Reader`.

Note that although fs.FS is a read-only interface ,you _can_ update marshalfs's backing objects via non-standard methods (SetFile/Remove/ReplaceAll). Also, each time you open a file, it will re-marshal the backing object. MarshalFS also uses a `sync.RWMutex` to provide some concurrency safety.

`marshalfs` only works with Go 1.16+. It can be thought of as a riff on [fstest.MapFS](https://golang.org/pkg/testing/fstest/#MapFS).

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
 * Reading a completely different data source, as though it were a file (TODO: example needed - prolly some helpers too).
 * Optionally, overlay this filesystem over a real `os.DirFS` filesystem, or any other `fs.FS`, using [mergefs](https://github.com/laher/mergefs).

Last but not least, if you just want to implement an exotic `fs.FS` filesystem, then marshalfs does some of the harder stuff for you.

### For testing Config

Test your config parsing without actually storing 'fixture' files on the filesystem. ...

 * e.g.: testing config files ... See [Example_forConfig()](./example_config_test.go) for a demonstration
 * e.g.2: injecting config data without writing directly to the filesystem:

```go
  mfs, err := marshalfs.New(json.Marshal, marshalfs.FilePaths{
      "config.json": marshalfs.NewFile(&myconfig{Env: "production", I: 3}),
      "config-staging.json": marshalfs.NewFile(&myconfig{Env: "staging", I: 2}),
      "config.yaml": marshalfs.NewFile(&myconfig{S: "production", I: 3}, marshalfs.WithMarshaler(yaml.Marshal)),
    })
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

## Caveats

 * This implementation is NOT computationally efficient. It keeps entire objects in RAM, and bytes in RAM too.
 * `fs.FS` is a read-only API. In the standard sense, so is this, currently.
  * The backing objects can change each time you open them, though
  * You _can_ update the backing objects using marshalfs.FS.SetFile()/marshalfs.FS.Remove()/marshalfs.FS.ReplaceAll()
    * ReplaceAll is a good option if you want to maintain your map outside of marshalfs.

## Incomplete plans

 * Support for a writable FS will likely be postponed until fs.FS supports writable files. The eventual design is unknown.
     * Probably something like `WithUnmarshaler(json.Unmarshal)`.
 * Helpers for 'dynamically updating objects':
   * Maybe some helpers for "file generators"
 * Maybe somehow copy mergefs into here?
 * Defining filesystem using globs + functions. Say, `person/*/*` to retrieve a person by `person/{lastname}/{firstname}`.
  * I initially put a lot of effort into this but found it hard to make it pass `fstest.TestFS`.
  * This would be awesome. You could use it to define a database 'driver' to back a filesystem.
  * Essentially, for a valid fs.FS, you need to be able to list the contents a directory.
  * Whilst this seems very acheivable - you'd need to supply a 'list directory' function, it makes the package a lot more complex. As such, I've put it off for now.

## Related Works

 * Standard Library:
   * [os.DirFS](https://tip.golang.org/pkg/os/) contains `os.DirFS` - this 'default' implementation is backed by an actual filesystem.
   * [fstest.MapFS](https://tip.golang.org/pkg/testing/fstest/) contains a memory-map implementation and a testing tool. The standard library contains a few other fs.FS implementations (like 'zip')
   * [embed.FS](https://tip.golang.org/pkg/embed/) provides access to files embedded in the running Go program.
 * An earlier work, [afero](https://github.com/spf13/afero) is a filesystem abstraction for Go, which has been the standard for filesystem abstractions up until go1.15. It's read-write in the usual sense (io.Writer), and it's a mature project. The interfaces look very different (lots of methods), so it's not really compatible.
 * [s3fs](https://github.com/jszwec/s3fs) is a fs.FS backed by the AWS S3 client
 * [mergefs](https://github.com/laher/mergefs) merge `fs.FS` filesystems together so that your FS can easily read from multiple sources.
 * [hashfs](https://pkg.go.dev/github.com/benbjohnson/hashfs) appends SHA256 hashes to filenames to allow for aggressive HTTP caching.
