---
adr: 0048
title: The key audit.skip of the section python leaves packages without a release on PyPI out of the scan
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

# ADR-0048: The key audit.skip of the section python leaves packages without a release on PyPI out of the scan

## Status

Accepted

## Context

`audit-python` exports `uv.lock` to a `pylock.toml` with `uv export`, and scans the export with `pip-audit --locked --strict`. pip-audit audits a package through its release on PyPI. It cannot audit a package from a git repository or from another index. pip-audit 2.10.1 states that `--strict` fails "the entire audit if dependency collection fails on any dependency".

thesm-os/engine-triton takes `dokimi-assert` from a commit of `dokimasia/assert-python` and `torch` from the CPU index of PyTorch, so `make check` failed there. In a probe project with `dokimi-assert` from git, `make audit-python` failed with this error:

```text
ERROR:pip_audit._cli:dokimi-assert: Dependency not found on PyPI and could not be audited: dokimi-assert (0.1.0)
```

## Decision

We will leave the packages of `python.audit.skip` out of the export of `audit-python`, because pip-audit cannot audit a package without a release on PyPI.

- `python.audit.skip` lists the names of PyPI packages, and is empty at the baseline. Its `Validate` refuses a name that PEP 508 does not allow, and a name that the list states twice.
- `audit-python` passes each name to `uv export` as `--no-emit-package`.
- pip-audit keeps `--strict`. A new dependency without a release on PyPI fails the scan until a person adds it to the list.

## Alternatives Considered

### pip-audit without --strict

Without `--strict`, pip-audit exits 0 and lists each package that it cannot audit as skipped. It lost because the scan would then pass each dependency that pip-audit cannot audit. A new dependency of that kind would pass without a person deciding it.

## Consequences

**Positive:**

- `make check` passes in a Python repository with dependencies from a git repository or from another index.
- pip-audit still scans the other packages. In the probe, it reported PYSEC-2026-215 for `idna` 3.10, and no known vulnerability for `idna` 3.15.
- `.ergon.yaml` lists each package that the scan leaves out.

**Negative:**

- pip-audit does not scan the packages of the list. The repository follows the advisories of such a package elsewhere, such as in the repository that publishes it.
- A package of the list stays out of the scan after PyPI gets a release of it, until a person removes it from the list.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#33 |
| The options of uv export | <https://docs.astral.sh/uv/reference/cli/#uv-export> |
| pip-audit | <https://github.com/pypa/pip-audit> at v2.10.1 |
