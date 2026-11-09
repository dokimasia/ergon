---
adr: 0073
title: A job keeps the directories of its caches in the cache of GitHub Actions
status: Accepted
date: 2026-11-09
supersedes: none
superseded-by: none
rfc: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0073: A job keeps the directories of its caches in the cache of GitHub Actions

## Status

Accepted

## Context

The producer of the GitHub files renders every job of `ci.yml` and `nightly.yml` from a `workflow.Job` that another producer contributes. A job keeps one directory between its runs: the tool directory of ergon, through the action `cache` of the section `github` (ADR-0031). The Go baseline needs more directories:

- The job `mutate-go` of `nightly.yml` keeps the history of dokimi-mutate-go. Every run changes the history, and ADR-0065 saves it after every run, also after a run that fails.
- The job `check-go` of `ci.yml` keeps the cache of golangci-lint, which golangci-lint-action keeps between builds with `@actions/cache`.

A producer of a language has no pin of `actions/cache`. The section `github` pins the action once for every workflow.

## Decision

We will let a job declare each directory that it keeps in the cache of GitHub Actions as a `workflow.Cache` in `Job.Caches`, because the producer of the GitHub files then renders every cache with the one pin of `actions/cache`.

- A cache has an ID, a name for the names of its steps, a path, the patterns of the files of its key, an interval in days, and the name of an input of `workflow_dispatch`.
- The key of a cache starts with its ID, the system, the architecture, the job and the runtime version of a matrix, as the key of the tools of ergon does. A cache with an interval then adds the number of whole intervals since the Unix epoch, which a step of the job computes. This prefix is the restore key, so a job restores the newest cache of its prefix.
- A cache with files ends its key in the digest of the files. One step of `actions/cache` restores the cache before the steps of the job and saves it after them when the key missed, as the cache of the tools of ergon and golangci-lint-action do.
- A cache without files ends its key in the ID and the attempt of the run. `actions/cache/restore` restores the cache before the steps of the job, and `actions/cache/save` saves it after them with `if: always()`, so a run that fails keeps what it wrote.
- The input is a boolean input of `workflow_dispatch` of `nightly.yml`, true by default. A run with the input false skips the restore. Only a job of `nightly.yml` takes an input, because `ci.yml` has no `workflow_dispatch`.

## Alternatives Considered

### A pin of actions/cache in each section of a language

Each language would pin `actions/cache` in its key `ci.actions` beside the action of its setup. Its producer would render the steps of its caches. It lost because two pins of one action can differ. Every language would also render the same steps.

### A directory inside the tool directory of ergon

A job would keep its directory inside the tool directory of ergon, whose cache the job already keeps. It lost because the tool cache saves only when its key misses. `ergon tool prune` also removes every file that the options do not name before the cache saves.

## Consequences

**Positive:**

- A producer keeps a directory between the runs of a job without a pin of its own.
- A job restores the newest cache of its prefix, also when the files of its key change.

**Negative:**

- A cache without files saves a new entry in every run. The cache of GitHub Actions removes an entry that no job read in seven days, and a repository has 10 GB of cache.
- `ergon tool ci prune` deletes only the replaced caches of the tools of ergon.

## References

| What | Where |
|---|---|
| The action that restores a cache | <https://github.com/actions/cache/blob/main/restore/README.md> |
| The action that saves a cache, and its use with `if: always()` | <https://github.com/actions/cache/blob/main/save/README.md> |
| The cache of golangci-lint-action | <https://github.com/golangci/golangci-lint-action/blob/main/src/cache.ts> |
| The eviction of the cache of GitHub Actions | <https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching> |
