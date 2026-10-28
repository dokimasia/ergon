---
adr: 0068
title: The step generate of Terraform writes the tables of the modules with terraform-docs
status: Accepted
date: 2026-10-28
supersedes: ADR-0062, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0068: The step generate of Terraform writes the tables of the modules with terraform-docs

## Status

Accepted

## Context

The README of a Terraform module lists the inputs and the outputs of the module, and terraform-docs writes those tables between the lines `<!-- BEGIN_TF_DOCS -->` and `<!-- END_TF_DOCS -->`. ergon had no tool for terraform-docs. A command line of `terraform.generate.commands` ran the terraform-docs of the `PATH`, which the job `check-terraform` does not install, so `verify-generate-terraform` failed in CI with such a line.

`verify-generate-<language>` runs the generators of a language and fails when the run changes a file of the repository. An empty list of command lines rendered neither `generate-<language>` nor `verify-generate-<language>`. The Makefile writes each word of a command line in quotes with `$` escaped, so a command line cannot run a variable of make, such as the command of a tool of ergon.

On 2026-10-28 terraform-docs v0.24.0 read `.terraform-docs.yml` from the working directory, wrote the README of each module that its recursive settings selected, and did not change a file on a second run. Without the file and without a formatter, it printed its usage and exited 0. With `--config .terraform-docs.yml`, it failed on a missing file.

## Decision

We will pin terraform-docs as a release binary of the section terraform and run it in the step generate when `generate.docs` is true, because `verify-generate-terraform` then fails on a README whose tables are out of date.

- `terraform.tools` has `terraform-docs`, pinned by version and a SHA-256 per platform.
- `generate.docs` is false at the baseline. When it is true, `generate-terraform` runs terraform-docs with `--config .terraform-docs.yml` on each path of the section, before the command lines of `generate.commands`.
- The step generate needs `generate.docs` or a command line. Terraform renders both of its targets for either.
- The variable `TERRAFORM_DOCS` of the Makefile runs the release of the section through ergon tool run.

## Alternatives Considered

### A step of its own with --output-check

A step `docs` would run `terraform-docs --output-check` in the gate. It lost because the step generate already fails on a generated file that is out of date. A step of its own would also add a step to the vocabulary that every section shares.

### A command line that names terraform-docs

The Makefile would run each command line whose first word is the name of a tool of the section through ergon tool run. It lost because such a line would run another program than the one that it names.

## Consequences

**Positive:**

- CI checks the tables of every module README with the terraform-docs of the baseline.

**Negative:**

- A repository with `generate.docs` writes `.terraform-docs.yml`, and its settings select the READMEs that terraform-docs writes.
- Terraform renders the targets of the step generate without a command line when `generate.docs` is true.

**Neutral:**

- A repository without `generate.docs` keeps its command lines and its targets as they are.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#63 |
| The configuration of terraform-docs | <https://terraform-docs.io/user-guide/configuration/> |
| The recursive settings of terraform-docs | <https://terraform-docs.io/user-guide/configuration/recursive/> |
| The step generate | ADR-0062 |
