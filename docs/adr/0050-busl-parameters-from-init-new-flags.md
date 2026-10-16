---
adr: 0050
title: ergon init new sets the parameters of BUSL-1.1 from its flags
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

# ADR-0050: ergon init new sets the parameters of BUSL-1.1 from its flags

## Status

Accepted

## Context

The LICENSE of BUSL-1.1 states four parameters before its terms: the Licensed Work, the Additional Use Grant, the Change Date and the Change License. They are options of the section `license` of `.ergon.yaml`, under `license.parameters`. The flags of `ergon init new` set its answers, and no flag set a parameter.

A new repository has no `.ergon.yaml`. To set up a repository under BUSL-1.1, a person wrote `.ergon.yaml` with the parameters before the first run of `ergon init new`. `ergon init new` adds the license header only to a `.ergon.yaml` that it creates, so `ergon license check` failed until the person also ran `ergon license fix`.

## Decision

We will add the flags `--licensed-work`, `--additional-use-grant`, `--change-date` and `--change-license` to `ergon init new` and write their values into `license.parameters` of `.ergon.yaml`, because one command then sets up a repository under BUSL-1.1 with a `.ergon.yaml` that has its license header.

- The name of each flag is the key of its parameter in `.ergon.yaml`.
- A flag without a value leaves its parameter as `.ergon.yaml` has it.
- `ergon init new` writes each value as a string, so YAML does not read a Change Date such as 2030-01-01 as a date.
- A flag with a value and a license other than BUSL-1.1 is a usage error, with the exit status 2.
- A parameter is an option and not an answer. The lock does not record a parameter, and `ergon init sync` does not have a flag for one. A repository changes a parameter in `.ergon.yaml`.
- The commands of `baseline.Repository` take such values as settings of `baseline.Options`, by the key of each option. A command resolves the options as if `.ergon.yaml` contained each setting, so `ergon init new` writes nothing when it fails.

## Alternatives Considered

### A .ergon.yaml that a person writes before ergon init new

A person writes `license.parameters` into a new `.ergon.yaml`, then runs `ergon init new` and `ergon license fix`. It lost because it takes three steps and a block of YAML from the documentation, and `ergon license check` fails between the second step and the third.

### An error of ergon init new that prints the block of .ergon.yaml

The parameters stay in `.ergon.yaml` alone, and the error of `ergon init new` prints the block that a person copies into the file. It lost because the person still writes the file before the first run and runs `ergon license fix` after it.

## Consequences

**Positive:**

- One run of `ergon init new` with the four flags writes a repository under BUSL-1.1 that passes `ergon init check` and `ergon license check`.
- A variable of the environment can set each flag, such as `ERGON_INIT_NEW_CHANGE_DATE`.

**Negative:**

- `ergon init new` has four flags that only one license reads, and its help lists them for every license.
- A flag shares its name with the key of its parameter, so a rename of the key renames the flag.
- A repository changes a parameter, such as the Change Date of a new release, by an edit of `.ergon.yaml`, because `ergon init sync` does not have a flag for it.

**Neutral:**

- The lock and the answers of `ergon init` do not change.

## References

| What | Where |
|---|---|
| The text of BUSL-1.1 | <https://github.com/spdx/license-list-data/tree/v3.29.0>, `text/BUSL-1.1.txt` |
