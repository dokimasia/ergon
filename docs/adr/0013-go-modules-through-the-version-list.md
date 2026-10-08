---
adr: 0013
title: The resolver finds the module of a Go package through @v/list
status: Proposed
date: 2026-10-08
supersedes: none
superseded-by: none
rfc: RFC-0005
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0013: The resolver finds the module of a Go package through @v/list

## Status

Proposed

## Context

A Go tool of the baseline is a package path, such as `github.com/golangci/golangci-lint/v2/cmd/golangci-lint`. `pin.Resolver` first finds the module that provides the package. RFC-0005 takes the longest prefix of the package path whose `@latest` the module proxy returns. `ergon init upgrade` resolves `go.dokimi.dev/ergon` the same way, through the first proxy of `GOPROXY` that is a URL.

The GOPROXY protocol makes `@latest` optional: "This endpoint is optional, and module proxies are not required to implement it." It also allows "a site serving from a fixed file system" as a proxy. Such a proxy serves `@v/list` and the files of each version, and no `@latest`.

On 2026-10-08, `ergon init upgrade` ran against a proxy of files that listed v0.3.0 and v0.3.1 of `go.dokimi.dev/ergon` in `@v/list`. It failed with "no module of the proxy has the package go.dokimi.dev/ergon". After an `@latest` document was added to the proxy, the same upgrade installed v0.3.1.

The go command reads `@v/list` first. Its reference states: "When resolving the latest version of a module, the go command will request $base/$module/@v/list, then, if no suitable versions are found, $base/$module/@latest." On the same day, proxy.golang.org responded 404 to `@v/list` of the package paths `golang.org/x/vuln/cmd/govulncheck` and `golang.org/x/perf/cmd/benchstat`. It responded 200 to the module `golang.org/x/vuln` with its versions, and 200 with an empty list to the module `golang.org/x/perf`, which has no tags.

## Decision

We will make the resolver find the module of a package path the way the go command does:

1. It requests `@v/list` of each prefix of the package path, the longest first. A 404 or a 410 means that the prefix is no module of the proxy.
2. A list with a stable version makes the prefix the module, and its versions the candidates.
3. An empty list makes the prefix the module when its `@latest` returns a document. The pseudo-version of that document is then the only candidate.
4. A prefix whose list is empty and whose `@latest` the proxy does not serve is no module, as for the go command.

## Alternatives Considered

### `@latest` with `@v/list` as a fallback

The resolver would keep `@latest` and read `@v/list` of the same prefix when `@latest` responds 404. It lost because it sends two requests for each prefix that is a package and not a module, on every proxy. It also keeps a rule of the module path that the go command does not have.

### A proxy that must serve `@latest`

ergon would state that its proxy must implement `@latest`. It lost because the protocol makes the endpoint optional, and a repository chooses its proxy for reasons other than ergon.

## Consequences

**Positive:**

- `ergon init upgrade` and the weekly baseline update work with every proxy of the GOPROXY protocol, including a proxy of files.
- The resolver and the go command agree on the module of each package.
- A tool in a subdirectory of a tagged module, such as golangci-lint, needs one request fewer: the list of its module is the request that finds the module.

**Negative:**

- A module without tags, such as `golang.org/x/perf`, still needs `@latest` for its pseudo-version. A proxy without `@latest` cannot update such a pin.
- The tests of the module path of `pin` change.

**Neutral:**

- The age and the major version of each candidate follow RFC-0005 unchanged.

## References

| What | Where |
|---|---|
| The GOPROXY protocol: the endpoints `@v/list` and `@latest`, and the order in which the go command requests them | https://go.dev/ref/mod#goproxy-protocol |
