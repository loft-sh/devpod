package ff

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"strings"
)

// ConfigFileParser interprets the config file represented by the reader
// and calls the set function for each parsed flag pair.
type ConfigFileParser func(r io.Reader, set func(name, value string) error) error

// Parse the flags in the flag set from the provided (presumably commandline)
// args. Additional options may be provided to have Parse also read from a
// config file, and/or environment variables, in that priority order.
func Parse(fs *flag.FlagSet, args []string, options ...Option) error {
	var c Context
	for _, option := range options {
		option(&c)
	}

	flag2env := map[*flag.Flag]string{}
	env2flag := map[string]*flag.Flag{}
	fs.VisitAll(func(f *flag.Flag) {
		var key string
		key = strings.ToUpper(f.Name)
		key = flagNameToEnvVar.Replace(key)
		key = maybePrefix(c.envVarPrefix != "", key, c.envVarPrefix)
		env2flag[key] = f
		flag2env[f] = key
	})

	// First priority: commandline flags (explicit user preference).

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("error parsing commandline arguments: %w", err)
	}

	provided := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	// Second priority: environment variables (session).

	if c.readEnvVars {
		var visitErr error
		fs.VisitAll(func(f *flag.Flag) {
			if visitErr != nil {
				return
			}

			if provided[f.Name] {
				return
			}

			key, ok := flag2env[f]
			if !ok {
				panic(fmt.Errorf("%s: invalid flag/env mapping", f.Name))
			}

			value := os.Getenv(key)
			if value == "" {
				return
			}

			for _, v := range maybeSplit(value, c.envVarSplit) {
				if err := fs.Set(f.Name, v); err != nil {
					visitErr = fmt.Errorf("error setting flag %q from environment variable %q: %w", f.Name, key, err)
					return
				}
			}
		})
		if visitErr != nil {
			return fmt.Errorf("error parsing environment variables: %w", visitErr)
		}
	}

	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	// Third priority: config file (host).

	var configFile string
	if c.configFileVia != nil {
		configFile = *c.configFileVia
	}

	if configFile == "" && c.configFileFlagName != "" {
		if f := fs.Lookup(c.configFileFlagName); f != nil {
			configFile = f.Value.String()
		}
	}

	if c.configFileOpenFunc == nil {
		c.configFileOpenFunc = func(s string) (iofs.File, error) {
			return os.Open(s)
		}
	}

	var (
		haveConfigFile  = configFile != ""
		haveParser      = c.configFileParser != nil
		parseConfigFile = haveConfigFile && haveParser
	)
	if parseConfigFile {
		f, err := c.configFileOpenFunc(configFile)
		switch {
		case err == nil:
			defer f.Close()
			if err := c.configFileParser(f, func(name, value string) error {
				if provided[name] {
					return nil
				}

				var (
					f1 = fs.Lookup(name)
					f2 = env2flag[name]
					f  *flag.Flag
				)
				switch {
				case f1 == nil && f2 == nil && c.ignoreUndefined:
					return nil
				case f1 == nil && f2 == nil && !c.ignoreUndefined:
					return fmt.Errorf("config file flag %q not defined in flag set", name)
				case f1 != nil && f2 == nil:
					f = f1
				case f1 == nil && f2 != nil:
					f = f2
				case f1 != nil && f2 != nil && f1 == f2:
					f = f1
				case f1 != nil && f2 != nil && f1 != f2:
					return fmt.Errorf("config file flag %q ambiguous: matches %s and %s", name, f1.Name, f2.Name)
				}

				if provided[f.Name] {
					return nil
				}

				if err := fs.Set(f.Name, value); err != nil {
					return fmt.Errorf("error setting flag %q from config file: %w", name, err)
				}

				return nil
			}); err != nil {
				return err
			}

		case errors.Is(err, iofs.ErrNotExist) && c.allowMissingConfigFile:
			// no problem

		default:
			return err
		}
	}

	fs.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})

	return nil
}

// Context contains private fields used during parsing.
type Context struct {
	configFileVia          *string
	configFileFlagName     string
	configFileParser       ConfigFileParser
	configFileOpenFunc     func(string) (iofs.File, error)
	allowMissingConfigFile bool
	readEnvVars            bool
	envVarPrefix           string
	envVarSplit            string
	ignoreUndefined        bool
}

// Option controls some aspect of Parse behavior.
type Option func(*Context)

