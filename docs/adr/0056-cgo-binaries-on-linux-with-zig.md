---
adr: 0056
title: Commands with cgo build every platform on Linux with zig
status: Accepted
date: 2026-10-21
supersedes: RFC-0007, in part
superseded-by: none
rfc: RFC-0009
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0056: Commands with cgo build every platform on Linux with zig

## Status

Accepted

## Context

eidos, techne and dokimi have a command that links the C grammars of tree-sitter through cgo. The managed binary release builds every command with `CGO_ENABLED=0`, so eidos and techne build their binaries in a hand-written job on a macOS runner, with an unpinned UPX from Homebrew.

zig 0.16.0 compiled the C code of `cmd/eidos` and `cmd/techne` for both architectures of macOS on Linux. Their link needed `libresolv`, CoreFoundation and Security, which Go's standard library imports on darwin. The archive of zig has no stubs of them. Text stubs of those libraries let both commands link.

## Decision

We will build the binaries of a command with cgo for every platform on the Linux runner of the job `pack`, with zig as the C compiler and stubs of the libraries of macOS that the Go toolchain imports, because the pinned UPX runs only there and no target needs a macOS host.

- `cgo: true` on a command of `go.binaries` builds it with cgo.
- zig is the release binary `go.tools.zig`, pinned by its version and a SHA-256 per platform.
- `ergon release pack` derives the stubs from the source of the Go toolchain that builds.
- The pinned UPX packs the Linux binaries of a command with cgo, as it packs those of a command without cgo.

## Alternatives Considered

### A macOS runner with UPX from Homebrew

The job `pack` runs on `macos-26` and installs UPX with Homebrew, as the jobs of eidos and techne do. It lost because a darwin build does not need a macOS host, and UPX is then not pinned.

### A macOS runner without UPX

The job `pack` runs on `macos-26` and packs no binary. It lost because the Linux binaries of a command with cgo would then differ from those of a command without cgo.

## Consequences

**Positive:**

- A repository lists a command with cgo under `go.binaries` and drops its own GoReleaser configuration and its job `binaries`.
- Every tool of the build has a pinned version and digest.

**Negative:**

- No run on macOS has checked the darwin binaries. Their load commands, their libraries and their signature match a binary that Go links on macOS.
- A command whose own C code needs a header or a framework of the macOS SDK does not build.
- Each pack of such a repository downloads zig, 55 MB.

## References

| What | Where |
|---|---|
| The limits of cgo in GoReleaser | https://goreleaser.com/resources/limitations/cgo/ |
| The download index of zig | https://ziglang.org/download/index.json |
