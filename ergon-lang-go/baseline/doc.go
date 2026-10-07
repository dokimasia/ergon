// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Go of ergon init: the configuration of golangci-lint, the
// fragments of Go of the shared files, the section go of .ergon.yaml, and the part of Go of the
// workflows.
//
// [Producer] renders .golangci.yml as a managed file from templates/managed/, and its fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile from templates/shared/.
//
// # Makefile
//
// The fragment of the Makefile runs each target in every module of go.work: fmt-go, lint-go,
// test-go, race-go, fuzz-go, bench-go, benchstat-go, mutate-go, generate-go and audit-go, and
// check-go, which requires the targets of the steps that the key check of the section names. Every
// tool runs through ergon tool run, which installs the version that the section names. The fragment
// states each option of a step as a variable, such as GO_FUZZ_TIME, which one run of make overrides
// on its command line. lint-go also runs ergon-go-vet and go mod tidy -diff, and generate-go fails
// when go generate changes a file of the repository.
//
// # Options
//
// [Options] is the section go: the tools, the package patterns, the steps of the gate, the options
// of each step, and the key ci of the job check-go. [Options.Contribution] returns the job check-go,
// the CodeQL analysis of go and the updates of the modules.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
