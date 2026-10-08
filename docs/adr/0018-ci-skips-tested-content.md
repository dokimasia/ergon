---
adr: 0018
title: ci.yml skips its jobs on content that already passed
status: Accepted
date: 2026-10-08
supersedes: ADR-0016, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0018: ci.yml skips its jobs on content that already passed

## Status

Accepted

## Context

On 2026-10-08 one release of ergon ran the whole gate three times. The runs tested the push of the commits to `main` (f81caba), the version pull request 8 (96bb11b), and the push of its merge (2b9d576). Only the first run tested new code:

- `version.yml` opens the version pull request only from a commit whose run of `ci.yml` passed. The version commit adds the output of `ergon release ci version`, which consists of versions, changelogs, requirements between the packages and lockfiles.
- The merge of the version pull request has the tree of its head, which its own run tested.

A repository whose gate takes 30 minutes spends 90 minutes of runners on each release.

Other tools avoid such runs:

- semantic-release writes `[skip ci]` into its release commit, so that the commit does not start a run.
- `fkirc/skip-duplicate-actions` skips a run when a run with the same tree hash passed. It reads the runs through the API of GitHub. For a matrix job that skips, it recommends a final job as the one required check, because the matrix does not report a check per runner.
- GitHub does not start a run for a commit message with `[skip ci]`, and leaves the checks of the commit pending. A pull request that requires those checks cannot merge.

## Decision

We will start every run of `ci.yml` with a job that checks whether a passed run already covers the content of the run, and skip every other job when one does:

- The job `skip` runs `ergon release ci skip`. Every other job needs it, and runs only when its output `skip` is `false`.
- A push takes the result of a passed run on its content, as `ergon release ci verify` finds it. The merge of a pull request whose run passed on the same tree is such a push.
- A version commit takes the result of the passed run of its first parent. `ergon release ci version` marks the head of the version pull request with the status `ergon/version`. The rule takes only a commit with that status in the state `success` and the tree of the run. A run of a pull request takes only this rule, because its checkout merges its head into the current base.
- `ergon release ci skip` writes a warning and `false` when it cannot read GitHub, so that the run tests the commit.
- The job `result` needs every job and fails when one failed or was cancelled. A branch requires the check `Result` alone.
- `version.yml` gets the permission `statuses: write`, and asks the GitHub App for the same permission.

## Alternatives Considered

### `[skip ci]` in the version commit

The version commit would not start a run, as the release commit of semantic-release does not. It lost because the version pull request and its merge would have no passed run. `ergon release ci verify` would then refuse the publish, and a required check of the pull request would remain pending.

### Skipping by tree alone

Only a passed run on the same tree would let the job `skip` skip. It lost because the tree of the version pull request is new, so a release would still run the gate twice.

### Reproducing the version commit in CI

The job `skip` would run `ergon release version` on the parent and compare the trees. It lost because the job would need the toolchains of the repository, and the Go toolchain runs `go mod tidy` in every module. Every push would spend that time, while the status proves the same for a commit that `version.yml` wrote.

### Skipping each step

Every job would start and skip its steps, so each runner of a matrix would still report its check. It lost because GitHub bills each started job for at least one minute, at the rate of its runner.

## Consequences

**Positive:**

- A release runs the gate once, on the push of its code. The runs of the version pull request and of its merge run the jobs `skip` and `result` alone.
- The push of a merge whose pull request passed on the same tree does not run the other jobs either.

**Negative:**

- Every run starts two more jobs, `skip` and `result`.
- A repository that requires the check of each runner of a matrix job blocks the version pull request, until it requires `Result` instead.
- A commit that a person pushes onto the branch of the version pull request has no status, so its run runs every job.
- A proposal of more than one commit marks its last commit, whose parent has no run, so its runs run every job.
- Anyone who can set a commit status can mark a commit as a version commit. That requires write access, which also allows a push to `main`.

**Neutral:**

- `release.yml` keeps `ergon release ci verify`, which accepts the passed run of a run that skipped its jobs.

## References

| What | Where |
|---|---|
| The release commit of semantic-release | https://github.com/semantic-release/git, the option `message` |
| The runs that skip-duplicate-actions skips, and required matrix jobs | https://github.com/fkirc/skip-duplicate-actions, "How does it work?" and "Frequently Asked Questions" |
| `[skip ci]` and the checks of a skipped run | https://docs.github.com/en/actions/how-tos/manage-workflow-runs/skip-workflow-runs |
| The combined status of a commit | https://docs.github.com/en/rest/commits/statuses |
