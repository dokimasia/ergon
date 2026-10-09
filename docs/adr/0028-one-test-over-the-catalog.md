---
adr: 0028
title: One test over the catalog renders every producer
status: Proposed
date: 2026-10-09
supersedes: RFC-0001, in part
superseded-by: none
rfc: RFC-0008
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0028: One test over the catalog renders every producer

## Status

Proposed

## Context

Each language renders its own files in its tests with the test kit of `ergon init`, and keeps golden copies of the workflows of the GitHub producer. The language modules contain 124 golden files with 5,806 lines, and 2,162 of these lines are copies of `ci.yml`, `security.yml` and `dependabot.yml`. dupl finds clones of 103 to 246 lines between their test files. A change of the GitHub producer rewrites golden files in up to 11 languages.

## Decision

We will render every producer in one test of `internal/app`, because the composition root is the one package that imports every producer.

The test covers the catalog and the base producers, and it keeps one golden tree of a repository with every language.

- The test checks for each producer that `Options` returns a new value on each call, that the options at the baseline pass `Validate`, and that the contribution passes `workflow.Contribution.Validate`.
- It renders `init new` with every language into one golden tree, and renders each language alone, which `baselinetest.Hygiene` checks without a golden file.
- The tests of a plugin check the rules of its language: `Validate`, the URLs of `Asset`, and templates rendered with options other than the baseline.

## Alternatives Considered

### A test kit beside the roles

A package `core/language/languagetest` checks the contract of a producer, and each plugin calls it in its tests. It lost because a test of `internal/app` reaches every producer that the binary registers, including a producer whose plugin does not call the kit.

### A golden tree per language in the root module

The rendering tests of each language move to the root module, each with its own tree under `testdata/golden/<language>/`. It lost because each tree keeps its copy of the workflows of the GitHub producer.

## Consequences

**Positive:**

- A change of the GitHub producer changes one `ci.yml`.
- A plugin that `internal/app` registers is in the test without a test file of its own.

**Negative:**

- A change of any producer changes the golden tree that every producer shares, so the review of one language reads a diff of the shared tree.
- A plugin's own tests do not render it at its baseline, so an error in a template fails the test of `internal/app` and not the tests of the plugin.

**Neutral:**

- The test kit `baselinetest` remains in `internal/baseline`.
