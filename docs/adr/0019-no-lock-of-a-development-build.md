---
adr: 0019
title: A build of ergon without a release does not write a lock of a release
status: Accepted
date: 2026-10-09
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0019: A build of ergon without a release does not write a lock of a release

## Status

Accepted

## Context

`.ergon/init.lock` records the version of the ergon that wrote it, and the action `setup-ergon` installs the release of that version in every job. A build of ergon without a release records the version `dev`:

- A `go build` of a checkout records the version `(devel)` of its main module.
- A `go install` at a commit records a pseudo-version.

On 2026-10-08 a local build of ergon ran `ergon init new` and `ergon init sync` in assert-go. Its lock recorded `dev`, so the first push failed in the job `skip`, whose `setup-ergon` asked GitHub for the release `vdev` and received 404.

A `go install go.dokimi.dev/ergon/cmd/ergon@latest` records the version that it resolved, such as `v0.5.0`, so such a build is a release.

## Decision

We will let a build of ergon without a release write only over a lock that such a build wrote:

- `New`, `Add`, `Remove` and `Sync` of the package `baseline` return an error that wraps `ErrDevelopmentBuild` for a new lock and for a lock that a release wrote, and leave every file as it is.
- A lock that records `dev` stays writable for such a build. The repository of ergon builds ergon from its own source in CI, and its lock records `dev`.
- `ergon init upgrade` in such a build runs the sync itself, so it refuses a lock of a release in the same way.

## Alternatives Considered

### A flag that sets the release in the lock

A flag such as `--as-release 0.5.0` would let a local build write a lock of a release. It lost because the lock would record a release whose baseline differs from the content of the build, so `ergon init check` in CI would report every managed file that the build changed after that release.

### setup-ergon installs the newest release for dev

The action would install the newest release when the lock records `dev`. It lost because the newest release renders another baseline than the build that wrote the lock, so the job `baseline` would fail on the next change of the baseline.

## Consequences

**Positive:**

- A repository's CI installs the release that its lock names, in every repository but ergon's own.

**Negative:**

- A person who builds ergon from a checkout runs a release of ergon to set up or sync another repository.

## References

| What | Where |
|---|---|
| The version that a build records | `go version -m`, and `debug.BuildInfo` in runtime/debug |
| The release of a build of ergon | `internal/buildinfo/version.go`, the function `Release` |
