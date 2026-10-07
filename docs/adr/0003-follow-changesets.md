---
adr: 0003
title: Follow changesets' file format, configuration and planning rules
status: Accepted
date: 2026-09-24
supersedes: none
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0003: Follow changesets' file format, configuration and planning rules

## Status

Accepted

## Context

`ergon release` needs a place where a pull request states what it releases, a rule for which dependents move with it, and a changelog format. stealthscale/stealth already runs changesets 3.0.2 through `changesets/action` v2, with 80 pending changeset files and `.changeset/config.json`. changesets itself publishes to npm only, and its maintainers have deferred other ecosystems to a later major version.

The tools that release several ecosystems differ from changesets in their file format, their configuration and their cascade. knope and sampo read change files, but both keep the quotes around a changeset's package name, so a changesets file matches no package in either.

## Decision

We will read changesets' `.changeset/*.md` files and `.changeset/config.json`, and plan releases by changesets v3's rules, because stealthscale/stealth then switches to ergon without rewriting its changesets, its configuration or its changelogs.

ergon writes `CHANGELOG.md` in the formats of `@changesets/cli/changelog` and `@changesets/changelog-github`, writes the publish plan in changesets v3's format, and adds only the `entrypoints` key to the configuration.

## Alternatives Considered

### sampo for every language except Go, and ergon for Go

sampo has change files, registry-checked publishing in six ecosystems and a transitive cascade. It lost because a repository with Go and another language would run two tools with two change-file directories and two CI flows, and sampo has no Gradle support.

### knope

knope tags Go modules with the right prefixes. It lost because it releases no dependents, never rewrites a Go `require` line, and rejects changesets' quoted keys.

### Nx release with version plans

Nx has file-based version plans and a per-language extension point. It lost because every Go repository would need Node and Nx. The Go plugins also leave `go.mod` unchanged.

### Conventional Commits as the bump source

The earlier ergon and release-please infer the bump from commit subjects. It lost because a commit subject is fixed at merge, while a changeset is reviewed with the pull request and can be edited until the release.

### Configuration in `.ergon.yaml`

The other ergon commands read `.ergon.yaml`. It lost because stealthscale/stealth would have to move its configuration, and changesets' keys already cover what `release` needs.

## Consequences

**Positive:**

- A repository that runs changesets switches by changing the `uses:` lines of its release workflow.
- changesets 3.0.3 is a reference implementation: its output on the same changeset files tests ergon's planner and changelog.

**Negative:**

- ergon has to track changesets' planning rules and changelog formats, and each changesets release can introduce a difference.
- changesets loads changelog functions from JavaScript modules. ergon supports the two built-in formats only, so a repository with a custom changelog module cannot switch.
- ergon refreshes lockfiles and changesets does not, so the comparison with changesets excludes lockfiles.

**Neutral:**

- Release configuration is in `.changeset/config.json`. The configuration of other ergon commands remains in `.ergon.yaml`.

## References

| What | Where |
|---|---|
| changesets v3 planner | https://github.com/changesets/changesets/blob/main/packages/assemble-release-plan/src/determine-dependents.ts |
| changesets and other ecosystems | https://github.com/changesets/changesets/pull/2124 |
| knope has no dependent cascade | https://github.com/knope-dev/knope/issues/1822 |
| sampo adapters | https://github.com/bruits/sampo/blob/main/crates/sampo-core/src/adapters.rs |
