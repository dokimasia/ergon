// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// The function whose text errorprefix checks: New of the package errors of the standard library.
const (
	errorsPath = "errors"
	errorsNew  = "New"
)

// prefixMessage is the message of a finding of errorprefix: the text and the prefix that it lacks.
const prefixMessage = "the text %q of errors.New does not start with %q, the name of its package"

// ErrorPrefix returns the analyzer errorprefix. It reports each call of errors.New, under any name
// of the import of errors, in a file other than a test file, whose text is a constant that does not
// start with the name of its package and a colon, such as clock: instant is zero, or with the name
// and a dot, for a qualifier such as kernel.patch: malformed. A text that is no constant, such as
// a variable, is not reported. The analyzer skips a generated file and every package that skip
// reports.
//
// The text of an error that starts with the name of its package states where the error comes from
// when a caller far from that package reports it.
func ErrorPrefix(skip Skip) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "errorprefix",
		Doc:  "report each errors.New whose text does not start with the name of its package",
		URL:  "https://pkg.go.dev/go.dokimi.dev/ergon/lang/go/analysis#ErrorPrefix",
		Run: func(pass *analysis.Pass) (any, error) {
			name := pass.Pkg.Name()
			for _, f := range scope(pass, skip) {
				if isTest(pass, f) {
					continue
				}
				ast.Inspect(f, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok || len(call.Args) != 1 {
						return true
					}
					fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
					if !ok || fn.Pkg() == nil || fn.Pkg().Path() != errorsPath || fn.Name() != errorsNew {
						return true
					}
					text := pass.TypesInfo.Types[call.Args[0]].Value
					if text == nil || text.Kind() != constant.String {
						return true
					}
					s := constant.StringVal(text)
					if !strings.HasPrefix(s, name+":") && !strings.HasPrefix(s, name+".") {
						pass.Reportf(call.Pos(), prefixMessage, s, name+":")
					}
					return true
				})
			}
			return nil, nil
		},
	}
}
