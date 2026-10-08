---
adr: 0016
title: The CI run of the version pull request is the gate of a release
status: Accepted
date: 2026-10-08
supersedes: ADR-0010
superseded-by: ADR-0018, in part
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0016: The CI run of the version pull request is the gate of a release

## Status

Accepted

## Context

ADR-0010 makes the release workflow wait for the CI run of its commit. The job wait reads the run of `ci.yml` every 15 seconds until it completes. On 2026-10-08 that job waited 27 minutes before the publish of v0.3.0. The runs of `ci.yml` for pushes to `main` share one concurrency group, and a second run of the earlier commit kept the group for 13 minutes. GitHub charges a private repository for the runner of the wait for its whole length.

On the same day, GitHub started two Release runs for the push of commit 096abbd. The second run started after the merge of version pull request 6, and opened pull request 7 from 096abbd. `ergon release ci version` did not check that its commit was still the head of `main`.

GitHub starts a workflow after another workflow completes through the event `workflow_run`. A publish cannot run on that event:

- crates.io refuses its token exchange for a run of `workflow_run` and recommends `push`, `release` or `workflow_dispatch`.
- A run of `workflow_run` has `GITHUB_SHA` set to the newest commit of the default branch, and not to the commit that the triggering run tested. The OIDC claim `sha` becomes the source digest of the Sigstore certificate, so npm provenance and PyPI attestations would record a commit that the release did not publish.

changesets, release-please and release-plz publish on a push to `main` and wait for no run. The checks of a pull request, the release pull request included, are the gate of its merge. release-plz states this as its preferred setup. Each of the three opens the release pull request with a token whose events start workflows. GitHub starts the runs of a pull request that `GITHUB_TOKEN` opens only after a maintainer approves them.

## Decision

We will make the CI run of the version pull request the gate of its release, and start each job of the release flow after the run that it needs:

- `version.yml` runs `ergon release ci version` when a run of `ci.yml` for a push to `main` succeeds, through `workflow_run`. It checks out the commit of that run. The command opens or updates the version pull request, and skips a commit that is no longer the head of `main`. The job does not need an OIDC token, so the refusal of crates.io does not apply.
- Every pull request that ergon proposes skips a commit that is no longer the head of its base: the version pull request, the upgrade of `baseline.yml`, and the weekly update of the baselines in ergon's own repository.
- `release.yml` publishes on a push to `main`, without a job that waits. Before the pack, `ergon release ci verify` checks that a run of `ci.yml` passed on the tree of the commit. It accepts a run of the commit itself, such as the run of a merge group, and a run of the head of a pull request that merged the commit with the same tree. Without such a run the job fails, and a rerun after the run of `main` passes publishes the release.
- A repository can store the client ID of a GitHub App in the variable `ERGON_APP_CLIENT_ID` and its private key in the secret `ERGON_APP_PRIVATE_KEY`. `version.yml` then opens the pull request with a token of the App, so its CI runs start without an approval.
- `ci.yml` gives the run of each push and of each merge group a concurrency group of its own. It still cancels a superseded run of a pull request.
- `ergon release ci wait` and the job wait are removed.

## Alternatives Considered

### The wait of ADR-0010

The release workflow would keep the job wait. It lost because the job occupies a runner for the length of the CI run, and the queue of the concurrency group of `ci.yml` adds to it.

### `workflow_run` for every job

`release.yml` would start when the run of `ci.yml` completes, and publish from that run. It lost because crates.io refuses the token exchange for that event. A provenance statement of npm or PyPI would also record the newest commit of `main` instead of the published commit.

### A publish on a tag that the release dispatches

The release would create the tags of the publish plan after the run of `ci.yml`, and dispatch a publish workflow on one of them. A dispatch from `GITHUB_TOKEN` starts a run, and `GITHUB_SHA` of a run on a tag is the commit of the tag. It lost because it tags before the uploads and needs an environment that allows the tags. It also adds a second release workflow, and none of the tools that we read publishes this way.

### The jobs of the release in `ci.yml`

The jobs of the release would follow the gate in `ci.yml` through `needs`. It lost because a failed publish would mark the status of `main` red, as ADR-0010 states.

### Branch protection alone

The release would publish on every push of a version commit and rely on required status checks, as changesets, release-please and release-plz do. It lost because ergon does not manage the settings of a repository (RFC-0004), and an administrator can merge past the rule. On 2026-10-08 the version pull request 6 was merged 14 seconds after it opened. Its CI run still waited for an approval, and GitHub completed that run at the merge with the conclusion `failure` and no job.

## Consequences

**Positive:**

- Each job of the release flow starts after the run that it needs has completed, so no runner idles.
- The publish runs on `push`, so crates.io accepts its token exchange, and provenance records the published commit.
- The version pull request opens only from a commit whose gate passed.
- A run of `ci.yml` for a push no longer waits for the run of an earlier push.

**Negative:**

- Without a GitHub App, a maintainer approves the CI run of each update of the version pull request.
- A version pull request that merges without a passing run publishes nothing until the run of `main` passes and a maintainer reruns the job verify.
- The merge of a version pull request runs the gate twice: on the pull request and on `main`.

**Neutral:**

- A run of `workflow_run` uses the workflow file of the default branch, so a change to `version.yml` takes effect once `main` contains it.

## References

| What | Where |
|---|---|
| The refusal of `workflow_run` by crates.io | https://github.com/rust-lang/crates.io/blob/main/src/controllers/trustpub/tokens/exchange/mod.rs, lines 120 to 129 |
| `GITHUB_SHA` of `workflow_run` and `workflow_dispatch` | https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows |
| The claim `sha` in the Sigstore certificate | https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md |
| The approval of runs of a pull request that `GITHUB_TOKEN` opens | https://docs.github.com/en/actions/concepts/security/github_token |
| release-plz: the checks of a pull request before a release | https://github.com/release-plz/release-plz/blob/main/website/docs/github/advanced.md |
| release-please: a token for CI on release pull requests | https://github.com/googleapis/release-please-action |
| changesets: Astro's release workflow | https://github.com/withastro/astro/blob/main/.github/workflows/release.yml |
| The token of a GitHub App in a workflow | https://github.com/actions/create-github-app-token |
