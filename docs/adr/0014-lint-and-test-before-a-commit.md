---
adr: 0014
title: The hooks lint and test each commit, and run the gate before each push
status: Accepted
date: 2026-10-08
supersedes: none
superseded-by: ADR-0015, in part
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0014: The hooks lint and test each commit, and run the gate before each push

## Status

Accepted

## Context

RFC-0004 renders `.pre-commit-config.yaml` with a hook that runs `make check` before each commit. `make check` runs the gate of every language of the repository. For Go, `check-go` runs the linters, the tests, the tests under the race detector and the vulnerability scan of every module.

Each commit of a series runs that whole gate, so a series of ten commits runs the race tests and the vulnerability scan ten times. The scan also queries its advisory database over the network, and a commit fails without network access.

CI runs `make check-<language>` for the push of every commit and for every pull request.

## Decision

We will run the linters and the tests before each commit. The whole gate runs before each push:

- The hooks `lint` and `test` run `make lint` and `make test` at the stage `pre-commit`.
- The hook `check` runs `make check` at the stage `pre-push`.
- The hooks of pre-commit-hooks run at the stage `pre-commit` alone. pre-commit-hooks declares `trailing-whitespace`, `end-of-file-fixer` and `check-added-large-files` for the stage `pre-push` as well, so the configuration limits these three to the stage `pre-commit`.
- `default_install_hook_types` names `pre-commit`, `commit-msg` and `pre-push`, so `pre-commit install` installs all three hooks.
- The hook `commitlint` still checks each commit message at the stage `commit-msg`.

## Alternatives Considered

### The gate before each commit

The hook `check` would keep the stage `pre-commit` of RFC-0004. It lost because a series of commits would repeat the race tests and the vulnerability scan for every commit. Each commit would also need network access.

### No gate before a push

The hooks would lint and test each commit. CI alone would run the gate. It lost because a push would then send a data race or a vulnerable dependency to the remote. CI would report it only minutes later.

## Consequences

**Positive:**

- The hook of a commit runs the linters and the tests only.
- The race tests and the vulnerability scan run once for each push, before git sends the commits to the remote.
- A push runs the same gate as CI, and no other hook.

**Negative:**

- A commit can contain a data race or a vulnerable dependency that the hook of the push reports later. The series then needs one more commit before the push.

**Neutral:**

- pre-commit 4.6.2 runs the hook of a push only for the commits that the remote-tracking branches of the remote lack. On 2026-10-08, the push of a tag at a commit that the remote had did not run the hook, and the push of a tag at a new commit ran it. `ergon release publish` on a workstation pushes its tags at HEAD, so it runs the gate only when HEAD is not yet on the remote.

## References

| What | Where |
|---|---|
| The stages of a hook of pre-commit | https://pre-commit.com/#confining-hooks-to-run-at-certain-stages |
| The hook types that `pre-commit install` installs | https://pre-commit.com/#top_level-default_install_hook_types |
