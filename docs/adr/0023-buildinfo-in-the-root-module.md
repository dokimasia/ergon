---
adr: 0023
title: ergon reads the version of its binary in a package of the root module
status: Accepted
date: 2026-10-09
supersedes: RFC-0007 and ADR-0021, in part
superseded-by: none
rfc: RFC-0007
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0023: ergon reads the version of its binary in a package of the root module

## Status

Accepted

## Context

RFC-0007 and ADR-0021 put the reading of the version of a binary into a module of its own, `go.dokimi.dev/ergon/buildinfo`, so that the commands of other repositories could import it. The package requires `golang.org/x/mod` for its semantic versions and pseudo-versions.

ergon is a command, and no repository imports a module of ergon. The module still cost ergon a vanity page on go.dokimi.dev, a tag in each release, and a step in every loop of the gate. Until its first release, the root module required it at the untagged version v0.0.0, so `go get` in the root module needed temporary directory replaces of the siblings.

## Decision

We will read the version of ergon in the package `internal/buildinfo` of the root module:

- The package keeps the rules of RFC-0007. It returns the version of a release for a build at its tag, and `dev` for `(devel)`, a pseudo-version and a build of a working tree with changes, each with the short commit and the date of the commit.
- `cmd/ergon` imports it, and ergon has no module `go.dokimi.dev/ergon/buildinfo`.

## Alternatives Considered

### The module go.dokimi.dev/ergon/buildinfo

It lost because no command outside ergon imports it, and a module costs a vanity page, a tag and a place in every loop of the gate.

### The earlier internal/buildinfo with flags of the linker

It lost because the managed GoReleaser configuration of RFC-0007 does not set an `-X` flag, and the earlier `Release` returned `0.5.0+dirty` for a build of a working tree with changes at a release tag.

## Consequences

**Positive:**

- ergon has one module fewer, and the root module does not require an untagged sibling.

**Negative:**

- A command of another repository that wants the same rules writes its own package.

**Neutral:**

- `ergon --version` prints the same version string as with the module.

## References

| What | Where |
|---|---|
| The design of the binary releases | RFC-0007, Binary releases |
| The decision of the binary releases | ADR-0021 |
