// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis

import (
	"go/ast"
	"go/constant"
	"regexp"
	"time"

	"golang.org/x/tools/go/analysis"
)

// The methods whose message skipexpiry checks: Skip and Skipf of a test, such as of testing.T.
const (
	skipMethod  = "Skip"
	skipfMethod = "Skipf"
)

// expires matches the clause of a message of a skip that states the day from which the skip is
// expired, as expires 2026-12-31, and captures the day.
var expires = regexp.MustCompile(`expires\s+(\d{4}-\d{2}-\d{2})`)

// SkipExpiry returns the analyzer skipexpiry. It reports each call of a method Skip or Skipf in a
// test file whose first argument is a constant text with the clause expires YYYY-MM-DD, when the
// day has passed: when it comes before the day of now in the location of now. A skip that expires
// today is not reported yet. The analyzer skips a generated file and every package that skip
// reports.
//
// A skip with an expiry states a fix that the repository deferred, and an expired one states a fix
// that the repository forgot.
func SkipExpiry(now func() time.Time, skip Skip) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "skipexpiry",
		Doc:  "report each skipped test whose message states an expiry that has passed",
		URL:  "https://pkg.go.dev/go.dokimi.dev/ergon/lang/go/analysis#SkipExpiry",
		Run: func(pass *analysis.Pass) (any, error) {
			today := now().Format(time.DateOnly)
			for _, f := range scope(pass, skip) {
				if !isTest(pass, f) {
					continue
				}
				ast.Inspect(f, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok || len(call.Args) == 0 {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || (sel.Sel.Name != skipMethod && sel.Sel.Name != skipfMethod) {
						return true
					}
					text := pass.TypesInfo.Types[call.Args[0]].Value
					if text == nil || text.Kind() != constant.String {
						return true
					}
					if day := expires.FindStringSubmatch(constant.StringVal(text)); len(day) > 1 && day[1] < today {
						pass.Reportf(call.Pos(), "the skip expired on %s, so fix the test and remove the skip", day[1])
					}
					return true
				})
			}
			return nil, nil
		},
	}
}
