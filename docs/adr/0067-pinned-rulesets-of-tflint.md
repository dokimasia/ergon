---
adr: 0067
title: lint-terraform loads the rulesets of lint.rulesets from releases that the section pins
status: Accepted
date: 2026-10-28
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0067: lint-terraform loads the rulesets of lint.rulesets from releases that the section pins

## Status

Accepted

## Context

The managed `.tflint.hcl` enables the terraform ruleset, which tflint bundles. Its comment told a repository to add the ruleset of its provider, such as tflint-ruleset-aws, in the local file. tflint loads such a ruleset from its plugin directory, and `tflint --init` downloads it there. `lint-terraform` did not run `tflint --init`:

- On a fresh machine, such as a runner of CI, tflint failed in every directory with `Plugin "aws" not found. Did you run "tflint --init"?`.
- The version of a ruleset in the local file is outside the baseline. Dependabot has no ecosystem for the plugins of tflint, and `ergon init upgrade` does not change a local file.

ergon tool run checks every release binary against a digest that the section pins, and the weekly update of the baseline moves each pin with its digests. The job of each language keeps the installs of ergon tool run in the cache of GitHub Actions. `ergon tool prune` removes from that cache every file that is not the install of a tool of the options.

tflint loads a plugin block without `source` and `version` from the file `tflint-ruleset-<name>` of its plugin directory, and `tflint --init` skips such a block. On 2026-10-28 tflint 0.64.0 loaded tflint-ruleset-aws 0.49.0, tflint-ruleset-azurerm 0.32.0 and tflint-ruleset-google 0.40.0 through links in one plugin directory. Of two plugin blocks with one name, it kept the later block.

## Decision

We will load the rulesets of `lint.rulesets` from release binaries that the section terraform pins, because ergon then checks the digest of each ruleset and the baseline moves its version.

- `terraform.tools` has `tflint-ruleset-aws`, `tflint-ruleset-azurerm` and `tflint-ruleset-google`, each pinned by version and a SHA-256 per platform, as `tflint` is.
- `terraform.lint.rulesets` lists the rulesets that `lint-terraform` loads: `aws`, `azurerm` or `google`. Its baseline is empty.
- The managed `.tflint.hcl` enables each ruleset of the list as a plugin without `source` and `version`.
- `lint-terraform` links the program of each listed ruleset into `.tflint.d/plugins` of the repository, which `.gitignore` ignores. It runs tflint with `TFLINT_PLUGIN_DIR` set to that directory.
- `lint-terraform` runs `tflint --init` before it lints. The command installs each plugin that the local file declares with a `source` and a `version` into the same directory.

## Alternatives Considered

### tflint --init alone

tflint would download each ruleset that the configuration declares, and check its signature. It lost because the version of each ruleset would remain outside the baseline. `ergon tool prune` would also remove the plugins from the cache of CI, because they are no tools of the options.

### The plugin directory of the install

`TFLINT_PLUGIN_DIR` would be the directory of the install of one ruleset in the tool cache. It lost because tflint reads one plugin directory, so a repository could load one ruleset.

### A plugin directory that ergon tool run builds

ergon tool run would link the rulesets for tflint and set its plugin directory. It lost because the runner installs the tools of every section, and only Terraform needs a plugin directory of tflint. The Makefile fragment of Terraform builds the directory with `ergon tool path`.

## Consequences

**Positive:**

- `lint-terraform` loads a ruleset of the list on a fresh machine, and the cache of CI keeps the install of each ruleset.
- The weekly update of the baseline moves the three rulesets with their digests, and `ergon init upgrade` brings them to a repository.

**Negative:**

- A ruleset of another source, such as tflint-ruleset-opa, remains in the local file. `tflint --init` downloads it on each run of CI, without a digest that ergon pins.
- A plugin block of the local file replaces the block of the same name that the list enables, because tflint keeps the later block.
- The weekly update moves each ruleset and tflint apart, so a ruleset can arrive that needs a newer tflint than the baseline pins.

**Neutral:**

- The terraform ruleset remains enabled with the preset `all`.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#62 |
| The plugins of tflint | <https://github.com/terraform-linters/tflint/blob/v0.64.0/docs/user-guide/plugins.md> |
| The plugin directory of tflint | <https://github.com/terraform-linters/tflint/blob/v0.64.0/plugin/discovery.go> |
