// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package version parses, orders and bumps the versions of Semantic Versioning 2.0.0 that a
// release plans, writes into manifests and changelogs, and tags.
//
// A [Version] is MAJOR.MINOR.PATCH with an optional pre-release and optional build metadata.
// [Parse] reads one without a leading v, as a manifest and a changelog state it. A Go tag adds the
// v itself. [Version.Compare] orders two versions by the precedence of the specification, and
// [Version.Bump] applies a [Bump], the level that a changeset names, by the rules of the inc
// function of node-semver, which changesets uses.
//
// # Bounds
//
// Each numeric component is at most [MaxComponent], 2^53 - 1, the bound of node-semver, so a
// version that ergon writes is one that changesets and npm read. A bump past the bound returns an
// error.
//
// # Errors
//
// [Parse] and [ParseBump] return an error that wraps [ErrInvalid], and [Version.Bump] one that
// wraps [ErrOverflow]. The text of each error names the value.
//
// # Dependency position
//
// Position 0 of ergon-core. Imports the standard library.
package version
