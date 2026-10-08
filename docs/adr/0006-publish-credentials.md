---
adr: 0006
title: OIDC for npm, PyPI and crates.io, and environment secrets for Maven Central
status: Accepted
date: 2026-09-24
supersedes: none
superseded-by: ADR-0010, in part
rfc: RFC-0002
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0006: OIDC for npm, PyPI and crates.io, and environment secrets for Maven Central

## Status

Accepted

## Context

The publish job uploads to four registries. npm, PyPI and crates.io accept a short-lived token that the job gets through GitHub's OIDC token, with no stored secret. assert-typescript and assert-python already publish this way.

Maven Central requires a PGP signature on every file and accepts Sigstore bundles only in addition to PGP. Its Central Portal accepts a user token that cannot be limited to one namespace, and it has no OIDC route. assert-java signs its bundle with repository secrets, and uploads it with secrets of a `maven-central` environment.

crates.io refuses the OIDC exchange when the workflow was triggered by `workflow_run` or `pull_request_target`. dokimasia/stealth triggers its release workflow with `workflow_run`.

## Decision

We will publish to npm, PyPI and crates.io through OIDC trusted publishing, and to Maven Central with a user token and a PGP signing subkey stored as secrets of a `release` environment, because the Central Portal is the only one of the four without OIDC.

The release workflow triggers on `push` to `main` and runs the repository's CI through `workflow_call`. The `release` environment has required reviewers, no self-review, no administrator bypass and deployments from `main` only.

## Alternatives Considered

### Repository secrets for Maven Central

Repository secrets need no environment and no reviewer. They lost because anyone with write access to the repository can read them.

### Keyless Sigstore signing instead of a PGP key

The `dev.sigstore.sign` Gradle plugin signs through the job's OIDC token, and the Central Portal validates `.sigstore.json` bundles. It lost because Maven Central still requires a `.asc` file for every artifact, and Sonatype has stated that Sigstore does not replace PGP.

### A `workflow_run` trigger

The release workflow runs after a separate CI workflow completes, as in dokimasia/stealth. It lost because crates.io refuses the OIDC exchange for that trigger.

## Consequences

**Positive:**

- The publish job has no stored credential for npm, PyPI or crates.io.
- A reviewer approves every publish that reads the Maven Central secrets.

**Negative:**

- Maven Central needs four long-lived secrets: the token username, the token password, the signing subkey and its passphrase.
- The Maven Central token covers every namespace of its account. Only a dedicated account with access to one namespace narrows it.
- A crate's first release needs a user owner's API token, because crates.io trusted publishing releases only existing crates.
- Naming the environment puts it in the OIDC `sub` claim, so every npm, PyPI and crates.io trusted-publisher configuration has to name `release`.
- The crates.io token expires 30 minutes after the exchange, so `lang/rust` exchanges one per publish chunk.

**Neutral:**

- Required reviewers on an environment need a public repository on GitHub Free, Pro and Team plans.

## References

| What | Where |
|---|---|
| crates.io trusted publishing | https://crates.io/docs/trusted-publishing |
| Refused triggers and the 30-minute token | https://github.com/rust-lang/crates.io/blob/ed382369c65cb636470f09c031a0b68352092079/src/controllers/trustpub/tokens/exchange/mod.rs |
| Central Portal user tokens | https://central.sonatype.org/publish/generate-portal-token/ |
| Maven Central signing requirements | https://central.sonatype.org/publish/requirements/gpg/ |
| Sigstore on the Central Portal | https://central.sonatype.org/news/20250128_sigstore_signature_validation_via_portal/ |
| Environment secrets and required reviewers | https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments#environment-secrets |
| The environment in the OIDC `sub` claim | https://docs.github.com/en/actions/reference/security/oidc |
