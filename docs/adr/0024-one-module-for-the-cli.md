---
adr: 0024
title: ergon has one module for its CLI and one for ergon-go-vet
status: Proposed
date: 2026-10-09
supersedes: ADR-0001, with ADR-0008 and RFC-0001 in part
superseded-by: none
rfc: RFC-0008
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0024: ergon has one module for its CLI and one for ergon-go-vet

## Status

Proposed

## Context

ergon has 14 Go modules: the root module, `core`, `service` and one module for each of 11 languages. A module per language keeps the dependencies of each language in a `go.sum` of its own. It also lets the tests of a language run with one toolchain.

The binary links every language, and no other repository imports a module of ergon. Every language module requires `service` for its tests. A Go module that requires a released module is released with it, so a fix of two files of `service` planned 13 releases. The Go module also contains ergon-go-vet. Its pin in the Go plugin is a literal of the same module, so each weekly update of the pin starts another release of the module.

## Decision

We will build the CLI as the one module `go.dokimi.dev/ergon` and ergon-go-vet as the module `go.dokimi.dev/ergon/vet`, because a module boundary inside the CLI isolates nothing that another repository uses.

Each module costs a tag, a changelog and a release for every change of a module that it requires.

- The vet module requires `golang.org/x/tools`, and `go.dokimi.dev/assert` for its tests. Other repositories install ergon-go-vet from it with `go install`.
- The two modules do not require each other, so `go.work` lists both and has no replace.
- The Go plugin pins ergon-go-vet as a literal, as it pins every other tool. A release of the vet module leaves the files of the root module unchanged.
- The 13 retired module paths keep their tags on the module proxy and get no further release.

## Alternatives Considered

### A module per language, without the test imports

The rendering tests of the language modules move to the root module, and each language module requires `core` alone. It lost because the modules keep 13 `go.mod` and `go.sum` files with their replaces, 14 tags for a change of `core`, and 14 runs of golangci-lint in each gate. The pin of ergon-go-vet also still changes its own module.

### One module, with ergon-go-vet

ergon-go-vet is a command of the root module, pinned at the version of the running ergon. It lost because `go install` of the command then loads the module graph of the CLI, whose `go.mod` lists 56 requirements, and because every release of ergon is then a release of ergon-go-vet, which each repository builds again.

## Consequences

**Positive:**

- A change of the CLI releases one module.
- A release of the vet module leads to one release of the root module, through the next baseline update of the pin.
- The gate runs golangci-lint, the tests and the vulnerability scan twice instead of 14 times.

**Negative:**

- ergon's own release no longer has requirements between sibling modules, so only the tests and other repositories exercise that part of the release flow.
- One `go.sum` lists the dependencies of every language.
- `go test ./...` at the root runs the tests of every language.

**Neutral:**

- The merge of the modules keeps every import path. The move of the packages under `internal/` is a decision of its own.
- The root module keeps its tags, `v<version>`.
