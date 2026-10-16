---
adr: 0046
title: A local TOML file is a TOML document that merges into its managed file
status: Accepted
date: 2026-10-14
supersedes: RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0046: A local TOML file is a TOML document that merges into its managed file

## Status

Accepted

## Context

ergon init merged a local YAML file into its managed file, and appended the lines of every other local file. The managed TOML files are `ruff.toml` of Python and `clippy.toml` of Rust. `ruff.toml` ends with the table `[lint.per-file-ignores]`, so the lines of a local `ruff.toml` added entries to that table. Such a local file was a fragment and not a valid configuration of ruff.

ruff reads every `ruff.toml` under the path that it checks, and uses the closest one for each file. `ruff check --no-fix .` failed to parse the fragment `.ergon/local/ruff.toml`. `make check` failed in every Python repository with a local `ruff.toml`.

A valid local `ruff.toml` is the configuration of a project of its own for ruff. In a probe repository, `ruff check .` wrote `.ergon/local/.ruff_cache/`. ergon init check then refused that directory as a local file of no managed path.

## Decision

We will merge a local TOML file into its managed TOML file as a valid TOML document of its own, because a fragment of TOML fails every tool that reads the local file as a configuration.

- Tables merge key by key. The local value replaces the managed value of a key that both have, unless both values are arrays, whose items the merge appends, or both are inline tables, which merge in the same way. A table of an array of tables follows the last table of that array.
- creachadair/tomledit parses and formats the documents, so the merge keeps the comments of both. go-toml/v2 checks that the local file and the merged document are valid TOML.
- The managed `ruff.toml` sets `extend-exclude = [".ergon"]`, so ruff neither reads the local file nor writes a cache under `.ergon`.
- The managed TOML templates have the layout that tomledit formats, so a merge changes the lines of the managed file that the local file changes and no others.

## Alternatives Considered

### Exclude the directory .ergon from ruff alone

`extend-exclude = [".ergon"]` alone stops ruff from reading the fragment. The issue asked for it. It lost because the local file would stay a fragment of TOML. Roy asked for the merge of two valid TOML files.

### Merge the decoded data

go-toml/v2 would decode both files into maps and encode the merged map. It lost because the encoding drops the comments of the managed file. The first of them is the header that ergon init writes, and ergon license skips a managed file by that header.

## Consequences

**Positive:**

- `ruff check .` and `ruff format --check .` run in a repository with a local `ruff.toml`.
- A local TOML file is a valid configuration of its tool, which an editor can check.
- A local TOML file adds keys to any table of the managed file, and adds tables of its own.

**Negative:**

- A local TOML file of the earlier form, the lines of a table without its header, merges into the top level of the managed file. Such a repository moves the lines under the header of their table.
- A local TOML file cannot remove an item of an array or a key of the managed file.
- The service module requires creachadair/tomledit 0.0.29, a release before 1.0.
- A managed TOML template keeps the layout of tomledit, such as an empty line before a key with a comment.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#28 |
| The discovery of the configuration of ruff | <https://docs.astral.sh/ruff/configuration/#config-file-discovery> |
| tomledit | <https://github.com/creachadair/tomledit> |
| The local files of ergon init | RFC-0004 |
