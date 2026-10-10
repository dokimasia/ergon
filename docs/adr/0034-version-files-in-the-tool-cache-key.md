---
adr: 0034
title: The key of a tool cache covers the files that set the version of the toolchain
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

# ADR-0034: The key of a tool cache covers the files that set the version of the toolchain

## Status

Accepted

## Context

The key of the tool cache of a job is the system, the architecture, the job, the runtime version of a matrix, and the digest of `.ergon.yaml` and `.ergon/init.lock`. The job of Go installs the version of `go.work` with `actions/setup-go`, and `ergon tool run` builds each Go module once for each version of the go command.

When `go.work` moves to a new version of Go and `.ergon.yaml` does not change, the key of the job still matches its cache. Each run then builds the Go modules again for the new version, and `actions/cache` saves nothing after a hit of the exact key.

## Decision

We will add the files from which the setup of a job reads the version of its toolchain to the digest of the key of its tool cache, because `ergon tool run` builds the Go modules of a job with the version of Go that the job set up.

- The setup of a job states these files as a pattern of `hashFiles` in `VersionFiles` of `workflow.Setup`.
- The setups of Go state `go.work`, which `actions/setup-go` reads.
- A setup without version files leaves the key as before.

## Alternatives Considered

### The key without the version files

The key would not change when `go.work` moves to a new version of Go. It lost because each run of each job would then build every Go module again and save nothing.

## Consequences

**Positive:**

- A new version of Go in `go.work` starts a new tool cache of each job, with the builds of its Go modules for that version.

**Negative:**

- A change of `go.work` that keeps the version of Go, such as a new module of the workspace, also makes each job save a new cache.

**Neutral:**

- Only the setups of Go state version files. `ergon tool run` installs the tools of the other toolchains under paths without the version of the toolchain.

## References

| What | Where |
|---|---|
| The cache of the tools of ergon | RFC-0004, Tools |
| The restore of the newest tool cache of a job | ADR-0031 |
| The removal of the tools that the options do not name | ADR-0033 |
| The version file of actions/setup-go | https://github.com/actions/setup-go, `go-version-file` |
