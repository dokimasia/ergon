// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package workspace discovers the crates of a Cargo workspace as the packages of a release.
//
// [Discover] reads the Cargo.toml at the root of a repository and the Cargo.toml of each member of
// its workspace, as Cargo finds the members: the crate at the root, the directories of
// workspace.members less those of workspace.exclude, and the crates of the paths of their
// dependencies inside the root. Each crate is a package of [Toolchain], with these requirements on
// the other crates:
//
//   - runtime requirements of dependencies and build-dependencies
//   - optional requirements of the dependencies with optional = true
//   - dev requirements of dev-dependencies
//
// # Versions
//
// A package has the version of package.version of its crate. A crate with version.workspace = true
// has workspace.package.version of the root, and the crates that share it have [VersionSource] as
// their Source, so a release gives them one version.
//
// # Manifests
//
// [ParseManifest] reads the keys of a Cargo.toml that a release reads and writes, with the range of
// bytes of each version and each version requirement in the file, so a release rewrites them in
// place and keeps every other byte. [Requirements] resolves the dependencies of a crate on the
// crates of a path, those that it inherits from workspace.dependencies included, to the file and
// the range of each version requirement.
//
// # Errors
//
// The errors wrap [ErrManifest] for a Cargo.toml that a release cannot read.
//
// # Dependency position
//
// Imports the standard library, github.com/bmatcuk/doublestar/v4 for the globs of the members of a
// workspace, github.com/pelletier/go-toml/v2 for the ranges of the values of a Cargo.toml,
// [go.dokimi.dev/ergon/core/version] and [go.dokimi.dev/ergon/core/workspace]. The package
// go.dokimi.dev/ergon/lang/rust and its package release import it.
package workspace
