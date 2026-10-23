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
// The fragment of the Makefile runs fmt-rust, lint-rust, test-rust, msrv-rust and audit-rust in the
// workspace of Cargo.toml, and check-rust, which requires the targets of the steps that the key
// check of the section names. lint-rust runs rustfmt, clippy with every pedantic lint and the lints
// of the documentation as errors, and rustdoc. clippy does not read lint levels from clippy.toml,
// so lint-rust sets the levels on the command line. test-rust runs cargo test with the arguments of
// the section, and then the doc tests of the workspace when cargo metadata lists a library crate.
// cargo test --all-targets leaves the doc tests out, and cargo test --doc fails for a workspace
// without a library. msrv-rust runs cargo-hack when cargo metadata lists a crate with a
// rust-version. cargo-hack installs the Rust of the rust-version of each published crate with
// rustup and checks the crate on it. It fails for a published crate without a rust-version.
// audit-rust and msrv-rust run their tools through ergon tool run, which installs a crate with
// cargo install --locked. CARGO_LLVM_COV runs the cargo-llvm-cov of the section for a target of
// .ergon/local/Makefile, because the gate does not measure coverage. With command lines in the key
// generate, the fragment also renders generate-rust, which runs them in order, and
// verify-generate-rust, which fails when they change a file. A line ##@ Rust starts the group of
// Rust in make help.
//
// # Options
//
// [Options] is the section rust: the tools, the steps of the gate, the options of test-rust, of the
// generators and of audit-rust, and the key ci of the job check-rust. [Options.Contribution]
// returns these parts of the workflows:
//
//   - the job check-rust
//   - the setup of Rust in the jobs of a release
//   - the CodeQL analysis of rust
//   - the updates of Cargo.lock
//   - the uploads to crates.io
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
