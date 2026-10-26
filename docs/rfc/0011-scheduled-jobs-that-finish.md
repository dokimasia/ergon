---
rfc: 0011
title: Scheduled jobs that finish
author: Roy Klopper
status: Draft
created: 2026-10-26
updated: 2026-10-26
discussion: none
supersedes: RFC-0005 and ADR-0020, in part
superseded-by: none
produces-adr: tbd
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0011: Scheduled jobs that finish

## Summary

`baseline.yml`, `nightly.yml` and ergon's own `update-baseline.yml` fail on most runs, and a rerun does not fix them. This proposal makes three changes:

- `baseline.yml` and `update-baseline.yml` commit their proposals with a token of the GitHub App of ergon that may change workflows.
- The Go steps mutate and bench take a list of the packages that they leave out, such as the conformance suites, the fixtures and the examples.
- The nightly job of mutate keeps the verdicts of its earlier runs and tests only the mutants whose code or tests changed. This needs a mode of dokimi-mutate-go that reuses verdicts.

## Motivation

### The Baseline job cannot commit a workflow

`ergon init ci upgrade` commits the changes of a sync through `createCommitOnBranch` with `GITHUB_TOKEN`. That token has no `workflows` scope, so GitHub refuses every commit that adds or changes a file under `.github/workflows/`, with "Resource not accessible by integration". A push of the same change fails with "refusing to allow a GitHub App to create or update workflow".

The runs of 2026-10-26 show the rule:

- The job failed in all 7 public repositories whose sync wrote a managed workflow: dokimi-typescript, interface, lint-go, mutate-go, techne, kanon and sharder.
- It opened a pull request in the 4 repositories whose logs were read and whose sync did not write a workflow: assert-go, dokimi, dokimi-go and eidos.
- The sync of thesm-os/core did not write a workflow either, and its pull request failed with 403 "GitHub Actions is not permitted to create or approve pull requests". The 9 repositories of thesm-os have "Allow GitHub Actions to create and approve pull requests" off. That setting configures `GITHUB_TOKEN`.

Most releases of ergon change a managed workflow, because the weekly update of the baseline moves the pins of the actions. In every repository that has not upgraded by hand, the job fails in most weeks.

On 2026-10-26 a run of `update-baseline` in a scratch clone, with the fixes of that day, rewrote four of ergon's own workflows: `codeql.yml`, `release.yml`, `security.yml` and `version.yml`. `update-baseline.yml` commits that proposal with `GITHUB_TOKEN` as well, so it fails the same way.

### The nightly jobs test packages that only tests use

The testing standards exempt conformance suites, the fixtures of generators, the corpus of dokimi and the examples from mutation testing. `make mutate-go` and `make bench-go` still run over them, because both run over `./...` of each module. On 2026-10-26:

- The production packages of sharder scored 99% to 100%. Its five suites scored 13% to 53%, and its examples 85% to 94%.
- Most corpus packages of dokimi scored 0%, because no test of the corpus covers the mutants of its generated suites.
- Bench Go of kanon spent 368 seconds on `internal/fixture/mapkey` and 324 on `internal/fixture/mapvalue`. Bench Go of dokimi spent about 40 seconds on each corpus package.

ergon pins dokimi-mutate-go at 719083c, a build without a flag that leaves a package out. Release v0.1.0 has `--skip`.

### A full mutation run does not fit a job

The Go baseline gives Mutate Go 60 minutes. GitHub cancels a job at its limit, so the run does not report a score. The Nightly of eidos, mutate-go, core, kanon and sharder ended as cancelled on every run from 2026-10-22 to 2026-10-26.

One worker on the runner tests 0.1 to 0.3 mutants a second in the large packages:

| Repository | Where the run stood at the limit |
|---|---|
| thesm-os/core | Four packages done, and mutant 714 of 1,004 in `btree` |
| thesm-os/kanon | `internal/kanon` at mutant 287 of 5,800, with 16 h 24 min left |
| dokimasia/mutate-go | `cmd/dokimi-mutate-go` at mutant 145 of 516, with 2 h 21 min left |

GitHub ends a job on a hosted runner after 6 hours, so no limit lets kanon finish a full run.

