---
adr: 0026
title: The options of a producer return its contribution, data and files
status: Proposed
date: 2026-10-09
supersedes: RFC-0001 and RFC-0004, in part
superseded-by: none
rfc: RFC-0008
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0026: The options of a producer return its contribution, data and files

## Status

Proposed

## Context

The roles of `ergon init` pass the options of a producer as the interface `language.Options`. All 16 producers implement `Configurable` and `Contributor`, and 14 of them assert their own options type in `Contribution` and fall back to the baseline for any other type. The engine passes each producer its own options, so the fallback runs in tests alone, and 29 test cases check these methods. `Data` and `Files` receive the options in the same way.

## Decision

We will make the options of a producer the receiver of `Contribution`, `Data` and `Files`, add `Options` to `Producer`, and delete `Configurable` and `Contributor`, because every producer has options and the engine always passes a producer its own options.

- `language.Options` has the methods `Validate` and `Contribution`.
- Options implement `language.Calculator` and `language.Placer`, whose methods receive the answers and the contributions.
- `render.Unit.Options` is never nil, and the engine calls `Contribution` without a type assertion.
- The release roles do not change.

## Alternatives Considered

### The roles as they are

Each producer keeps `Contribution(o language.Options)` with its type assertion. It lost because the assertion and its fallback repeat in 14 producers and 29 test cases, for a branch that the engine never takes.

## Consequences

**Positive:**

- The methods of a producer receive typed options. The 16 forwarding methods, their fallbacks and the helpers `own` of the GitHub and license producers are deleted.

**Negative:**

- The change edits `core`, the engine and all 16 producers in one commit.
- A producer without a part of the workflows still implements `Contribution`, which returns the zero value.

**Neutral:**

- `LocalChecker` remains a role of the producer, because it checks files and not options.
