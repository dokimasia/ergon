// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of TypeScript of ergon init: the fragments of TypeScript of the
// shared files, the section typescript of .ergon.yaml, and the job of TypeScript in the workflows.
//
// [Producer] renders its fragments of .editorconfig, .gitignore and the Makefile from
// templates/shared/. The js toolchain of the JavaScript module renders biome.json, the targets
// fmt-js, lint-js and audit-js, the CodeQL analysis of javascript-typescript and the npm updates,
// once for a repository with JavaScript, TypeScript or both.
//
// # Makefile
//
// The fragment of the Makefile runs lint-typescript, which requires lint-js and runs tsc through
// ergon tool run with the type-safety options of tsc, test-typescript with npm test, and
// check-typescript, which requires the targets of the steps that the key check of the section
// names. audit-typescript requires audit-js. With a command in the key generate, the fragment also
// renders generate-typescript, which runs it, and verify-generate-typescript, which fails when it
// changes a file. A line ##@ TypeScript starts the group of TypeScript in make help.
//
// # Options
//
// [Options] is the section typescript: the version of tsc, the steps of the gate, and the options
// of test-typescript and of the generators. [Options.Contribution] returns the job
// check-typescript, which runs the setup of the js toolchain.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option], [go.dokimi.dev/ergon/core/workflow], and the root package of
// the JavaScript module for the name of the js toolchain. The root package of the module imports
// it.
package baseline
