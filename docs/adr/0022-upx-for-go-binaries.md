---
adr: 0022
title: The release of a Go module packs the Linux binaries of its commands with UPX
status: Accepted
date: 2026-10-09
supersedes: RFC-0007, in part
superseded-by: none
rfc: RFC-0007
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0022: The release of a Go module packs the Linux binaries of its commands with UPX

## Status

Accepted

## Context

RFC-0007 builds the binaries of the commands of a Go module in the job pack of `release.yml`, and leaves UPX out of scope. A binary of ergon is 19 to 21 MB.

We packed the six binaries of ergon 0.6.0 with UPX 5.2.1 on 2026-10-09, and cataloged them with syft 1.54.1:

| Platform | Unpacked | `--best` | `--best --lzma` |
|---|---|---|---|
| linux/amd64 | 20.4 MB | 6.2 MB | 5.3 MB |
| linux/arm64 | 19.0 MB | 5.4 MB | 4.3 MB |
| windows/amd64 | 21.0 MB | 6.3 MB | 5.4 MB |
| windows/arm64 | 19.3 MB | 5.6 MB | 5.6 MB |
| darwin/amd64, darwin/arm64 | 20.8 MB, 19.4 MB | Refused | Refused |

- UPX refuses a macOS binary with `CantPackException: macOS is currently not supported (try --force-macos)`. Its release notes disable macOS support "until we fix compatibility with macOS 13+".
- syft finds the 55 Go modules of the Linux binary when UPX packed it with LZMA, and none when UPX packed it with its default method. syft reconstructs only ELF binaries that UPX packed with LZMA, so it finds none of the 57 Go modules of a packed Windows binary.
- A packed binary decompresses itself on each start. `ergon --version` on linux/amd64 takes 8.2 ms unpacked, 44.1 ms packed with `--best` and 212.8 ms packed with `--best --lzma`, as a mean of 20 runs.
- `go version -m` cannot read the build information of a packed binary. The binary still reports its version: `ergon --version` printed `ergon 0.6.0 (14b3b7b, 2026-10-09)` after the pack.
- GoReleaser runs the program `upx` from `PATH` in its section `upx`, and skips the step without an error when `PATH` has no `upx`.
- UPX publishes its Linux programs as `.tar.xz` archives, a format that `ergon tool run` did not unpack.

## Decision

We will pack each Linux binary of a command with UPX in the release of its module:

- The managed GoReleaser configuration builds the Linux targets of a command in the build `<name>-upx` and its other targets in the build `<name>`. A post hook of the build `<name>-upx` runs `ergon tool run go.upx -- --best --lzma --quiet` on each binary, so the SBOM of a Linux archive still lists the Go modules of the binary.
- macOS binaries stay unpacked, because UPX refuses them. Windows binaries stay unpacked, because the SBOM of a packed Windows binary lists none of its Go modules.
- `go.tools.upx` pins UPX 5.2.1 as a release binary, with the SHA-256 of the asset of linux/amd64, linux/arm64 and windows/amd64. UPX does not publish a program for macOS or for Windows on ARM.
- `ergon tool run` unpacks a `.tar.xz` asset with `github.com/ulikunitz/xz`, a decoder in Go.

## Alternatives Considered

### The section `upx` of GoReleaser

It lost because GoReleaser runs `upx` from `PATH` and skips the step without an error when it finds none. A release would then publish unpacked binaries without a failure. A hook runs UPX through `ergon tool run`, which installs the pinned version and fails the pack when UPX fails.

### UPX's default method, or `--best` without LZMA

It lost because syft finds none of the 55 Go modules of a binary that UPX packed with these methods, so the SBOMs of the release would not list the dependencies. The method starts in 44.1 ms instead of 212.8 ms.

### Packed Windows binaries

It lost because syft finds none of the 57 Go modules of a Windows binary that UPX packed with any method.

### Packed macOS binaries with `--force-macos`

It lost because UPX disabled its support for macOS until it works on macOS 13 and later.

## Consequences

**Positive:**

- The Linux archives and packages are about four times smaller: 5.3 MB instead of 20.4 MB on amd64, and 4.3 MB instead of 19.0 MB on arm64.

**Negative:**

- Each start of a packed binary takes about 205 ms more, the time that UPX takes to decompress the binary.
- `go version -m` cannot read the build information of a packed binary.
- The pack runs UPX for about 4 seconds per Linux binary.
- `ergon release pack` fails on macOS for a command with a Linux target, because UPX does not publish a program for macOS. The job pack runs on Linux.

**Neutral:**

- The macOS and Windows binaries, their archives and their SBOMs do not change.

## References

| What | Where |
|---|---|
| The design of the binary releases | RFC-0007, Binary releases |
| The UPX section and the build hooks of GoReleaser | https://goreleaser.com/customization/builds/upx/, https://goreleaser.com/customization/builds/hooks/ |
| The releases of UPX and its release notes | https://github.com/upx/upx/releases, https://github.com/upx/upx/blob/devel/NEWS |
| The UPX support of syft's Go cataloger | https://github.com/anchore/syft/blob/main/syft/pkg/cataloger/golang/upx.go |
| The xz decoder | https://github.com/ulikunitz/xz |
