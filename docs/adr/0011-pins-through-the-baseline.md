---
adr: 0011
title: Repositories receive their pins through the baseline
status: Accepted
date: 2026-10-08
supersedes: none
superseded-by: none
rfc: RFC-0005
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0011: Repositories receive their pins through the baseline

## Status

Accepted

## Context

A repository of ergon init pins the releases of its tools, the actions of its workflows, GNU make and the hooks of pre-commit in `.ergon.yaml`. Each release of ergon embeds the baseline value of each pin. `ergon init sync` moves a pin that still has the earlier baseline value to the new one, and keeps a pin that the repository changed.

The repositories need current tools, and nothing updates the pins of the baseline. A person writes each newer release into the literals of the producers by hand, with the digests of each release binary and the commit of each action. Dependabot does not read `.ergon.yaml`. It can only edit the files that ergon renders from it, and `ergon init check` reports such an edit.

A tool and its managed configuration change together. `.golangci.yml` states the configuration format of golangci-lint. `biome.json` takes its schema from the version of biome. A repository cannot change a managed file to fit a newer tool.

## Decision

We will update pins only in ergon's baseline and deliver them through releases of ergon, because a tool and its managed configuration change together and ergon's gate tests them together.

A weekly command in ergon's repository resolves the newest stable release of each pin that is at least seven days old and of the same major version. It opens the change as a pull request. A repository moves to the newest baseline with `ergon init upgrade`. Its managed workflow `baseline.yml` opens that change as a pull request. ergon refuses a local file of `dependabot.yml` that updates a file that ergon renders.

## Alternatives Considered

### An update in each repository

`ergon init update` would write the newest release of each pin into `.ergon.yaml` of each repository. It lost because every value that it writes differs from the record of the lock. `sync` never moves such a pin again, so every repository would update each pin on its own. A newer tool that needs a change of its managed configuration would fail the gate of the repository, which cannot change the managed file.

### Dependabot for the rendered files

A local file of `dependabot.yml` would add the ecosystems `github-actions` and `pre-commit`, so Dependabot updates the action pins and the hooks where ergon renders them. It lost because each such pull request edits a managed file. `ergon init check` fails the gate, and `ergon init sync --force` writes the old version back. Dependabot has no ecosystem for the tools of `.ergon.yaml`.

### Renovate in ergon's repository

The regex managers of Renovate would update the version strings in the literals of the producers. It lost because a release binary also needs the digest of the asset of each platform, which a regex manager does not compute.

## Consequences

**Positive:**

- One review in ergon covers each new release of a tool, its digests and its managed configuration, for every repository.
- A repository receives each baseline as one pull request, which its own gate tests.
- A pin that a repository changed remains its choice across upgrades, and the upgrade lists it.

**Negative:**

- A newer tool arrives in a repository only with a release of ergon, at least seven days after the tool's release. For a fix that cannot wait, a person overrides the pin in `.ergon.yaml` of the repository.
- ergon's repository runs a weekly workflow that reads seven registries and the releases of GitHub. A person reviews each of its pull requests.
- The team updates the resolvers when a registry changes its API.

**Neutral:**

- The dependency manifests and their lockfiles remain the work of Dependabot.

## References

| What | Where |
|---|---|
| Dependabot updates `.pre-commit-config.yaml` | https://github.blog/changelog/2026-03-10-dependabot-now-supports-pre-commit-hooks/ |
| Renovate updates the template of copier and runs copier | https://docs.renovatebot.com/modules/manager/copier/ |
