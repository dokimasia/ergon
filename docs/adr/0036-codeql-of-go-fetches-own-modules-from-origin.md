---
adr: 0036
title: The CodeQL analysis of Go fetches the modules of its repository from their origin
status: Accepted
date: 2026-10-10
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0036: The CodeQL analysis of Go fetches the modules of its repository from their origin

## Status

Accepted

## Context

The autobuild of CodeQL for Go runs `go mod tidy -e` in each module of a repository before it builds. `go mod tidy` does not use `go.work`, so it resolves each requirement through the module proxy and the checksum database. The autobuilder has no option that skips the tidy, because `CODEQL_EXTRACTOR_GO_BUILD_COMMAND` replaces only the build after it.

The version commit of a release requires the new versions of the sibling modules before the publish pushes their tags. On 2026-10-10 the CodeQL runs of the version pull request of ergon 0.8.0 and of its merge requested `go.dokimi.dev/ergon/core@v0.7.0`, `go.dokimi.dev/ergon/service@v0.8.0` and the eleven language modules from proxy.golang.org, at 12:38 and 12:49 UTC. The publish pushed their tags at 12:54 UTC. At 13:00 UTC sum.golang.org still returned "unknown revision" for each of them, so `go install go.dokimi.dev/ergon/cmd/ergon@v0.8.0` failed. The CodeQL run of the version pull request of ergon 0.7.0 requested its versions the same way.

proxy.golang.org caches a version that someone requested before its tag existed for up to 30 minutes. The go command fetches a module whose path matches GOPRIVATE from its origin, without the proxy and the checksum database.

## Decision

We will run the CodeQL analysis of Go with GOPRIVATE set to the paths of the modules of `go.work`, because the go command then fetches those modules from their origin, and the public proxy and checksum database never receive a request for a version before its tag.

- A CodeQL analysis of `workflow.CodeQL` has steps. `codeql.yml` runs them before the analysis, in the runs of the language of the analysis alone, once the repository has a file of its files.
- The Go producer contributes two steps. The first runs setup-go at the version of `go.work`. The second writes `GOPRIVATE`, the module paths that `go list -m` prints, separated by commas, to `$GITHUB_ENV`.
- The tidy of a version without a tag still fails inside the run. The analysis still succeeds, as the runs of 2026-10-10 did.

## Alternatives Considered

### The build mode manual

`codeql.yml` would build each module of `go.work` with commands of its own, without the autobuilder and its tidy. It lost because CodeQL then extracts only the packages that those commands build, so the analysis would cover other code than the autobuild covers.

### A skip of CodeQL on the version pull request

The job `skip` of `ci.yml` would also skip the CodeQL analysis of the version pull request. It lost because the analysis of the push of the merge runs before the publish as well, and code scanning compares the next pull request with the analysis of that push.

## Consequences

**Positive:**

- `go install` resolves a release of a repository with more than one Go module as soon as the publish pushes its tags.

**Negative:**

- The tidy fetches the modules of the repository with git from their origin, which is slower than the proxy. The fetch also needs the server of their import paths to respond to the runner.
- A producer that adds a step to its CodeQL analysis changes `codeql.yml` of every repository with that language.

**Neutral:**

- The autobuild of Go builds with the version of `go.work` that setup-go installs, in place of the Go of the runner image and a download of the toolchain.

## References

| What | Where |
|---|---|
| The tidy of the autobuilder of CodeQL for Go | https://github.com/github/codeql/blob/main/go/extractor/cli/go-autobuilder/go-autobuilder.go, `tryUpdateGoModAndGoSum` |
| A version that someone requested before its tag | https://proxy.golang.org/, FAQ |
| GOPRIVATE, the proxy and the checksum database | `go help private` of Go 1.27.2 |
| The comparison of a pull request with the analysis of its base branch | https://docs.github.com/en/enterprise-server@2.22/github/finding-security-vulnerabilities-and-errors-in-your-code/setting-up-code-scanning-for-a-repository |
| The job skip of ci.yml | ADR-0018 |
