---
adr: 0040
title: The archives and the packages of a release include the NOTICE of a repository under Apache-2.0
status: Accepted
date: 2026-10-10
supersedes: RFC-0007, in part
superseded-by: none
rfc: RFC-0007
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0040: The archives and the packages of a release include the NOTICE of a repository under Apache-2.0

## Status

Accepted

## Context

The managed configuration of GoReleaser puts the binary, `LICENSE`, `README.md` and the completions into each archive of a command. Each Linux package installs `LICENSE` as `/usr/share/doc/<name>/copyright`. ergon init writes a `NOTICE` beside `LICENSE` for a repository under Apache-2.0, and for no other license.

Section 4(d) of the Apache License 2.0 requires a redistribution of a work that has a `NOTICE` to include a readable copy of it. The archives and the packages of an Apache-2.0 repository left the `NOTICE` out. kanon's own configuration of GoReleaser had put its `NOTICE` into each archive, and its move to the managed configuration would have dropped it.

## Decision

We will put the `NOTICE` of a repository under Apache-2.0 into each archive and each package of its releases, because the Apache License 2.0 requires a distribution of the work to include its `NOTICE`.

- An archive has `NOTICE` beside `LICENSE`.
- A package installs `NOTICE` as `/usr/share/doc/<name>/NOTICE`, beside the copyright file.
- The configuration of a repository under any other license has neither entry.

## Alternatives Considered

### A glob NOTICE* in every configuration

Every configuration would list a glob `NOTICE*` among the files of an archive, as it lists `LICENSE*`. GoReleaser 2.18.2 skips an archive glob that matches no file and logs a warning. This alternative came up while the fix was built, from the source of GoReleaser. It lost because a package of nFPM names one file, which must exist. A repository without a `NOTICE` would also log a warning for each archive.

## Consequences

**Positive:**

- The binary release of a repository under Apache-2.0 includes its `NOTICE`, as the license requires.
- kanon keeps the `NOTICE` in its archives after its move to the managed configuration.

**Negative:**

- The entry follows the license of the repository. A command with a license of its own in a repository under Apache-2.0 takes the `NOTICE` of the repository, as it takes its `LICENSE`.

**Neutral:**

- The source archive of a release already contains `NOTICE`, because it contains the tree of the release.

## References

| What | Where |
|---|---|
| The archives and the packages of a release | RFC-0007 |
| The NOTICE of a redistribution | The Apache License 2.0, section 4(d) |
