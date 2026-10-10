---
adr: 0038
title: The binaries of a release build outside the workspace against a proxy of the released modules
status: Accepted
date: 2026-10-10
supersedes: RFC-0007, in part
superseded-by: none
rfc: RFC-0007
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0038: The binaries of a release build outside the workspace against a proxy of the released modules

## Status

Accepted

## Context

`ergon release pack` ran GoReleaser in the checkout of the release, whose `go.work` made the go command build in workspace mode. A binary then recorded each other module of the workspace without a version. The binary of ergon 0.8.1 recorded `go.dokimi.dev/ergon/core` and the eleven language modules as `(devel)`, so its SBOMs listed them without a version. `go install` of the release resolves the same modules at the versions that `go.mod` requires, and checks them against the hashes of `go.sum`.

The tags of the release do not exist when the pack runs. `ergon release version` serves the released modules from a module proxy in a temporary directory, with zips of a snapshot of the working tree, so that `go mod tidy` writes their hashes into `go.sum` before the tags exist.

## Decision

We will run GoReleaser outside the workspace, against a proxy of zips of the released modules before the configured proxy, because the go command then resolves the modules of the repository as `go install` resolves them, and writes their versions into the binaries.

- `ergon release pack` writes the zip of each module of the release at its version into a module proxy in a temporary directory, as `ergon release version` does, and removes the directory after the runs.
- GoReleaser runs with `GOWORK=off` and `GOFLAGS=-modcacherw`, a module cache in the temporary directory, and a `GOPROXY` of the proxy of the release, the download cache of the module cache of the go command, and the configured `GOPROXY`. `GONOSUMDB` adds the module paths of the repository.
- The build reads `go.mod` and `go.sum` without changes. A `go.sum` whose hash of a released module differs from the content of that module fails the pack, as it would fail `go install` of the release.

## Alternatives Considered

### The build in the workspace

GoReleaser would keep building with `go.work`, as before. It lost because the binaries and their SBOMs would keep recording the modules of the repository without a version, and the binaries of a release would differ from the binaries that `go install` builds.

## Consequences

**Positive:**

- A binary of a release records each module of the repository at its version and with its hash, as a binary of `go install` does.
- The pack fails a release that `go install` would refuse because of `go.sum`.

**Negative:**

- The pack writes a zip of each released module, and extracts the modules into a module cache of its own, which adds to the time of the job.
- A module of a command that requires a module of the repository through a directory replace builds with the directory of that replace, which `go install` refuses.

**Neutral:**

- The proxy and the module cache are in a temporary directory, so the module cache of the go command does not receive the content of a version before its tag.

## References

| What | Where |
|---|---|
| The pack of the binaries of a release | RFC-0007 |
| The proxy of the released modules | ADR-0004 |
| The modules of a release that syft reads | ADR-0037 |
