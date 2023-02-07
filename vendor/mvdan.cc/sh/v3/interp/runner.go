// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/pattern"
	"mvdan.cc/sh/v3/syntax"
)

func (r *Runner) fillExpandConfig(ctx context.Context) {
	r.ectx = ctx
	r.ecfg = &expand.Config{
		Env: expandEnv{r},
		CmdSubst: func(w io.Writer, cs *syntax.CmdSubst) error {
			switch len(cs.Stmts) {
			case 0: // nothing to do
				return nil
			case 1: // $(<file)
				word := catShortcutArg(cs.Stmts[0])
				if word == nil {
					break
				}
				path := r.literal(word)
				f, err := r.open(ctx, path, os.O_RDONLY, 0, true)
				if err != nil {
					return err
				}
				_, err = io.Copy(w, f)
				f.Close()
				return err
			}
			r2 := r.Subshell()
			r2.stdout = w
			r2.stmts(ctx, cs.Stmts)
			r.lastExpandExit = r2.exit
			return r2.err
		},
		ProcSubst: func(ps *syntax.ProcSubst) (string, error) {
			if runtime.GOOS == "windows" {
				return "", fmt.Errorf("TODO: support process substitution on Windows")
			}
			if len(ps.Stmts) == 0 { // nothing to do
				return os.DevNull, nil
			}

			if r.rand == nil {
				r.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
			}
			dir := os.TempDir()

			// We can't atomically create a random unused temporary FIFO.
			// Similar to os.CreateTemp,
			// keep trying new random paths until one does not exist.
			// We use a uint64 because a uint32 easily runs into retries.
			var path string
			try := 0
			for {
				path = fmt.Sprintf("%s/sh-interp-%x", dir, r.rand.Uint64())
				err := mkfifo(path, 0o666)
				if err == nil {
					break
				}
				if !os.IsExist(err) {
					return "", fmt.Errorf("cannot create fifo: %v", err)
				}
				if try++; try > 100 {
					return "", fmt.Errorf("giving up at creating fifo: %v", err)
				}
			}

			r2 := r.Subshell()
			stdout := r.origStdout
			r.wgProcSubsts.Add(1)
			go func() {
				defer r.wgProcSubsts.Done()
				switch ps.Op {
				case syntax.CmdIn:
					f, err := os.OpenFile(path, os.O_WRONLY, 0)
					if err != nil {
						r.errf("cannot open fifo for stdout: %v", err)
						return
					}
					r2.stdout = f
					defer func() {
						if err := f.Close(); err != nil {
							r.errf("closing stdout fifo: %v", err)
						}
						os.Remove(path)
					}()
				default: // syntax.CmdOut
					f, err := os.OpenFile(path, os.O_RDONLY, 0)
					if err != nil {
						r.errf("cannot open fifo for stdin: %v", err)
						return
					}
					r2.stdin = f
					r2.stdout = stdout

					defer func() {
						f.Close()
						os.Remove(path)
					}()
				}
				r2.stmts(ctx, ps.Stmts)
			}()
			return path, nil
		},
	}
	r.updateExpandOpts()
}

// catShortcutArg checks if a statement is of the form "$(<file)". The redirect
// word is returned if there's a match, and nil otherwise.
func catShortcutArg(stmt *syntax.Stmt) *syntax.Word {
	if stmt.Cmd != nil || stmt.Negated || stmt.Background || stmt.Coprocess {
		return nil
	}
	if len(stmt.Redirs) != 1 {
		return nil
	}
	redir := stmt.Redirs[0]
	if redir.Op != syntax.RdrIn {
		return nil
	}
	return redir.Word
}

func (r *Runner) updateExpandOpts() {
	if r.opts[optNoGlob] {
		r.ecfg.ReadDir = nil
	} else {
		r.ecfg.ReadDir = func(s string) ([]os.FileInfo, error) {
			return r.readDirHandler(r.handlerCtx(context.Background()), s)
		}
	}
	r.ecfg.GlobStar = r.opts[optGlobStar]
	r.ecfg.NullGlob = r.opts[optNullGlob]
	r.ecfg.NoUnset = r.opts[optNoUnset]
}

func (r *Runner) expandErr(err error) {
	if err != nil {
		r.errf("%v\n", err)
		r.exitShell(context.TODO(), 1)
	}
}

