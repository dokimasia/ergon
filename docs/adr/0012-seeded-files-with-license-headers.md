---
adr: 0012
title: ergon init writes the license header into the files it seeds
status: Proposed
date: 2026-10-08
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0012: ergon init writes the license header into the files it seeds

## Status

Proposed

## Context

`ergon init` writes the managed files, the seeded files that are absent, and `.ergon.yaml`. `ergon license` skips the managed files, because they are the rendering of ergon. It checks every other file that has a comment syntax, so it checks the seeded files and `.ergon.yaml`, which the repository maintains.

On 2026-10-08, `ergon init new` with the languages Go, TypeScript and Python wrote 37 files into an empty repository. `ergon license check` then exited 1, and reported a missing header for each of these files:

- `README.md`, `CONTRIBUTING.md` and `SECURITY.md`
- `.ergon.yaml` and `.changeset/README.md`
- the `README.md` of `docs`, `docs/adr`, `docs/architecture`, `docs/rfc` and `docs/roadmap`

The job `license` of the rendered `ci.yml` runs that command, so the first push of a new repository fails CI.

RFC-0003 has a person run `ergon license fix` after `ergon init`. Its measurements started from clones that `ergon init new --force` had adopted. Neither the help of `ergon init new` nor its output names that step, and no hook or target of the baseline runs it.

## Decision

We will make `ergon init` write the license header of the section `license` into each file that it seeds and into `.ergon.yaml` when it creates them. A repository then passes `ergon license check` directly after `ergon init new`.

- `new`, `add` and `sync` add the header to each file that they seed, with the comment style and the text that `ergon license fix` writes for its path.
- The header states the year of the answer `year`, which is also the year of `LICENSE`.
- `ergon init` skips a seeded file without a comment syntax, as `ergon license fix` does.
- `ergon init` adds headers only to the files that it writes. The headers of the other files of a repository remain the work of `ergon license fix`.

## Alternatives Considered

### A step that a person runs

The output of `ergon init new` and the seeded `README.md` would tell a person to run `ergon license fix` before the first commit. It lost because the first CI run of a new repository still fails whenever the person skips the step. The lock already records the owner, the license and the year of every header.

### No check of the seeded files

`ergon license` would skip the seeded files, as it skips the managed files. It lost because the repository maintains the seeded files like its own source. A skip would leave them without a header permanently, while RFC-0003 puts a header on every file of the repository that has a comment syntax.

### `ergon license fix` over the whole repository

`ergon init new` would run `ergon license fix` after it writes the baseline. It lost because `ergon init new --force` adopts existing repositories. The fix would then rewrite the headers of every source file of such a repository in the commit of the baseline, including the vendored files that `license.exclude` does not name yet.

## Consequences

**Positive:**

- A new repository passes `ergon license check`, and the job `license` of its first CI run.
- The seeded files have the same header as the rest of the repository from their first commit.

**Negative:**

- `ergon init` depends on the header rules of `service/licenses`, so a change of those rules also changes the seeded files of new repositories.
- The goldens of every producer with a seeded file change once.

**Neutral:**

- The managed files keep their first lines, such as `# Managed by ergon init.`, and `ergon license` still skips them.
- After `ergon init sync --owner` or `--license`, `ergon license fix` updates the headers of the seeded files, as it updates every other header.

## References

| What | Where |
|---|---|
| The files that `ergon license` checks and skips | RFC-0003, Licenses |
| The classes of file of `ergon init` | RFC-0004, Repository initialization |
