---
adr: 0031
title: The tool cache of a job has its key without the digest as its restore key
status: Accepted
date: 2026-10-10
supersedes: RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0031: The tool cache of a job has its key without the digest as its restore key

## Status

Accepted

## Context

A job of `ci.yml` or `nightly.yml` that runs tools keeps the tool directory of ergon in the cache of GitHub Actions. The key of the cache is the system, the architecture, the job, the runtime version of a matrix, and the digest of `.ergon.yaml` and `.ergon/init.lock`. A changed tool or a new version of ergon changes the key, so the job installs every tool again.

The digest also changes with each option that is not a tool. On 2026-10-10 the macOS job of ergon's commit 9ed9473 missed its cache and downloaded 243 modules: 201 for golangci-lint, 40 for ergon and 2 for govulncheck. The Linux job of techne missed its cache on the same day.

`ergon tool run` installs each tool under its kind, its name, its version and its platform. It installs a Go module also under the version of the go command. A tool directory of an earlier key contains each tool whose version did not change, and `ergon tool run` reuses those tools.

## Decision

We will give the cache step of each job that runs tools its key without the digest as the restore key, because `ergon tool run` reuses each tool of a restored directory whose version did not change.

- The restore key is `ergon-tools-<system>-<architecture>-<job>-`, with the runtime version of a matrix before the last dash.
- When the key misses, `actions/cache` restores the newest cache whose key starts with the restore key, in the scope of the run. The job saves the directory under its new key after it succeeds.

## Alternatives Considered

### The key without a restore key

Each change of the digest would miss the whole cache. It lost because a change of an option that is not a tool installs every tool again, as the 243 modules of the macOS job of 2026-10-10 show.

### The release binary of golangci-lint

The job would download the release binary of golangci-lint in place of its `go install`. It lost because golangci-lint refuses a module whose Go version is newer than the Go that built the binary. A repository could then move to a new Go only after golangci-lint released a binary built with it.

## Consequences

**Positive:**

- A change of `.ergon.yaml` or of the lock installs only the tools whose versions changed.

**Negative:**

- A job that restored an earlier cache saves the whole directory under its new key, with the tools of the earlier versions. Each new version of a tool or of Go adds a build to the newest cache of the job. The cache becomes smaller only after a run that does not find a cache of the job.

**Neutral:**

- The key does not change, so a run whose key matches a cache restores that cache as before.

## References

| What | Where |
|---|---|
| The cache of the tools of ergon | RFC-0004, Tools |
| The restore keys of actions/cache, and the newest of several caches that match | https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching |
| The check of golangci-lint against the Go that built it | `pkg/goutil/version.go:32` of golangci-lint v2.14.0 |
