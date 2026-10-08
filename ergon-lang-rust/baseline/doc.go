// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Rust of ergon init: the configuration of clippy, the
// fragments of Rust of the shared files, the section rust of .ergon.yaml, and the part of Rust of
// the workflows.
//
// [Producer] renders clippy.toml as a managed file from templates/managed/, and its fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile from templates/shared/.
//
// # Makefile
//
// The fragment of the Makefile runs fmt-rust, lint-rust, test-rust and audit-rust in the workspace
// of Cargo.toml, and check-rust, which requires the targets of the steps that the key check of the
// section names. lint-rust runs rustfmt, clippy with every pedantic lint and the lints of the
// documentation as errors, and rustdoc. clippy reads no lint level from clippy.toml, so lint-rust
// sets the levels on the command line. audit-rust runs cargo-audit through ergon tool run, which
// installs the crate with cargo install --locked. With a command in the key generate, the fragment
// also renders generate-rust, which runs it, and verify-generate-rust, which fails when it changes
// a file. A line ##@ Rust starts the group of Rust in make help.
//
// # Options
//
// [Options] is the section rust: the tools, the steps of the gate, the options of test-rust, of the
// generators and of audit-rust, and the key ci of the job check-rust. [Options.Contribution]
// returns the job check-rust, the CodeQL analysis of rust and the updates of Cargo.lock.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
