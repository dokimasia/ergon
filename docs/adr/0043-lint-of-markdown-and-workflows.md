---
adr: 0043
title: The targets lint-common and lint-github lint the Markdown files and the workflows
status: Accepted
date: 2026-10-11
supersedes: RFC-0004, in part
superseded-by: none
rfc: RFC-0004
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0043: The targets lint-common and lint-github lint the Markdown files and the workflows

## Status

Accepted

## Context

`make lint` runs the linters of each language, and the hooks of `.pre-commit-config.yaml` run it before each commit. It did not check the Markdown files or the workflows:

- The job `docs` of `ci.yml` linted the Markdown files with markdownlint-cli2-action and the rules of `.markdownlint.yml`, so a defect showed only after a push.
- GitHub found a defect in the expressions, the jobs or the runner labels of a workflow only when it ran the workflow.

Every tool of `make lint` runs through `ergon tool run`, which installs the version that `.ergon.yaml` pins. A tool of no language runs as a release binary with a digest per platform, as commitlint does, so a repository of Go does not need another runtime.

We measured seven tools on 2026-10-11 on ergon, and the tools of Markdown and of spelling also on the ledger and the kernel:

| Tool | Published as | Findings | Defects |
|---|---|---|---|
| markdownlint-cli2 0.23.3 | npm package | 0 in the 95, 113 and 90 Markdown files of ergon, the ledger and the kernel | 0 |
| rumdl 0.2.79 | Release binaries | 2 in ergon, 16 in the ledger, 1 in the kernel | 0 |
| actionlint 1.7.12 | Release binaries | 19 in the 8 workflows of ergon, each the runner label `ubuntu-26.04` | 0 |
| yamllint 1.38.0 | PyPI package | 1,805 in 109 files of ergon | 0 |
| zizmor 1.30.1 | Release binaries | 17 in the workflows of ergon | 0 |
| typos 1.51.1 | Release binaries | 280 in ergon, the ledger and the kernel | 0 |
| editorconfig-checker 4.0.2 | Release binaries | 1,343 in ergon | 0 |

rumdl reads the rules of `.markdownlint.yml`. Its two findings in ergon were a blank line of a list item in a changelog, which markdownlint accepts, and a relative link of a golden file of a test.

## Decision

We will lint the Markdown files with rumdl in the target `lint-common` and the workflows with actionlint in the target `lint-github`, because both run as release binaries that ergon pins by digest, and both pass the files of ergon with the configuration that ergon init renders.

- `common.tools.rumdl` pins rumdl. `lint-common` runs `rumdl check --no-cache .` with the rules of `.markdownlint.yml`. rumdl skips the files that `.gitignore` ignores, and the managed `.markdownlintignore` adds `**/testdata/**`.
- `.markdownlint.yml` sets `list_item_empty_lines` of MD009. The release flow writes a blank line of a list item in a changelog with the indentation of the item.
- The job `docs` runs `make lint-common` through ergon in place of markdownlint-cli2-action, and `.ergon.yaml` no longer has the key `common.ci.actions.markdownlint`.
- `github.tools.actionlint` pins actionlint. `lint-github` runs it with `-shellcheck=` and `-pyflakes=`, because ergon pins neither program.
- The managed `.github/actionlint.yaml` lists the runners of the section `github` as labels, because actionlint 1.7.12 does not know `ubuntu-26.04`.
- The job `workflows` of the section `github` runs `make lint-github`.
- `lint` and `check` depend on both targets.
- ergon leaves out a group of `.ergon.yaml` whose options the producer no longer has and whose values equal the record of the lock, so an upgrade of ergon removes `common.ci.actions.markdownlint`.

## Alternatives Considered

### markdownlint-cli2 in lint-common

markdownlint-cli2 is the reference implementation of the rules, and it found nothing in three repositories. It lost because npm alone publishes it, so the pre-commit hook would need Node.js in every repository, a repository of Go included.

### yamllint

Under its default rules yamllint reported these findings in ergon:

- 1,413 lines over 80 characters.
- 232 comments with one space before `#`, the form of the pin of an action.
- 109 files without `---`.
- 51 keys `on`.

ergon renders each of those forms on purpose. With the four rules turned off, yamllint found nothing. It lost because it adds Python through uv to every repository and did not find a defect. actionlint parses the workflows, and the hooks parse every YAML file with check-yaml.

### zizmor

zizmor failed the workflows of ergon offline:

- It reported a high finding on the trigger `workflow_run` of `version.yml`, which opens the version pull request from a passed run of `ci.yml`.
- It reported 16 low findings that ask for a self-repository syntax `$/` in place of `./`.

It lost because every repository would fail it on managed files that the repository cannot edit. Its online audits also need the API of GitHub.

### typos

typos reported 280 findings in ergon, the ledger and the kernel, and none was a typo. The findings came from the French text of CeCILL, hashes, the paths of SVG files, country codes and the title of RFC 9562. It lost on that measurement.

### editorconfig-checker

editorconfig-checker reported 1,343 findings in ergon and no defect. The findings came from tabs in the Go blocks of Markdown files, spaces in the raw strings of Go, the padding of license texts and the tabs of the Makefile templates. It lost on that measurement.

## Consequences

**Positive:**

- The hooks lint the Markdown files and the workflows before each commit, and CI runs the same targets with the same pins.
- `.ergon.yaml` pins one action fewer.

**Negative:**

- rumdl has rules that markdownlint does not have. They reported 16 findings in an RFC of the ledger and 1 in an RFC of the kernel.
- rumdl released 0.2.79 on 2026-10-09 and releases most weeks. The baseline update moves its pin often, and a new release can add findings.
- actionlint does not check the scripts of the steps with shellcheck.
- `ci.yml` has one job more, and the job `docs` installs ergon and its tool cache.
- actionlint accepts every runner of the section `github` as a label, also a runner that GitHub no longer offers.

**Neutral:**

- The rules of `.markdownlint.yml` apply as before, with one option of MD009 more.

## References

| What | Where |
|---|---|
| rumdl | <https://github.com/rvben/rumdl> at v0.2.79 |
| The configuration of actionlint | <https://github.com/rhysd/actionlint/blob/main/docs/config.md> at v1.7.12 |
| The rule MD009 of markdownlint | <https://github.com/DavidAnson/markdownlint/blob/main/doc/md009.md> |
| The audits of zizmor | <https://docs.zizmor.sh/audits/> |
| The managed files and the jobs of `ci.yml` | RFC-0004 |
