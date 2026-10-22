---
adr: 0059
title: A job of nightly.yml proposes the updates of the copies of spec with the GitHub App of ergon
status: Accepted
date: 2026-10-22
supersedes: RFC-0010 and ADR-0058, in part
superseded-by: none
rfc: RFC-0010
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0059: A job of nightly.yml proposes the updates of the copies of spec with the GitHub App of ergon

## Status

Accepted

## Context

An implementation got `spec.yml`, which called the reusable workflow `propose.yml` of stealth-scale/spec and passed it the secret `ERGON_APP_PRIVATE_KEY` by name. `propose.yml` created the token of the GitHub App from the variable `ERGON_APP_CLIENT_ID` and that secret. spec is a program of its own, but its workflow read the names of ergon's GitHub App. spec also needed a change before an implementation outside stealth-scale could pass the key.

`spec propose` needs a token with write access to the contents and the pull requests of the implementation. The token of the job has that access when the job grants it. The token of the App adds one property. GitHub starts the checks of a pull request that the token of the App opens. It holds the checks of a pull request that the token of the job opens until a maintainer approves them. `version.yml` already creates the token of the App when the repository has the variable. Without the variable it uses the token of the job.

## Decision

We will propose the updates of the copies in a job of `nightly.yml` that ergon renders with the token of its GitHub App, because the App is ergon's concern, and spec then reads only `GITHUB_TOKEN`.

- `workflow.Job` gets the key `App`. A job with it creates the token of the App before its setup when the repository has the variable `ERGON_APP_CLIENT_ID`, with write access to the contents and the pull requests. A step reads that token, or the token of the job without an App, through the expression `workflow.AppToken`.
- Only a job of `nightly.yml` may set `App`. A pull request from a fork runs `ci.yml` without the secrets of the repository, so the creation of the token would fail there.
- An implementation gets the job `propose-spec` of `nightly.yml`. It grants `contents: write` and `pull-requests: write`, and runs `ergon tool run spec.spec -- propose --repo "$GITHUB_REPOSITORY"` with that token in `GITHUB_TOKEN`.
- ergon init renders no `spec.yml`, and the section `spec` has no pin of `propose.yml`.

## Alternatives Considered

### spec.yml with propose.yml and a declared secret

`propose.yml` would declare the secret `ERGON_APP_PRIVATE_KEY`. `spec.yml` would pass it by name. It lost because spec would keep the names of ergon's App. The repository would also pass the secret to a workflow of another organization.

### Proposals with the token of the job alone

The job would open the pull requests with its own token. It lost because the checks of each proposal would wait for the approval of a maintainer. The App starts them, as it does for the version pull request.

## Consequences

**Positive:**

- spec reads only `GITHUB_TOKEN`, and the names of the App of ergon stay in ergon.
- A proposal runs the release of spec that the gate runs, from the tool cache of ergon.

**Negative:**

- `workflow.Job` has a key that only a job of `nightly.yml` accepts.
- The proposals run on the schedule of `nightly.yml`, every day, in place of every weekday.
- An implementation without a nightly job gets `nightly.yml` for the proposals, with the job that prunes the tool caches.
- Without the App, the checks of a proposal wait for the approval of a maintainer.

## References

| What | Where |
|---|---|
| The checks of a pull request that `GITHUB_TOKEN` opens | <https://docs.github.com/en/actions/concepts/security/github_token> |
| The secret of `propose.yml` | <https://github.com/stealth-scale/spec/issues/3> |
