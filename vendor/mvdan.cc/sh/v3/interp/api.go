// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// Package interp implements an interpreter that executes shell
// programs. It aims to support POSIX, but its support is not complete
// yet. It also supports some Bash features.
package interp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

// A Runner interprets shell programs. It can be reused, but it is not safe for
// concurrent use. You should typically use New to build a new Runner.
//
// Note that writes to Stdout and Stderr may be concurrent if background
// commands are used. If you plan on using an io.Writer implementation that
// isn't safe for concurrent use, consider a workaround like hiding writes
// behind a mutex.
//
// To create a Runner, use New. Runner's exported fields are meant to be
// configured via runner options; once a Runner has been created, the fields
// should be treated as read-only.
type Runner struct {
	// Env specifies the initial environment for the interpreter, which must
	// be non-nil.
	Env expand.Environ

	writeEnv expand.WriteEnviron

	// Dir specifies the working directory of the command, which must be an
	// absolute path.
	Dir string

	// Params are the current shell parameters, e.g. from running a shell
	// file or calling a function. Accessible via the $@/$* family of vars.
	Params []string

	// Separate maps - note that bash allows a name to be both a var and a
	// func simultaneously.
	// Vars is mostly superseded by Env at this point.
	// TODO(v4): remove these

	Vars  map[string]expand.Variable
	Funcs map[string]*syntax.Stmt

	alias map[string]alias

	// callHandler is a function allowing to replace a simple command's
	// arguments. It may be nil.
	callHandler CallHandlerFunc

	// execHandler is a function responsible for executing programs. It must be non-nil.
	execHandler ExecHandlerFunc

	// openHandler is a function responsible for opening files. It must be non-nil.
	openHandler OpenHandlerFunc

	// readDirHandler is a function responsible for reading directories during
	// glob expansion. It must be non-nil.
	readDirHandler ReadDirHandlerFunc

	// statHandler is a function responsible for getting file stat. It must be non-nil.
	statHandler StatHandlerFunc

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	ecfg *expand.Config
	ectx context.Context // just so that Runner.Subshell can use it again

	lastExpandExit int // used to surface exit codes while expanding fields

	// didReset remembers whether the runner has ever been reset. This is
	// used so that Reset is automatically called when running any program
	// or node for the first time on a Runner.
	didReset bool

	usedNew bool

	// rand is used mainly to generate temporary files.
	rand *rand.Rand

	// wgProcSubsts allows waiting for any process substitution sub-shells
	// to finish running.
	wgProcSubsts sync.WaitGroup

	filename string // only if Node was a File

	// >0 to break or continue out of N enclosing loops
	breakEnclosing, contnEnclosing int

	inLoop    bool
	inFunc    bool
	inSource  bool
	noErrExit bool

	// track if a sourced script set positional parameters
	sourceSetParams bool

	err          error // current shell exit code or fatal error
	handlingTrap bool  // whether we're currently in a trap callback
	shellExited  bool  // whether the shell needs to exit

	// The current and last exit status code. They can only be different if
	// the interpreter is in the middle of running a statement. In that
	// scenario, 'exit' is the status code for the statement being run, and
	// 'lastExit' corresponds to the previous statement that was run.
	exit     int
	lastExit int

	bgShells errgroup.Group

	opts runnerOpts

	origDir    string
	origParams []string
	origOpts   runnerOpts
	origStdin  io.Reader
	origStdout io.Writer
	origStderr io.Writer

	// Most scripts don't use pushd/popd, so make space for the initial PWD
	// without requiring an extra allocation.
	dirStack     []string
	dirBootstrap [1]string

	optState getopts

	// keepRedirs is used so that "exec" can make any redirections
	// apply to the current shell, and not just the command.
	keepRedirs bool

	// Fake signal callbacks
	callbackErr  string
	callbackExit string
}

type alias struct {
	args  []*syntax.Word
	blank bool
}

func (r *Runner) optByFlag(flag byte) *bool {
	for i, opt := range &shellOptsTable {
		if opt.flag == flag {
			return &r.opts[i]
		}
	}
	return nil
}

