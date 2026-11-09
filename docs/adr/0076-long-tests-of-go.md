---
adr: 0076
title: The step longtest runs the Go tests that the gate skips in short mode
status: Accepted
date: 2026-11-09
supersedes: none
superseded-by: none
rfc: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0076: The step longtest runs the Go tests that the gate skips in short mode

## Status

Accepted

## Context

A Go test that takes too long for each push skips itself in short mode, as `TestTimeConsuming` in the documentation of package `testing` does. `go help testflag` describes `-short` as the flag that tells long-running tests to shorten their run time. The Go project runs the tests of all.bash in short mode, and its longtest builders run them without `-short`.

A repository can add `-short` to `go.test.args` and `go.race.args`, so `check-go` skips its long tests. No job of the baseline then runs them. The key `nightly` of the section `go` accepts only fuzz, bench and mutate.

thesm-os/sharder runs a deterministic simulation of each failure trace of its first milestone: 38 traces over 4 presets, 152 simulations. One run of the 152 simulations took 992 seconds of CPU time. Its job `check-go` took 10 to 24 minutes against its limit of 30, and its step race runs every test under the race detector.

## Decision

We will add a step `longtest` that runs the Go tests without the args of the gate, because a long test needs a job of its own on a schedule.

- `core/option` gains the step `longtest`, after `race` in the order of the targets.
- The section `go` gains the key `longtest`, the args of `longtest-go`, with `-count=1` at the baseline.
- `longtest-go` runs `go test -timeout $(GO_LONGTEST_TIMEOUT)` with those args in every module of `go.work`. `GO_LONGTEST_TIMEOUT` is 0 by default, which turns the limit of go test off.
- The keys `check` and `nightly` of the section `go` accept `longtest`. The baseline runs it in neither.
- The job `longtest-go` of `nightly.yml` runs `make longtest-go GO_LONGTEST_TIMEOUT=<minutes>m`, where `<minutes>` is the limit of the job minus 10 minutes, as in the job `mutate-go` of ADR-0074. A test binary that runs longer panics with the stacks of its goroutines before GitHub cancels the job. The section refuses a limit of 10 minutes or less for `longtest`.
- A repository skips its long tests in `check-go` with `-short` in `go.test.args` and `go.race.args`.

## Alternatives Considered

### A -timeout in go.longtest.args

A repository would set `-timeout` in the args of the step, as the request in dokimasia/ergon#78 states. It lost because the timeout and `go.nightly.longtest` would have to change together.

### -short in the args of test and race at the baseline

The gate would skip the long tests of every repository. It lost because a repository without the nightly job of `longtest` would then run its long tests nowhere.

### A step named after its use, such as simulation

It lost because the Go project calls this mode the long-test mode, and names its builders longtest.

## Consequences

**Positive:**

- A repository runs its long tests each night, and `check-go` stays within its limit.

**Negative:**

- A test that skips in short mode runs only in `nightly.yml`. A defect that only such a test finds fails the next nightly run, and not the push that caused it.

## References

| What | Where |
|---|---|
| `-short` in `go help testflag` | <https://pkg.go.dev/cmd/go#hdr-Testing_flags> |
| `TestTimeConsuming` in the documentation of package `testing` | <https://pkg.go.dev/testing#hdr-Skipping> |
| The short mode of `cmd/dist test` | <https://github.com/golang/go/blob/go1.27.3/src/cmd/dist/test.go> |
| The request for the step | <https://github.com/dokimasia/ergon/issues/78> |
