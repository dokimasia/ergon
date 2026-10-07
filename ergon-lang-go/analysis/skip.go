// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// testSuffix ends the name of a test file, and the import path of the external test package of a
// package.
const testSuffix = "_test"

// Skip reports whether an analyzer skips the package of an import path, such as a package that the
// key lint.exclude of the section go names.
type Skip func(path string) bool

// scope returns the files of the package of pass that an analyzer reads: none for a package that
// skip reports, and otherwise every file that a tool did not generate, as [ast.IsGenerated]
// reports it. It asks skip for the import path of the package, without the suffix _test of an
// external test package, so a skipped package skips its tests too.
func scope(pass *analysis.Pass, skip Skip) []*ast.File {
	if skip(strings.TrimSuffix(pass.Pkg.Path(), testSuffix)) {
		return nil
	}
	files := make([]*ast.File, 0, len(pass.Files))
	for _, f := range pass.Files {
		if !ast.IsGenerated(f) {
			files = append(files, f)
		}
	}
	return files
}

// isTest reports whether f is a test file of the package of pass: a file whose name ends in
// _test.go.
func isTest(pass *analysis.Pass, f *ast.File) bool {
	return strings.HasSuffix(pass.Fset.File(f.Pos()).Name(), testSuffix+".go")
}