func (r *Runner) arithm(expr syntax.ArithmExpr) int {
	n, err := expand.Arithm(r.ecfg, expr)
	r.expandErr(err)
	return n
}

func (r *Runner) fields(words ...*syntax.Word) []string {
	strs, err := expand.Fields(r.ecfg, words...)
	r.expandErr(err)
	return strs
}

func (r *Runner) literal(word *syntax.Word) string {
	str, err := expand.Literal(r.ecfg, word)
	r.expandErr(err)
	return str
}

func (r *Runner) document(word *syntax.Word) string {
	str, err := expand.Document(r.ecfg, word)
	r.expandErr(err)
	return str
}

func (r *Runner) pattern(word *syntax.Word) string {
	str, err := expand.Pattern(r.ecfg, word)
	r.expandErr(err)
	return str
}

// expandEnviron exposes Runner's variables to the expand package.
type expandEnv struct {
	r *Runner
}

var _ expand.WriteEnviron = expandEnv{}

func (e expandEnv) Get(name string) expand.Variable {
	return e.r.lookupVar(name)
}

func (e expandEnv) Set(name string, vr expand.Variable) error {
	e.r.setVarInternal(name, vr)
	return nil // TODO: return any errors
}

func (e expandEnv) Each(fn func(name string, vr expand.Variable) bool) {
	e.r.writeEnv.Each(fn)
}

func (r *Runner) handlerCtx(ctx context.Context) context.Context {
	hc := HandlerContext{
		Env:    &overlayEnviron{parent: r.writeEnv},
		Dir:    r.Dir,
		Stdin:  r.stdin,
		Stdout: r.stdout,
		Stderr: r.stderr,
	}
	return context.WithValue(ctx, handlerCtxKey{}, hc)
}

func (r *Runner) setErr(err error) {
	if r.err == nil {
		r.err = err
	}
}

func (r *Runner) out(s string) {
	io.WriteString(r.stdout, s)
}

func (r *Runner) outf(format string, a ...interface{}) {
	fmt.Fprintf(r.stdout, format, a...)
}

func (r *Runner) errf(format string, a ...interface{}) {
	fmt.Fprintf(r.stderr, format, a...)
}

func (r *Runner) stop(ctx context.Context) bool {
	if r.err != nil || r.Exited() {
		return true
	}
	if err := ctx.Err(); err != nil {
		r.err = err
		return true
	}
	if r.opts[optNoExec] {
		return true
	}
	return false
}

func (r *Runner) stmt(ctx context.Context, st *syntax.Stmt) {
	if r.stop(ctx) {
		return
	}
	r.exit = 0
	if st.Background {
		r2 := r.Subshell()
		st2 := *st
		st2.Background = false
		r.bgShells.Go(func() error {
			return r2.Run(ctx, &st2)
		})
	} else {
		r.stmtSync(ctx, st)
	}
	r.lastExit = r.exit
}

func (r *Runner) stmtSync(ctx context.Context, st *syntax.Stmt) {
	defer r.wgProcSubsts.Wait()
	oldIn, oldOut, oldErr := r.stdin, r.stdout, r.stderr
	for _, rd := range st.Redirs {
		cls, err := r.redir(ctx, rd)
		if err != nil {
			r.exit = 1
			break
		}
		if cls != nil {
			defer cls.Close()
		}
	}
	if r.exit == 0 && st.Cmd != nil {
		r.cmd(ctx, st.Cmd)
	}
	if st.Negated {
		r.exit = oneIf(r.exit == 0)
	} else if _, ok := st.Cmd.(*syntax.CallExpr); !ok {
	} else if r.exit != 0 && !r.noErrExit && r.opts[optErrExit] {
		// If the "errexit" option is set and a simple command failed,
		// exit the shell. Exceptions:
		//
		//   conditions (if <cond>, while <cond>, etc)
		//   part of && or || lists
		//   preceded by !
		r.exitShell(ctx, r.exit)
	} else if r.exit != 0 {
		r.trapCallback(ctx, r.callbackErr, "error")
	}
	if !r.keepRedirs {
		r.stdin, r.stdout, r.stderr = oldIn, oldOut, oldErr
	}
}