## Detailed design

### Components

| Component | Where | Change |
|---|---|---|
| `baseline.yml` | template of `service/baseline/github` | Creates the token of the App with the permission `workflows`, and runs `ergon init ci upgrade` with it |
| `update-baseline.yml` | ergon's repository | Creates the same token for `update-baseline -propose` |
| `ergon init ci upgrade` | `internal/cli` | Names the workflows and the App when GitHub refuses a commit of a workflow |
| `Mutate`, `Bench` | `core/option` | The key `skip` |
| Makefile of Go | template of `ergon-lang-go` | `mutate-go` passes `--skip` in the flags of dokimi-mutate-go v0.1.0, and `bench-go` leaves out the packages of `skip` |
| Baseline of Go | `ergon-lang-go/baseline` | Pins dokimi-mutate-go v0.1.0 |
| Nightly job of mutate | `ergon-lang-go/baseline` | Restores and saves the history of dokimi-mutate-go in the cache of GitHub Actions |
| History of a run | dokimi-mutate-go and mutate-spec | `--history`, a request to those repositories |

### The token of the App

`baseline.yml` creates the token of the App before it upgrades, as `version.yml` does before the version pull request:

```yaml
    steps:
      - uses: actions/checkout@<commit> # <release>
        with:
          persist-credentials: false
      - name: Create a token of the GitHub App
        id: app
        if: vars.ERGON_APP_CLIENT_ID != ''
        uses: actions/create-github-app-token@<commit> # <release>
        with:
          client-id: ${{ vars.ERGON_APP_CLIENT_ID }}
          private-key: ${{ secrets.ERGON_APP_PRIVATE_KEY }}
          permission-contents: write
          permission-pull-requests: write
          permission-workflows: write
      - uses: ./.github/actions/setup-ergon
      - name: Upgrade ergon
        env:
          GITHUB_TOKEN: ${{ steps.app.outputs.token || github.token }}
        run: |
          ergon init ci upgrade
```

- The App needs the repository permission Workflows with write access, and each installation must grant it. The action refuses a permission that the installation lacks.
- The token of the App opens the pull request, so the setting "Allow GitHub Actions to create and approve pull requests" plays no part. GitHub starts the checks of the pull request without the approval of a maintainer.
- `update-baseline.yml` gets the same step, with the token in `GITHUB_TOKEN` of `update-baseline -propose`. dokimasia/ergon has the variable `ERGON_APP_CLIENT_ID` already.
- A repository without the App keeps `GITHUB_TOKEN`. Its upgrades that do not change a workflow open their pull requests as today.

When GitHub refuses the commit of a proposal that changes a file under `.github/workflows/`, `ergon init ci upgrade` returns an error that lists the files and the remedy:

```text
ergon: the upgrade changes .github/workflows/ci.yml and .github/workflows/nightly.yml, which a token
without the permission workflows cannot commit: set the variable ERGON_APP_CLIENT_ID and the secret
ERGON_APP_PRIVATE_KEY of a GitHub App with the permission workflows
```

### Packages that a step leaves out

`Mutate` and `Bench` of `core/option` gain the key `skip`:

```go
// Package option (core/option).

// Mutate is the options of the step mutate: the packages that it leaves out, the limit of the run,
// and how many mutants run at once.
type Mutate struct {
	// Timeout is the limit of the whole run, as a duration of at least 0s. 0s sets no limit.
	Timeout string `yaml:"timeout"`

	// Args are the arguments that follow the step's command, each on one line.
	Args []string `yaml:"args"`

	// Skip are the patterns of the packages whose mutants the step does not test, such as
	// ./internal/fixture/..., each on one line. The step resolves them in each module.
	Skip []string `yaml:"skip"`

	// Workers is the number of mutants that run at once, at least 1.
	Workers int `yaml:"workers"`
}

// Bench is the options of the step bench: the packages that it leaves out, which benchmarks run,
// for how long, and how often.
type Bench struct {
	// Match is a regular expression of the names of the benchmarks that run, on one line and
	// without a single quote, which the Makefile writes in single quotes.
	Match string `yaml:"match"`

	// Time is the time of each run of a benchmark: a positive duration, such as 1s, or a positive
	// number of iterations, such as 100x.
	Time string `yaml:"time"`

	// Args are the arguments that follow the step's command, each on one line.
	Args []string `yaml:"args"`

	// Skip are the patterns of the packages whose benchmarks the step does not run, each on one
	// line. The step resolves them in each module.
	Skip []string `yaml:"skip"`

	// Count is the number of runs of each benchmark, at least 1.
	Count int `yaml:"count"`
}
```

