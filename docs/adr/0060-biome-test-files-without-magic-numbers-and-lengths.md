---
adr: 0060
title: The managed biome.json turns off noMagicNumbers and the two length rules in test files
status: Accepted
date: 2026-10-23
supersedes: RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0060: The managed biome.json turns off noMagicNumbers and the two length rules in test files

## Status

Accepted

## Context

The managed `biome.json` turns on every stable group of Biome's rules as an error. These rules report the layout of a spec of vitest:

- `style/noMagicNumbers` reports each number that a spec states, such as the result that a call must return.
- `complexity/noExcessiveLinesPerFunction` reports a function of more than 50 lines. A spec has a `describe` for each export with an `it` for each case, and each case adds lines to the callback of its `describe`.
- `style/noExcessiveLinesPerFile` reports a file of more than 300 lines. The testing standard requires one spec per source file, so the spec of a module with many exports passes 300 lines.

assert-typescript at 5ae8aba had 2,608 errors of `noMagicNumbers` under `tests`, 279 of `noExcessiveLinesPerFunction` and 63 of `noExcessiveLinesPerFile`.

The managed configurations of Python and Go already exempt tests. ruff skips PLR2004, its rule of magic values, in the tests of pytest. golangci-lint skips `goconst` in `_test.go`, because a test states its values inline. ruff has no rule for the length of a file, and the Go baseline turns on no rule for the length of a function or a file.

## Decision

We will turn off `noMagicNumbers`, `noExcessiveLinesPerFunction` and `noExcessiveLinesPerFile` in the test files of the managed `biome.json`, because a spec states its values inline and grows with its cases, as the baselines of Python and Go already accept.

- An override of `biome.json` turns off the three rules for the paths of the globs `**/tests/**`, `**/*.test.*` and `**/*.spec.*`.
- The rules remain errors in every other file. In a probe with Biome 2.5.15, `Math.max(3, 4)` in a file under `src` still failed `noMagicNumbers`.

## Alternatives Considered

### Named constants and suppression comments in the specs

A spec would declare a constant for each value that it states, and a `biome-ignore` comment would precede each long callback of `describe`. It lost because assert-typescript would need 2,608 constants and a comment on each of 279 long functions. Each comment would repeat one decision in every spec that it applies to.

## Consequences

**Positive:**

- A spec states its values inline and keeps the cases of one module in one file.
- assert-typescript loses the 2,950 errors of the three rules under `tests` without a change to its specs.

**Negative:**

- Biome does not report the three rules in the helpers under `tests` either.
- Biome does not report them in any file whose name contains `.test.` or `.spec.`.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#50 |
| `noMagicNumbers` | <https://biomejs.dev/linter/rules/no-magic-numbers/> |
| `noExcessiveLinesPerFunction` | <https://biomejs.dev/linter/rules/no-excessive-lines-per-function/> |
| `noExcessiveLinesPerFile` | <https://biomejs.dev/linter/rules/no-excessive-lines-per-file/> |
| The exceptions of the managed configurations | RFC-0004 |
