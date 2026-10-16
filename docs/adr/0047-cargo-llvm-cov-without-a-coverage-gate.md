---
adr: 0047
title: rust.tools pins cargo-llvm-cov, and the gate of Rust does not measure coverage
status: Accepted
date: 2026-10-16
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0047: rust.tools pins cargo-llvm-cov, and the gate of Rust does not measure coverage

## Status

Accepted

## Context

The section `rust` of `.ergon.yaml` had one tool, `cargo-audit`. ergon refuses a key that a section does not have, so a repository could not pin another crate there. `ergon tool run rust.cargo-llvm-cov` failed with this error:

```text
ergon: options: invalid .ergon.yaml: rust: decoding failed due to the following error(s):

'tools' has invalid keys: cargo-llvm-cov
```

A repository measures the coverage of its Rust tests with cargo-llvm-cov, a subcommand of cargo that `cargo install` builds. `Cargo.toml` has no place for such a tool. A Python repository has one: pytest runs in the environment of the project, so the repository adds pytest-cov to its own dependencies.

RFC-0004 leaves coverage out of the gate of every language, and assigns coverage gates to another component.

## Decision

We will pin cargo-llvm-cov in `rust.tools` without a coverage target, because a repository needs a pinned release of the tool and RFC-0004 keeps coverage gates out of the baseline.

- `rust.tools.cargo-llvm-cov` pins cargo-llvm-cov 0.9.1, the newest release on crates.io on 2026-10-16. `ergon tool run` installs it with `cargo install --locked`, as it installs cargo-audit.
- The targets of the baseline do not run cargo-llvm-cov. The fragment of the Makefile sets `CARGO_LLVM_COV` to its command for the targets that a repository adds.
- A repository adds its coverage target to `.ergon/local/Makefile`, such as a target that runs `$(CARGO_LLVM_COV) llvm-cov --workspace --fail-under-lines 90`.

## Alternatives Considered

### A target coverage-rust with a threshold of the section rust

The issue asked for a target `coverage-rust` that runs `cargo llvm-cov` with a threshold of lines from the section `rust`. It lost because a target with a threshold is a coverage gate, which RFC-0004 leaves to another component. The other ten languages have no coverage target either.

## Consequences

**Positive:**

- `ergon tool run rust.cargo-llvm-cov` installs cargo-llvm-cov 0.9.1 and runs it, on a workstation and in CI.
- The baseline update moves the pin of cargo-llvm-cov with the other pins of crates.

**Negative:**

- A repository writes its own coverage target and chooses its own threshold.
- cargo-llvm-cov needs the rustup component `llvm-tools-preview`. Without it, a run fails with `failed to find llvm-tools-preview`. A repository adds the component, such as in the key `components` of `rust-toolchain.toml`.
- The fragment of the Makefile has a variable that the targets of the baseline do not use.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#32 |
| cargo-llvm-cov | <https://github.com/taiki-e/cargo-llvm-cov> at v0.9.1 |
| The gate of ergon init | RFC-0004 |
