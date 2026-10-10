---
adr: 0032
title: nightly.yml deletes the tool caches that newer caches replaced
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

# ADR-0032: nightly.yml deletes the tool caches that newer caches replaced

## Status

Accepted

## Context

Each change of `.ergon.yaml` or of the lock makes each job that runs tools save a cache under a new key, on each of its runners. A job restores its exact key, or the newest cache of its restore key. A replaced cache only serves a run whose key equals the key of that cache, such as a run of a pull request whose base has the earlier `.ergon.yaml`.

GitHub deletes a cache 7 days after its last restore. When the caches of a repository pass 10 GB, GitHub deletes caches in the order of their last restore, the oldest first.

On 2026-10-10 dokimasia/ergon had these caches:

- 60 caches of 10.55 GB in all, most of them caches of `actions/setup-go`
- 23 tool caches of 1,098 MB
- 19 tool caches of 911 MB that a newer cache of the same job and branch replaced

`nightly.yml` rendered only for a repository with a nightly step of a language.

## Decision

We will add the job `prune-tools` to `nightly.yml` of each repository with a job that runs tools, because a tool cache that a newer cache of its job replaced uses the storage of the repository until GitHub deletes it.

- The job runs `ergon tool ci prune` on the Linux runner of the section `github`, with the permissions `actions: write` and `contents: read`.
- `ergon tool ci prune` lists the caches whose keys start with `ergon-tools-`. Caches with the same ref and the same key up to its last dash belong to one job. The command keeps the newest cache of each job and deletes the others. Of two caches of the same time, it keeps the cache with the higher ID.
- `nightly.yml` renders for each repository with a job that runs tools, also when no language has a nightly step.

## Alternatives Considered

### A step of gh and jq in the workflow

The job would run `gh cache list` and `gh cache delete` with a filter of jq. It lost because the managed workflows run their logic through commands of ergon, such as `ergon release ci skip`, whose tests cover that logic. A filter in a workflow has no test.

### The replaced caches of every action

The prune could also cover the replaced caches of `actions/setup-go` and of the other setup actions, which were most of the 10.55 GB. It lost because ergon does not define the keys of those actions. A workflow of a repository can use two current caches whose keys differ only after their last dash.

## Consequences

**Positive:**

- The storage of a repository keeps one tool cache for each job, system, architecture, runtime version and branch.

**Negative:**

- Each repository with a job that runs tools now has `nightly.yml`, and its job `prune-tools` runs each night.
- The job deletes whole caches. The newest cache of a job still contains the tools of earlier versions, which a restore by its restore key brought into it.
- After a prune, a run of a pull request whose base has the earlier `.ergon.yaml` does not find the cache of its key, and restores the newest cache of its job.

**Neutral:**

- GitHub deletes the newest cache of a closed pull request 7 days after its last restore, as before.

## References

| What | Where |
|---|---|
| The nightly workflow | ADR-0020 |
| The restore of the newest tool cache of a job | ADR-0031 |
| The endpoints that list and delete caches | https://docs.github.com/en/rest/actions/cache |
| The permission `actions: write` of a workflow that deletes caches | https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manage-caches |
| The deletion of caches by GitHub | https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching |
