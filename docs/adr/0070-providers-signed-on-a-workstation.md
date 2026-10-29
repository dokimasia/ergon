---
adr: 0070
title: A release of a Terraform provider remains a draft until a maintainer signs it on a workstation
status: Accepted
date: 2026-10-29
supersedes: RFC-0007 and ADR-0021, in part
superseded-by: none
rfc: RFC-0012
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0070: A release of a Terraform provider remains a draft until a maintainer signs it on a workstation

## Status

Accepted

## Context

The Terraform Registry publishes a provider from the GitHub Releases of its repository. It requires zip archives, a manifest, a file `SHA256SUMS` and a binary GPG signature of that file by a key of the type RSA or DSA. The managed binary release attaches tar.gz archives, `checksums.txt` and a cosign signature, and no GPG signature. Its job `publish` publishes each release after it uploads the assets.

HashiCorp's scaffolding signs in its release workflow with a private key from a secret. The publishing guide of the Registry also describes a release that a person signs before the draft is published.

## Decision

We will release a command with `registry: terraform` as a draft with the assets of the Registry, which a maintainer signs with a GPG key and publishes with `ergon release sign` on a workstation, because no job of a workflow may have a signing key.

- `ergon release pack` builds the zips, the manifest and `SHA256SUMS` of the provider, and cosign signs `SHA256SUMS` in the job `pack`.
- `ergon release publish` creates the tag, the draft and its assets, and leaves the draft unpublished.
- `ergon release sign` signs only a `SHA256SUMS` whose cosign bundle names the release workflow of the base branch at the commit of the tag, and whose lines match the digests of the assets of the draft.
- gpg must verify the signature before the upload and the publish. The key of the signature must be the key of `--key`, and its type must be RSA or DSA.

## Alternatives Considered

### The key in a secret of the workflow

The release workflow imports the private key from a secret, as HashiCorp's scaffolding does. It lost because every workflow that runs with the secret can read the key, and the secret keeps the key after each job.

### A local GoReleaser configuration

The repository writes the release of the Registry into its local GoReleaser configuration. It lost because the overlay keeps the managed archives and builds beside the local ones, and the pack attaches no manifest.

### A signature without the checks of the draft

`ergon release sign` signs the `SHA256SUMS` of the draft as it finds it. It lost because a token with write access to the contents of the repository could replace the assets of the draft before the signature.

## Consequences

**Positive:**

- A repository lists its provider under `go.binaries` and drops its own GoReleaser configuration and release workflow.
- The private key of a release never leaves the smart card of the maintainer.
- A maintainer signs only the checksums that the release workflow built at the tagged commit.

**Negative:**

- The Registry receives a release only after a maintainer runs `ergon release sign`.
- `ergon release sign` needs a token with write access to the contents of the repository on the workstation.
- cosign does not sign the SBOMs of a provider, because `SHA256SUMS` lists the zips and the manifest alone.
- The interface `Releaser` of the module `service` gains the method `Draft`.

## References

| What | Where |
|---|---|
| The publishing guide of the Terraform Registry | <https://developer.hashicorp.com/terraform/registry/providers/publishing> |
| The release workflow of HashiCorp's scaffolding | <https://github.com/hashicorp/terraform-provider-scaffolding-framework/blob/97f18e9cf02fa8ffee72d6d86df036233d1d4f2d/.github/workflows/release.yml> |
| The request | <https://github.com/dokimasia/ergon/issues/68> |
