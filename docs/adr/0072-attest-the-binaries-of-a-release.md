---
adr: 0072
title: The job pack attests the binary of each command beside the assets of a release
status: Accepted
date: 2026-10-29
supersedes: RFC-0007, in part
superseded-by: none
rfc: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0072: The job pack attests the binary of each command beside the assets of a release

## Status

Accepted

## Context

The job `pack` of `release.yml` attests every file under `dist/assets/` with SLSA build provenance: the archives, the packages, the source archive, the checksums, their signature and the SBOMs. `gh attestation verify` finds the attestation of a file by its digest. A person who installs a command through its cask, a package or an archive runs the binary alone. The digest of the binary does not match an attested file, so `gh attestation verify "$(command -v <name>)"` does not find an attestation of an installed command.

GoReleaser builds each binary into its dist directory, and UPX packs each Linux binary there before GoReleaser archives and packages it. The archive and the package of a platform contain the bytes of that file. `ergon release pack` removes the dist directory after it copies the assets.

## Decision

We will attest the binary of each command and platform of a release in the job `pack`, beside the assets, because the binary is the file that a person runs and verifies.

- `ergon release pack` copies each binary of a run of GoReleaser into `dist/binaries/<tag>/<target>/<binary>`, after UPX has packed it. `<target>` is the directory of the build in the dist directory of GoReleaser, such as `ergon-upx_linux_amd64_v1`.
- The step that attests the assets also attests every file under `dist/binaries/`.
- A release attaches no binary. The artifact of the job `pack` leaves `dist/binaries/` out, so the jobs `publish` and `homebrew` do not download the binaries.

## Alternatives Considered

### The archives and the packages alone

The job `pack` attests the assets of a release alone. A person verifies the archive or the package that they downloaded. It lost because a cask and a package manager do not keep the archive beside the installed binary, so a person cannot check the binary that they run.

### The binaries from the archives

A step of the job `pack` unpacks each archive and attests the binary inside it. It lost because that step would read every archive format of a release in shell, while the packer already has each binary.

## Consequences

**Positive:**

- `gh attestation verify "$(command -v <name>)" --repo <owner>/<repo>` checks the build provenance of an installed binary.
- A repository that attested its binaries in a job of its own, such as techne before it moved to the managed release, keeps that check.

**Negative:**

- The attestation of a release has one more subject for each platform of each command.
- The binaries of a release before this change have no attestation of their own.
- `dist/binaries/` exists in the job `pack` alone, so a step of another job cannot read a binary.

## References

| What | Where |
|---|---|
| The inputs of actions/attest | <https://github.com/actions/attest> |
| The verification of an attestation by the digest of a file | <https://cli.github.com/manual/gh_attestation_verify> |
| The exclusion of a path from the artifact of actions/upload-artifact | <https://github.com/actions/upload-artifact> |
