---
adr: 0071
title: A publish marks the release of the root package as the latest release of GitHub
status: Accepted
date: 2026-10-29
supersedes: RFC-0005, in part
superseded-by: none
rfc: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0071: A publish marks the release of the root package as the latest release of GitHub

## Status

Accepted

## Context

GitHub marks one release of a repository as its latest release. The page of the repository shows it, and `gh release view` without a tag returns it. The field `make_latest` of a release defaults to `true` for a newly published release, so each release becomes the latest release when GitHub publishes it. GitHub cannot mark a draft or a prerelease as the latest release.

`ergon release publish` creates each release as a draft, and publishes it without `make_latest`. It publishes the packages in chunks of dependency order, so the release that it publishes last keeps the mark. In a repository whose submodules require the root module, the root module is in the first chunk. After the publish of `v0.7.0`, stealth-scale/go-cli showed `examples/v0.0.8` as its latest release. The root module of ergon requires its submodules, so ergon's own publish released the root module last, and that order alone made it the latest release.

The legacy rule of GitHub sorts the releases by the date of the commit of each release. The releases of one publish share their commit.

## Decision

We will publish the release of the root package with `make_latest` set to `true`, and every other release with `make_latest` set to `false`, because the page of a repository and the tools that read its latest release expect the release of the root package.

- A package is the root package when its directory is the root of the repository.
- A prerelease of the root package is published with `make_latest` set to `false`, so the mark remains on the earlier release.
- A repository without a root package keeps the default of GitHub.
- `ergon release sign` publishes the drafts of a provider, which is always the root package, in ascending order of their versions, so the newest release of a run becomes the latest release.

## Alternatives Considered

### `make_latest` set to `legacy`

GitHub then chooses the latest release by the date of its commit. It lost because the releases of one publish share their commit, so a release of a submodule could still win the mark.

### A comparison with the latest release of the repository

`ergon release sign` reads the latest release through the API of GitHub, and marks a draft only when its version is higher. It lost because the latest release can belong to another package with another format of tag, such as `examples/v0.0.8`, whose version ergon cannot compare with the version of the draft.

## Consequences

**Positive:**

- The page of a repository and `gh release view` show the release of the root package.
- A publish of submodules alone leaves the mark on the latest release of the root package.

**Negative:**

- `forge.Client.Publish`, `release.Releaser`, `release.TagForge` and `release.SignForge` take the mark, so their implementations outside ergon break.
- A later run of `ergon release sign` that publishes an older draft than the latest release moves the mark to the older release.
- In a repository with two root packages, such as a Go module and an npm package at its root, the release that the publish creates last keeps the mark.

## References

| What | Where |
|---|---|
| The fields of a release and the latest release in the API of GitHub | <https://docs.github.com/en/rest/releases/releases> |
| The request | <https://github.com/dokimasia/ergon/issues/69> |
