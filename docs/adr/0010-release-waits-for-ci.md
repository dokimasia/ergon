---
adr: 0010
title: The release workflow waits for the CI run of its commit
status: Superseded
date: 2026-10-08
supersedes: ADR-0006, in part
superseded-by: ADR-0016
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0010: The release workflow waits for the CI run of its commit

## Status

Superseded by ADR-0016

## Context

Every push to `main` starts `ci.yml` and `release.yml` in parallel. The run of `ci.yml` runs the gate, and its status is the status of `main`. The run of `release.yml` opens or updates the version pull request, or publishes the packages of the publish plan.

ADR-0006 keeps the trigger of `release.yml` at `push`, because crates.io refuses its OIDC token exchange for a workflow that `workflow_run` triggered. The release workflow cannot start after the CI workflow. ADR-0006 runs the gate inside the release workflow through `workflow_call`, and RFC-0002 does so only before a publish.

On 2026-10-08 the CI run of commit `768f7d7` failed on Windows, while the Release run of the same push updated version pull request 5. The job version had run beside the gate of its commit. A publish also ran the gate twice: once in the CI run of its push, and once in the release workflow.

The status of `main` is separate from the release flow, so the gate cannot move into the release workflow alone. A failed publish would otherwise mark the checks of `main` red.

## Decision

We will make the release workflow wait for the CI run of its commit. Every release step then follows a passed gate of the same commit, and each push runs the gate once.

- A job wait runs `ergon release ci wait` after select-mode, for the modes version and publish. The command finds the newest run of `ci.yml` for the commit of HEAD and the event `push` through the REST API of GitHub.
- The command asks every 15 seconds and gives GitHub 5 minutes to create the run. It exits 1 unless the run completes with the conclusion `success`.
- The jobs version and pack need the job wait.
- `ci.yml` keeps its trigger `push` and loses `workflow_call`, because no workflow calls it.
- `release.yml` keeps its trigger `push`, as ADR-0006 requires.

## Alternatives Considered

### The gate inside the release workflow for every mode

The job version would call `ci.yml` through `workflow_call` before it opens the version pull request. It lost because `ci.yml` keeps its own run on each push for the status of `main`. Every push to `main` would then run the gate twice.

### The gate inside the release workflow alone

`ci.yml` would lose its trigger `push`, and the release workflow would run the gate once for each push. It lost because the checks of `main` would belong to the Release workflow. A failed publish would then mark them red.

### One workflow for the gate and the release

The jobs of the release would follow the jobs of the gate in `ci.yml` through `needs`. It lost for the same reason: the status of the workflow on `main` would include every release job.

### A `workflow_run` trigger

The release workflow would start when the CI workflow completes. crates.io refuses the OIDC token exchange for that trigger, as ADR-0006 states.

## Consequences

**Positive:**

- A red `main` does not open or update a version pull request, and it publishes nothing.
- The gate runs once for each push to `main`, also for a publish.
- `main` keeps the status of its own CI run.

**Negative:**

- The job wait keeps a Linux runner busy for the length of the CI run of its commit. GitHub charges a private repository $0.006 a minute for that runner on 2026-10-08, and the runners of a public repository are free.
- A rerun that turns a failed CI run green leaves the Release run of its commit failed. A rerun of the failed jobs of the Release run then continues the release.
- A newer push can cancel the pending CI run of an older commit. The Release run of the older commit then fails at the job wait, and the Release run of the newer push continues.
- The job wait needs the permission `actions: read`.

**Neutral:**

- The concurrency group of `ci.yml` no longer needs the prefix that kept a called run apart from the group of its caller.

## References

| What | Where |
|---|---|
| The runs of a workflow, newest first, filtered by commit and event | https://docs.github.com/rest/actions/workflow-runs#list-workflow-runs-for-a-workflow |
| crates.io refuses the OIDC token exchange for `workflow_run` | https://github.com/rust-lang/crates.io/blob/ed382369c65cb636470f09c031a0b68352092079/src/controllers/trustpub/tokens/exchange/mod.rs |
| The prices of GitHub Actions | https://docs.github.com/en/billing/concepts/product-billing/github-actions |
