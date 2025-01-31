package qf1006

import (
	"go/ast"
	"go/token"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/go/ast/astutil"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "QF1006",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title: `Lift \'if\'+\'break\' into loop condition`,
		Before: `
for {
    if done {
        break
    }
    ...
}`,

		After: `
for !done {
    ...
}`,
		Since:    "2021.1",
		Severity: lint.SeverityHint,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var checkForLoopIfBreak = pattern.MustParse(`(ForStmt nil nil nil if@(IfStmt nil cond (BranchStmt "BREAK" nil) nil):_)`)

func run(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := code.Match(pass, checkForLoopIfBreak, node)
		if !ok {
			return
		}

		pos := node.Pos() + token.Pos(len("for"))
		r := astutil.NegateDeMorgan(m.State["cond"].(ast.Expr), false)

		// FIXME(dh): we're leaving behind an empty line when we
		// delete the old if statement. However, we can't just delete
		// an additional character, in case there closing curly brace
		// is followed by a comment, or Windows newlines.
		report.Report(pass, m.State["if"].(ast.Node), "could lift into loop condition",
			report.Fixes(edit.Fix("Lift into loop condition",
				edit.ReplaceWithString(edit.Range{pos, pos}, " "+report.Render(pass, r)),
				edit.Delete(m.State["if"].(ast.Node)))))
	}
	code.Preorder(pass, fn, (*ast.ForStmt)(nil))
	return nil, nil
}
