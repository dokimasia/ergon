---
adr: 0027
title: A template function writes the generate and check targets of each language
status: Proposed
date: 2026-10-09
supersedes: none
superseded-by: none
rfc: RFC-0008
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0027: A template function writes the generate and check targets of each language

## Status

Proposed

## Context

The Makefile fragments of 10 languages declare the same targets: `<SECTION>_GENERATE`, `generate-<section>`, `verify-generate-<section>`, `check-<section>`, and their entries in `.PHONY`, `generate` and `check`. They differ only in the name of the section and the title of the language. 10 test files repeat one test of these targets. The fragment of Go runs its generators in each module of `go.work`, so its targets differ.

## Decision

We will write these targets with the template function `targets` of the engine, because the 10 fragments that declare them differ only in the name of the section and the title of the language.

- A fragment calls `{{% targets "bash" "Bash" .Options.Generate .Options.Check %}}` after its own targets.
- The help of each target does not change, so `make help` prints the same lines.
- The fragment of Go keeps its own targets.
- One test of the engine covers the function.

## Alternatives Considered

### A partial template

The common producer renders `partials/targets.tmpl`, and each fragment calls it with `{{% template "targets" . %}}`. It lost because a template action passes one value to the template that it calls. The partial needs the section and the title beside the options, and the data of a template contains only the answers, the options, the computed values and the contributions.

### The copies as they are

Each fragment keeps its own targets. It lost because a change of these targets edits 10 templates and 10 tests.

## Consequences

**Positive:**

- A change of the targets of the generators or of the gate edits one function and one test.

**Negative:**

- The text of these targets is Go code in the engine, beside the function `steps`, and no longer a template.
- The targets of the generators move behind the other targets of a language, so the first `ergon init sync` after the change rewrites the Makefile of each repository.

**Neutral:**

- The fragment of Go keeps its own targets.
