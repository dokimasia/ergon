// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package rust declares Rust and its toolchain, Cargo.
//
// [Register] adds the toolchain and the language to a catalog. The toolchain discovers the crates
// of a Cargo workspace with its package workspace, and releases them to crates.io with the roles of
// its package release, which run cargo through the [Tools] of a composition root.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/workspace], and its packages baseline, release and workspace. Only the
// root module of ergon imports it.
package rust
