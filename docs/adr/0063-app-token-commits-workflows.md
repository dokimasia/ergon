---
adr: 0063
title: The scheduled proposals commit with a token of the GitHub App that may change workflows
status: Accepted
date: 2026-10-26
supersedes: RFC-0005, in part
superseded-by: none
rfc: RFC-0011
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0063: The scheduled proposals commit with a token of the GitHub App that may change workflows

## Status

Accepted

## Context

`baseline.yml` runs `ergon init ci upgrade`, which commits the changes of a sync through `createCommitOnBranch` with `GITHUB_TOKEN`. ergon's own `update-baseline.yml` commits its proposal the same way. `GITHUB_TOKEN` has no `workflows` scope, so GitHub refuses every commit that adds or changes a file under `.github/workflows/`, with "Resource not accessible by integration".

On 2026-10-26 the job failed in all 7 public repositories whose sync wrote a managed workflow. It opened a pull request in the 4 repositories whose logs were read and whose sync did not write one. Most releases of ergon change a managed workflow, because the weekly update of the baseline moves the pins of the actions. A run of `update-baseline` in a scratch clone with the fixes of that day rewrote four of ergon's own workflows.

`version.yml` already creates a token of the GitHub App of ergon when the repository has the variable `ERGON_APP_CLIENT_ID`, and opens the version pull request with it. The setting "Allow GitHub Actions to create and approve pull requests" configures `GITHUB_TOKEN` alone.

## Decision

We will create a token of the GitHub App of ergon with write access to workflows in `baseline.yml` and `update-baseline.yml`, and commit their proposals with it, because `GITHUB_TOKEN` cannot commit a change of a workflow.

- The step "Create a token of the GitHub App" runs when the repository has the variable `ERGON_APP_CLIENT_ID`. It requests write access to the contents, the pull requests and the workflows. The proposal reads that token, or `GITHUB_TOKEN` in a repository without the App.
- The App needs the repository permission Workflows with write access, and each installation must grant it.
- When GitHub refuses the commit of a proposal that changes a workflow, `ergon init ci upgrade` returns an error that contains the paths of those workflows and the variable and the secret that the App needs.
- `update-baseline.yml` is no managed file, so its step is written in the file itself.

## Alternatives Considered

### A personal access token with the scope workflow

The workflows would pass a token of a person that may change workflows, from a secret. It lost because the token belongs to a person and expires. ergon already creates the token of its App for the version pull request and for the proposals of spec.

### An issue in place of the pull request

The job would open an issue with the commands of the upgrade when the sync changes a workflow. It lost because a person then runs the upgrade by hand in most weeks, which is the work that the job exists to remove.

### A proposal without the workflows

The proposal would contain every changed file but the workflows. It lost because the managed files of a release change together. `ergon init check` fails a tree with the files of one release and the workflows of another.

## Consequences

**Positive:**

- An upgrade that changes a workflow opens its pull request.
- GitHub starts the checks of the proposal without the approval of a maintainer, and the setting of pull requests from `GITHUB_TOKEN` plays no part.

**Negative:**

- A key of the App with write access to workflows can change the CI of each repository of its installation.
- A repository without the App, its variable and its secret still fails the job when the sync changes a workflow. The error of the job contains the remedy.

**Neutral:**

- A repository without the App still opens the proposals that do not change a workflow with `GITHUB_TOKEN`.

## References

| What | Where |
|---|---|
| The scopes of `GITHUB_TOKEN`, which include no `workflows` | <https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax> |
| The setting that lets `GITHUB_TOKEN` create pull requests | <https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository> |
| The input `permission-workflows` of the action that creates the token | <https://github.com/actions/create-github-app-token/blob/main/README.md> |
