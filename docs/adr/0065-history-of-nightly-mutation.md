---
adr: 0065
title: The nightly mutation reuses the verdicts of its earlier runs
status: Accepted
date: 2026-10-26
supersedes: ADR-0020, in part
superseded-by: none
rfc: RFC-0011
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0065: The nightly mutation reuses the verdicts of its earlier runs

## Status

Accepted

## Context

The job Mutate Go of `nightly.yml` runs `make mutate-go`, which tests every mutant of each module, with a limit of 60 minutes. GitHub cancels a job at its limit, so the run does not report a score. The Nightly of eidos, mutate-go, core, kanon and sharder ended as cancelled on every run from 2026-10-22 to 2026-10-26.

One worker on the runner tests 0.1 to 0.3 mutants a second in the large packages. The run of core stopped at mutant 714 of 1,004 in `btree`. The run of kanon stopped at mutant 287 of 5,800 in `internal/kanon`, with 16 h 24 min left for that package alone. GitHub ends a job on a hosted runner after 6 hours.

PIT and StrykerJS keep the results of earlier runs and test a mutant again only when its code or its tests changed. The key of a mutant in mutate-spec stays the same when lines elsewhere in its file change, so a run can look up an unchanged mutant in the record of an earlier run.

mutate-go declined an earlier request to reuse the record of a whole package. The digest of the inputs of a record does not cover the environment variables that the tests read, or the files outside the package that they open, so a reused record can keep a stale verdict.

## Decision

We will keep the verdicts of each nightly mutation in the cache of GitHub Actions and test only the mutants whose package or tests changed, because a full run of a large module does not fit a job of GitHub.

- dokimi-mutate-go gains the flag `--history dir`, with the variable `DOKIMI_MUTATE_GO_HISTORY`. It reuses the verdict of a mutant whose key, package and tests are unchanged since the run that tested it. It tests every other mutant and writes the records of the run to `dir`. The rule belongs to mutate-spec. ergon's part needs a release of dokimi-mutate-go with the flag.
- The reuse follows PIT and StrykerJS:
  - A killed mutant counts as killed while the files of its package and the tests that killed it are unchanged.
  - A surviving mutant counts as a survivor while no test that covers its package changed.
  - A change of `go.sum` or of a package that the mutated package imports tests the mutant again.
- The nightly job of mutate restores the newest cache of its history, and runs `make mutate-go` with `DOKIMI_MUTATE_GO_HISTORY` and with `DOKIMI_MUTATE_GO_TIMEOUT` at the limit of the job minus 10 minutes. It saves the history with `actions/cache/save` and `if: always()`.
- A run that its timeout ends fails, also when it killed every mutant that it tested.
- A run of `workflow_dispatch` with the input `history: false` tests every mutant.

## Alternatives Considered

### A higher limit for Mutate Go

Each repository would raise `go.nightly.mutate` until its run finishes. It lost because GitHub ends a job on a hosted runner after 6 hours, and `internal/kanon` alone needs about 16.

### Only the lines of the changes since the last passing run

The job would mutate the lines of `git diff` since the commit of its last passing run, through `--diff`. It lost because a run that fails keeps its anchor, so the diff grows until it exceeds the limit and no run passes again. A change of a test alone also tests no mutant of the code that it covers.

### A matrix of jobs, one for each package

The nightly workflow would run a job for each package of a module. It lost because one package exceeds the limit of a job alone, and the matrix runs the setup of Go and of ergon once for each package.

### More workers

`go.mutate.workers` would rise to the number of processors. It lost because the runner has 4 processors, which makes a run at most four times as fast. Tests that share a resource, such as a port, then fail each other, and each such failure counts as a detection.

## Consequences

**Positive:**

- A module whose full run exceeds the limit of a job still gets a verdict for each mutant. After its history is complete, each night tests only the mutants of the changes of the day.
- A run that its timeout ends reports the score of the mutants that it tested.

**Negative:**

- The flag needs a design in mutate-spec and a release of dokimi-mutate-go. Until then the Nightly of kanon, eidos, core, mutate-go and sharder cannot pass.
- A large module fails its nightly job until its history has a verdict for each of its mutants. For kanon that takes about 20 nights.
- The cache of GitHub Actions removes an entry that no job read in seven days, and a repository has 10 GB of cache.
- A reused verdict can be wrong when a change outside the files of the rule changes the behaviour, such as a file that a test reads.

## References

| What | Where |
|---|---|
| The incremental analysis of PIT | <https://pitest.org/quickstart/incremental_analysis/> |
| The incremental mode of StrykerJS | <https://stryker-mutator.io/docs/stryker-js/incremental/> |
| The limit of 6 hours of a job on a hosted runner | <https://docs.github.com/en/actions/reference/limits> |
| The eviction of the cache of GitHub Actions | <https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching> |
| The declined reuse of the record of a whole package | <https://github.com/dokimasia/mutate-go/issues/32> |
| The request for the rule of a run that reuses earlier verdicts | <https://github.com/dokimasia/mutate-spec/issues/35> |
| The request for `--history` | <https://github.com/dokimasia/mutate-go/issues/50> |
