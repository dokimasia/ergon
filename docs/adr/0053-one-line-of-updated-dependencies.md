---
adr: 0053
title: A changelog entry has one line of updated dependencies
status: Accepted
date: 2026-10-17
supersedes: RFC-0002 and ADR-0003, in part
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0053: A changelog entry has one line of updated dependencies

## Status

Accepted

## Context

ergon writes `CHANGELOG.md` in the formats of changesets 3.0.3. The format `@changesets/cli/changelog` writes a line `- Updated dependencies [<commit>]` for each changeset of the packages that a release requires, and then the list of the updated requirements. A commit with six changesets repeats its line six times. The entry 0.3.3 of `ergon-lang-typescript/CHANGELOG.md` had seven such lines for two commits. Its entry 0.3.2 had fourteen lines for six commits.

`@changesets/changelog-github` writes one such line with a link for each changeset. It repeats the link of a commit in the same way.

## Decision

We will write each commit of the changesets of the updated dependencies once in one line of the entry because a line or a link for each changeset repeats the same commit.

- The format `@changesets/cli/changelog` writes `- Updated dependencies [64b7050, 67ba789]`, with the short commits in the order of the plan. Changesets without a commit give `- Updated dependencies` once.
- The format `@changesets/changelog-github` writes the link of each commit once.
- An entry whose updated requirements have no changesets lists the requirements alone, as changesets does.
- Every other line of an entry renders as changesets 3.0.3 renders it.
- The entries of earlier releases in the changelogs of ergon get the same line.

## Alternatives Considered

### The line of each changeset

ergon writes the dependency lines as changesets does. The output of changesets 3.0.3 on the same changeset files then checks every line of an entry. It lost because a release of many changesets repeats the line of one commit. The repeated lines add nothing to the entry.

## Consequences

**Positive:**

- An entry lists each commit of its updated dependencies once.

**Negative:**

- The output of changesets 3.0.3 no longer checks the dependency line of an entry.
- A repository that switches from changesets writes its next entries in the shorter form, above entries with a line for each changeset.

**Neutral:**

- The notes of the published GitHub releases keep a line for each changeset.

## References

| What | Where |
|---|---|
| The dependency line of `@changesets/cli/changelog` | <https://github.com/changesets/changesets/blob/main/packages/changelog-git/src/index.ts> |
