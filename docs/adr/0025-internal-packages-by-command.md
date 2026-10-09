---
adr: 0025
title: The packages of ergon are internal and grouped by command
status: Proposed
date: 2026-10-09
supersedes: RFC-0001, in part
superseded-by: none
rfc: RFC-0008
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0025: The packages of ergon are internal and grouped by command

## Status

Proposed

## Context

The module `service` contains the packages of four commands, the clients of git and GitHub, and the base producers, under the name of a layer. These packages, and the packages of `core` and of the languages, are public, although no other repository imports them.

The command layer also contains the code of protocols. `internal/cli/upgrade.go` fetches `checksums.txt` over HTTP and parses GOPROXY, while the tool runner already downloads the assets of releases and the pin resolver already reads the module proxy.

## Decision

We will put every package of the CLI under `internal/` and group the packages by command, because ergon is a command and other repositories do not import its packages.

A directory named after a layer groups packages that change for different reasons, so the command packages and the clients of git and GitHub are directories of their own.

- The kernel is `internal/core`. The command packages are `internal/baseline`, `internal/release`, `internal/licenses`, `internal/tool` and `internal/pin`. The clients are `internal/vcs` and `internal/forge`, and the plugins are `internal/lang/<language>`.
- `internal/app` registers the plugins and lists the base producers. It is the only package that imports a plugin.
- `internal/cli` parses the flags of a command, calls one command package and writes the output. The lookup of a digest in `checksums.txt` moves to `internal/tool`, the parsing of GOPROXY to `internal/pin`, and the proposal of the working tree to `internal/release`.
- depguard checks the imports of each directory.

## Alternatives Considered

### Public packages

The packages keep their paths, such as `go.dokimi.dev/ergon/core/version`. It lost because a public package promises an API that ergon does not keep. The paths of the root module would also overlap the retired module paths, such as `go.dokimi.dev/ergon/lang/go`, which the module proxy keeps.

### The directory service

The command packages and the clients remain under `service/`. It lost because the name states a layer and no command, and the clients of git and GitHub change for other reasons than the commands that call them.

## Consequences

**Positive:**

- The go command refuses an import of ergon's packages from any package outside `go.dokimi.dev/ergon/...`, so a package can change its API without a release that announces the change.
- The code of a command is in the package with the name of the command.

**Negative:**

- The move rewrites 824 import lines in 266 files, and every open branch conflicts with it.
- Each import path is longer by `internal/`.
- pkg.go.dev does not show the documentation of the packages.

**Neutral:**

- A plugin keeps its structure inside `internal/lang/<language>`: a root package with `Register`, and one package for each command that it supports.
