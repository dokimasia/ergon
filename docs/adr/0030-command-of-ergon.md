---
adr: 0030
title: The command that runs ergon is an option of the section common
status: Accepted
date: 2026-10-10
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0030: The command that runs ergon is an option of the section common

## Status

Accepted

## Context

The targets of the managed Makefile run their tools with `$(ERGON) tool run`. `ERGON` was `ergon`, the ergon on the PATH. The commit-msg hook of `.pre-commit-config.yaml` ran `ergon tool run common.commitlint` in the same way.

ergon's own repository renders its managed files with the ergon of its source, so its `.ergon.yaml` has each new option before a release of ergon has it. A release decodes `.ergon.yaml` strictly and refuses an option that it does not know. On 2026-10-10 ergon 0.6.1 refused the file of the commit that added `go.lint.plugins` with `'lint' has invalid keys: plugins`, and the pre-push hook failed. CI has no such gap: the action `setup-ergon` of ergon's repository builds ergon from the source of the commit.

`go run go.dokimi.dev/ergon/cmd/ergon` runs the ergon of the source from any module directory of the repository. The go command finds the package through `go.work`. Go caches the executable of `go run`, and on 2026-10-10 the second and later runs took 0.07 s.

## Decision

We will make the command that runs ergon in the Makefile and in the hooks an option, `common.ergon`, because ergon's repository needs the ergon of its source in its targets and hooks, as its CI has.

- `common.ergon` is a command as its words, `[ergon]` at the baseline. The Makefile sets `ERGON ?=` to the command, and the commit-msg hook runs the command with `tool run common.commitlint`.
- A word has only letters, digits and the characters `_ @ % + = : , . / ^ -`, so the Makefile, the shell and pre-commit read each word as it is.
- ergon's repository sets `common.ergon` to `[go, run, go.dokimi.dev/ergon/cmd/ergon]`.

## Alternatives Considered

### ergon installed from the checkout

The README of ergon's repository asked for `go install ./cmd/ergon` before `make check`. It lost because the shell runs the first ergon on the PATH, and on the workstation of the maintainer the ergon of Homebrew comes before `~/go/bin`. Each change of the source also needs another install.

### An override of `ERGON` in `.ergon/local/Makefile`

The local file of the Makefile would set `ERGON` for ergon's repository. It lost because the targets expand `$(ERGON)` when make reads the managed part, before it reads the local part. The commit-msg hook also runs ergon without make.

## Consequences

**Positive:**

- The targets and the hooks of ergon's repository run the ergon of the tree that they check, without an installed ergon.

**Negative:**

- The first run after a change of the source compiles ergon before it runs a tool.
- A commit of ergon whose staged tree does not compile fails its hooks with the build error of ergon.

**Neutral:**

- Every other repository keeps `[ergon]`, the ergon on the PATH.

## References

| What | Where |
|---|---|
| The tools of the gate and the Makefile | RFC-0004, Tools |
| The ergon of the source in CI | `.ergon/local/.github/actions/setup-ergon/action.yml` of ergon's repository |
| The cache of the executables of `go run` | https://go.dev/doc/go1.24#go-command |
