---
adr: 0042
title: The toolkit vite-plus runs the gates of JavaScript and TypeScript through vp
status: Accepted
date: 2026-10-11
supersedes: RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0042: The toolkit vite-plus runs the gates of JavaScript and TypeScript through vp

## Status

Accepted

## Context

A repository of Vite+ formats, lints and checks the types of every package with `vp check`, and runs its tests with `vp test`. `vp check` runs Oxfmt, Oxlint and tsgolint with the blocks `fmt` and `lint` of `vite.config.ts`. In dokimasia/interface that file extends `@stealthscale/vite-config/preset/workspace`, a published package with the rules of every repository of stealthscale.

The gates of JavaScript and TypeScript run Biome with the managed `biome.json`, and `tsc` at the root of the repository. The request in dokimasia/ergon#22 measured three conflicts in dokimasia/interface:

- `vp check` passed its 31 files of TypeScript and JSON, while `biome ci` reported 41 errors. Each error contradicted a rule of the repository's configuration, such as the line width of 100 of Oxfmt against the 88 of Biome.
- `tsc` with the flags of `lint-typescript` listed 2 of the 21 files of TypeScript. The root `tsconfig.json` of a workspace of pnpm includes the files of the root alone.
- After `vp fmt`, `ergon init check` reported 7 of the 22 managed files as edited, because Oxfmt also formats YAML and Markdown.

## Decision

We will add the key `toolkit` to the section `js`, with the values `biome` and `vite-plus`, because a repository of Vite+ states its rules of format, lint and types in `vite.config.ts`, which the managed `biome.json` and `tsc` contradict.

- For vite-plus, `fmt-js` runs `vp fmt`, and `lint-js` runs `vp check`. `lint-typescript` does not run `tsc`. The targets of the tests run `vp test --run`.
- The package manager runs the `vp` of the packages of the repository, as `pnpm exec vp` or `npm exec -- vp`.
- ergon init renders no `biome.json` for vite-plus. It renders a managed `.prettierignore` with `.ergon/`, `.ergon.yaml` and the path of every managed file, and Oxfmt reads that file.
- The templates of a language read the choices of the section of its toolchain as `.Toolchain`. A template reads the paths of the managed files of the rendering as `.Managed`. The renderer renders twice to know them, and refuses a template that renders a managed file in one of the two renderings alone.
- `vite.config.ts`, `package.json`, `pnpm-workspace.yaml` and `tsconfig.json` remain files of the repository.

## Alternatives Considered

### A fixed list in .prettierignore

The template of `.prettierignore` would list the paths at which the producers render their managed files. It lost because the list would repeat the templates of every other producer, and miss the files that a producer places, such as the configuration of GoReleaser of a module with a command.

### vp through ergon tool run

`.ergon.yaml` would pin `vp`, and `ergon tool run` would start it, as it starts Biome. It lost because a test imports Vitest from the packages of the repository. A `vp` from the cache of ergon would run a second copy of Vitest beside the one that the tests import.

## Consequences

**Positive:**

- A repository of Vite+ checks its sources with its own rules, through one formatter, one linter and one type checker.
- `vp fmt` leaves the managed files as ergon renders them, so `ergon init check` passes after it.

**Negative:**

- For vite-plus, ergon does not enforce a type check. `vp check` checks the types only when the block `lint.options` of `vite.config.ts` enables `typeCheck`. The strict flags that `lint-typescript` passes to `tsc` for biome do not apply.
- ergon manages no linter configuration of a repository of Vite+. Its rules come from `vite.config.ts` and the packages that it extends.
- `.ergon.yaml` does not pin `vp`. The lockfile of the repository pins it, and Dependabot updates it.
- Every rendering of ergon init runs the templates twice.

**Neutral:**

- The section `js` keeps the pin of Biome for vite-plus, and no target runs it.

## References

| What | Where |
|---|---|
| The request | dokimasia/ergon#22 |
| The checks of vp check | <https://viteplus.dev/guide/check> |
| The ignore files of Oxfmt | <https://oxc.rs/docs/guide/usage/formatter/ignore-files.html> |
| The linter configuration of each language | RFC-0004 |
