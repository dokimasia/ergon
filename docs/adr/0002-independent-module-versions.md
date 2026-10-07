---
adr: 0002
title: Independent module versions, with the root module as an entry point
status: Accepted
date: 2026-09-24
supersedes: none
superseded-by: none
rfc: RFC-0001
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0002: Independent module versions, with the root module as an entry point

## Status

Accepted

## Context

ergon has seven modules and one binary. `go install go.dokimi.dev/ergon/cmd/ergon@<version>` ignores `go.work` and every `replace`, and resolves versions from the root module's `require` lines only. Minimal version selection takes the highest of the required minimums, so it never selects a sibling release that no `go.mod` requires.

A sibling release is therefore in the binary only when the root module is released with a rewritten `require`. changesets bumps a dependent only when the new version leaves its range. A Go requirement is a minimum that every later version satisfies, so that rule never releases the root module.

Nothing outside the repository imports ergon's modules, so a module version makes no compatibility promise to anyone.

## Decision

We will version each ergon module independently and release the root module whenever a module it depends on is released, because `go install` builds the binary from the root module's `require` lines only.

`.changeset/config.json` names the root module in `entrypoints`. The planner gives an entry point a patch when a module it depends on, directly or through another module, is released, and rewrites its `require` lines.

## Alternatives Considered

### One version for every module

A `fixed` group puts all seven modules at one version, released together. OpenTelemetry Go groups its stable modules this way because its users import them together. It lost because nobody imports ergon's modules, so a change to `core` would tag seven modules, and six of those tags would point at unchanged content.

### Release every dependent of a released module

`updateInternalDependents: "always"` releases every module that depends on a released one. The earlier ergon cascaded this way. It lost because a change to `core` would still release all seven modules, since every module depends on `core`.

## Consequences

**Positive:**

- A change to one module releases two modules: that module and the root module.
- Each module's version and changelog show when that module changed.
- The root module's tag is the ergon version, and goreleaser builds from it.

**Negative:**

- `.changeset/config.json` has a key, `entrypoints`, that changesets does not define.
- The root module is released far more often than any other module.
- `GOWORK=off` builds of the root module resolve siblings from their tags, so the root module cannot carry a `replace` for a sibling. `go install <pkg>@<version>` refuses a module whose `go.mod` contains one.

**Neutral:**

- Other Go repositories with a binary in a multi-module workspace, such as techne, use the same `entrypoints` setting.

## References

| What | Where |
|---|---|
| Minimal version selection | https://go.dev/ref/mod#minimal-version-selection |
| `go install pkg@version` and `replace` | https://go.dev/ref/mod#go-install, `cmd/go/internal/load/pkg.go:3462-3468` in go1.27.1 |
| OpenTelemetry Go module sets | https://github.com/open-telemetry/opentelemetry-go/blob/main/versions.yaml |
