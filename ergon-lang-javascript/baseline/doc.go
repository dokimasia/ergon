// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of JavaScript of ergon init, and the producer of the js
// toolchain, which renders what JavaScript shares with TypeScript.
//
// [Producer] renders the fragments of JavaScript of .editorconfig and the Makefile from
// templates/javascript/. [Toolchain] renders biome.json as a managed file, and its fragments of
// .gitattributes, .gitignore and the Makefile, from templates/js/, once for a repository with
// JavaScript, TypeScript or both. biome.json is JSON, so it opens with no managed comment.
//
// # Makefile
//
// The fragment of the js toolchain runs fmt-js and lint-js with Biome through ergon tool run, and
// audit-js with npm audit. The fragment of JavaScript runs test-javascript with npm test, and
// check-javascript, which requires the targets of the steps that the key check of the section
// names. lint-javascript and audit-javascript require lint-js and audit-js.
//
// # Options
//
// [ToolchainOptions] is the section js: the version of Biome, the paths that it checks, the lowest
// severity that fails npm audit, and the key ci of the setup of Node.js.
// [ToolchainOptions.Contribution] returns that setup, the CodeQL analysis of javascript-typescript
// and the updates of the npm packages. [Options] is the section javascript: the steps of the gate
// and the options of test-javascript. [Options.Contribution] returns the job check-javascript,
// which runs the setup of the js toolchain.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
