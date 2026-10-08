---
adr: 0017
title: make generate runs the generators of every language, and make help groups the targets
status: Accepted
date: 2026-10-08
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0017: make generate runs the generators of every language, and make help groups the targets

## Status

Accepted

## Context

RFC-0004 gives the Makefile the aggregates `fmt`, `lint`, `test`, `audit` and `check`, and only Go has the step `generate`. `generate-go` runs `go generate` and fails when the run changes a file, so the step `generate` of `check` fails on generated code that is out of date. A person who runs `generate-go` to update the generated code gets a failure from every run that changes a file.

A repository of any other language has no target for its generators, and `.ergon.yaml` has no key for one.

`make help` prints every target in one list, in the order of the Makefile. A repository with three languages lists about thirty targets without a heading.

Kubernetes keeps the update and the check of generated code apart. `hack/update-codegen.sh` writes the generated files. `hack/verify-codegen.sh` runs it in a temporary worktree of HEAD and fails on a difference. The Makefile that Kubebuilder scaffolds has a target `generate` that writes the generated code. Lines `##@ <name>` group its targets in `make help`.

## Decision

We will separate the target that writes the generated files from the target that checks them. Every language gets the step `generate`, and `make help` groups the targets by producer:

- The aggregate `generate` runs `generate-<language>` of every language that has one.
- The step `generate` of every language section has the keys `command` and `args`: the command line of the generators and the arguments that follow it. Go's command is `go generate`, which `generate-go` runs in every module of `go.work`. The other languages have no command at the baseline, and have the targets of the step only when their section sets one.
- `generate-<language>` runs the command, and fails only when the command fails.
- `verify-generate-<language>` runs the command, and fails when the run changes a file of the repository. Its message contains the target that updates the files. The step `generate` of `check` runs it.
- A line `##@ <name>` starts a group of `make help`. The groups are Common, then one group for each language and each shared toolchain, in the order of the Makefile. The local file starts a group for the repository's own targets with a line of its own.

## Alternatives Considered

### One target that writes and checks

`generate-<language>` would keep the contract of RFC-0004. `make generate` would run it for every language. It lost because the command that updates the generated code would fail whenever it updates a file.

### The check in a temporary worktree

`verify-generate-<language>` would run the generators in a worktree of HEAD and leave the working tree unchanged, as Kubernetes does. It lost because a worktree of HEAD lacks the uncommitted changes, and the hook `pre-push` runs `check` on a working tree that can have them. The check compares the changes of the working tree before and after the run instead, as `generate-go` did.

### Groups by the suffix of a target

`make help` would put each target into the group of the section that its suffix names, such as `go` for `test-go`. It lost because a target of the repository's own, such as `deploy-docs`, would go into a group named after its suffix.

## Consequences

**Positive:**

- `make generate` updates the generated code of every language, and succeeds when the generators do.
- A repository of any language sets the command of its generators in `.ergon.yaml`, and its gate checks them.
- `make help` shows the targets of each language under its name.

**Negative:**

- `verify-generate-<language>` leaves the files that the generators changed in the working tree. The person commits them, or reverts them.
- A local file without a line `##@ <name>` shows its targets in the group of the last language.

**Neutral:**

- The step `generate` is the one step whose target is not `<step>-<language>`.

## References

| What | Where |
|---|---|
| The update and the check of the generated code of Kubernetes | https://github.com/kubernetes/kubernetes/blob/master/hack/verify-codegen.sh, and `hack/lib/verify-generated.sh` |
| The groups of `make help` in the Makefile that Kubebuilder scaffolds | https://github.com/kubernetes-sigs/kubebuilder/blob/master/testdata/project-v4/Makefile |
| The steps and the Makefile of `ergon init` | RFC-0004, Repository initialization |
