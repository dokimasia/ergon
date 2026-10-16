---
adr: 0044
title: The key code-security renders the security workflows, and the Scorecard runs in a public repository alone
status: Accepted
date: 2026-10-14
supersedes: RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0044: The key code-security renders the security workflows, and the Scorecard runs in a public repository alone

## Status

Accepted

## Context

ergon init renders `security.yml` and `codeql.yml` in every repository. In a private repository each job of `security.yml` failed. The last runs of thesm-os/ledger, thesm-os/kernel, thesm-os/sharder and dokimasia/interface failed on each push and each weekly schedule. Each of them is a private repository of an organisation on GitHub's free plan without GitHub Advanced Security.

- The CodeQL analysis failed with "Resource not accessible by integration" when it read the run of its workflow. GitHub's starter workflow of CodeQL grants `actions: read` for a private repository, and the job `analyze` of `codeql.yml` did not grant it.
- GitHub's documentation states that code scanning in a private repository needs a GitHub Code Security license, and that the dependency review action needs GitHub Code Security or GitHub Advanced Security there.
- The Scorecard failed in its query of the commits with the same message. Its action supports a private repository only with GitHub Advanced Security, and then with read permissions that ergon does not grant.

A scheduled run has no event payload, and the context `github` has no property of the visibility. A scheduled job therefore queries the API of GitHub to learn whether its repository is private.

## Decision

We will render `security.yml` and `codeql.yml` only when the new key `github.code-security` is true, and run the Scorecard only in a public repository, because a private repository without GitHub Code Security fails every job of `security.yml`.

- `github.code-security` is true at the baseline. A private repository without GitHub Code Security sets it to false, and ergon init renders neither workflow.
- The job `analyze` of `codeql.yml` and each job `codeql-<language>` of `security.yml` grant `actions: read`. A called workflow can only keep or reduce the permissions of its caller.
- A job `visibility` queries the API of GitHub for the visibility of the repository, on the schedule alone. The Scorecard needs it and runs when the repository is public.

## Alternatives Considered

### The visibility from the event payload

The Scorecard would run under the condition `!github.event.repository.private`. It lost because a scheduled run, the only run of the Scorecard, has no event payload. The condition would be true in a private repository too.

### The Scorecard in a private repository with a license

With GitHub Advanced Security, the Scorecard would run in a private repository too, with the read permissions that its action lists for such a repository. It lost because the results of a private repository are not public, and no private repository on ergon has that license.

## Consequences

**Positive:**

- A private repository without GitHub Code Security does not run a job that fails for the license.
- A private repository with GitHub Code Security runs the CodeQL analysis and the dependency review.

**Negative:**

- A private repository sets `code-security: false` by hand. ergon init cannot read the license of a repository.
- The weekly schedule runs one job more, which asks the API before the Scorecard.
- A private repository without the key has no CodeQL analysis and no dependency review.

## References

| What | Where |
|---|---|
| The issue | dokimasia/ergon#26 |
| The starter workflow of CodeQL | <https://github.com/actions/starter-workflows/blob/main/code-scanning/codeql.yml> |
| The availability of code scanning | <https://docs.github.com/en/code-security/code-scanning/introduction-to-code-scanning/about-code-scanning> |
| The availability of the dependency review action | <https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/about-dependency-review> |
| The Scorecard action in a private repository | <https://github.com/ossf/scorecard-action#readme> |
| The events of a schedule | <https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows> |
| The GitHub files and their jobs | RFC-0004 |
