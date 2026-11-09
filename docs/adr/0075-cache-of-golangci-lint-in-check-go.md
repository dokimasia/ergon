---
adr: 0075
title: The job check-go keeps the cache of golangci-lint
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

# ADR-0075: The job check-go keeps the cache of golangci-lint

## Status

Accepted

## Context

The job `check-go` of `ci.yml` keeps the tools of ergon in the cache of GitHub Actions, and setup-go keeps the build cache and the module cache of Go. No step keeps the cache of golangci-lint, so `make lint-go` analyses every package of every module again in each run. The run 41931688092 of thesm-os/theke on 2026-11-04 spent 7.2 minutes in `lint-go` on ubuntu-26.04 and 2.4 minutes on macos-26. On windows-2025 the `run.timeout` of 5 minutes ended golangci-lint, and the job reached its limit of 30 minutes.

golangci-lint 2.15.0 keeps its cache in `GOLANGCI_LINT_CACHE`, and in the directory `golangci-lint` under `os.UserCacheDir()` when the variable is not set. The managed Makefile does not set the variable.

golangci-lint-action keeps the same directory with `@actions/cache`:

- The key is `golangci-lint.cache-`, the system, the number of whole intervals since the Unix epoch and the SHA-1 of `go.mod`.
- The restore key ends after the interval, so a run restores the newest cache of its interval.
- The action saves the cache only when the key missed.
- The interval is the input `cache-invalidation-interval`, 7 days by default. A new interval starts an empty cache, so the directory does not grow without a bound.

## Decision

We will keep the cache of golangci-lint in the job `check-go` as a cache of ADR-0073 with files and an interval, because golangci-lint-action keeps it the same way.

- The directory is `golangci-lint` under the user cache directory of each system: `~/.cache` on Linux, `~/Library/Caches` on macOS and `~/AppData/Local` on Windows.
- The files of the key are each `go.sum`, `go.work`, `.golangci.yml`, `.ergon.yaml` and `.ergon/init.lock`. A change of the dependencies, of the version of Go, of the rules or of the version of golangci-lint changes the results of most packages, so the run saves a new cache.
- The interval is the key `go.lint.cache-invalidation-interval`, 7 days at the baseline. 0 turns the cache off.
- The job keeps the cache only when `go.check` names `lint`.

## Alternatives Considered

### A key that each run makes new

Each run would save a new cache and restore the newest one, as the request in dokimasia/ergon#76 states. It lost because each run on each runner then saves an entry. Once a repository reaches its 10 GB of cache, GitHub removes the entries that no job read for the longest time. The history of the nightly mutation is read once a day, so the entries of lint would remove it first.

### The go.mod of the root, as golangci-lint-action

The key would end in the digest of the root `go.mod` alone. It lost because a workspace keeps its dependencies in the `go.sum` of each module. A new version of golangci-lint or new rules would also restore a cache that the run cannot use, and the run would not save a new cache until the interval ends.

## Consequences

**Positive:**

- `lint-go` analyses only the packages that changed since the run that saved the cache of its key.

**Negative:**

- A key keeps the cache of the run that saved it. Later runs with the same key analyse every package that changed since that run, until the files of the key or the interval change.
- Each runner of the job saves an entry for each interval and each digest of the files of the key.

## References

| What | Where |
|---|---|
| The cache of golangci-lint-action | <https://github.com/golangci/golangci-lint-action/blob/main/src/cache.ts> |
| The input `cache-invalidation-interval` of golangci-lint-action | <https://github.com/golangci/golangci-lint-action/blob/main/README.md> |
| The cache directory of golangci-lint 2.15.0 | <https://github.com/golangci/golangci-lint/blob/v2.15.0/internal/go/cache/default.go> |
| The request for the cache | <https://github.com/dokimasia/ergon/issues/76> |
