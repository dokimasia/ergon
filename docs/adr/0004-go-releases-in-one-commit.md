---
adr: 0004
title: Go releases in one commit through a file proxy
status: Accepted
date: 2026-09-24
supersedes: none
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0004: Go releases in one commit through a file proxy

## Status

Accepted

## Context

A Go module's version is a git tag. A dependent's `go.sum` needs the hash of the sibling version it requires, and that version does not exist until its tag is pushed. `go mod tidy` with `GOWORK=off` cannot resolve it before then.

The earlier ergon released a Go workspace one dependency layer at a time: tag the layer, push, rewrite the dependents' `require` lines, run `go mod tidy`, commit. A version pull request is one commit, and it cannot contain a push between layers. The layer pipeline also depends on proxy.golang.org during the release. One eidos release stopped on `unknown revision backend/golang@v1.7.0` because the proxy had cached an earlier lookup of that version.

A module's `h1:` hash covers only its file names and bytes. `golang.org/x/mod/zip` and the go command read those files with the same `git archive` invocation, so the hash is computable from a commit before any tag exists.

## Decision

We will release every Go module in a repository in one commit, and compute the dependents' `go.sum` entries through a `file://` proxy built from a snapshot of that commit, because a version pull request is one commit and the hash of a module version depends only on its files.

For each module that another module requires without a directory `replace`, in dependency order, `lang/go/release` rewrites the `require` lines, snapshots the tree with `git commit-tree`, writes the module zip with `zip.CreateFromVCS`, and runs `go mod tidy` in each dependent against the file proxy.

## Alternatives Considered

### The dependency-layer pipeline

The earlier ergon published each layer's tags, then pinned and tidied the next layer against them. It lost because it needs a commit and a push per layer, which one version pull request cannot contain, and it depends on proxy.golang.org while the release is in progress.

## Consequences

**Positive:**

- A Go release of any number of modules is one commit and one set of tags.
- The release makes no network request for the repository's own modules.
- A dependent's `go.sum` at the tag verifies against the sibling it requires. A two-module spike confirmed this with `go mod verify` against an empty module cache.

**Negative:**

- A cycle among modules that require each other without a `replace` has no fixed point for `go.sum`, so ergon refuses to release it.
- `lang/go/release` runs git itself, through `zip.CreateFromVCS`, as the one exception to `service/vcs` running all git commands.
- The file proxy runs one `go mod tidy` per dependent. Tidy needs the network for third-party modules.
- A version pull request merged while `main` changed a released module leaves a stale `go.sum`. The next release run detects the mismatch and opens a pull request that rewrites `go.sum`.

**Neutral:**

- The spike checked the hashes against a proxy rebuilt from the tag, not against proxy.golang.org. Consumers verify against sum.golang.org and never read a dependency's `go.sum`.

## References

| What | Where |
|---|---|
| `dirhash.Hash1` | `golang.org/x/mod` v0.40.0, `sumdb/dirhash/hash.go` |
| `zip.CreateFromVCS` and its `git archive` call | `golang.org/x/mod` v0.40.0, `zip/zip.go:619` and `filesInGitRepo` |
| The go command's `git archive` call | `cmd/go/internal/modfetch/codehost/git.go:928`, go1.27.1 |
| proxy.golang.org caching | https://proxy.golang.org/ |
