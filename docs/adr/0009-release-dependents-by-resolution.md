---
adr: 0009
title: Dependent releases by consumer resolution
status: Accepted
date: 2026-10-07
supersedes: ADR-0002 and ADR-0003, in part
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0009: Dependent releases by consumer resolution

## Status

Accepted

## Context

A release of a package matters to a consumer only once the consumer's resolver picks the new version.

- npm, Cargo, pip and uv pick the newest version that a requirement admits.
- Go's minimal version selection keeps the build list when a dependency releases a new version. A release is in a consumer's build only once some `go.mod` in that build requires it.
- Maven takes the nearest definition of a dependency, and Gradle takes the highest requested version. A published POM states the version that its build used.

changesets releases a dependent only when the new version leaves the dependent's range. Where the resolver picks the newest admitted version, that rule gives every consumer the new version. For Go and Gradle it gives the new version to nobody who installs a dependent, because the dependent's published requirement names the old version.

ADR-0002 releases the root module of a Go repository through the key `entrypoints`, so the binary gets the new version. The consumers of the other dependents do not. A user of `go.dokimi.dev/ergon/service` gets a fix in `core` only after `service` is released for another reason.

Tools that release the packages of one repository follow three policies:

- The AWS SDK for Go v2 releases at patch every module that requires a changed module, directly or through other modules, and raises their `require` lines. Its tool runs `calculaterelease` and then `updaterequires`.
- release-please's workspace plugins, Nx 22, Lerna and sampo release every dependent at patch, in every ecosystem.
- cargo-release rewrites the requirements of dependents and releases none of them.

## Decision

We will release a dependent when a consumer that resolves its requirement would not receive the new version, because that rule gives every release to the consumers of every dependent. Wherever the resolver picks the newest version that a range admits, the rule is changesets' rule.

Each toolchain classifies a version under a requirement with `Versioner.Resolve`:

- **Excluded:** the requirement does not admit the version.
- **Pinned:** the requirement admits the version, and a consumer that resolves the requirement alone gets the version that it names.
- **Selected:** a consumer that resolves the requirement alone gets the version.

The planner releases a dependent at patch when the new version is Excluded or Pinned under a requirement outside the dev section. With `updateInternalDependents: "always"`, a Selected version also releases it. `version` rewrites a requirement on a released sibling when the new version is Excluded or Pinned under it, and a requirement on any other sibling whose current version is Pinned under it. The key `entrypoints` no longer exists.

## Alternatives Considered

### changesets' rule, with a list of entry points

The rule of ADR-0002 releases only the programs that `entrypoints` names. A fix in `core` then never gets into the build of a user of a library module such as `service`. Each Go repository would also have to name its programs by hand.

### Release every dependent in every ecosystem

release-please, Nx 22, Lerna and sampo do this. It lost because the consumers of npm, Cargo and Python packages already receive a new version through the range, so those releases give them nothing. An npm plan would also differ from changesets' plan on the same files, which removes changesets as the reference implementation that ADR-0003 relies on.

### Rewrite requirements without releasing dependents

cargo-release does this. It lost because a raised `require` line in a module without a release is in no consumer's build, and `main` would contain `go.mod` changes that no tag contains.

### One version for every module

ergon's modules have independent versions, as ADR-0002 decides, and a fixed group would also release modules that require nothing that changed. OpenTelemetry Go releases its modules in sets that share one version.

## Consequences

**Positive:**

- One rule serves every toolchain. The planner has no branch for Go, and `.changeset/config.json` has only changesets' keys.
- A released package requires its siblings at the versions it was tested with.
- No npm requirement is Pinned, so an npm plan is the plan of changesets 3.0.3 on the same files.

**Negative:**

- A Go module's release also releases the modules that require it, directly or through other modules. A fix in ergon's `core` releases all fourteen modules, and thirteen of those releases change only `go.mod`, `go.sum` and `CHANGELOG.md`.
- A dependent takes a patch even when its dependency breaks an API that the dependent re-exports.
- Maven's nearest definition can still give a consumer an older version through a declaration nearer to the consumer.

**Neutral:**

- `updateInternalDependents` keeps changesets' two values. `"out-of-range"` releases a dependent whose consumers would not receive the new version, which is changesets' meaning for npm.

## References

| What | Where |
|---|---|
| Minimal version selection keeps the build list | https://go.dev/ref/mod#minimal-version-selection |
| Cargo picks the highest compatible version | https://doc.rust-lang.org/cargo/reference/resolver.html |
| pip installs the latest version that satisfies the constraints | https://pip.pypa.io/en/stable/cli/pip_install/ |
| Maven's nearest definition and soft requirements | https://maven.apache.org/guides/introduction/introduction-to-dependency-mechanism.html, https://maven.apache.org/pom.html#Dependency_Version_Requirement_Specification |
| Gradle selects the highest requested version | https://docs.gradle.org/current/userguide/graph_resolution.html |
| The AWS SDK for Go v2 releases dependency updates | https://github.com/awslabs/aws-go-multi-module-repository-tools/blob/main/release/release.go |
| release-please's workspace plugins | https://github.com/googleapis/release-please/blob/main/docs/manifest-releaser.md |
| Nx release and `updateDependents` | https://nx.dev/docs/guides/nx-release/update-dependents |
| cargo-release and `dependent-version` | https://github.com/crate-ci/cargo-release/blob/master/docs/reference.md |
| OpenTelemetry Go module sets | https://github.com/open-telemetry/opentelemetry-go-build-tools/blob/main/multimod/README.md |
