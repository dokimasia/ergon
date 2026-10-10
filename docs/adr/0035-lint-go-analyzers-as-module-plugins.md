---
adr: 0035
title: golangci-lint runs the analyzers of lint-go as module plugins in place of ergon-go-vet
status: Accepted
date: 2026-10-10
supersedes: RFC-0004 and ADR-0029, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0035: golangci-lint runs the analyzers of lint-go as module plugins in place of ergon-go-vet

## Status

Accepted

## Context

`lint-go` ran golangci-lint, its format check and ergon-go-vet in every module. ergon-go-vet was a command of `ergon-lang-go` with the analyzers `errorprefix` and `skipexpiry`, and `go.tools.ergon-go-vet` pinned it. `go.lint.exclude` named the package patterns that it skipped.

The weekly baseline update would move the pin of ergon-go-vet to a release of `ergon-lang-go` that does not have the command, because the analyzers moved to the module `go.dokimi.dev/lint`. The release v0.2.0 of that module has the package `go.dokimi.dev/lint/golangci`, which registers `errorprefix` and `skipexpiry` as two module plugins of golangci-lint.

`ergon tool run` builds the module plugins of `go.lint.plugins` into golangci-lint, and the baseline of that option is empty. The lock records the whole map as one option. The baseline update resolves the pin of each tool of the kind Go module, but not the entries of `go.lint.plugins`.

## Decision

We will build the analyzers of lint-go into golangci-lint as the module plugins of the option `go.lint.analyzers`, in place of the step of ergon-go-vet, because golangci-lint applies its exclusions and its `//nolint` directives to the findings of a module plugin as it does to those of its own linters.

- `go.lint.analyzers` is the `<package>@<version>` of the package that registers `errorprefix` and `skipexpiry`. Its baseline is `go.dokimi.dev/lint/golangci@v0.2.0`, and the baseline update moves it as it moves the pin of any Go module.
- The tag `plugins:"lint"` of `go.tools.golangci-lint` names the group `go.lint`, which implements `option.Linters`. Its method `Linters` returns `errorprefix` and `skipexpiry` of `go.lint.analyzers`, and each plugin of `go.lint.plugins`. A plugin of `go.lint.plugins` with the name of an analyzer replaces the analyzer.
- `ergon tool run` builds golangci-lint with each linter of the group. The managed `.golangci.yml` enables each of them, with the type `module` under `linters.settings.custom`.
- `lint-go` runs golangci-lint and its format check alone. The options `go.tools.ergon-go-vet` and `go.lint.exclude` are removed. A repository excludes a finding of an analyzer in its local `.golangci.yml`, as it excludes the finding of any other linter.
- Every command that reads `.ergon.yaml` leaves out an option that its producer no longer has when the lock records the option with the same value. In a repository that never changed `go.tools.ergon-go-vet` and `go.lint.exclude`, `ergon init sync` removes both. In a repository that changed one of them, the strict decode fails with its key until the repository deletes the key.

## Alternatives Considered

### The analyzers as entries of go.lint.plugins

The baseline of `go.lint.plugins` would name `errorprefix` and `skipexpiry`. It lost because the baseline update does not resolve the entries of `go.lint.plugins`. Their pin would not follow the releases of lint-go.

Because the lock records the whole map as one option, `ergon init sync` would also keep the analyzers at their old version in a repository that adds assertlint or another plugin to `go.lint.plugins`.

### dokimi-lint-go in a step beside golangci-lint

As it ran ergon-go-vet, `lint-go` would run the command `dokimi-lint-go` of lint-go after golangci-lint. It lost because a second tool needs exclusions of its own, as `go.lint.exclude` was for ergon-go-vet. golangci-lint applies one configuration of exclusions to every linter that it runs.

## Consequences

**Positive:**

- One run of golangci-lint reports the findings of the analyzers with the exclusions, the `//nolint` directives and the output of every other linter.
- The baseline update moves the pin of the analyzers to each new release of lint-go.

**Negative:**

- Every Go repository of the baseline builds golangci-lint with `golangci-lint custom`, also without plugins of its own. The first run after a change of the plugins, of golangci-lint or of the go command builds golangci-lint twice: once with `go install` and once with the plugins. The second build needs git, and access to github.com and to the module proxy.
- A golangci-lint that ergon did not build, such as the one of an editor, refuses the configuration of every Go repository of the baseline.
- A repository that set `go.lint.exclude` moves its patterns into the exclusions of its local `.golangci.yml`, and deletes the key.

**Neutral:**

- The analyzers of lint-go v0.2.0 report the same findings as those of ergon-go-vet. Only the URL of their documentation differs.

## References

| What | Where |
|---|---|
| The gate of Go and the options of its steps | RFC-0004, Each language's gate and Settings |
| The module plugins of go.lint.plugins | ADR-0029 |
| The package that registers the analyzers | https://github.com/dokimasia/lint-go/tree/v0.2.0/golangci |
| The pins that the baseline update resolves | `ergon-service/pin/pin.go`, `Find` |
