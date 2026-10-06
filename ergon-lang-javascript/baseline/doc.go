// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline renders what JavaScript and the js toolchain contribute to a repository that
// ergon init sets up: their fragments of the shared files.
//
// [Initializer] renders the fragments of JavaScript, among them its job check-javascript of the
// workflow of the gate. [Toolchain] renders the files that JavaScript and TypeScript share, once
// for a repository with either or both: the configuration of Biome, [Biome], the targets fmt-js,
// lint-js and audit-js of the Makefile, the CodeQL analysis of javascript-typescript and the npm
// updates of Dependabot.
//
// The templates under templates/ mirror the paths of the shared files, and end in .tmpl. The
// templates of the toolchain are under templates/toolchain/.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language]. The root package of the module imports it.
package baseline
