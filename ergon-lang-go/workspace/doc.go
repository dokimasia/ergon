// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package workspace discovers the Go modules of a repository as the packages of the toolchain
// go.
//
// # Modules
//
// [Modules] returns the modules of a repository: each directory that go.work uses, in the order
// of go.work, or the module at the root of a repository without go.work. [Discoverer.Discover]
// returns each module as a package of [Toolchain] with its module path, its directory, its
// require lines on the other modules of the repository, and its version.
//
// # Versions
//
// go.mod has no version field. The version of a module is the higher of the highest version that a
// heading of its CHANGELOG.md names and the highest version that a tag of the module names. The go
// command reads the tag v1.2.3 for the module at the root of a repository, and the tag
// <dir>/v1.2.3 for a module in the directory <dir>. A module with neither has the zero version,
// the version of a module that has never been released.
//
// # Errors
//
// Each error names the file that caused it: a go.work or a go.mod that the go command refuses, a
// directory of go.work without a go.mod, and the error of reading a file or the tags.
//
// # Dependency position
//
// Imports the standard library, golang.org/x/mod/modfile, [go.dokimi.dev/ergon/core/version] and
// [go.dokimi.dev/ergon/core/workspace].
package workspace
