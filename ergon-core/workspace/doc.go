// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package workspace names the toolchains and the languages of ergon.
//
// A [Toolchain] builds packages, such as the go command or Gradle. A [Language] is the language
// of source files, such as Go or Kotlin. Both are registered names: the catalog of
// [go.dokimi.dev/ergon/core/language] accepts each name once.
//
// # Names
//
// A valid name is a lowercase ASCII letter followed by lowercase ASCII letters and digits, such
// as "go", "csharp" or "jvm". Configuration and reports spell a toolchain or a language by its
// name, so a rename is a breaking change.
//
// # Dependency position
//
// Position 0 of ergon-core. The package imports no package.
package workspace
