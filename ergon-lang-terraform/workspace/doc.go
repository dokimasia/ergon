// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package workspace discovers a repository of Terraform as one package of the toolchain terraform.
//
// # Package
//
// A Terraform module has no manifest with a name or a version, and a consumer calls a module of a
// repository from git with the tag of a version. [Discoverer.Discover] returns the repository as
// one package at its root when the lock of ergon init lists [Language] among the languages of the
// repository. The name of the package is the name of the repository in the answers of the lock.
//
// # Versions
//
// The version of the package is the higher of the highest version that a heading of its
// CHANGELOG.md names and the highest version that a tag v<version> of the repository names. A
// package with neither has the zero version, the version of a package that has never been released.
//
// # Errors
//
// Discover returns the error of reading the answers of the lock, of opening the repository, of
// reading CHANGELOG.md, with its file, and of reading the tags.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/changeset],
// [go.dokimi.dev/ergon/core/language], [go.dokimi.dev/ergon/core/version] and
// [go.dokimi.dev/ergon/core/workspace]. The root package of the module imports it.
package workspace