A repository lists its test-only packages in `.ergon.yaml`:

```yaml
go:
  mutate:
    skip:
      - ./internal/fixture/...
      - ./examples/...
  bench:
    skip:
      - ./internal/fixture/...
```

- The baseline of `skip` is empty in both steps. The test-only packages of the repositories follow no single name, such as `corpus/` in dokimi, `internal/fixture/` in kanon and `<package>test` in sharder.
- Each step has a key of its own. A fixture can have benchmarks that a repository keeps while it leaves out the mutants. The benchmarks of kanon's fixtures measure the codecs that kanon generates.
- The baseline pins dokimi-mutate-go v0.1.0. `mutate-go` passes `--workers`, `--timeout` and `--skip` in the flags of that release.
- `bench-go` runs `go test -bench` on the packages of `go list` of `go.paths` without the packages of `go list` of `go.bench.skip`, in each module.
- A pattern that matches no package of a module is no error, because one list serves every module of the repository.

### Verdicts that carry over between runs

dokimi-mutate-go gains a history of its verdicts. This proposal states the contract that ergon needs. The rule itself belongs to mutate-spec, which defines the key of a mutant and the record of a run:

```text
--history dir   Directory of the records of earlier runs. The command reuses the verdict of a
                mutant whose key, package and tests are unchanged since the run that tested it,
                tests every other mutant, and writes the records of this run to dir. The summary
                counts reused verdicts with the others. env DOKIMI_MUTATE_GO_HISTORY
```

The reuse rule follows PIT and StrykerJS:

- A killed mutant counts as killed while the files of its package and the tests that killed it are unchanged.
- A surviving mutant counts as a survivor while no test that covers its package changed.
- A change of `go.sum` or of a package that the mutated package imports tests the mutant again.

The nightly job of mutate uses the history:

1. It restores the newest cache of the history of its job, as the jobs restore the tools of ergon.
2. It runs `make mutate-go` with `DOKIMI_MUTATE_GO_HISTORY` set to that directory, and with `DOKIMI_MUTATE_GO_TIMEOUT` set to the limit of the job minus 10 minutes. The run then ends with a score before GitHub cancels the job.
3. It saves the history under a new key with `actions/cache/save` and `if: always()`, so a run that fails keeps the verdicts of the mutants that it tested.

kanon's `internal/kanon` alone needs about 16 hours, so at 50 minutes a night its history covers every mutant after about 20 nights. From then on, each night tests the mutants of the changes of the day. A run of `workflow_dispatch` with the input `history: false` tests every mutant.

A run that its timeout ends fails, also when it killed every mutant that it tested. A pass would report a score of part of the module.

### Failure handling

| Failure | Effect | Recovery |
|---|---|---|
| The repository lacks the variable or the secret of the App | `baseline.yml` uses `GITHUB_TOKEN`. An upgrade that changes a workflow fails with the error of the upgrade | Set the variable and the secret, or upgrade with `ergon init upgrade` and commit by hand |
| The App lacks the permission Workflows | The step of the token fails, and so does the job | Grant the permission and approve it in the installation |
| A pattern of `skip` matches no package | The step runs every package of the module | None needed |
| The cache of the history expired or was deleted | The run tests every mutant again, over as many nights as a first run | None needed |
| The run reaches `DOKIMI_MUTATE_GO_TIMEOUT` | The job fails with the score of the mutants that ran, and the history keeps their verdicts | The next night continues |

Invariants:

- A proposal of an upgrade contains every managed file of the sync, or the job fails.
- A reused verdict comes from a run of the same key, package and tests.
- The step mutate leaves out only the packages that `skip` names.

