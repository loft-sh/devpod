# ff [![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/peterbourgon/ff/v3) [![Latest Release](https://img.shields.io/github/v/release/peterbourgon/ff?style=flat-square)](https://github.com/peterbourgon/ff/releases/latest) ![Build Status](https://github.com/peterbourgon/ff/actions/workflows/test.yml/badge.svg?branch=main)

ff stands for flags-first, and provides an opinionated way to populate a
[flag.FlagSet](https://golang.org/pkg/flag#FlagSet) with configuration data from
the environment. By default, it parses only from the command line, but you can
enable parsing from environment variables (lower priority) and/or a
configuration file (lowest priority).

Building a commandline application in the style of `kubectl` or `docker`?
Consider [package ffcli](https://pkg.go.dev/github.com/peterbourgon/ff/v3/ffcli),
a natural companion to, and extension of, package ff.

## Usage

Define a flag.FlagSet in your func main.

```go
import (
	"flag"
	"os"
	"time"

	"github.com/peterbourgon/ff/v3"
)

func main() {
	fs := flag.NewFlagSet("my-program", flag.ContinueOnError)
	var (
		listenAddr = fs.String("listen-addr", "localhost:8080", "listen address")
		refresh    = fs.Duration("refresh", 15*time.Second, "refresh interval")
		debug      = fs.Bool("debug", false, "log debug information")
		_          = fs.String("config", "", "config file (optional)")
	)
```

Then, call ff.Parse instead of fs.Parse.
[Options](https://pkg.go.dev/github.com/peterbourgon/ff/v3#Option)
are available to control parse behavior.

```go
	err := ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("MY_PROGRAM"),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
	)
```

This example will parse flags from the commandline args, just like regular
package flag, with the highest priority. (The flag's default value will be used
only if the flag remains unset after parsing all provided sources of
configuration.)

Additionally, the example will look in the environment for variables with a
`MY_PROGRAM` prefix. Flag names are capitalized, and separator characters are
converted to underscores. In this case, for example, `MY_PROGRAM_LISTEN_ADDR`
would match to `listen-addr`.

Finally, if a `-config` file is specified, the example will try to parse it
using the PlainParser, which expects files in this format.


```
listen-addr localhost:8080
refresh 30s
debug true
```

You could also use the JSONParser, which expects a JSON object.

```json
{
	"listen-addr": "localhost:8080",
	"refresh": "30s",
	"debug": true
}
```

Or, you could write your own config file parser.

```go
// ConfigFileParser interprets the config file represented by the reader
// and calls the set function for each parsed flag pair.
type ConfigFileParser func(r io.Reader, set func(name, value string) error) error
```

## Flags and env vars

One common use case is to allow configuration from both flags and env vars.

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3"
)

func main() {
	fs := flag.NewFlagSet("myservice", flag.ContinueOnError)
	var (
		port  = fs.Int("port", 8080, "listen port for server (also via PORT)")
		debug = fs.Bool("debug", false, "log debug information (also via DEBUG)")
	)
	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVars()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("port %d, debug %v\n", *port, *debug)
}
```

```
$ env PORT=9090 myservice
port 9090, debug false
$ env PORT=9090 DEBUG=1 myservice -port=1234
port 1234, debug true
```

## Error handling

In general, you should call flag.NewFlagSet with the flag.ContinueOnError error 
handling strategy, which, somewhat confusingly, is the only way that ff.Parse can
return errors. (The other strategies terminate the program on error. Rude!) This 
is [the only way to detect certain types of parse failures][90], in addition to 
being good practice in general.

[90]: https://github.com/peterbourgon/ff/issues/90
