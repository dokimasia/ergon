// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Command ergon-go-vet runs the analyzers errorprefix and skipexpiry of
// [go.dokimi.dev/ergon/lang/go/analysis] over the packages of its arguments and their tests, as go
// vet runs its analyzers. lint-go of the baseline of ergon init runs it in every module of Go, with
// a flag -exclude for each pattern of the key lint.exclude of the section go:
//
//	ergon-go-vet -exclude=./internal/legacy/... ./...
//
// The flag -exclude takes a package pattern of go list, such as ./internal/legacy/..., and the
// analyzers skip each package that it matches with its external test package. The command resolves
// the pattern with go list while it parses its flags.
//
// The command exits as go vet does: with 0 for packages without a finding, 3 when an analyzer
// reports a finding, 1 for an error, such as a package that does not compile, and 2 for a command
// line that it refuses, such as a pattern of -exclude that go list does not resolve.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/lang/go/analysis],
// golang.org/x/tools/go/analysis/multichecker and golang.org/x/tools/go/packages.
package main
