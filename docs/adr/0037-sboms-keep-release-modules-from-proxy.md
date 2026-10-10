---
adr: 0037
title: syft does not read a module of a release from the module proxy
status: Accepted
date: 2026-10-10
supersedes: RFC-0007, in part
superseded-by: none
rfc: RFC-0007
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0037: syft does not read a module of a release from the module proxy

## Status

Accepted

## Context

The managed GoReleaser configuration writes an SBOM of each archive and each package with syft, and its argument `--enrich all` makes syft download the zip of each Go module of a binary from the module proxy to read its license. The job pack of `release.yml` runs before the publish creates the tags of the release. A binary records its own module at the version of the release.

On 2026-10-10 syft wrote the SBOMs of ergon 0.8.1 at 13:48 UTC, and the publish pushed the tags at 13:49 UTC. Afterwards sum.golang.org still returned "unknown revision" for `go.dokimi.dev/ergon@v0.8.1`, so `go install go.dokimi.dev/ergon/cmd/ergon@v0.8.1` failed. lint-go 0.2.0 failed the same way. In a probe, syft with `--enrich all` requested the zip of the main module of a binary at its version from a local proxy, and requested nothing with `GOPRIVATE` set to that module.

GoReleaser 2.18.2 runs an SBOM command with eight variables of its own environment alone: `HOME`, `USER`, `USERPROFILE`, `TMPDIR`, `TMP`, `TEMP`, `PATH` and `LOCALAPPDATA`. It adds the `env` of the entry of `sboms`, which it renders as templates. syft adds the patterns of `GOPRIVATE` and `GONOPROXY` to the modules that it does not read from the proxy.

## Decision

We will set `GOPRIVATE` in the `env` of each entry of `sboms` of the managed GoReleaser configuration to the modules of the release, because syft then does not read a module of the release from the proxy before the publish creates its tag.

- `ergon release pack` passes the paths of the modules of the release to GoReleaser in `ERGON_MODULES`, separated by commas.
- Each entry of `sboms` has `env: ["GOPRIVATE={{ .Env.ERGON_MODULES }}"]`.

## Alternatives Considered

### The argument --enrich all removed

syft would not read a license from the proxy. It lost because the SBOMs would then lose the licenses of every module outside the repository.

### GOPRIVATE in the environment of GoReleaser

`ergon release pack` would set `GOPRIVATE` for GoReleaser alone. It lost because GoReleaser passes no other variable of its environment to an SBOM command.

### The template variable ModulePath

The entries of `sboms` would set `GOPRIVATE={{ .ModulePath }}`, without a variable from `ergon release pack`. It lost because GoReleaser takes `ModulePath` from the first line of `go list -m`. In a workspace, that line is the path of the first module of `go.work`, which is not the module of a command in another directory.

## Consequences

**Positive:**

- The tag of the root module of a release resolves through the proxy as soon as the publish pushes it.

**Negative:**

- The managed configuration reads `ERGON_MODULES`, so a run of GoReleaser outside `ergon release pack` sets it, as it sets `ERGON_VERSION`.

**Neutral:**

- An SBOM still has no license from the proxy for a module of the release, whose zip does not exist when syft runs. syft still reads the licenses of the other modules from the proxy.

## References

| What | Where |
|---|---|
| The SBOMs of a release | RFC-0007 |
| The environment of an SBOM command | `internal/pipe/sbom/sbom.go` of GoReleaser v2.18.2 |
| The module path of a GoReleaser project | `internal/pipe/gomod/gomod.go` of GoReleaser v2.18.2 |
| The modules that syft does not read from the proxy | `syft/pkg/cataloger/golang/config.go` of syft v1.54.1 |
| The modules of a release kept out of the proxy in CodeQL | ADR-0036 |