// WithConfigFile tells Parse to read the provided filename as a config file.
// Requires WithConfigFileParser, and overrides WithConfigFileFlag. Because
// config files should generally be user-specifiable, this option should rarely
// be used; prefer WithConfigFileFlag.
func WithConfigFile(filename string) Option {
	return WithConfigFileVia(&filename)
}

// WithConfigFileVia tells Parse to read the provided filename as a config file.
// Requires WithConfigFileParser, and overrides WithConfigFileFlag. This is
// useful for sharing a single root level flag for config files among multiple
// ffcli subcommands.
func WithConfigFileVia(filename *string) Option {
	return func(c *Context) {
		c.configFileVia = filename
	}
}

// WithConfigFileFlag tells Parse to treat the flag with the given name as a
// config file. Requires WithConfigFileParser, and is overridden by
// WithConfigFile.
//
// To specify a default config file, provide it as the default value of the
// corresponding flag. See also: WithAllowMissingConfigFile.
func WithConfigFileFlag(flagname string) Option {
	return func(c *Context) {
		c.configFileFlagName = flagname
	}
}

// WithConfigFileParser tells Parse how to interpret the config file provided
// via WithConfigFile or WithConfigFileFlag.
func WithConfigFileParser(p ConfigFileParser) Option {
	return func(c *Context) {
		c.configFileParser = p
	}
}

// WithAllowMissingConfigFile tells Parse to permit the case where a config file
// is specified but doesn't exist.
//
// By default, missing config files cause Parse to fail.
func WithAllowMissingConfigFile(allow bool) Option {
	return func(c *Context) {
		c.allowMissingConfigFile = allow
	}
}

// WithEnvVarNoPrefix is an alias for WithEnvVars.
//
// DEPRECATED: prefer WithEnvVars.
var WithEnvVarNoPrefix = WithEnvVars

// WithEnvVars tells Parse to set flags from environment variables. Flag
// names are matched to environment variables by capitalizing the flag name, and
// replacing separator characters like periods or hyphens with underscores.
//
// By default, flags are not set from environment variables at all.
func WithEnvVars() Option {
	return func(c *Context) {
		c.readEnvVars = true
	}
}

// WithEnvVarPrefix is like WithEnvVars, but only considers environment
// variables beginning with the given prefix followed by an underscore. That
// prefix (and underscore) are removed before matching to flag names. This
// option is also respected by the EnvParser config file parser.
//
// By default, flags are not set from environment variables at all.
func WithEnvVarPrefix(prefix string) Option {
	return func(c *Context) {
		c.readEnvVars = true
		c.envVarPrefix = prefix
	}
}

// WithEnvVarSplit tells Parse to split environment variables on the given
// delimiter, and to make a call to Set on the corresponding flag with each
// split token.
func WithEnvVarSplit(delimiter string) Option {
	return func(c *Context) {
		c.envVarSplit = delimiter
	}
}

// WithIgnoreUndefined tells Parse to ignore undefined flags that it encounters
// in config files. By default, if Parse encounters an undefined flag in a
// config file, it will return an error. Note that this setting does not apply
// to undefined flags passed as arguments.
func WithIgnoreUndefined(ignore bool) Option {
	return func(c *Context) {
		c.ignoreUndefined = ignore
	}
}

// WithFilesystem tells Parse to use the provided filesystem when accessing
// files on disk, for example when reading a config file. By default, the host
// filesystem is used, via [os.Open].
func WithFilesystem(fs embed.FS) Option {
	return func(c *Context) {
		c.configFileOpenFunc = fs.Open
	}
}

var flagNameToEnvVar = strings.NewReplacer(
	"-", "_",
	".", "_",
	"/", "_",
)

func maybePrefix(doPrefix bool, key string, prefix string) string {
	if doPrefix {
		key = strings.ToUpper(prefix) + "_" + key
	}
	return key
}

func maybeSplit(value, split string) []string {
	if split == "" {
		return []string{value}
	}
	return strings.Split(value, split)
}

// StringConversionError was returned by config file parsers in certain cases.
//
// DEPRECATED: this error is no longer returned by anything.
type StringConversionError struct {
	Value interface{}
}

// Error implements the error interface.
func (e StringConversionError) Error() string {
	return fmt.Sprintf("couldn't convert %q (type %T) to string", e.Value, e.Value)
}
