// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"flag"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"go.dokimi.dev/ergon/lang/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/packages"
)

// excludeFlag is the flag that names a package pattern that the analyzers skip.
const excludeFlag = "exclude"

// excluded are the import paths of the packages that the patterns of the flag -exclude match.
type excluded map[string]bool

// String returns the import paths of e, sorted and separated by commas.
func (e excluded) String() string {
	return strings.Join(slices.Sorted(maps.Keys(e)), ",")
}

// Set adds the import path of each package that the package pattern of go list matches to e, such
// as ./internal/legacy/..., which it resolves with go list in the working directory. It returns the
// error of go list, and the error of a pattern that go list does not resolve, such as a directory
// that does not exist, which the flag package reports as an invalid value of -exclude.
func (e excluded) Set(pattern string) error {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName}, pattern)
	if err != nil {
		return fmt.Errorf("go list: %w", err)
	}
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return errors.New(pkg.Errors[0].Msg)
		}
		e[pkg.PkgPath] = true
	}
	return nil
}

// main runs the analyzers errorprefix and skipexpiry over the packages of the arguments, with the
// packages of each -exclude skipped, and exits as go vet does.
func main() {
	exclude := excluded{}
	flag.Var(exclude, excludeFlag,
		"a package `pattern` that the analyzers skip, such as ./internal/legacy/..., once for each pattern")
	skip := func(path string) bool { return exclude[path] }
	multichecker.Main(analysis.ErrorPrefix(skip), analysis.SkipExpiry(time.Now, skip))
}
