---
adr: 0057
title: The dependency review leaves the repository's own Go modules out of its license check
status: Accepted
date: 2026-10-22
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0057: The dependency review leaves the repository's own Go modules out of its license check

## Status

Accepted

## Context

The dependency review of ergon's version pull request of 2026-10-22 listed 23 dependencies with an unknown license, such as `../ergon-core` at 0.12.0. Each one is a directory replace in the `go.mod` of a module, such as `go.dokimi.dev/ergon/core => ../ergon-core`. GitHub's dependency graph uses the path of the replace as the name of such a dependency and the version of its requirement as its version. The dependency then has no source repository, so neither the graph nor the action finds its license. The same comparison gives every `go.dokimi.dev/ergon/...` module the license MIT.

The modules keep the replaces so that each of them builds and tidies outside the workspace. The latest version pull requests of techne and eidos listed 23 and 53 such dependencies.

The action fails a review for a forbidden license and for a license that it cannot resolve. It only lists an unknown license, so a new third-party dependency without a license appears among the repository's own modules on every version pull request.

## Decision

We will leave the directory replaces of the modules of `go.work` out of the license check of the dependency review, because these modules have the license of the repository.

- The Go producer contributes two steps to the job `dependency-review` of `security.yml`. Both run when the repository has `go.work`.
- The first step installs the version of `go.work` with setup-go, without the cache of the modules.
- The second step runs `go mod edit -json` on the `go.mod` of each module of `go.work`. For each directory replace of a module of `go.work`, it appends the package URL `pkg:golang/<path>` to the environment variable `ERGON_OWN_PACKAGES`.
- The job passes `ERGON_OWN_PACKAGES` to the input `allow-dependencies-licenses` of the action when a producer contributes steps to it.
- The license check still covers a directory replace of a module outside `go.work`, such as a fork of a third-party module in the tree.

## Alternatives Considered

### A license file in each module directory

The key `license.directories` renders a `LICENSE` into the directory of each module. It lost because each module already has the license of the repository. The zip of `go.dokimi.dev/ergon/core@v0.11.0` contains the `LICENSE` of the repository root, and the dependency graph gives every module path the license MIT. GitHub would still use the path of each replace as the name of the dependency.

### Requirements without directory replaces

Each module requires its siblings at their versions alone, as the root module does, and the graph then lists each sibling under its module path. It lost because a module would build and tidy outside the workspace only against released siblings. The change would also touch 11 modules of ergon, 16 of techne and 14 of eidos.

### No license check

The input `license-check: false` removes the list. It lost because the managed review does not set an allow list or a deny list, so the unknown licenses are the only output of its license check. The review would then also stop reporting the unknown license of a third-party dependency.

### A list that ergon init renders

ergon init writes the package URLs into `security.yml` when it renders the file. It lost because a rendering does not read a file of the repository. The list would also miss a module that the repository adds before its next sync.

## Consequences

**Positive:**

- The license check of a pull request lists the unknown licenses of third-party dependencies alone.
- Every repository with Go and `go.work` receives the steps through `ergon init sync`.

**Negative:**

- The review job of each pull request in such a repository installs Go and runs jq. GitHub's Ubuntu runner images include jq.
- The action matches an exclusion by its name alone. A directory replace of a module outside `go.work` whose path has the same text in another `go.mod` is left out of the license check too.
- The steps depend on the names that GitHub gives directory replaces. If GitHub lists them under their module paths, the list matches no dependency, and the modules show the license of the repository.

**Neutral:**

- The CodeQL analysis of Go already installs Go and runs `go list -m` in a step of its own.

## References

| What | Where |
|---|---|
| The unknown licenses and the exclusions of the action | <https://github.com/actions/dependency-review-action/blob/v5.0.0/src/licenses.ts> |
| The match of an exclusion by type and name | <https://github.com/actions/dependency-review-action/blob/v5.0.0/src/purl.ts> |
| The comparison of the dependency graph | <https://docs.github.com/en/rest/dependency-graph/dependency-review> |
| A variable of the environment file in the expressions of a later step | <https://github.com/actions/runner/blob/main/src/Runner.Worker/FileCommandManager.cs> |
| The software of the Ubuntu 26.04 runner image | <https://github.com/actions/runner-images/blob/main/images/ubuntu/Ubuntu2604-Readme.md> |
