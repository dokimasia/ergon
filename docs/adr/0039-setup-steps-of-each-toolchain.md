---
adr: 0039
title: The key ci.steps of each toolchain adds steps to the setup of its jobs
status: Accepted
date: 2026-10-10
supersedes: RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0039: The key ci.steps of each toolchain adds steps to the setup of its jobs

## Status

Accepted

## Context

The job of a toolchain, such as `check-go`, first runs the steps that set up its toolchain. It then runs the target of the Makefile. A repository adds content to a managed workflow through the file of the same path under `.ergon/local/`. The merge of that file appends the items of a list. A step of the local file then runs after the step that runs the target.

The tests of the language backends of eidos compile the code that eidos generates with `tsc`, `javac` and `cargo`. In CI a test fails when its compiler is missing. A missing compiler then cannot hide a regression. The job of Go installs Go alone. eidos had no way to install the compilers before `make check-go`. The jobs of every toolchain have the same structure. A repository with another language has the same gap.

## Decision

We will add the key `ci.steps` to the section of every toolchain, whose steps each job of the toolchain runs after the setup of the toolchain and before ergon, because a repository whose tests need another program cannot install it before the target of the Makefile runs.

- The key is in the CI options of every toolchain section: `go`, `jvm`, `js`, `csharp`, `php`, `python`, `rust`, `terraform` and `bash`. The jobs of Java and Kotlin run the steps of `jvm`. The jobs of JavaScript and TypeScript run the steps of `js`.
- A step has the keys of a step of a workflow: `name`, `id`, `if`, `env`, and either `uses` with its inputs under `with`, or `run`. `uses` is an action pinned as `uses`, `commit` and `release`, as every action of `.ergon.yaml` is. `run` is the list of the lines of a command of bash. ergon checks each step as it checks the steps of its own jobs.
- The steps run in the check job of the toolchain and in its nightly jobs. They follow the setup steps of the toolchain. They run under the condition on the files that the setup states.

## Alternatives Considered

### A local job in eidos

The local `ci.yml` of eidos would add a Linux job that installs the compilers and runs the tests of the backends. The managed job would set `CI=false`. A test there would then skip when its compiler is missing. It lost because the managed job would hide a missing compiler. The added job would also repeat a part of the gate on one runner.

### The key in the section go alone

Only the section `go` would have `ci.steps`. That met the need of eidos. It lost because a repository with another language has the same gap.

### One list under the section common

One list of steps would run in the jobs of every toolchain. It lost because a step installs what the tests of one toolchain need. The list would also run in the jobs of the other toolchains. A `setup-node` of the list would run beside the `setup-node` of the toolchain `js` and its matrix of versions of Node.js.

## Consequences

**Positive:**

- A repository installs the programs that its tests need in the job that runs the tests. The steps run on every runner of the matrix.
- The steps of a repository follow the rules of the steps of ergon. An action is pinned to a commit, and a command has no empty line.

**Negative:**

- The section of each toolchain and the lock have one key more.
- An input of an action is a string in `.ergon.yaml`, such as `node-version: "24"`. The decoder of the options converts no kinds.
- The repository updates the pins of its steps in `.ergon.yaml` itself. Dependabot does not read `.ergon.yaml`. The update of the pins of the baseline does not reach the steps of a repository either.

**Neutral:**

- The steps of a toolchain whose setup has a condition on its files run only when the repository has those files.

## References

| What | Where |
|---|---|
| The options and the jobs of each language | RFC-0004 |
| The steps of the CodeQL analysis of Go | ADR-0036 |
