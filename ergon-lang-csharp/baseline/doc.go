// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline renders what C# contributes to a repository that ergon init sets up: the
// analyzer configuration, [GlobalConfig], and its fragments of the shared files.
//
// The templates under templates/ mirror the paths of the shared files, and end in .tmpl.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language]. The root package of the module imports it.
package baseline
