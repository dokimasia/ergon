---
rfc: 0010
title: Conformance definitions with spec
author: Roy Klopper
status: Accepted
created: 2026-10-22
updated: 2026-10-22
discussion: none
supersedes: none
superseded-by: none
produces-adr: ADR-0058
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0010: Conformance definitions with spec

## Summary

ergon init gains the producer `spec`, which runs [spec](https://github.com/stealth-scale/spec) in a repository that defines a conformance definition or vendors copies of one. The key `spec.roles` of `.ergon.yaml` states what the repository is to spec, and a repository without a role gets nothing of spec. ergon pins spec as a release binary and adds its checks to the Makefile and to `ci.yml`. An implementation also gets `spec.yml`, which calls the reusable workflow of spec on each weekday to propose the updates of its copies.

## Motivation

spec vendors the conformance definition of a polyglot library into each implementation of the library. The repository of a definition declares the section `definition` of `.spec.yaml` and writes the manifest of its files with `spec manifest`. An implementation declares the section `implements`, and `spec sync` writes each copy with its `manifest.json`. The README of spec prescribes three parts of CI for an implementation:

- A job that installs spec with the action `setup-spec` and runs `spec check --remote`.
- A test run with the variable `SPEC_RESULTS`, followed by `spec verify` of the report of the runner.
- A workflow on a schedule that calls the reusable workflow `propose.yml` of spec with `secrets: inherit`. `propose.yml` opens the pull requests that update the copies.

A repository on ergon's baseline adds the job through the local file of `ci.yml` and writes the workflow by hand, because ergon renders none of these parts. Its pins of spec are then not part of the weekly update of the baseline. `setup-spec` builds spec from source with `go install` on each run, so the job needs Go and the time of a build.

GitHub passes secrets through `secrets: inherit` only to a reusable workflow of the same organization or enterprise. dokimasia and stealth-scale are two organizations on the free plan. A call of `propose.yml` from a repository of dokimasia does not pass `ERGON_APP_PRIVATE_KEY`, so the step that creates the token of the GitHub App fails. `propose.yml` does not declare the secret, and a caller cannot pass the key by name either.

Each release of spec attaches an archive of the program for Linux, macOS and Windows on amd64 and arm64, and `checksums.txt`. The tool runner of ergon installs such an archive as it installs actionlint.

`spec check` refuses a configuration file without the section `implements`. `spec manifest` refuses one without the section `definition`, so each check needs the role of the repository.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| Producer `spec` | `service/baseline/spec` | The section `spec`, the targets of spec in the Makefile, the job `spec` of `ci.yml`, and `spec.yml` |
| Release binary | `service/baseline/spec` | `Spec`, the archive of spec from the releases of stealth-scale/spec, pinned for Linux, macOS and Windows on amd64 and arm64 |
| Registration | `internal/cli` | The producer `spec` after the producer of the license files in `Producers` |
| Secret of `propose.yml` | stealth-scale/spec | The declaration of `ERGON_APP_PRIVATE_KEY` under `on.workflow_call.secrets` |

### The options

```yaml
spec:
  # What the repository is to spec: definition, for the repository of a conformance definition,
  # and implementation, for a repository with copies of definitions. A repository without a role
  # gets nothing of spec.
  roles: [implementation]
  tools:
    # The program spec, which checks the copies of the definitions and the manifest of a definition.
    spec:
      sha256:
        darwin/amd64: <sha256>
        darwin/arm64: <sha256>
        linux/amd64: <sha256>
        linux/arm64: <sha256>
        windows/amd64: <sha256>
        windows/arm64: <sha256>
      version: 0.2.0
  ci:
    actions:
      # The reusable workflow of spec that spec.yml calls.
      propose:
        uses: stealth-scale/spec/.github/workflows/propose.yml
        commit: 78d70fd5be63102e43f8dd9628d4594914bb390e
        release: v0.2.0
    timeout: 10
```

```go
// Package spec (service/baseline/spec).

// Name is the name of the producer of spec in the lock, and of its section of .ergon.yaml.
const Name = "spec"

// Role is what a repository is to spec. Each role is the name of the section of .spec.yaml that
// states it.
type Role string

// The roles of a repository.
const (
	// Definition is the repository of a conformance definition, whose .spec.yaml has the section
	// definition.
	Definition Role = "definition"

	// Implementation is a repository with copies of definitions, whose .spec.yaml has the section
	// implements.
	Implementation Role = "implementation"
)

// Valid reports whether r is Definition or Implementation.
func (r Role) Valid() bool

// Options are the options of the section spec of .ergon.yaml.
type Options struct {
	// Roles are what the repository is to spec, each once. A repository without a role gets no
	// target, job or workflow of spec.
	Roles []Role `yaml:"roles"`

	// Tools are the tools of spec.
	Tools Tools `yaml:"tools"`

	// CI are the pin of the reusable workflow of spec and the limit of the job spec.
	CI option.CI[Actions] `yaml:"ci"`
}

// Validate returns an error that wraps option.ErrInvalid for a role that is not valid and for a
// role that Roles repeats.
func (o *Options) Validate() error

// Tools are the tools of the section spec.
type Tools struct {
	// Spec is the program spec.
	Spec Spec `yaml:"spec"`
}

// Spec is the release binary of spec, stealth-scale/spec on GitHub.
type Spec struct {
	option.Binary `yaml:",inline"`
}

// Asset returns the archive of spec for p from the release of the version of s on GitHub. The
// archive is spec_<version>_<os>_<arch>.tar.gz on every platform, with the program spec at its
// root, or spec.exe on Windows. Asset returns an error that wraps option.ErrNoAsset for a platform
// that is not valid.
func (s Spec) Asset(p option.Platform) (option.Asset, error)

// Repository returns stealth-scale/spec.
func (Spec) Repository() string

// Actions are the pins of the actions of the section spec.
type Actions struct {
	// Propose is the reusable workflow propose.yml of spec, which spec.yml calls.
	Propose workflow.Action `yaml:"propose"`
}
```

The weekly update of the baseline resolves `tools.spec` from the releases of stealth-scale/spec, as it resolves actionlint. It resolves `ci.actions.propose` from the same releases, as it resolves the pin of checkout.

### The targets

The producer adds the group Spec to the Makefile. Each role adds its targets:

| Role | Target | Command | Aggregate |
|---|---|---|---|
| implementation | `check-spec` | `spec check` | `check` |
| definition | `generate-spec` | `spec manifest` | `generate` |
| definition | `verify-generate-spec` | `spec manifest --check` | `check` |

Each target runs spec through `ergon tool run spec.spec`. `spec check` does not read GitHub. It fails for a copy that differs from its manifest or comes from a checkout, and for a copied file that differs from its source. `spec manifest --check` fails for a manifest that differs from the files of the definition, and it does not write the manifest.

### The job spec of ci.yml

The producer contributes the job `spec` as a check of text on the Linux runner of the section `github`. The job installs ergon and keeps spec in the tool cache of the job. It grants `contents: read`, and its limit is `spec.ci.timeout`. Its steps follow the roles:

- An implementation runs `ergon tool run spec.spec -- check --remote` with the token of the job in `GITHUB_TOKEN`. The run also fails for a copy whose commit is not on the track of its entry, and it reports whether the head of the track has a newer copy.
- A definition runs `ergon tool run spec.spec -- manifest --check`.

### spec.yml

An implementation gets the managed workflow `spec.yml`:

```yaml
name: Spec

on:
  schedule:
    - cron: "41 6 * * 1-5"
  workflow_dispatch:

permissions: {}

concurrency:
  group: ${{ github.workflow }}
  cancel-in-progress: false

jobs:
  propose:
    name: Propose
    uses: stealth-scale/spec/.github/workflows/propose.yml@78d70fd5be63102e43f8dd9628d4594914bb390e # v0.2.0
    with:
      version: v0.2.0
    secrets:
      ERGON_APP_PRIVATE_KEY: ${{ secrets.ERGON_APP_PRIVATE_KEY }}
```

- The input `version` is the version of `tools.spec`, so a proposal runs the release of spec that the gate runs.
- `spec.yml` passes the private key of the GitHub App by name. That pass needs `propose.yml` to declare the secret `ERGON_APP_PRIVATE_KEY` under `on.workflow_call.secrets`. The reusable workflow reads `ERGON_APP_CLIENT_ID` from the variables of the calling repository.
- `propose.yml` runs `spec propose --repo` with the token of the App. The checks of a pull request from that token start without an approval.
- The schedule is fixed, as the schedules of `baseline.yml` and `security.yml` are.

### Failure handling

| Failure | Effect |
|---|---|
| A copy differs from its manifest, comes from a checkout, or has a copied file that differs from its source | `check-spec` and the job `spec` fail and name the copy |
| The commit of a copy is not on the track of its entry | The job `spec` fails. `check-spec` passes, because it does not read GitHub |
| The manifest of a definition differs from its files | `verify-generate-spec` and the job `spec` fail. `make generate` writes the manifest |
| A role without its section in `.spec.yaml` | spec exits with 2 and names the section, and the target or the job fails |
| GitHub refuses a request of `check --remote` | spec exits with 3, and the job `spec` fails |
| The repository lacks the variable `ERGON_APP_CLIENT_ID` or the secret `ERGON_APP_PRIVATE_KEY` | The run of `spec.yml` fails at the token of the App |

### Out of scope

- `spec verify`. No runner of an implementation writes a report to `SPEC_RESULTS`, so a verification would fail every run. A verification needs a report file per language and a step after the tests of each language. This proposal does not design them.
- `spec sync`, `spec try`, `spec status` and `spec propose --all`. A person runs them through `ergon tool run spec.spec --`.

## Alternatives considered

### A. Commands of spec that read the roles

`spec check` would also check the manifest of a definition, and `spec propose` would pass an implementation without the section `implements`. ergon would then render spec from one key of the section, without roles.

**Why not:** without an option, ergon cannot leave `spec.yml` out of the repository of a definition, because a rendering depends on the answers, the options and the contributions alone. Such a repository would run a proposal with nothing to do on each weekday. `make generate` could not write the manifest of a definition either. The alternative also needs two changes of spec.

### B. The action setup-spec in the job

The job installs spec with `setup-spec`, as the README of spec shows.

**Why not:** the action builds spec from source on each run, so the job needs Go and the time of a build. The tool runner installs the archive of a release, checks its SHA-256 and keeps it in the tool cache of the job.

### C. A job of ergon in nightly.yml

ergon runs `spec propose --repo` with the pinned binary in a job of `nightly.yml`, with the token of the App.

**Why not:** no job of `nightly.yml` creates the token of an App. `workflow.Job` would need a key for such a token. The job would also repeat `propose.yml`, the workflow that spec keeps for the proposals of an implementation.

### D. A catalog entry that ergon init add selects

`ergon init add spec` would add spec to the answers. Only a repository that uses spec would then have the section `spec`.

**Why not:** the catalog has languages and toolchains. spec is neither. The answers would need a second list beside the languages, and that list costs more than one section of pins in each `.ergon.yaml`.

### E. Proposals from the repository of the definition

The repository of a definition runs `spec propose --all` after each push to main.

**Why not:** the run needs a token of the App for the repository of every implementation, while the run of an implementation needs a token for its own repository alone. An implementation in another organization needs a run of its own anyway. spec models a proposal as a run of the implementation, and ergon follows that model.

### F. Local files in each repository

Each repository keeps the job in the local file of `ci.yml` and writes its workflow by hand.

**Why not:** every repository repeats the same job and workflow, and their pins of spec are not part of the weekly update of the baseline.

## Drawbacks

- Every `.ergon.yaml` gets the section `spec`, 24 lines in the form above, also in a repository without a role.
- The roles restate the sections of `.spec.yaml`. A role without its section fails the target and the job with the error of spec.
- spec has two pins: the binary and the commit of `propose.yml`. The weekly update moves both.
- `propose.yml` installs spec with `go install` and the newest stable Go on each run, outside the tool cache.
- `spec.yml` needs a change of `propose.yml` in spec before it can run.

## References

| What | Where |
|---|---|
| spec, its configuration and its CI | <https://github.com/stealth-scale/spec/blob/main/README.md> |
| The action setup-spec | <https://github.com/stealth-scale/spec/blob/main/.github/actions/setup-spec/action.yml> |
| The reusable workflow propose.yml | <https://github.com/stealth-scale/spec/blob/main/.github/workflows/propose.yml> |
| The section that `spec check` requires | <https://github.com/stealth-scale/spec/blob/main/implementation/config.go> |
| The section that `spec manifest` requires | <https://github.com/stealth-scale/spec/blob/main/definition/config.go> |
| The assets of spec v0.2.0 | <https://github.com/stealth-scale/spec/releases/tag/v0.2.0> |
| `secrets: inherit` within one organization or enterprise | <https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows> |
| A report of secrets that do not reach a workflow of another organization | <https://github.com/orgs/community/discussions/65766> |
| The secret of `propose.yml` | <https://github.com/stealth-scale/spec/issues/3> |
| The variables of the calling repository in a reusable workflow | <https://docs.github.com/en/actions/reference/workflows-and-actions/variables> |
| The scope of a token of a GitHub App | <https://github.com/actions/create-github-app-token/blob/main/README.md> |
