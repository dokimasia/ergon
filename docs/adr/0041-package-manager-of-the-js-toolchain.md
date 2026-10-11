---
adr: 0041
title: The key package-manager of the section js selects npm or pnpm
status: Accepted
date: 2026-10-11
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0041: The key package-manager of the section js selects npm or pnpm

## Status

Accepted

## Context

The js toolchain runs npm in every target and job:

- The setup of the jobs `check-javascript` and `check-typescript` runs `npm ci`.
- `audit-js` runs `npm audit`, and the targets of the tests run `npm test`.
- `ergon tool run` starts an npm package, such as Biome or `tsc`, with npx.

`npm ci` and `npm audit` read `package-lock.json`, which a repository of pnpm does not have. The request in dokimasia/ergon#21 measured npm 11.20.0 in a directory whose `package.json` pins pnpm in `devEngines.packageManager`. npm refused each of its commands there with the error `EBADDEVENGINES`.

The repository dokimasia/interface is a workspace of pnpm, because the packages of stealthscale that it builds on publish catalogs of pnpm. Without a language its gate checked none of its TypeScript. `ergon license fix` also wrote a header into its `pnpm-lock.yaml`, and pnpm removes that header at its next install.

## Decision

We will add the key `package-manager` to the section `js`, with the values `npm` and `pnpm`, because a repository whose packages pnpm installs cannot run its gate through npm.

- For pnpm, the setup of the jobs runs the action `pnpm/setup` of the key `js.ci.actions.setup-pnpm` with `require-lockfile: true`. The action installs pnpm, the Node.js of `devEngines.runtime` or of the matrix, and the packages of `pnpm-lock.yaml`. A matrix of versions passes its version as the input `runtime`.
- For pnpm, the targets of the tests run `pnpm test`, and `audit-js` runs `pnpm audit`.
- `ergon tool run` starts an npm package with `pnpm dlx` for pnpm, and with npx for npm. The command reads the package manager through the interface `option.PackageManaged`, which the options of the section `js` implement.
- `ergon license` does not write a header into `pnpm-lock.yaml`, for either package manager.
- Dependabot keeps the ecosystem `npm`, which its documentation names for pnpm.

## Alternatives Considered

### The section js by its name in ergon tool run

`ergon tool run` would read the key `package-manager` from the section with the name `js`. It lost because the command does not import a language module, and it would then depend on the name of one toolchain. The interface leaves the key with the section that declares it.

### No header in pnpm-lock.yaml for pnpm alone

The producer of the licenses would skip `pnpm-lock.yaml` only when the section `js` sets pnpm. It lost because pnpm removes the header at its next install of `pnpm-lock.yaml`. A repository whose section sets npm can have that file too.

## Consequences

**Positive:**

- A workspace of pnpm takes the languages JavaScript and TypeScript, and its gate checks its sources.
- One step of the setup installs pnpm, Node.js and the packages, and fails without `pnpm-lock.yaml`.

**Negative:**

- `pnpm/setup` v3 installs pnpm 11 or newer. A repository on pnpm 10 cannot select pnpm.
- Without `devEngines.runtime` or a version file of Node.js, `pnpm/setup` does not install Node.js, and the jobs run the Node.js of the runner image.
- The documentation of Dependabot lists pnpm v7 to v10. No run has shown yet that Dependabot updates a lockfile that pnpm 12 writes.
- The section `js` and the lock have two keys more: `package-manager` and the pin `setup-pnpm`.

**Neutral:**

- A repository of npm renders the same targets and jobs as with the earlier release of ergon.

## References

| What | Where |
|---|---|
| The request | dokimasia/ergon#21 |
| The inputs of pnpm/setup | <https://github.com/pnpm/setup> at v3.1.1 |
| The package managers that Dependabot supports | <https://docs.github.com/en/code-security/dependabot/ecosystems-supported-by-dependabot/supported-ecosystems-and-repositories> |
| The options and the jobs of each language | RFC-0004 |