func (r *Runner) cmd(ctx context.Context, cm syntax.Command) {
	if r.stop(ctx) {
		return
	}

	tracingEnabled := r.opts[optXTrace]
	trace := r.tracer()

	switch x := cm.(type) {
	case *syntax.Block:
		r.stmts(ctx, x.Stmts)
	case *syntax.Subshell:
		r2 := r.Subshell()
		r2.stmts(ctx, x.Stmts)
		r.exit = r2.exit
		r.setErr(r2.err)
	case *syntax.CallExpr:
		// Use a new slice, to not modify the slice in the alias map.
		var args []*syntax.Word
		left := x.Args
		for len(left) > 0 && r.opts[optExpandAliases] {
			als, ok := r.alias[left[0].Lit()]
			if !ok {
				break
			}
			args = append(args, als.args...)
			left = left[1:]
			if !als.blank {
				break
			}
		}
		args = append(args, left...)
		r.lastExpandExit = 0
		fields := r.fields(args...)
		if len(fields) == 0 {
			for _, as := range x.Assigns {
				vr := r.assignVal(as, "")
				r.setVar(as.Name.Value, as.Index, vr)

				if !tracingEnabled {
					continue
				}

				// Strangely enough, it seems like Bash prints original
				// source for arrays, but the expanded value otherwise.
				// TODO: add test cases for x[i]=y and x+=y.
				if as.Array != nil {
					trace.expr(as)
				} else if as.Value != nil {
					val, err := syntax.Quote(vr.String(), syntax.LangBash)
					if err != nil { // should never happen
						panic(err)
					}
					trace.stringf("%s=%s", as.Name.Value, val)
				}
				trace.newLineFlush()
			}
			// If interpreting the last expansion like $(foo) failed,
			// and the expansion and assignments otherwise succeeded,
			// we need to surface that last exit code.
			if r.exit == 0 {
				r.exit = r.lastExpandExit
			}
			break
		}

		type restoreVar struct {
			name string
			vr   expand.Variable
		}
		var restores []restoreVar

		for _, as := range x.Assigns {
			name := as.Name.Value
			origVr := r.lookupVar(name)

			vr := r.assignVal(as, "")
			// Inline command vars are always exported.
			vr.Exported = true

			restores = append(restores, restoreVar{name, origVr})

			r.setVarInternal(name, vr)
		}

		trace.call(fields[0], fields[1:]...)
		trace.newLineFlush()

		r.call(ctx, x.Args[0].Pos(), fields)
		for _, restore := range restores {
			r.setVarInternal(restore.name, restore.vr)
		}
	case *syntax.BinaryCmd:
		switch x.Op {
		case syntax.AndStmt, syntax.OrStmt:
			oldNoErrExit := r.noErrExit
			r.noErrExit = true
			r.stmt(ctx, x.X)
			r.noErrExit = oldNoErrExit
			if (r.exit == 0) == (x.Op == syntax.AndStmt) {
				r.stmt(ctx, x.Y)
			}
		case syntax.Pipe, syntax.PipeAll:
			pr, pw, err := os.Pipe()
			if err != nil {
				r.setErr(err)
				return
			}
			r2 := r.Subshell()
			r2.stdout = pw
			if x.Op == syntax.PipeAll {
				r2.stderr = pw
			} else {
				r2.stderr = r.stderr
			}
			r.stdin = pr
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				r2.stmt(ctx, x.X)
				pw.Close()
				wg.Done()
			}()
			r.stmt(ctx, x.Y)
			pr.Close()
			wg.Wait()
			if r.opts[optPipeFail] && r2.exit != 0 && r.exit == 0 {
				r.exit = r2.exit
			}
			r.setErr(r2.err)
		}
	case *syntax.IfClause:
		oldNoErrExit := r.noErrExit
		r.noErrExit = true
		r.stmts(ctx, x.Cond)
		r.noErrExit = oldNoErrExit

		if r.exit == 0 {
			r.stmts(ctx, x.Then)
			break
		}
		r.exit = 0
		if x.Else != nil {
			r.cmd(ctx, x.Else)
		}
	case *syntax.WhileClause:
		for !r.stop(ctx) {
			oldNoErrExit := r.noErrExit
			r.noErrExit = true
			r.stmts(ctx, x.Cond)
			r.noErrExit = oldNoErrExit

			stop := (r.exit == 0) == x.Until
			r.exit = 0
			if stop || r.loopStmtsBroken(ctx, x.Do) {
				break
			}
		}
	case *syntax.ForClause:
		switch y := x.Loop.(type) {
		case *syntax.WordIter:
			name := y.Name.Value
			items := r.Params // for i; do ...

			inToken := y.InPos.IsValid()
			if inToken {
				items = r.fields(y.Items...) // for i in ...; do ...
			}

			for _, field := range items {
				r.setVarString(name, field)
				trace.stringf("for %s in", y.Name.Value)
				if inToken {
					for _, item := range y.Items {
						trace.string(" ")
						trace.expr(item)
					}
				} else {
					trace.string(` "$@"`)
				}
				trace.newLineFlush()
				if r.loopStmtsBroken(ctx, x.Do) {
					break
				}
			}
		case *syntax.CStyleLoop:
			if y.Init != nil {
				r.arithm(y.Init)
			}
			for y.Cond == nil || r.arithm(y.Cond) != 0 {
				if r.exit != 0 || r.loopStmtsBroken(ctx, x.Do) {
					break
				}
				if y.Post != nil {
					r.arithm(y.Post)
				}
			}
		}
	case *syntax.FuncDecl:
		r.setFunc(x.Name.Value, x.Body)
	case *syntax.ArithmCmd:
		r.exit = oneIf(r.arithm(x.X) == 0)
	case *syntax.LetClause:
		var val int
		for _, expr := range x.Exprs {
			val = r.arithm(expr)

			if !tracingEnabled {
				continue
			}

			switch v := expr.(type) {
			case *syntax.Word:
				qs, err := syntax.Quote(r.literal(v), syntax.LangBash)
				if err != nil {
					return
				}
				trace.stringf("let %v", qs)
			case *syntax.BinaryArithm, *syntax.UnaryArithm:
				trace.expr(x)
			case *syntax.ParenArithm:
				// TODO
			}
		}

		trace.newLineFlush()
		r.exit = oneIf(val == 0)
	case *syntax.CaseClause:
		trace.string("case ")
		trace.expr(x.Word)
		trace.string(" in")
		trace.newLineFlush()
		str := r.literal(x.Word)
		for _, ci := range x.Items {
			for _, word := range ci.Patterns {
				pattern := r.pattern(word)
				if match(pattern, str) {
					r.stmts(ctx, ci.Stmts)
					return
				}
			}
		}
	case *syntax.TestClause:
		if r.bashTest(ctx, x.X, false) == "" && r.exit == 0 {
			// to preserve exit status code 2 for regex errors, etc
			r.exit = 1
		}
	case *syntax.DeclClause:
		local, global := false, false
		var modes []string
		valType := ""
		switch x.Variant.Value {
		case "declare":
			// When used in a function, "declare" acts as "local"
			// unless the "-g" option is used.
			local = r.inFunc
		case "local":
			if !r.inFunc {
				r.errf("local: can only be used in a function\n")
				r.exit = 1
				return
			}
			local = true
		case "export":
			modes = append(modes, "-x")
		case "readonly":
			modes = append(modes, "-r")
		case "nameref":
			valType = "-n"
		}
		for _, as := range x.Args {
			for _, as := range r.flattenAssign(as) {
				name := as.Name.Value
				if strings.HasPrefix(name, "-") {
					switch name {
					case "-x", "-r":
						modes = append(modes, name)
					case "-a", "-A", "-n":
						valType = name
					case "-g":
						global = true
					default:
						r.errf("declare: invalid option %q\n", name)
						r.exit = 2
						return
					}
					continue
				}
				if !syntax.ValidName(name) {
					r.errf("declare: invalid name %q\n", name)
					r.exit = 1
					return
				}
				var vr expand.Variable
				if !as.Naked {
					vr = r.assignVal(as, valType)
				}
				if global {
					vr.Local = false
				} else if local {
					vr.Local = true
				}
				for _, mode := range modes {
					switch mode {
					case "-x":
						vr.Exported = true
					case "-r":
						vr.ReadOnly = true
					}
				}
				if as.Naked {
					if vr.Exported || vr.Local || vr.ReadOnly {
						r.setVarInternal(name, vr)
					}
				} else {
					r.setVar(name, as.Index, vr)
				}
			}
		}
	case *syntax.TimeClause:
		start := time.Now()
		if x.Stmt != nil {
			r.stmt(ctx, x.Stmt)
		}
		format := "%s\t%s\n"
		if x.PosixFormat {
			format = "%s %s\n"
		} else {
			r.outf("\n")
		}
		real := time.Since(start)
		r.outf(format, "real", elapsedString(real, x.PosixFormat))
		// TODO: can we do these?
		r.outf(format, "user", elapsedString(0, x.PosixFormat))
		r.outf(format, "sys", elapsedString(0, x.PosixFormat))
	default:
		panic(fmt.Sprintf("unhandled command node: %T", x))
	}
}

