<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# Decision records

Each ADR records one decision and what it cost. Once an ADR is accepted, do not change its argument. A later decision supersedes it, and both records link to each other.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-one-module-per-language.md) | One module per language | Accepted, superseded in part by ADR-0008 |
| [0002](0002-independent-module-versions.md) | Independent module versions, with the root module as an entry point | Accepted, superseded in part by ADR-0009 |
| [0003](0003-follow-changesets.md) | Follow changesets' file format, configuration and planning rules | Accepted, superseded in part by ADR-0009 |
| [0004](0004-go-releases-in-one-commit.md) | Go releases in one commit through a file proxy | Accepted |
| [0005](0005-signed-version-commits.md) | Version commits through createCommitOnBranch, split by size | Accepted |
| [0006](0006-publish-credentials.md) | OIDC for npm, PyPI and crates.io, and environment secrets for Maven Central | Accepted, superseded in part by ADR-0010 |
| [0007](0007-toolchains-and-languages.md) | Toolchains and languages are separate catalog entries | Accepted, superseded in part by ADR-0008 |
| [0008](0008-toolchains-in-language-modules.md) | The npm and Gradle toolchains are in the JavaScript and Java modules | Accepted |
| [0009](0009-release-dependents-by-resolution.md) | Dependent releases by consumer resolution | Accepted |
| [0010](0010-release-waits-for-ci.md) | The release workflow waits for the CI run of its commit | Superseded by ADR-0016 |
| [0011](0011-pins-through-the-baseline.md) | Repositories receive their pins through the baseline | Accepted |
| [0012](0012-seeded-files-with-license-headers.md) | ergon init writes the license header into the files it seeds | Proposed |
| [0013](0013-go-modules-through-the-version-list.md) | The resolver finds the module of a Go package through @v/list | Proposed |
| [0014](0014-lint-and-test-before-a-commit.md) | The hooks lint and test each commit, and run the gate before each push | Accepted, superseded in part by ADR-0015 |
| [0015](0015-hook-targets-in-ergon-yaml.md) | The targets of the hooks are options of .ergon.yaml | Accepted |
| [0016](0016-version-pull-request-gates-the-release.md) | The CI run of the version pull request is the gate of a release | Accepted, superseded in part by ADR-0018 |
| [0017](0017-generate-and-help-groups.md) | make generate runs the generators of every language, and make help groups the targets | Accepted |
| [0018](0018-ci-skips-tested-content.md) | ci.yml skips its jobs on content that already passed | Accepted |
| [0019](0019-no-lock-of-a-development-build.md) | A build of ergon without a release does not write a lock of a release | Accepted |
| [0020](0020-nightly-workflow.md) | nightly.yml runs the long steps of a language on a schedule | Accepted |