// New creates a new Runner, applying a number of options. If applying any of
// the options results in an error, it is returned.
//
// Any unset options fall back to their defaults. For example, not supplying the
// environment falls back to the process's environment, and not supplying the
// standard output writer means that the output will be discarded.
func New(opts ...RunnerOption) (*Runner, error) {
	r := &Runner{
		usedNew:        true,
		execHandler:    DefaultExecHandler(2 * time.Second),
		openHandler:    DefaultOpenHandler(),
		readDirHandler: DefaultReadDirHandler(),
		statHandler:    DefaultStatHandler(),
	}
	r.dirStack = r.dirBootstrap[:0]
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	// Set the default fallbacks, if necessary.
	if r.Env == nil {
		Env(nil)(r)
	}
	if r.Dir == "" {
		if err := Dir("")(r); err != nil {
			return nil, err
		}
	}
	if r.stdout == nil || r.stderr == nil {
		StdIO(r.stdin, r.stdout, r.stderr)(r)
	}
	return r, nil
}

// RunnerOption is a function which can be passed to New to alter Runner behaviour.
// To apply option to existing Runner call it directly,
// for example interp.Params("-e")(runner).
type RunnerOption func(*Runner) error

// Env sets the interpreter's environment. If nil, a copy of the current
// process's environment is used.
func Env(env expand.Environ) RunnerOption {
	return func(r *Runner) error {
		if env == nil {
			env = expand.ListEnviron(os.Environ()...)
		}
		r.Env = env
		return nil
	}
}