## Alternatives considered

### A. A personal access token with the scope workflow

The workflows pass a token of a person that may change workflows, from a secret.

**Why not:** the token belongs to a person and expires. ergon already creates the token of its App for the version pull request and for the proposals of spec.

### B. An issue in place of the pull request

The job opens an issue with the commands of the upgrade when the sync changes a workflow, as `baseline.yml` did before ergon proposed upgrades.

**Why not:** a person then runs the upgrade by hand in most weeks, which is the work that the job exists to remove.

### C. A proposal without the workflows

The job commits every changed file but the workflows.

**Why not:** the managed files of a release change together. `ergon init check` fails a tree with the files of one release and the workflows of another.

### D. A higher limit for Mutate Go

Each repository raises `go.nightly.mutate` until its run finishes.

**Why not:** GitHub ends a job on a hosted runner after 6 hours, and `internal/kanon` alone needs about 16.

### E. Only the lines of the changes since the last passing run

The job mutates the lines of `git diff` since the commit of its last passing run, through `--diff`.

**Why not:** a run that fails keeps its anchor, so the diff grows until it exceeds the limit and no run passes again. A change of a test alone also tests no mutant of the code that it covers.

### F. A matrix of jobs, one for each package

The nightly workflow runs a job for each package of a module.

**Why not:** one package exceeds the limit of a job alone, and the matrix runs the setup of Go and of ergon once for each package.

### G. More workers

`go.mutate.workers` rises to the number of processors.

**Why not:** the runner has 4 processors, which makes a run at most four times as fast. Tests that share a resource, such as a port, then fail each other, and each such failure counts as a detection.

### H. A convention for test-only packages

ergon leaves out every package whose path ends in `test`, and every package below `examples/`, without a key.

**Why not:** dokimi's `corpus/` and kanon's `internal/fixture/` follow neither name.

## Drawbacks

- Each repository that runs the Baseline job needs the App with the permission Workflows, its variable and its secret. A repository without them fails the job in most weeks, as today, with a clearer error.
- A key of the App with write access to workflows can change the CI of each repository of its installation. Today the key has write access to contents and pull requests.
- Each repository lists its test-only packages in two keys. A new suite costs time in the nightly mutation until someone adds it to the list.
- The flags of the Makefile change with dokimi-mutate-go v0.1.0, so a repository that sets `go.mutate.args` with the single-dash flags of 719083c fails until it moves to the GNU form.
- The history needs a design in mutate-spec and a release of dokimi-mutate-go before ergon builds its part. Until then the Nightly of kanon, eidos, core, mutate-go and sharder cannot pass.
- A large module fails its nightly job for weeks while its first history fills: about 20 nights for kanon.
- GitHub Actions keeps the history in its cache. The cache removes an entry that no job read in seven days, and a repository has 10 GB of cache.
- A verdict that a run reuses can be wrong when a change outside the files of the rule changes the behaviour, such as a file that a test reads. PIT and StrykerJS state the same limit.

## Unresolved and future work

- This proposal does not run the mutation of a pull request. The gate of a pull request has no step mutate.
- Billing, such as a spending limit that stops the jobs of an organization, is outside ergon.

## References

| What | Where |
|---|---|
| The scopes of `GITHUB_TOKEN`, which include no `workflows` | <https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax> |
| `GITHUB_TOKEN` cannot change workflow files | <https://github.com/orgs/community/discussions/35410> |
| The setting that lets `GITHUB_TOKEN` create pull requests | <https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository> |
| The limit of 6 hours of a job on a hosted runner | <https://docs.github.com/en/actions/reference/limits> |
| The scope of a token of a GitHub App, `permission-workflows` | <https://github.com/actions/create-github-app-token/blob/main/README.md> |
| The flags of dokimi-mutate-go v0.1.0, `--skip` among them | <https://github.com/dokimasia/mutate-go/blob/v0.1.0/cmd/dokimi-mutate-go/doc.go> |
| The incremental analysis of PIT | <https://pitest.org/quickstart/incremental_analysis/> |
| The incremental mode of StrykerJS | <https://stryker-mutator.io/docs/stryker-js/incremental/> |
