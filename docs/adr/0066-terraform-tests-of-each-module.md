---
adr: 0066
title: test-terraform tests each module and validates every other configuration
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

# ADR-0066: test-terraform tests each module and validates every other configuration

## Status

Accepted

## Context

`test-terraform` ran `terraform init` and `terraform test` in the root of the repository. `terraform test` runs the test files of the root module alone, from the root and from its `tests` directory. A repository of modules keeps each module in `modules/<name>` and its tests in `modules/<name>/tests`, and has no configuration in its root:

- `make test-terraform` reported 0 tests and passed in such a repository.
- In seven such repositories with 2,732 test runs, `make check` passed without running one of them.
- `TERRAFORM_TEST_ARGS` cannot move the run into a module, because `-chdir` is an option of `terraform` that precedes the subcommand.

The gate of each language has a slot for a static analyzer, and the slot of Terraform was empty. `terraform validate` checks the syntax of a configuration and its consistency with the schemas of its providers. On 2026-10-28 Terraform 1.16.5 refused `terraform validate` in a module that declares `configuration_aliases`, with the error "Provider configuration not present". It accepted the same module through an example that configures the aliases. A test file of the module configured the aliases itself, and `terraform test` passed there.

## Decision

We will run `terraform test` in each directory with test files and `terraform validate` in each other directory with a configuration, because the gate then checks every module and every example of a repository.

- `test-terraform` lists the files under the paths of the section with `git ls-files`, so it skips `.terraform` and every other path that git ignores.
- The test files of a directory are the `.tftest.hcl` and `.tftest.json` files in it and in its `tests` directory. A directory with a `.tf` or a `.tf.json` file has a configuration.
- Each directory runs `terraform init -backend=false -input=false` first. `TERRAFORM_TEST_ARGS` passes its arguments to `terraform test` alone.
- `test-terraform` runs every directory, and fails at the end when one of them failed.
- The Makefile runs Terraform as the command of the variable `TERRAFORM`, which is `terraform` by default.

## Alternatives Considered

### A key with the directories of the modules

The section would list the directories that `test-terraform` runs. It lost because a repository would add each new module by hand, and a module that the list misses passes the gate without its tests.

### terraform validate in every directory with a configuration

`test-terraform` would validate each directory, also a directory with tests. It lost because Terraform refuses `terraform validate` in a module with `configuration_aliases`, so such a module would fail the gate while its tests pass.

### The directories from find

A `find` over the paths would list the directories. It lost because `git ls-files` skips the paths that git ignores, such as the modules that `terraform init` downloads into `.terraform`. The Makefile fragments of Java, Kotlin and Bash list their files the same way.

## Consequences

**Positive:**

- The gate runs the tests of every module, and validates each example and each module without tests.

**Negative:**

- A module that declares `configuration_aliases` needs a test file, because `terraform validate` refuses it alone.
- `terraform init` runs once for each directory. It downloads the providers of each directory unless the variable `TF_PLUGIN_CACHE_DIR` of the environment sets a directory of cached providers.
- `terraform init` writes `.terraform.lock.hcl` into each directory that it initializes. The repository commits or ignores these files.
- `test-terraform` needs a working tree of git.

**Neutral:**

- A repository whose tests are in its root runs them in the root.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#61 |
| The test files of `terraform test` | <https://developer.hashicorp.com/terraform/cli/commands/test> |
| `terraform validate` | <https://developer.hashicorp.com/terraform/cli/commands/validate> |
| `configuration_aliases` | <https://developer.hashicorp.com/terraform/language/modules/develop/providers> |
