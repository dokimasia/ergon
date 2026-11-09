---
adr: 0074
title: The job mutate-go keeps the history of dokimi-mutate-go and sets the limit of its run through GO_MUTATE_TIMEOUT
status: Accepted
date: 2026-11-09
supersedes: ADR-0065 and RFC-0011, in part
superseded-by: none
rfc: RFC-0011
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0074: The job mutate-go keeps the history of dokimi-mutate-go and sets the limit of its run through GO_MUTATE_TIMEOUT

## Status

Accepted

## Context

ADR-0065 decides that the nightly job of mutate keeps the verdicts of its earlier runs in the cache of GitHub Actions. Its part in ergon waited for a release of dokimi-mutate-go with `--history`. Release v0.2.0 of 2026-10-28 adds the flag, and v0.3.0 of 2026-11-03 is the newest release. The baseline pins v0.1.0, so the job `mutate-go` runs every mutant in each run. The Nightly of thesm-os/core ended as cancelled in each of its runs from 2026-11-04 to 2026-11-08. In the run of 2026-11-08, GitHub cancelled its job Mutate Go at the limit of 60 minutes.

ADR-0065 and RFC-0011 set the limit of the run through `DOKIMI_MUTATE_GO_TIMEOUT`. The recipe of `mutate-go` passes `--timeout $(GO_MUTATE_TIMEOUT)`, and the help of dokimi-mutate-go v0.3.0 states that a flag takes precedence over its variable. With the flag in the recipe, the variable does not limit the run, and GitHub cancels the job before the run reports a score.

`--history` of v0.3.0 keeps one record for each package, so one directory serves every module of a repository. The run writes its records back unless it has a run error. A mutant that `--timeout` does not start keeps the verdict that the run reused.

The Go jobs of the Nightly runs of dokimasia/assert-go and thesm-os/sharder on 2026-11-08 started their step of make 41 to 98 seconds after the job started.

## Decision

We will build ergon's part of ADR-0065 with a cache of ADR-0073, and set the limit of the run through `GO_MUTATE_TIMEOUT`, because the flag of the recipe overrides `DOKIMI_MUTATE_GO_TIMEOUT`.

- The baseline pins dokimi-mutate-go v0.3.0.
- The job `mutate-go` keeps `${{ runner.temp }}/dokimi-mutate-go-history` as a cache without files: the ID `dokimi-mutate-go-history` and the input `history` of `workflow_dispatch`.
- The step of the job runs `make mutate-go GO_MUTATE_TIMEOUT=<minutes>m` with `DOKIMI_MUTATE_GO_HISTORY` set to the directory. `<minutes>` is the limit of the job minus 10 minutes, as ADR-0065 decides.
- The section `go` refuses a limit of 10 minutes or less for `mutate` under `nightly`, which does not leave time for the run.

This record supersedes the variable of the limit in ADR-0065 and in RFC-0011. Every other part of both records stands.

## Alternatives Considered

### DOKIMI_MUTATE_GO_TIMEOUT in the environment of the step

The step would set the variable that ADR-0065 names. It lost because the recipe passes `--timeout` in every run, and the flag takes precedence over the variable.

### The limit in the key go.mutate.timeout

The job would pass no limit, and each repository would keep `go.mutate.timeout` below `go.nightly.mutate`. It lost because the two keys would have to change together, while the renderer reads the limit of the job from `go.nightly.mutate`.

### dokimi-mutate-go v0.2.0

The first release with `--history` lost to v0.3.0, which fixes two defects of v0.2.0. The control runs of v0.2.0 passed a test binary that failed a test and exited with the status 0. Its memory ceiling of a test binary on Linux included the memory of the process that started the binary.

## Consequences

**Positive:**

- A module whose full run exceeds the limit of its job gets a verdict for each of its mutants after enough nights.
- A run that reaches its limit fails with the score of the mutants that it tested, and the history keeps their verdicts.

**Negative:**

- The pin of v0.3.0 was six days old on 2026-11-09. The weekly update of the baseline takes a release only once it is seven days old.
- A repository whose `go.nightly.mutate` is 10 minutes or less fails `ergon init` until it raises the limit.

## References

| What | Where |
|---|---|
| The help of dokimi-mutate-go v0.3.0 | <https://github.com/dokimasia/mutate-go/blob/v0.3.0/cmd/dokimi-mutate-go/doc.go> |
| The changelog of dokimi-mutate-go | <https://github.com/dokimasia/mutate-go/blob/v0.3.0/CHANGELOG.md> |
| The request for ergon's part of ADR-0065 | <https://github.com/dokimasia/ergon/issues/74> |
