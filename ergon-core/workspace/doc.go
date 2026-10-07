// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package workspace names the toolchains and the languages of ergon, and describes the packages
// that a toolchain discovers.
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
// # Packages
//
// A [Package] is one unit that a toolchain releases, as its discovery returns it: its name, its
// directory, its version and its [Dependency] on each other package of the repository, with the
// [Kind] of the section that declares it.
//
// # Dependency position
//
// Position 0 of ergon-core. Imports [go.dokimi.dev/ergon/core/version].
package workspace
