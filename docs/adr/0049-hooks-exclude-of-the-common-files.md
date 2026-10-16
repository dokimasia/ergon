---
adr: 0049
title: The key common.hooks.exclude keeps paths out of the hooks that change or refuse a file
status: Accepted
date: 2026-10-16
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0049: The key common.hooks.exclude keeps paths out of the hooks that change or refuse a file

## Status

Accepted

## Context

thesm-os/engine-triton keeps copies of files of model checkpoints from the Hugging Face Hub in `models/`, and its build checks the SHA-256 of each copy. The hooks of the managed `.pre-commit-config.yaml` change or refuse such a copy:

- `end-of-file-fixer` appends a newline to a copy that ends without one, such as the `tokenizer.json` of `Qwen/Qwen3.8-27B`.
- `check-added-large-files` with `--maxkb=500` refuses that `tokenizer.json`, which has 12,809,320 bytes.

ergon merges a local YAML file into its managed file by appending the lists of the local file. A local entry of `end-of-file-fixer` with an `exclude` adds a second hook, so the managed hook still changes the copies.

## Decision

We will write the regular expression of `common.hooks.exclude` as the key `exclude` of `trailing-whitespace`, `end-of-file-fixer`, `mixed-line-ending` and `check-added-large-files`, because these four hooks change or refuse the content of a file, while the other hooks only check it.

- The key is empty at the baseline, and the template then does not write an `exclude`.
- `trailing-whitespace` skips the Markdown files with `exclude_types: [markdown]` in place of `exclude: '\.md$'`. Its key `exclude` then contains the expression of the section alone. identify 2.6.20 gives the files `*.md` and `*.markdown` the type `markdown`.
- ergon writes the expression as it is. pre-commit compiles it with the `re` module of Python, whose syntax differs from that of Go, and refuses a configuration whose expression does not compile.

## Alternatives Considered

### The expression joined with \.md$ in the exclude of trailing-whitespace

The issue proposed one `exclude` of `trailing-whitespace` that joins `\.md$` with the expression of the section. It lost because Python refuses an expression with global flags after its start. On Python 3.14.8, `re.compile` of `\.md$|(?x)^models/` failed with "global flags not at the start of the expression".

### A top-level exclude in the local file

A repository can write a top-level `exclude` in `.ergon/local/.pre-commit-config.yaml`. It lost because pre-commit applies that key to every hook, so `check-json` and `detect-private-key` also skip the paths.

## Consequences

**Positive:**

- A repository keeps the files of its expression byte for byte through the hooks of each commit.
- The other hooks still check those files. In a probe with the expression `^models/`, `end-of-file-fixer` left a copy under `models/` unchanged, and `check-json` refused an invalid JSON file in the same directory.

**Negative:**

- ergon does not check the expression. With an expression that does not compile, every run of the hooks fails with `InvalidConfigError` until the repository fixes `.ergon.yaml`.
- `trailing-whitespace` also skips the files `*.markdown`, which `\.md$` did not match.

**Neutral:**

- `.pre-commit-config.yaml` changes in every repository at its next `ergon init sync`, because `exclude_types` replaces `exclude` in `trailing-whitespace`.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#34 |
| The keys of a hook | <https://pre-commit.com/#pre-commit-configyaml---hooks> |
| The types of a file | <https://github.com/pre-commit/identify> at v2.6.20 |
| The local files of ergon init | RFC-0004 |
