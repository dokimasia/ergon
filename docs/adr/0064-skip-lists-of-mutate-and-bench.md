---
adr: 0064
title: The Go steps mutate and bench leave out the packages of the key skip
status: Accepted
date: 2026-10-26
supersedes: none
superseded-by: none
rfc: RFC-0011
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0064: The Go steps mutate and bench leave out the packages of the key skip

## Status

Accepted

## Context

The testing standards exempt conformance suites, the fixtures of generators, the corpus of a generator and examples from mutation testing. `make mutate-go` and `make bench-go` still run over them, because both run over `./...` of each module. On 2026-10-26:

- The production packages of sharder scored 99% to 100%. Its five suites scored 13% to 53%, and its examples 85% to 94%.
- Most corpus packages of dokimi scored 0%, because no test of the corpus covers the mutants of its generated suites.
- Bench Go of kanon spent 368 seconds on `internal/fixture/mapkey` and 324 on `internal/fixture/mapvalue`.

The baseline pinned dokimi-mutate-go at 719083c, a build without a flag that leaves a package out. Release v0.1.0 has `--skip`, which takes package patterns, and it takes its flags in the GNU form.

## Decision

We will add the key `skip`, a list of package patterns, to the steps mutate and bench of Go, and pin dokimi-mutate-go v0.1.0, because the packages that only tests use cost the nightly jobs much of their time and most of their survivors.

- The baseline of `skip` is empty in both steps.
- `mutate-go` passes the patterns of `go.mutate.skip` to `--skip`, beside `--workers` and `--timeout`.
- `bench-go` runs `go test -bench` on the packages of `go list` of `go.paths` without the packages of `go list` of `go.bench.skip`, in each module.
- A pattern that matches no package of a module is no error, because one list serves every module of the repository.

## Alternatives Considered

### A convention for test-only packages

ergon would leave out every package whose path ends in `test` and every package below `examples/`. A repository would not need a key. It lost because dokimi's `corpus/` and kanon's `internal/fixture/` follow neither name.

### One list for both steps

One key would list the packages that both steps leave out. It lost because a repository can keep the benchmarks of a fixture whose mutants it leaves out. The benchmarks of kanon's fixtures measure the codecs that kanon generates.

## Consequences

**Positive:**

- Mutate Go and Bench Go spend their time on the production packages, and Mutate Go reports the survivors of those packages alone.

**Negative:**

- Each repository lists its test-only packages in two keys. A new suite costs time in the nightly mutation until someone adds it.
- A repository whose `go.mutate.args` holds single-dash flags fails `mutate-go` until it moves to the GNU form.

**Neutral:**

- Both steps still run over every package of a repository that does not list one.

## References

| What | Where |
|---|---|
| The flags of dokimi-mutate-go v0.1.0, `--skip` among them | <https://github.com/dokimasia/mutate-go/blob/v0.1.0/cmd/dokimi-mutate-go/doc.go> |
