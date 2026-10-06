// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline renders what Rust contributes to a repository that ergon init sets up: the
// configuration of clippy, [Clippy], and its fragments of the shared files.
//
// The templates under templates/ mirror the paths of the shared files, and end in .tmpl.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language]. The root package of the module imports it.
package baseline
