---
adr: 0061
title: A local JSON file is a JSON document that merges into its managed file
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

# ADR-0061: A local JSON file is a JSON document that merges into its managed file

## Status

Accepted

## Context

ergon init merged a local YAML or TOML file into its managed file, and appended the lines of every other local file. The one managed JSON file is `biome.json` of the js toolchain. A local JSON file appended to it gave a file that is not JSON, so RFC-0004 stated that a local file cannot extend `biome.json`.

A repository had no place for a decision about a rule of Biome. assert-typescript at 5ae8aba had 315 errors of six rules that its design requires. `useMaxParams` reports each assertion, because the standard fixes the parameters of an assertion. `noBitwiseOperators` reports the integers and the floats of the property engine. Each site needed a suppression comment, while assert-python records the same kind of decision once in its local `ruff.toml`.

Biome reads a `biome.json` below the root of a project as a nested configuration. A `biome.json` under `.ergon/local` stopped Biome 2.5.15 with "Found a nested root configuration".

Biome's formatter puts an array on one line when the line fits the line width, and it checks the format of `biome.json` itself.

## Decision

We will merge a local JSON file into its managed JSON file as a JSON document of its own, because a repository then records a decision about a rule of Biome once in its local file.

- Objects merge member by member. For a name that both objects have, the local value replaces the managed value, unless both values are objects, which merge, or both are arrays, to which the merge appends the local items. A member of the local object that the managed object lacks follows the members of the managed object.
- The local overrides of `biome.json` follow the managed overrides. Biome applies the last override that matches a file, so a local override takes precedence over a managed one.
- The merge writes the document with an indentation of two spaces and each member and each item on a line of its own, as `JSON.stringify(value, null, 2)` writes it. Go's `encoding/json/jsontext` reads and writes both documents. It keeps the order of the members, and it refuses a duplicate name, a comment and any content after the value.
- The managed `biome.json` has the same layout, so a merge changes only the lines that the local file changes. An override sets `json.formatter.expand` to `always` for `biome.json`, so Biome's formatter accepts that layout. Biome formats `package.json` in the same way by default.
- `files.includes` of the managed `biome.json` has the force-ignore `!!**/.ergon`, so Biome does not read a configuration under `.ergon`.

## Alternatives Considered

### An extends of the local file in the managed biome.json

The managed `biome.json` would name `.ergon/local/biome.json` in its key `extends`, and Biome would merge the two files itself. It lost because Biome applies the extending file last. Biome's configuration reference states that order, and that the last matching override takes precedence. An override of the managed file would then take precedence over a local override of the same rule, such as `useImportExtensions`.

### The layout of Biome's formatter at the line width

The merge would put an array on one line when the line fits the line width of `biome.json`, as Biome's formatter does by default. It lost because the merge of the overlay would then need the line width of the js toolchain. Every other managed file merges without a line width.

## Consequences

**Positive:**

- A repository relaxes a rule of Biome in one override of its local file, with the globs of the paths that the decision covers.
- A local `biome.json` is a valid configuration of Biome, which an editor checks against the schema of Biome.

**Negative:**

- The managed `biome.json` has more lines, because each item of an array has a line of its own.
- A local JSON file cannot remove an item of an array or a member of an object of the managed file.
- The merge writes each string with the escapes of `jsontext`, so a local `é` becomes `é`.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#51 |
| The merge of a local TOML file | ADR-0046 |
| `extends`, `overrides` and `json.formatter.expand` of Biome | <https://biomejs.dev/reference/configuration/> |
| `encoding/json/jsontext` | <https://pkg.go.dev/encoding/json/jsontext> |
| The local files of ergon init | RFC-0004 |
