---
adr: 0015
title: The targets of the hooks are options of .ergon.yaml
status: Accepted
date: 2026-10-08
supersedes: ADR-0014, in part
superseded-by: none
rfc: RFC-0006
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0015: The targets of the hooks are options of .ergon.yaml

## Status

Accepted

## Context

ADR-0014 runs `make lint` and `make test` before each commit and `make check` before each push, in every repository. Each target runs every language and every package of the repository, so the time of a hook grows with the size of the repository.

A team whose tests take minutes cannot move them to the push. The local file `.ergon/local/.pre-commit-config.yaml` cannot do it either, because ergon appends the lists of a local YAML file to the rendering. A local file adds a hook, and cannot remove or move a hook of the rendering.

## Decision

We will make the targets of each stage options of the section `common` of `.ergon.yaml`:

- `hooks.pre-commit` and `hooks.pre-push` each list aggregate targets of the Makefile: `fmt`, `lint`, `test`, `audit` or `check`.
- The defaults are the targets of ADR-0014.
- The template renders one hook for each target of each stage, with the target as its id, in the order of the list. An empty list turns the hooks of its stage off.
- Each hook runs its target for every language and every package of the repository.

## Alternatives Considered

### Hooks for the languages and the packages of a change

Each hook would run its target only for the languages and the packages that the commit or the push affects, through the packages that ergon discovers and the packages that require them. It lost because each language would have to declare the files of its sources, and each fragment of the Makefile would have to take a list of packages. The selection would also miss a dependency between two languages, such as a TypeScript client that a Go program generates.

### `SKIP` and `--no-verify`

A person skips a hook with the variable `SKIP` of pre-commit, or every hook with `git commit --no-verify`. It lost because each acts on one command of one person, and the repository does not record it.

## Consequences

**Positive:**

- A repository chooses what runs before a commit and before a push, and keeps that choice across upgrades of ergon.
- A repository that keeps the defaults has the hooks of ADR-0014.

**Negative:**

- A repository that moves its tests from the commit to the push finds a failing test only when it pushes.

**Neutral:**

- CI runs the gate of every language on every push, whatever the hooks run.

## References

| What | Where |
|---|---|
| The design of the options and of the rendering | RFC-0006, Configurable hooks |
