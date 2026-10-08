---
adr: 0020
title: nightly.yml runs the long steps of a language on a schedule
status: Accepted
date: 2026-10-09
supersedes: none
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0020: nightly.yml runs the long steps of a language on a schedule

## Status

Accepted

## Context

The gate of a language, `check-<language>`, runs the steps that the key `check` of its section lists, and `ci.yml` runs it on every push and pull request. Some steps take too long for that:

- `fuzz-go` fuzzes each fuzz target for the time that `go.fuzz.time` states, and takes up to two hours in a large repository.
- `bench-go` runs each benchmark six times, and takes up to 45 minutes.
- `mutate-go` runs the tests against every mutant of each module, and takes up to an hour.

assert-go ran its fuzz targets in a job of its local `ci.yml`, which added five minutes to every push. ergon init manages no scheduled workflow, so a repository that runs such a step on a schedule writes the workflow itself.

## Decision

We will render a managed workflow, `nightly.yml`, that runs the steps of each language that its section lists under the key `nightly`:

- The workflow runs on a schedule and on `workflow_dispatch`. The key `nightly.schedule` of the section `github` states the schedule as a cron expression, `0 3 * * *` by default.
- A producer contributes the jobs of the workflow in `workflow.Contribution.Nightly`, as it contributes the jobs of `ci.yml` in `Jobs`. Each job runs on the Linux runner of the section `github`, unless its setup lists runners.
- The key `nightly` of a language section maps each of its steps to the limit of the step's job in minutes, as `nightly: {fuzz: 120, bench: 45, mutate: 60}`. Each step of each language is a job of its own, `<step>-<language>`, which runs `make <step>-<language>`. The jobs run in parallel, and one that fails leaves the others running.
- Go has the key, with fuzz for 120 minutes, bench for 45 and mutate for 60. A language gets the key when its producer has a step that is too long for each push: fuzz, bench or mutate. Today only Go has such a step.
- ergon init renders no `nightly.yml` when no producer contributes a job.
- A failed run fails as any run does, and GitHub notifies the people who watch the repository.

## Alternatives Considered

### A local workflow in each repository

Each repository would write its own scheduled workflow in its local files. It lost because every repository repeats the same jobs, and the pins of their actions miss the updates of the baseline.

### The long steps in ci.yml under a condition

`ci.yml` would gain the trigger `schedule`, and the long steps would run only on that event. It lost because the job `result` and the skip of ADR-0018 work on the content of a commit, and a scheduled run tests no new content.

### An issue for a failed run

A failed run would open or update an issue. It lost because the run already fails visibly, and an issue needs the permission `issues: write` and a rule for when to close it.

## Consequences

**Positive:**

- The gate of each push runs the fast steps alone, and the long steps still run every day.
- A repository changes the steps or the schedule in `.ergon.yaml`, and the baseline keeps the pins of the workflow.

**Negative:**

- A long step reports a defect up to a day after the push that caused it.
- The nightly jobs of Go use up to almost four hours of runners each day at the default limits.

**Neutral:**

- ergon init does not render `nightly.yml` for a repository without a step under `nightly`, so its workflows do not change.
