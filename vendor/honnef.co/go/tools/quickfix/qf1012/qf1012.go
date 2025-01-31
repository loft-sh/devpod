package qf1012

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/knowledge"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "QF1012",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title:    `Use \'fmt.Fprintf(x, ...)\' instead of \'x.Write(fmt.Sprintf(...))\'`,
		Since:    "2022.1",
		Severity: lint.SeverityHint,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var (
	checkWriteBytesSprintfQ = pattern.MustParse(`
	(CallExpr
		(SelectorExpr recv (Ident "Write"))
		(CallExpr (ArrayType nil (Ident "byte"))
			(CallExpr
				fn@(Or
					(Symbol "fmt.Sprint")
					(Symbol "fmt.Sprintf")
					(Symbol "fmt.Sprintln"))
				args)
	))`)

	checkWriteStringSprintfQ = pattern.MustParse(`
	(CallExpr
		(SelectorExpr recv (Ident "WriteString"))
		(CallExpr
			fn@(Or
				(Symbol "fmt.Sprint")
				(Symbol "fmt.Sprintf")
				(Symbol "fmt.Sprintln"))
			args))`)
)

func run(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if m, ok := code.Match(pass, checkWriteBytesSprintfQ, node); ok {
			recv := m.State["recv"].(ast.Expr)
			recvT := pass.TypesInfo.TypeOf(recv)
			if !types.Implements(recvT, knowledge.Interfaces["io.Writer"]) {
				return
			}

			name := m.State["fn"].(*types.Func).Name()
			newName := "F" + strings.TrimPrefix(name, "S")
			msg := fmt.Sprintf("Use fmt.%s(...) instead of Write([]byte(fmt.%s(...)))", newName, name)

			args := m.State["args"].([]ast.Expr)
			fix := edit.Fix(msg, edit.ReplaceWithNode(pass.Fset, node, &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("fmt"),
					Sel: ast.NewIdent(newName),
				},
				Args: append([]ast.Expr{recv}, args...),
			}))
			report.Report(pass, node, msg, report.Fixes(fix))
		} else if m, ok := code.Match(pass, checkWriteStringSprintfQ, node); ok {
			recv := m.State["recv"].(ast.Expr)
			recvT := pass.TypesInfo.TypeOf(recv)
			if !types.Implements(recvT, knowledge.Interfaces["io.StringWriter"]) {
				return
			}
			// The type needs to implement both StringWriter and Writer.
			// If it doesn't implement Writer, then we cannot pass it to fmt.Fprint.
			if !types.Implements(recvT, knowledge.Interfaces["io.Writer"]) {
				return
			}

			name := m.State["fn"].(*types.Func).Name()
			newName := "F" + strings.TrimPrefix(name, "S")
			msg := fmt.Sprintf("Use fmt.%s(...) instead of WriteString(fmt.%s(...))", newName, name)

			args := m.State["args"].([]ast.Expr)
			fix := edit.Fix(msg, edit.ReplaceWithNode(pass.Fset, node, &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("fmt"),
					Sel: ast.NewIdent(newName),
				},
				Args: append([]ast.Expr{recv}, args...),
			}))
			report.Report(pass, node, msg, report.Fixes(fix))
		}
	}
	code.Preorder(pass, fn, (*ast.CallExpr)(nil))
	return nil, nil
}
