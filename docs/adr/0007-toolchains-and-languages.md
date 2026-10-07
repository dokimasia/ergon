---
adr: 0007
title: Toolchains and languages are separate catalog entries
status: Accepted
date: 2026-10-05
supersedes: none
superseded-by: ADR-0008, in part
rfc: RFC-0001
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0007: Toolchains and languages are separate catalog entries

## Status

Accepted

## Context

ergon supports ten languages, and each has a module of its own, because every command is implemented per language. Java and Kotlin share one build toolchain, Gradle. TypeScript's toolchain, npm, pnpm or bun, also builds JavaScript packages.

`release` works on packages, and a Gradle project with Java and Kotlin sources is one package. `init` renders each language's configuration and gate tools, which differ between Java and Kotlin. If each language registered discovery and the release roles, ergon would discover and release a mixed Gradle project twice.

## Decision

We will register toolchains and languages as separate catalog entries, because a package belongs to exactly one build toolchain while its sources can be in more than one language.

A toolchain entry provides discovery and the package roles, and a language entry provides the language roles. `lang/jvm` and `lang/js` register as the toolchains `jvm` and `js`. Java and Kotlin name `jvm`, and TypeScript names `js`. The other seven languages are their own toolchains, and their module registers both entries. `workspace.Package` records its `Toolchain`, not a language. `init` also asks the toolchains of the chosen languages, so `lang/jvm` renders the Gradle build that Java and Kotlin share once.

## Alternatives Considered

### One catalog entry per language

Each language registers discovery and the release roles itself. It lost because Java and Kotlin would each register Gradle, and a mixed Gradle project would be discovered and released twice.

### One catalog entry per toolchain

`jvm` registers every role for Java and Kotlin, and `js` for TypeScript. It lost because `lang/jvm` would then contain the configuration of Java alone and of Kotlin alone, such as each language's formatter and linter. `ergon-lang` contains only code that two languages share. `init`'s answers also name languages, not toolchains.

## Consequences

**Positive:**

- A package is discovered and released once, whichever languages its sources use.
- A language module contains only the code specific to its language.

**Negative:**

- The catalog has two registries, and `internal/app` calls `RegisterToolchain` before `Register`.
- A command over packages and a command over languages iterate different entries, so each role states which kind it is.
- The release vocabulary names toolchains, so a changeset key that two toolchains share takes the form `<toolchain>:<name>`.

**Neutral:**

- For seven of the ten languages, the toolchain and the language have the same name and come from the same module.