func (r *Runner) trapCallback(ctx context.Context, callback, name string) {
	if callback == "" {
		return // nothing to do
	}
	if r.handlingTrap {
		return // don't recurse, as that could lead to cycles
	}
	r.handlingTrap = true

	p := syntax.NewParser()
	// TODO: do this parsing when "trap" is called?
	file, err := p.Parse(strings.NewReader(callback), name+" trap")
	if err != nil {
		r.errf(name+"trap: %v\n", err)
		// ignore errors in the callback
		return
	}
	r.stmts(ctx, file.Stmts)

	r.handlingTrap = false
}

// exitShell exits the current shell session with the given status code.
func (r *Runner) exitShell(ctx context.Context, status int) {
	if status != 0 {
		r.trapCallback(ctx, r.callbackErr, "error")
	}
	r.trapCallback(ctx, r.callbackExit, "exit")

	r.shellExited = true
	// Restore the original exit status. We ignore the callbacks.
	r.exit = status
}

func (r *Runner) flattenAssign(as *syntax.Assign) []*syntax.Assign {
	// Convert "declare $x" into "declare value".
	// Don't use syntax.Parser here, as we only want the basic
	// splitting by '='.
	if as.Name != nil {
		return []*syntax.Assign{as} // nothing to do
	}
	var asgns []*syntax.Assign
	for _, field := range r.fields(as.Value) {
		as := &syntax.Assign{}
		parts := strings.SplitN(field, "=", 2)
		as.Name = &syntax.Lit{Value: parts[0]}
		if len(parts) == 1 {
			as.Naked = true
		} else {
			as.Value = &syntax.Word{Parts: []syntax.WordPart{
				&syntax.Lit{Value: parts[1]},
			}}
		}
		asgns = append(asgns, as)
	}
	return asgns
}

