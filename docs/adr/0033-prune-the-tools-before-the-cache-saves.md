---
adr: 0033
title: A job removes the tools that the options do not name before its tool cache saves
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

# ADR-0033: A job removes the tools that the options do not name before its tool cache saves

## Status

Accepted

## Context

A job that runs tools restores the newest cache of its job when its key misses. After the job succeeds, `actions/cache` saves the whole tool directory under the new key. The restored directory contains the tools of the earlier key, so each new version of a tool, and each new version of Go, adds a build to the newest cache of the job. The content of a saved cache cannot change, and `actions/cache` saves nothing after a hit of the exact key.

`ergon tool run` installs each tool under a path that the tool's kind, name, version and platform decide. It installs a Go module also under the version of the go command. The options of the repository name the version of every tool.

## Decision

We will end each job that runs tools with `ergon tool prune`, which removes each file of the tool directory that is not the install of a tool of the options, because a cache that a job saves should contain only the tools that the options of its commit name.

- `ergon tool prune` keeps the install of each tool of each section: the program of a release binary and of a Go module, the directory of the program that `golangci-lint custom` builds with the plugins of its section, the root of a crate, the jar of a Maven artifact, and the project of the Composer packages of a section. It removes every other file and directory, such as an earlier version of a tool, a build for another version of Go, and the files of an install that did not finish.
- The step runs only when the key of the cache missed, because `actions/cache` saves nothing after a hit.
- When `go env GOVERSION` does not report a version, as in a job without Go, the command keeps every Go module of the directory.
- The command walks and removes through an `os.Root` of the directory, so a symbolic link in the directory does not lead a removal outside it.

## Alternatives Considered

### The job prune-tools of nightly.yml alone

The nightly job deletes each tool cache that a newer cache of its job replaced. It lost because it can only delete a whole cache. The newest cache of a job contains the builds of each earlier version that the job restored and saved again, and the nightly job cannot remove them from that cache.

## Consequences

**Positive:**

- The cache that a job saves contains the tools that the options of its commit name, and no build of an earlier version.

**Negative:**

- Each job that runs tools has one more step.
- `ergon tool prune` on a workstation also removes the tools of the other repositories that share the tool directory.

**Neutral:**

- When the key of a job matches a cache, the job skips both the prune and the save, as before.

## References

| What | Where |
|---|---|
| The steps of a job, and the cache of the tools of ergon | RFC-0004, Jobs and Tools |
| The restore of the newest tool cache of a job | ADR-0031 |
| The deletion of the tool caches that newer caches replaced | ADR-0032 |
| A saved cache, whose content cannot change, and the save after a miss | https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching |
| The save of actions/cache after a job that succeeds | https://github.com/actions/cache, v6.1.0, `action.yml` |
