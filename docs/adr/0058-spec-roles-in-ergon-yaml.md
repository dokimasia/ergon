---
adr: 0058
title: The key spec.roles runs spec in the gate and the proposals of a repository
status: Accepted
date: 2026-10-22
supersedes: none
superseded-by: ADR-0059, in part
rfc: RFC-0010
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0058: The key spec.roles runs spec in the gate and the proposals of a repository

## Status

Accepted

## Context

spec vendors a conformance definition into each implementation of a polyglot library. A repository on ergon's baseline added the checks of spec through the local file of `ci.yml` and wrote the workflow of its proposals by hand. Those pins of spec were outside the weekly update of the baseline.

ergon init renders from the answers, the options and the contributions alone, so it cannot read `.spec.yaml` to find the role of the repository. The checks of a repository depend on its role, because `spec check` needs the section `implements` and `spec manifest` needs the section `definition`.

## Decision

We will render the targets, the job and the workflow of spec from the key `spec.roles` of `.ergon.yaml`, because a rendering cannot read the sections of `.spec.yaml`.

- A repository without a role gets nothing of spec.
- ergon pins spec as a release binary of stealth-scale/spec and runs it through `ergon tool run spec.spec`.
- An implementation gets `check-spec` in the Makefile, `spec check --remote` in the job `spec` of `ci.yml`, and `spec.yml`, which calls the reusable workflow `propose.yml` of spec on each weekday.
- A definition gets `generate-spec` and `verify-generate-spec` in the Makefile, and `spec manifest --check` in the job `spec`.
- `spec.yml` passes `ERGON_APP_PRIVATE_KEY` to `propose.yml` by name, because GitHub passes secrets through `secrets: inherit` only within one organization or enterprise.

## Alternatives Considered

### Commands of spec that read the roles

`spec check` would also check the manifest of a definition. `spec propose` would pass a repository without the section `implements`. It lost because ergon still could not leave `spec.yml` out of the repository of a definition. `make generate` could not write the manifest of a definition either.

### The action setup-spec

The job would install spec with the action of spec. It lost because the action builds spec from source on each run, so the job needs Go and the time of a build.

## Consequences

**Positive:**

- A repository adds spec to its gate and to its proposals with one key, and the weekly update of the baseline moves its pins.
- The job `spec` installs spec from the tool cache, without Go.

**Negative:**

- Every `.ergon.yaml` has the section `spec`, also in a repository without a role.
- The roles restate the sections of `.spec.yaml`. A role without its section fails the target and the job.
- `spec.yml` fails until `propose.yml` declares the secret `ERGON_APP_PRIVATE_KEY`.

## References

| What | Where |
|---|---|
| spec, its configuration and its CI | <https://github.com/stealth-scale/spec/blob/main/README.md> |
| The secret of `propose.yml` | <https://github.com/stealth-scale/spec/issues/3> |