func match(pat, name string) bool {
	expr, err := pattern.Regexp(pat, 0)
	if err != nil {
		return false
	}
	rx := regexp.MustCompile("(?m)^" + expr + "$")
	return rx.MatchString(name)
}

func elapsedString(d time.Duration, posix bool) string {
	if posix {
		return fmt.Sprintf("%.2f", d.Seconds())
	}
	min := int(d.Minutes())
	sec := math.Mod(d.Seconds(), 60.0)
	return fmt.Sprintf("%dm%.3fs", min, sec)
}

func (r *Runner) stmts(ctx context.Context, stmts []*syntax.Stmt) {
	for _, stmt := range stmts {
		r.stmt(ctx, stmt)
	}
}

func (r *Runner) hdocReader(rd *syntax.Redirect) io.Reader {
	if rd.Op != syntax.DashHdoc {
		hdoc := r.document(rd.Hdoc)
		return strings.NewReader(hdoc)
	}
	var buf bytes.Buffer
	var cur []syntax.WordPart
	flushLine := func() {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(r.document(&syntax.Word{Parts: cur}))
		cur = cur[:0]
	}
	for _, wp := range rd.Hdoc.Parts {
		lit, ok := wp.(*syntax.Lit)
		if !ok {
			cur = append(cur, wp)
			continue
		}
		for i, part := range strings.Split(lit.Value, "\n") {
			if i > 0 {
				flushLine()
				cur = cur[:0]
			}
			part = strings.TrimLeft(part, "\t")
			cur = append(cur, &syntax.Lit{Value: part})
		}
	}
	flushLine()
	return &buf
}

