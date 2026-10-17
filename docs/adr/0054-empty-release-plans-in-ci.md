---
adr: 0054
title: CI treats an empty release plan as a plan without changesets
status: Accepted
date: 2026-10-17
supersedes: RFC-0002, in part
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0054: CI treats an empty release plan as a plan without changesets

## Status

Accepted

## Context

`version.yml` runs `ergon release ci version` after each run of `ci.yml` that passes on a push to `main`. The command writes the release plan as `ergon release version` does. The write removes the changesets of the plan. The command then opened a version pull request for every plan with changesets.

`ergon release status` asks for a changeset of each package that a branch changes. A change of documentation or tests then adds a changeset that does not release a package. Such a changeset is empty or lists its packages at the level none. The repository of ergon adds a changeset of the root module at none with each decision record. A release plan is empty when every release of the plan is at none. The version pull request of an empty plan removes the changesets and does not release a package.

changesets/action v2.1.2 does not create a pull request when every changeset is empty. It creates one when a changeset lists a package at none. The parser of changesets returns a release of the level none for such a package.

`ergon release ci select-mode` returned version while `.changeset` had changesets. A changeset at none or without packages blocked the job publish until a version pull request removed it.

## Decision

We will treat an empty release plan as a plan without changesets in `ergon release ci version` and `ergon release ci select-mode` because the version pull request of an empty plan does not release a package.

- `ergon release ci version` proposes only the stale lockfiles for an empty plan, as it does in a repository without changesets. It writes `No unreleased changesets found.` when every lockfile is current.
- `ergon release ci select-mode` chooses the mode of an empty plan from the publish plan and the stale lockfiles alone.
- CI does not remove the changesets of an empty plan. The next version pull request removes them together with the changesets that release a package.
- `ergon release version` on a workstation still removes them, as `changeset version` does.

## Alternatives Considered

### Skip only a plan of empty changesets

CI skips the version pull request when every changeset is empty, as changesets/action v2.1.2 does. It lost because a changeset at none does not release a package either. The repository of ergon adds such a changeset with each decision record.

### A version pull request for every plan with changesets

CI proposes every plan with changesets. It lost because the version pull request of an empty plan does not release a package.

## Consequences

**Positive:**

- A change with changesets at none or without packages does not open a version pull request.
- A changeset at none or without packages does not block a publish.

**Negative:**

- `.changeset` can contain changesets without a pending release until the next version pull request removes them.
- CI keeps such changesets, and `ergon release version` on a workstation removes them.
- `ergon release ci version` writes `No unreleased changesets found.` while such changesets exist.
- select-mode differs from changesets/action, which does not publish while any changeset exists.

## References

| What | Where |
|---|---|
| The rule of changesets/action v2.1.2 for empty changesets | <https://github.com/changesets/action/blob/v2.1.2/src/index.ts> |
| The releases that changesets parses from the front matter | <https://github.com/changesets/changesets/blob/main/packages/parse/src/index.ts> |
