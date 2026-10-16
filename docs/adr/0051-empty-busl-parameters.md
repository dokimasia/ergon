---
adr: 0051
title: A parameter of BUSL-1.1 may be empty
status: Accepted
date: 2026-10-16
supersedes: RFC-0003, in part
superseded-by: none
rfc: RFC-0003
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0051: A parameter of BUSL-1.1 may be empty

## Status

Accepted

## Context

`licenses.Config.Validate` refused BUSL-1.1 while one of its four parameters was empty, at the root of a repository and in each directory under BUSL-1.1. Each command that reads the section `license` of `.ergon.yaml` failed until every parameter had a value, `ergon init new` included.

A repository can start under BUSL-1.1 before it decides each parameter, such as the Change Date of its first release. The flags of `ergon init new` that set the parameters are optional.

## Decision

We will accept an empty parameter of BUSL-1.1 at the root and in each directory, and write its line of LICENSE with the label alone, because a repository then starts under BUSL-1.1 before it decides each parameter.

- The line of an empty Change Date is `Change Date:`, without trailing spaces.
- `Validate` still refuses a parameter that spans lines, because LICENSE states each parameter on one line.

## Alternatives Considered

### Each parameter required

`Validate` refuses BUSL-1.1 until each parameter has a value, and its error lists each empty parameter. It lost because a repository without a value for one parameter cannot run any command of ergon, `ergon init new` included.

## Consequences

**Positive:**

- `ergon init new --license BUSL-1.1` sets up a repository without the flags of the parameters.
- A directory under BUSL-1.1 can leave a parameter empty, as the root can.

**Negative:**

- ergon does not report an empty parameter. A repository can publish a LICENSE of BUSL-1.1 without a Change Date, and its Change License then does not apply on a known date.

**Neutral:**

- `.ergon.yaml` states an empty parameter as `""`.

## References

| What | Where |
|---|---|
| The text of BUSL-1.1 | <https://github.com/spdx/license-list-data/tree/v3.29.0>, `text/BUSL-1.1.txt` |
