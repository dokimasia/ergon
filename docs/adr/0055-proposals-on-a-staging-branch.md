---
adr: 0055
title: A proposal is committed on a staging branch, and its branch moves once
status: Accepted
date: 2026-10-18
supersedes: ADR-0005 and RFC-0002, in part
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0055: A proposal is committed on a staging branch, and its branch moves once

## Status

Accepted

## Context

`release.Propose` first pointed the branch of its pull request at the head of the base branch, because `createCommitOnBranch` appends a commit to the head of an existing branch. It then committed on that branch. `ergon release ci version`, `ergon init ci upgrade` and the weekly update of the baseline open or update their pull requests through `Propose`.

While the branch points at the head of the base, its open pull request has no commits of its own. GitHub closes such a pull request and records the bot of the push as the actor of the close. The Version runs of thesm-os/core closed the version pull requests 60, 66 and 70 this way. In each, the close and the force push of the bot to the head of `main` have the same second. ergon opened pull request 68 two seconds after the close of 66, and a maintainer reopened 70 by hand.

changesets/action v2.1.2 commits through the same mutation with ghcommit v3.0.0. ghcommit builds the commit on a temporary branch and then force-updates the branch of the pull request to the new commit. Its source gives the reason: GitHub closes the pull request of a branch that is reset to the base.

## Decision

We will build the commits of a proposal on the branch `ergon-staging/<branch>` and move the branch of the pull request to the last of them in one update, because a branch that points at the head of its base closes its open pull request.

- `Propose` points `ergon-staging/<branch>` at the head of the base and makes the same commits there as before.
- It then moves the branch of the pull request from its earlier commit to the last new commit with one update of the ref.
- It deletes the staging branch, also after a commit that fails.
- The commits keep the signature of GitHub. The update of the ref moves the branch to the commits that the mutation made.

## Alternatives Considered

### Reopen the pull request after the commits

`Propose` keeps the reset and reopens a pull request that the reset closed. It lost because every update then writes a close and a reopen into the pull request. A run that fails between the two also leaves the pull request closed.

### Commits through the Git Database REST API

A commit of the REST API takes any parent, so no branch has to point at the head of the base. It lost because GitHub leaves such commits unsigned.

## Consequences

**Positive:**

- An update keeps the open pull request open: the version pull request and the pull requests of the baseline.

**Negative:**

- Each proposal makes three more requests to GitHub: the lookup, the creation and the deletion of the staging branch.
- A run that stops between the creation and the deletion leaves `ergon-staging/<branch>` behind. The next run of the same command moves it to the head of the base and deletes it.
- A workflow that runs on a push to any branch also runs for the commits that the token of a GitHub App makes on the staging branch. The managed workflows run on a push to `main` alone.

## References

| What | Where |
|---|---|
| The temporary branch of ghcommit v3.0.0 | <https://github.com/changesets/ghcommit/blob/v3.0.0/src/core.ts> |
| The version pull requests that the reset closed | <https://github.com/thesm-os/core/pull/60>, <https://github.com/thesm-os/core/pull/66>, <https://github.com/thesm-os/core/pull/70> |
