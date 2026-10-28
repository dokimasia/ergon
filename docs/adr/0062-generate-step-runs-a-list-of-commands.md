---
adr: 0062
title: The step generate runs a list of command lines
status: Accepted
date: 2026-10-23
supersedes: ADR-0017, in part
superseded-by: ADR-0068, in part
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0062: The step generate runs a list of command lines

## Status

Accepted

## Context

The step `generate` of each language section had the keys `command` and `args`, one command line of the generators. `generate-<language>` ran that line, and `verify-generate-<language>` failed when the run changed a file of the repository. Go ran the line with the paths of the section in every module.

A module whose code comes from two generators could give the step only one of them:

- theke writes its enumerations, doubles and conformance suites with `dokimi run`, and the canonical codecs of its records with kanon through `go generate`. Its step ran `dokimi run` alone, so `verify-generate-go` did not check the codecs that kanon writes.
- core runs kanon alone, with `go generate -run 'go tool kanon'`.

Kubernetes runs each of its 15 generators in order in `hack/update-codegen.sh`, and `hack/verify-codegen.sh` fails on a difference in the output of any of them.

## Decision

We will replace the keys `command` and `args` of the step `generate` with the key `commands`, a list of command lines, because `verify-generate-<language>` then checks the output of every generator of a repository.

- Each line of `commands` is a command with its arguments, one word on each line of the list. `generate-<language>` runs the lines in order and stops at the first line that fails. `verify-generate-<language>` runs the lines and then compares the files of the repository once.
- Go runs each line with the paths of the section in every module, as it ran the one line.
- An empty list renders neither target, as an empty `command` did. Go's list at the baseline has the one line `go generate`.
- The variable of the Makefile that a run of make overrides is `GO_GENERATE_COMMANDS` for Go and `<LANGUAGE>_GENERATE` for every other language. It joins the lines with `&&`.
- ergon refuses the keys `command` and `args` of a section as unknown keys. A repository writes its command line and its arguments as one line of `commands`.

## Alternatives Considered

### The key commands beside command and args

A repository would keep `command` and `args` for one generator and add `commands` for two or more. It lost because a step would then have two ways to state its generators, and ergon would have to refuse a section that sets both.

## Consequences

**Positive:**

- theke lists `dokimi run` and kanon in one step, and `verify-generate-go` checks the output of both.
- Every language section states its generators with the same key.

**Negative:**

- Each repository changes the keys of the step in `.ergon.yaml` once, by hand, before ergon accepts the section.
- The Makefile has no `GO_GENERATE_COMMAND` and `GO_GENERATE_ARGS`. A run of make overrides all the lines of Go in one variable.
- The line of the log of a module names the step `generate` and not the command that runs.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#53 |
| The generators of Kubernetes | <https://github.com/kubernetes/kubernetes/blob/master/hack/update-codegen.sh> |
| The step generate | ADR-0017 |
