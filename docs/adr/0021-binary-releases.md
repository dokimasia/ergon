---
adr: 0021
title: Each release of a Go module attaches the signed binaries of its commands
status: Accepted
date: 2026-10-09
supersedes: none
superseded-by: ADR-0023, in part
rfc: RFC-0007
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0021: Each release of a Go module attaches the signed binaries of its commands

## Status

Accepted

## Context

ergon and the repositories of ThesmOS build the binaries of their releases with GoReleaser configurations and workflows that each repository writes by hand. The configurations drifted apart, and an update of GoReleaser breaks a configuration that uses a deprecated key.

GoReleaser without Pro refuses a tag with a directory prefix, such as `lint/v0.1.0`. ergon's `binaries.yml` attaches the archives after `ergon release publish` has published the release, which a repository with immutable releases refuses.

## Decision

We will build the binaries of the commands of a Go module in each release of the module:

- `go.binaries` of `.ergon.yaml` lists the commands, each with its module, its package, its platforms, its completions, its Linux packages and its cask. `go.homebrew.tap` names the tap of the casks.
- ergon init renders a managed GoReleaser configuration for each module with a command.
- Every command gets checksums, a cosign signature, SBOMs and SLSA build provenance, and is built with CGO off, `-trimpath` and `-s -w`. Go writes the version from the tag of the module, without linker flags.
- The job `pack` of `release.yml` runs GoReleaser in snapshot mode at the local tag of the module, and attests the assets. `ergon release publish` attaches them to a draft release and then publishes the release.
- The job `homebrew` commits the casks to the tap with a token of the GitHub App of ergon.
- The module `go.dokimi.dev/ergon/buildinfo` reads the version of a binary.

## Alternatives Considered

### A configuration that each repository writes

It lost because the configurations drift, and each update of GoReleaser's pin would break them repository by repository.

### A job after the publish

It lost because a repository with immutable releases refuses the upload, and because a published release has no binaries until the job ends.

## Consequences

**Positive:**

- A release has its signed and attested assets from the moment it is published.
- A repository states its commands in `.ergon.yaml`, and the baseline maintains the configuration and the pins.

**Negative:**

- The job `pack` of a repository with commands gets `id-token: write`.
- A failure of GoReleaser blocks the publish of every package of the publish plan.

**Neutral:**

- A repository without commands renders no GoReleaser configuration, and its `release.yml` does not change.

## References

| What | Where |
|---|---|
| The design of the options, the configuration, the release flow and the version of a binary | RFC-0007, Binary releases |
