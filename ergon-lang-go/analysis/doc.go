// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package analysis is the analyzers of ergon-go-vet, which lint-go of the baseline of ergon init
// runs in every module of Go: [ErrorPrefix] and [SkipExpiry].
//
//   - errorprefix reports a text of errors.New that does not start with the name of its package.
//   - skipexpiry reports a skipped test whose message states an expiry that has passed.
//
// Each is an analyzer of golang.org/x/tools/go/analysis, so ergon-go-vet runs both as go vet runs
// its analyzers, over the packages of its arguments with their tests.
//
// # Scope
//
// Both analyzers skip a file that a tool generated, as [go/ast.IsGenerated] reports it, and every
// package that their [Skip] reports, with its external test package. ergon-go-vet builds the Skip
// from the patterns of the key lint.exclude of the section go. errorprefix reads the files other
// than the test files, and skipexpiry the test files.
//
// # Dependency position
//
// Imports the standard library, golang.org/x/tools/go/analysis and
// golang.org/x/tools/go/types/typeutil. cmd/ergon-go-vet imports it.
package analysis
