// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package release implements the release roles of the toolchain rust for the crates of a Cargo
// workspace.
//
// [Versioner] reads the version requirements of Cargo by the rules of the semver crate 1.0.28, the
// parser and the matcher of Cargo, and moves them to a new version as cargo-release does. It
// writes the versions and the requirements of a release into the Cargo.toml files in place, and
// refreshes Cargo.lock with cargo update. [Packer] packages and verifies the crates of a release
// with cargo package, in the job pack of the release workflow, which has no token of a registry.
// [Publisher] reads the versions of a crate from the sparse index of crates.io, and uploads the
// crates with cargo publish without a second verification.
//
// # Errors
//
// The errors wrap [ErrRequirement] for a version requirement, [ErrEdit] for an edit of a manifest
// that changed since its discovery, and [ErrCrate] for a crate without its .crate file.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/version], [go.dokimi.dev/ergon/core/workspace] and
// [go.dokimi.dev/ergon/lang/rust/workspace]. The package go.dokimi.dev/ergon/lang/rust imports it.
package release
