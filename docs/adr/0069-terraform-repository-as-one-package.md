---
adr: 0069
title: ergon release releases a repository of Terraform as one package
status: Accepted
date: 2026-10-28
supersedes: none
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0069: ergon release releases a repository of Terraform as one package

## Status

Accepted

## Context

A consumer calls a Terraform module from git with a tag in `ref`, such as `?ref=v1.0.0`. The Terraform Registry publishes a version of a module for each tag that is a semantic version, optionally prefixed with a `v`. ergon release 0.15.0 discovered no package in a repository of Terraform modules, so `version.yml` and `release.yml` neither versioned nor tagged it. The maintainer wrote `CHANGELOG.md` and pushed the tag by hand, and no workflow created the release of GitHub.

- Terraform has no manifest with a name or a version. A go.mod has no version either, so the toolchain of Go reads the version of a module from the headings of its `CHANGELOG.md` and from its tags.
- The lock of ergon init records the name of the repository and its languages.
- A file of a repository belongs to the package with the longest directory that contains it.
- A publish creates the tag of each released package, and a release of GitHub whose notes are the section of the version in the changelog of the package.

## Decision

We will discover a repository whose lock of ergon init lists the language terraform as one package of the toolchain terraform, because `version.yml` and `release.yml` then release it as they release a Go module.

- The package is the root of the repository. Its name is the name of the repository in the answers of `.ergon/init.lock`.
- Its version is the higher of the newest heading of `CHANGELOG.md` and the highest tag `v<version>`.
- Its tag is `v<version>`, also in a repository with packages of other toolchains.
- A release writes `CHANGELOG.md` and no other file of the package. Its publish creates the tag and the release of GitHub, and does not upload a file.
- A repository without a lock, or with a lock whose languages do not include terraform, has no package of Terraform.

## Alternatives Considered

### Every repository with a .tf file

The discovery would find the configuration of Terraform in the tree. It lost because a repository of another language with Terraform files, such as test fixtures or a deployment, would gain a package that covers its whole tree.

### The name of the root directory

The package would have the name of the directory of its working tree. It lost because a clone into another directory, such as a worktree, gives a changeset another package name than CI does.

### One name for every Terraform repository

Every repository would name its package `terraform`. It lost because a changeset and a version pull request would then not identify the repository that they release.

## Consequences

**Positive:**

- `version.yml` opens the version pull request of a repository of Terraform, and `release.yml` tags it and creates its release of GitHub.

**Negative:**

- The job `changeset` of `ci.yml` fails a pull request that changes a file of such a repository without a changeset.
- The discovery reads the lock of ergon init, so a repository that ergon init did not set up releases no Terraform.
- A Go module at the root of a repository takes the same tags `v<version>` as the package of Terraform.

**Neutral:**

- The webhook of the Terraform Registry publishes the new tag.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#64 |
| The selection of a revision of a module from git | <https://developer.hashicorp.com/terraform/language/modules/sources> |
| The releases of a module on the Terraform Registry | <https://developer.hashicorp.com/terraform/registry/modules/publish> |
