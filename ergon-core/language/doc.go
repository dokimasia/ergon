// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package language declares the toolchains and the languages of ergon, and the catalog that
// registers them.
//
// A [Toolchain] builds packages, such as the go command or Gradle. A [Declaration] states a
// language and the toolchain that builds it: Java and Kotlin name the toolchain jvm, and Go names
// the toolchain go. A composition root registers each toolchain with [RegisterToolchain] and each
// language with [Register] into one [Catalog], before any command reads the catalog.
//
// # Roles
//
// A language registers its roles with Register, and a command selects one with [Role].
// [Initializer] is the role of ergon init. It renders the files of a language and the language's
// fragments of the shared files, such as [GitIgnore] and [CI].
//
// # Errors
//
// RegisterToolchain and Register leave the catalog unchanged when they return an error. Each
// error wraps one of [ErrInvalidName], [ErrRegistered] and [ErrUnknownToolchain], and quotes the
// name that caused it.
//
// # Concurrency
//
// Registration writes the catalog and is not safe for concurrent use. The methods of a Catalog
// only read it, so they are safe for concurrent use once the last registration has returned.
//
// # Dependency position
//
// Position 1 of ergon-core. Imports the standard library and
// [go.dokimi.dev/ergon/core/workspace].
package language
