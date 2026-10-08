---
rfc: 0006
title: Configurable hooks
author: Roy Klopper
status: Accepted
created: 2026-10-08
updated: 2026-10-08
discussion: none
supersedes: RFC-0004, in part
superseded-by: none
produces-adr: ADR-0015
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0006: Configurable hooks

## Summary

The hooks of `.pre-commit-config.yaml` run targets of the Makefile before each commit and before each push. ADR-0014 runs `make lint` and `make test` before a commit and `make check` before a push, in every repository.

With this RFC, `.ergon.yaml` sets the targets of each stage, and an empty list turns the hooks of a stage off. A repository then chooses what runs before a commit and before a push. The defaults remain the targets of ADR-0014.

## Motivation

Each target of the Makefile runs every language of the repository, and every package of each language. The targets of Go run in every module of `go.work`, so the time of a hook grows with the size of the repository. On ergon's repository, with its 14 Go modules, four CPU cores and warm caches, `make lint test` took 15.6 seconds on 2026-10-08.

The hooks are the same in every repository. A team whose tests take minutes cannot move them from the commit to the push, and a team cannot add its full gate before each commit either. The local file `.ergon/local/.pre-commit-config.yaml` cannot help, because ergon appends the lists of a local YAML file to the rendering. A local file adds a hook, and cannot remove or move a hook of the rendering.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| Hook options | `service/baseline/common` | The key `hooks` of the section `common`: the targets of each stage |
| Hook configuration | the template of `.pre-commit-config.yaml` | One hook for each target of each stage |

### The options

The section `common` gains the key `hooks`:

```yaml
common:
  # The targets of the Makefile that the hooks of .pre-commit-config.yaml run, at each stage of git.
  # An empty list turns the hooks of its stage off.
  hooks:
    # The targets that run before each commit, in order: fmt, lint, test, audit or check.
    pre-commit: [lint, test]
    # The targets that run before each push, in order: fmt, lint, test, audit or check.
    pre-push: [check]
```

- A target is an aggregate target of the Makefile: `fmt`, `lint`, `test`, `audit` or `check`. Each language adds its own targets to them.
- ergon refuses another value, and a target that a stage lists twice. A target may appear in both stages.
- An empty list turns the hooks of its stage off.
- The defaults are the targets of ADR-0014. An upgrade moves a list that still has its baseline value to the baseline of the new release, and keeps a list that the repository changed, as for every option.

```go
// Package common (service/baseline/common).

// Target is an aggregate target of the Makefile that a hook of .pre-commit-config.yaml runs.
type Target string

// Validate returns an error that wraps option.ErrInvalid for a t that is none of the aggregate
// targets.
func (t Target) Validate() error

// Targets are the targets that the hooks of one stage run, in order.
type Targets []Target

// Validate returns an error that wraps option.ErrInvalid for a target that is not valid, and for a
// target that t names twice. An empty Targets is valid, and turns the hooks of its stage off.
func (t Targets) Validate() error

// Hooks are the targets that the hooks of .pre-commit-config.yaml run at each stage of git.
type Hooks struct {
	PreCommit Targets `yaml:"pre-commit"`
	PrePush   Targets `yaml:"pre-push"`
}
```

### The hook configuration

`.pre-commit-config.yaml` has one hook for each target of each stage, in the order of its list. The hook of a target has the id and the name of the target, so `SKIP=test` skips the tests at every stage:

```yaml
  - repo: local
    hooks:
      # Each commit runs the targets of the Makefile that common.hooks.pre-commit of .ergon.yaml
      # lists.
      - id: lint
        name: make lint
        entry: make lint
        language: system
        pass_filenames: false

      # Each push runs the targets of the Makefile that common.hooks.pre-push of .ergon.yaml lists.
      - id: check
        name: make check
        entry: make check
        language: system
        pass_filenames: false
        stages: [pre-push]
```

- The hooks of pre-commit-hooks and the hook `commitlint` keep the stages of ADR-0014.
- `default_install_hook_types` keeps `pre-commit`, `commit-msg` and `pre-push`, also when a stage has no target. The hook of such a stage then runs nothing, and a repository that lists a target later does not need to run `pre-commit install` again.
- pre-commit 4.6.2 accepts one id at two stages, as `pre-commit validate-config` showed on 2026-10-08.

### Failure handling

| Failure | State afterwards | Recovery |
|---|---|---|
| `hooks` lists a value that is no aggregate target, or a target twice | `ergon init check` and every command that writes fail, and name the key | Correct the list in `.ergon.yaml` |
| A target fails | The hook exits with the status of make, and git stops the commit or the push | Fix the cause, or skip the hook with `SKIP=<target>` |

Invariants:

- The hooks run the targets that `.ergon.yaml` lists, in the order of each list, and no other target of the Makefile.
- CI runs `make check-<language>` for every language, whatever the hooks run.

## Alternatives considered

### A. The fixed targets of ADR-0014

Every repository runs `make lint` and `make test` before each commit and `make check` before each push.

**Why not:** a repository cannot adapt the hooks to the time of its gate.

### B. `SKIP` and `--no-verify`

A person skips a hook with the variable `SKIP` of pre-commit, or skips every hook with `git commit --no-verify`.

**Why not:** each acts on one command of one person. The repository does not record it. `--no-verify` also skips the hygiene hooks and the check of the commit message.

### C. Hooks in the local file

A repository adds its hooks in `.ergon/local/.pre-commit-config.yaml`.

**Why not:** a local file can add a hook and cannot remove or move a hook that ergon renders, because ergon appends the lists of a local YAML file to the rendering.

### D. Hooks for the languages and the packages of a change

Each hook runs its target only for the languages and the packages that the commit or the push affects.

**Why not:** each language would have to declare the files of its sources, and each fragment of the Makefile would have to take a list of packages. The selection would also miss a dependency between two languages, such as a TypeScript client that a Go program generates.

## Drawbacks

- A repository that moves its tests from the commit to the push finds a failing test only when it pushes, after the commits that it already made.
- `common` gains one more key to validate and to document.

## Unresolved and future work

- This RFC does not change `ci.yml`, which runs the gate of every language on every push.

## References

| What | Where |
|---|---|
| The stages of a hook of pre-commit | https://pre-commit.com/#confining-hooks-to-run-at-certain-stages |
| The hook types that `pre-commit install` installs | https://pre-commit.com/#top_level-default_install_hook_types |
| The environment variable `SKIP` of pre-commit | https://pre-commit.com/#temporarily-disabling-hooks |
