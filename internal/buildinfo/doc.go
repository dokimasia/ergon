// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package buildinfo reads the version of the running ergon from the build information that the go
// command writes into it.
//
// A build at a tag records the version of the main module, such as v0.6.0 at the tag v0.6.0, and go
// install of a version records that version. A build of a git checkout also records the commit,
// its time, and whether the working tree had changes. The binary needs no -X flag of the linker for
// any of it.
//
// [Read] returns the [Info] of the running binary, and [From] the Info of any build information.
// [Info.String] writes the version string of the flag --version, such as 0.6.0 (a5eb82a,
// 2026-10-08).
//
// # Releases
//
// The version of an Info is the version of a release without its v, or [Development] for a build
// that no release tags: a build without build information, a build of the go command that records
// (devel), a build at a commit after the last tag, which records a pseudo-version, and a build of a
// working tree with changes.
//
// # Dependency position
//
// Imports the standard library, golang.org/x/mod/module and golang.org/x/mod/semver. cmd/ergon
// imports it.
package buildinfo