// Dir sets the interpreter's working directory. If empty, the process's current
// directory is used.
func Dir(path string) RunnerOption {
	return func(r *Runner) error {
		if path == "" {
			path, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("could not get current dir: %w", err)
			}
			r.Dir = path
			return nil
		}
		path, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("could not get absolute dir: %w", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("could not stat: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
		r.Dir = path
		return nil
	}
}

// Params populates the shell options and parameters. For example, Params("-e",
// "--", "foo") will set the "-e" option and the parameters ["foo"], and
// Params("+e") will unset the "-e" option and leave the parameters untouched.
//
// This is similar to what the interpreter's "set" builtin does.
func Params(args ...string) RunnerOption {
	return func(r *Runner) error {
		fp := flagParser{remaining: args}
		for fp.more() {
			flag := fp.flag()
			if flag == "-" {
				// TODO: implement "The -x and -v options are turned off."
				if args := fp.args(); len(args) > 0 {
					r.Params = args
				}
				return nil
			}
			enable := flag[0] == '-'
			if flag[1] != 'o' {
				opt := r.optByFlag(flag[1])
				if opt == nil {
					return fmt.Errorf("invalid option: %q", flag)
				}
				*opt = enable
				continue
			}
			value := fp.value()
			if value == "" && enable {
				for i, opt := range &shellOptsTable {
					r.printOptLine(opt.name, r.opts[i])
				}
				continue
			}
			if value == "" && !enable {
				for i, opt := range &shellOptsTable {
					setFlag := "+o"
					if r.opts[i] {
						setFlag = "-o"
					}
					r.outf("set %s %s\n", setFlag, opt.name)
				}
				continue
			}
			opt := r.optByName(value, false)
			if opt == nil {
				return fmt.Errorf("invalid option: %q", value)
			}
			*opt = enable
		}
		if args := fp.args(); args != nil {
			// If "--" wasn't given and there were zero arguments,
			// we don't want to override the current parameters.
			r.Params = args

			// Record whether a sourced script sets the parameters.
			if r.inSource {
				r.sourceSetParams = true
			}
		}
		return nil
	}
}

// CallHandler sets the call handler. See CallHandlerFunc for more info.
func CallHandler(f CallHandlerFunc) RunnerOption {
	return func(r *Runner) error {
		r.callHandler = f
		return nil
	}
}

// ExecHandler sets the command execution handler. See ExecHandlerFunc for more info.
func ExecHandler(f ExecHandlerFunc) RunnerOption {
	return func(r *Runner) error {
		r.execHandler = f
		return nil
	}
}

// OpenHandler sets file open handler. See OpenHandlerFunc for more info.
func OpenHandler(f OpenHandlerFunc) RunnerOption {
	return func(r *Runner) error {
		r.openHandler = f
		return nil
	}
}

// ReadDirHandler sets the read directory handler. See ReadDirHandlerFunc for more info.
func ReadDirHandler(f ReadDirHandlerFunc) RunnerOption {
	return func(r *Runner) error {
		r.readDirHandler = f
		return nil
	}
}

// StatHandler sets the stat handler. See StatHandlerFunc for more info.
func StatHandler(f StatHandlerFunc) RunnerOption {
	return func(r *Runner) error {
		r.statHandler = f
		return nil
	}
}

// StdIO configures an interpreter's standard input, standard output, and
// standard error. If out or err are nil, they default to a writer that discards
// the output.
func StdIO(in io.Reader, out, err io.Writer) RunnerOption {
	return func(r *Runner) error {
		r.stdin = in
		if out == nil {
			out = io.Discard
		}
		r.stdout = out
		if err == nil {
			err = io.Discard
		}
		r.stderr = err
		return nil
	}
}

func (r *Runner) optByName(name string, bash bool) *bool {
	if bash {
		for i, optName := range bashOptsTable {
			if optName == name {
				return &r.opts[len(shellOptsTable)+i]
			}
		}
	}
	for i, opt := range &shellOptsTable {
		if opt.name == name {
			return &r.opts[i]
		}
	}
	return nil
}

type runnerOpts [len(shellOptsTable) + len(bashOptsTable)]bool

var shellOptsTable = [...]struct {
	flag byte
	name string
}{
	// sorted alphabetically by name; use a space for the options
	// that have no flag form
	{'a', "allexport"},
	{'e', "errexit"},
	{'n', "noexec"},
	{'f', "noglob"},
	{'u', "nounset"},
	{'x', "xtrace"},
	{' ', "pipefail"},
}

var bashOptsTable = [...]string{
	// sorted alphabetically by name
	"expand_aliases",
	"globstar",
	"nullglob",
}

// To access the shell options arrays without a linear search when we
// know which option we're after at compile time. First come the shell options,
// then the bash options.
const (
	optAllExport = iota
	optErrExit
	optNoExec
	optNoGlob
	optNoUnset
	optXTrace
	optPipeFail

	optExpandAliases
	optGlobStar
	optNullGlob
)

// Reset returns a runner to its initial state, right before the first call to
// Run or Reset.
//
// Typically, this function only needs to be called if a runner is reused to run
// multiple programs non-incrementally. Not calling Reset between each run will
// mean that the shell state will be kept, including variables, options, and the
// current directory.
func (r *Runner) Reset() {
	if !r.usedNew {
		panic("use interp.New to construct a Runner")
	}
	if !r.didReset {
		r.origDir = r.Dir
		r.origParams = r.Params
		r.origOpts = r.opts
		r.origStdin = r.stdin
		r.origStdout = r.stdout
		r.origStderr = r.stderr
	}
	// reset the internal state
	*r = Runner{
		Env:            r.Env,
		callHandler:    r.callHandler,
		execHandler:    r.execHandler,
		openHandler:    r.openHandler,
		readDirHandler: r.readDirHandler,
		statHandler:    r.statHandler,

		// These can be set by functions like Dir or Params, but
		// builtins can overwrite them; reset the fields to whatever the
		// constructor set up.
		Dir:    r.origDir,
		Params: r.origParams,
		opts:   r.origOpts,
		stdin:  r.origStdin,
		stdout: r.origStdout,
		stderr: r.origStderr,

		origDir:    r.origDir,
		origParams: r.origParams,
		origOpts:   r.origOpts,
		origStdin:  r.origStdin,
		origStdout: r.origStdout,
		origStderr: r.origStderr,

		// emptied below, to reuse the space
		Vars:     r.Vars,
		dirStack: r.dirStack[:0],
		usedNew:  r.usedNew,
	}
	if r.Vars == nil {
		r.Vars = make(map[string]expand.Variable)
	} else {
		for k := range r.Vars {
			delete(r.Vars, k)
		}
	}
	// TODO(v4): Use the supplied Env directly if it implements enough methods.
	r.writeEnv = &overlayEnviron{parent: r.Env}
	if !r.writeEnv.Get("HOME").IsSet() {
		home, _ := os.UserHomeDir()
		r.setVarString("HOME", home)
	}
	if !r.writeEnv.Get("UID").IsSet() {
		r.setVar("UID", nil, expand.Variable{
			Kind:     expand.String,
			ReadOnly: true,
			Str:      strconv.Itoa(os.Getuid()),
		})
	}
	if !r.writeEnv.Get("GID").IsSet() {
		r.setVar("GID", nil, expand.Variable{
			Kind:     expand.String,
			ReadOnly: true,
			Str:      strconv.Itoa(os.Getgid()),
		})
	}
	r.setVarString("PWD", r.Dir)
	r.setVarString("IFS", " \t\n")
	r.setVarString("OPTIND", "1")

	r.dirStack = append(r.dirStack, r.Dir)
	r.didReset = true
}

// exitStatus is a non-zero status code resulting from running a shell node.
type exitStatus uint8

func (s exitStatus) Error() string { return fmt.Sprintf("exit status %d", s) }

// NewExitStatus creates an error which contains the specified exit status code.
func NewExitStatus(status uint8) error {
	return exitStatus(status)
}

// IsExitStatus checks whether error contains an exit status and returns it.
func IsExitStatus(err error) (status uint8, ok bool) {
	var s exitStatus
	if errors.As(err, &s) {
		return uint8(s), true
	}
	return 0, false
}

// Run interprets a node, which can be a *File, *Stmt, or Command. If a non-nil
// error is returned, it will typically contain a command's exit status, which
// can be retrieved with IsExitStatus.
//
// Run can be called multiple times synchronously to interpret programs
// incrementally. To reuse a Runner without keeping the internal shell state,
// call Reset.
//
// Calling Run on an entire *File implies an exit, meaning that an exit trap may
// run.
func (r *Runner) Run(ctx context.Context, node syntax.Node) error {
	if !r.didReset {
		r.Reset()
	}
	r.fillExpandConfig(ctx)
	r.err = nil
	r.shellExited = false
	r.filename = ""
	switch x := node.(type) {
	case *syntax.File:
		r.filename = x.Name
		r.stmts(ctx, x.Stmts)
		if !r.shellExited {
			r.exitShell(ctx, r.exit)
		}
	case *syntax.Stmt:
		r.stmt(ctx, x)
	case syntax.Command:
		r.cmd(ctx, x)
	default:
		return fmt.Errorf("node can only be File, Stmt, or Command: %T", x)
	}
	if r.exit != 0 {
		r.setErr(NewExitStatus(uint8(r.exit)))
	}
	if r.Vars != nil {
		r.writeEnv.Each(func(name string, vr expand.Variable) bool {
			r.Vars[name] = vr
			return true
		})
	}
	return r.err
}

// Exited reports whether the last Run call should exit an entire shell. This
// can be triggered by the "exit" built-in command, for example.
//
// Note that this state is overwritten at every Run call, so it should be
// checked immediately after each Run call.
func (r *Runner) Exited() bool {
	return r.shellExited
}

// Subshell makes a copy of the given Runner, suitable for use concurrently
// with the original. The copy will have the same environment, including
// variables and functions, but they can all be modified without affecting the
// original.
//
// Subshell is not safe to use concurrently with Run. Orchestrating this is
// left up to the caller; no locking is performed.
//
// To replace e.g. stdin/out/err, do StdIO(r.stdin, r.stdout, r.stderr)(r) on
// the copy.
func (r *Runner) Subshell() *Runner {
	if !r.didReset {
		r.Reset()
	}
	// Keep in sync with the Runner type. Manually copy fields, to not copy
	// sensitive ones like errgroup.Group, and to do deep copies of slices.
	r2 := &Runner{
		Dir:            r.Dir,
		Params:         r.Params,
		callHandler:    r.callHandler,
		execHandler:    r.execHandler,
		openHandler:    r.openHandler,
		readDirHandler: r.readDirHandler,
		statHandler:    r.statHandler,
		stdin:          r.stdin,
		stdout:         r.stdout,
		stderr:         r.stderr,
		filename:       r.filename,
		opts:           r.opts,
		usedNew:        r.usedNew,
		exit:           r.exit,
		lastExit:       r.lastExit,

		origStdout: r.origStdout, // used for process substitutions
	}
	// Env vars and funcs are copied, since they might be modified.
	// TODO(v4): lazy copying? it would probably be enough to add a
	// copyOnWrite bool field to Variable, then a Modify method that must be
	// used when one needs to modify a variable. ideally with some way to
	// catch direct modifications without the use of Modify and panic,
	// perhaps via a check when getting or setting vars at some level.
	oenv := &overlayEnviron{parent: expand.ListEnviron()}
	r.writeEnv.Each(func(name string, vr expand.Variable) bool {
		vr2 := vr
		// Make deeper copies of List and Map, but ensure that they remain nil
		// if they are nil in vr.
		vr2.List = append([]string(nil), vr.List...)
		if vr.Map != nil {
			vr2.Map = make(map[string]string, len(vr.Map))
			for k, vr := range vr.Map {
				vr2.Map[k] = vr
			}
		}
		oenv.Set(name, vr2)
		return true
	})
	r2.writeEnv = oenv
	r2.Funcs = make(map[string]*syntax.Stmt, len(r.Funcs))
	for k, v := range r.Funcs {
		r2.Funcs[k] = v
	}
	r2.Vars = make(map[string]expand.Variable)
	if l := len(r.alias); l > 0 {
		r2.alias = make(map[string]alias, l)
		for k, v := range r.alias {
			r2.alias[k] = v
		}
	}

	r2.dirStack = append(r2.dirBootstrap[:0], r.dirStack...)
	r2.fillExpandConfig(r.ectx)
	r2.didReset = true
	return r2
}
