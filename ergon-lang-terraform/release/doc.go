// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package release releases a repository of Terraform for ergon release.
//
// # Roles
//
// [Versioner] is the versioner of the toolchain terraform. A package of Terraform has no manifest
// with a version and no requirement on another package of the repository, so a release changes no
// file of the package but the CHANGELOG.md that the command writes. [Tagger] names the tag of a
// release v<version>. Terraform reads a module from git at the tag in the ref of its source, and the
// Terraform Registry publishes a version of a module for each such tag. The toolchain has neither a
// packer nor a publisher: a publish creates the tag and its release of GitHub.
//
// # Errors
//
// [Versioner.Resolve] and [Versioner.Rewrite] return an error that wraps [ErrRequirement] for every
// requirement.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/version] and [go.dokimi.dev/ergon/core/workspace]. The root package of
// the module imports it.
package release