func (r *Runner) redir(ctx context.Context, rd *syntax.Redirect) (io.Closer, error) {
	if rd.Hdoc != nil {
		r.stdin = r.hdocReader(rd)
		return nil, nil
	}
	orig := &r.stdout
	if rd.N != nil {
		switch rd.N.Value {
		case "1":
		case "2":
			orig = &r.stderr
		}
	}
	arg := r.literal(rd.Word)
	switch rd.Op {
	case syntax.WordHdoc:
		r.stdin = strings.NewReader(arg + "\n")
		return nil, nil
	case syntax.DplOut:
		switch arg {
		case "1":
			*orig = r.stdout
		case "2":
			*orig = r.stderr
		}
		return nil, nil
	case syntax.RdrIn, syntax.RdrOut, syntax.AppOut,
		syntax.RdrAll, syntax.AppAll:
		// done further below
	// case syntax.DplIn:
	default:
		panic(fmt.Sprintf("unhandled redirect op: %v", rd.Op))
	}
	mode := os.O_RDONLY
	switch rd.Op {
	case syntax.AppOut, syntax.AppAll:
		mode = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case syntax.RdrOut, syntax.RdrAll:
		mode = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := r.open(ctx, arg, mode, 0o644, true)
	if err != nil {
		return nil, err
	}
	switch rd.Op {
	case syntax.RdrIn:
		r.stdin = f
	case syntax.RdrOut, syntax.AppOut:
		*orig = f
	case syntax.RdrAll, syntax.AppAll:
		r.stdout = f
		r.stderr = f
	default:
		panic(fmt.Sprintf("unhandled redirect op: %v", rd.Op))
	}
	return f, nil
}

func (r *Runner) loopStmtsBroken(ctx context.Context, stmts []*syntax.Stmt) bool {
	oldInLoop := r.inLoop
	r.inLoop = true
	defer func() { r.inLoop = oldInLoop }()
	for _, stmt := range stmts {
		r.stmt(ctx, stmt)
		if r.contnEnclosing > 0 {
			r.contnEnclosing--
			return r.contnEnclosing > 0
		}
		if r.breakEnclosing > 0 {
			r.breakEnclosing--
			return true
		}
	}
	return false
}

type returnStatus uint8

func (s returnStatus) Error() string { return fmt.Sprintf("return status %d", s) }

func (r *Runner) call(ctx context.Context, pos syntax.Pos, args []string) {
	if r.stop(ctx) {
		return
	}
	if r.callHandler != nil {
		var err error
		args, err = r.callHandler(r.handlerCtx(ctx), args)
		if err != nil {
			// handler's custom fatal error
			r.setErr(err)
			return
		}
	}
	name := args[0]
	if body := r.Funcs[name]; body != nil {
		// stack them to support nested func calls
		oldParams := r.Params
		r.Params = args[1:]
		oldInFunc := r.inFunc
		r.inFunc = true

		// Functions run in a nested scope.
		// Note that Runner.exec below does something similar.
		origEnv := r.writeEnv
		r.writeEnv = &overlayEnviron{parent: r.writeEnv, funcScope: true}

		r.stmt(ctx, body)

		r.writeEnv = origEnv

		r.Params = oldParams
		r.inFunc = oldInFunc
		if code, ok := r.err.(returnStatus); ok {
			r.err = nil
			r.exit = int(code)
		}
		return
	}
	if isBuiltin(name) {
		r.exit = r.builtinCode(ctx, pos, name, args[1:])
		return
	}
	r.exec(ctx, args)
}

func (r *Runner) exec(ctx context.Context, args []string) {
	err := r.execHandler(r.handlerCtx(ctx), args)
	if status, ok := IsExitStatus(err); ok {
		r.exit = int(status)
		return
	}
	if err != nil {
		// handler's custom fatal error
		r.setErr(err)
		return
	}
	r.exit = 0
}

func (r *Runner) open(ctx context.Context, path string, flags int, mode os.FileMode, print bool) (io.ReadWriteCloser, error) {
	f, err := r.openHandler(r.handlerCtx(ctx), path, flags, mode)
	// TODO: support wrapped PathError returned from openHandler.
	switch err.(type) {
	case nil:
	case *os.PathError:
		if print {
			r.errf("%v\n", err)
		}
	default: // handler's custom fatal error
		r.setErr(err)
	}
	return f, err
}

func (r *Runner) stat(ctx context.Context, name string) (os.FileInfo, error) {
	path := absPath(r.Dir, name)
	return r.statHandler(ctx, path, true)
}

func (r *Runner) lstat(ctx context.Context, name string) (os.FileInfo, error) {
	path := absPath(r.Dir, name)
	return r.statHandler(ctx, path, false)
}
