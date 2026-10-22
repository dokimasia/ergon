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
| [0003](0003-follow-changesets.md) | Follow changesets' file format, configuration and planning rules | Accepted, superseded in part by ADR-0009 and ADR-0053 |
| [0004](0004-go-releases-in-one-commit.md) | Go releases in one commit through a file proxy | Accepted |
| [0005](0005-signed-version-commits.md) | Version commits through createCommitOnBranch, split by size | Accepted, superseded in part by ADR-0055 |
| [0006](0006-publish-credentials.md) | OIDC for npm, PyPI and crates.io, and environment secrets for Maven Central | Accepted, superseded in part by ADR-0010 |
| [0007](0007-toolchains-and-languages.md) | Toolchains and languages are separate catalog entries | Accepted, superseded in part by ADR-0008 |
| [0008](0008-toolchains-in-language-modules.md) | The npm and Gradle toolchains are in the JavaScript and Java modules | Accepted |
| [0009](0009-release-dependents-by-resolution.md) | Dependent releases by consumer resolution | Accepted |
| [0010](0010-release-waits-for-ci.md) | The release workflow waits for the CI run of its commit | Superseded by ADR-0016 |
| [0011](0011-pins-through-the-baseline.md) | Repositories receive their pins through the baseline | Accepted |
| [0012](0012-seeded-files-with-license-headers.md) | ergon init writes the license header into the files it seeds | Accepted |
| [0013](0013-go-modules-through-the-version-list.md) | The resolver finds the module of a Go package through @v/list | Proposed |
| [0014](0014-lint-and-test-before-a-commit.md) | The hooks lint and test each commit, and run the gate before each push | Accepted, superseded in part by ADR-0015 |
| [0015](0015-hook-targets-in-ergon-yaml.md) | The targets of the hooks are options of .ergon.yaml | Accepted |
| [0016](0016-version-pull-request-gates-the-release.md) | The CI run of the version pull request is the gate of a release | Accepted, superseded in part by ADR-0018 |
| [0017](0017-generate-and-help-groups.md) | make generate runs the generators of every language, and make help groups the targets | Accepted |
| [0018](0018-ci-skips-tested-content.md) | ci.yml skips its jobs on content that already passed | Accepted |
| [0019](0019-no-lock-of-a-development-build.md) | A build of ergon without a release does not write a lock of a release | Accepted |
| [0020](0020-nightly-workflow.md) | nightly.yml runs the long steps of a language on a schedule | Accepted |
| [0021](0021-binary-releases.md) | Each release of a Go module attaches the signed binaries of its commands | Accepted, superseded in part by ADR-0023 |
| [0022](0022-upx-for-go-binaries.md) | The release of a Go module packs the Linux binaries of its commands with UPX | Accepted |
| [0023](0023-buildinfo-in-the-root-module.md) | ergon reads the version of its binary in a package of the root module | Accepted |
| [0024](0024-one-module-for-the-cli.md) | ergon has one module for its CLI and one for ergon-go-vet | Proposed |
| [0025](0025-internal-packages-by-command.md) | The packages of ergon are internal and grouped by command | Proposed |
| [0026](0026-options-implement-producer-roles.md) | The options of a producer return its contribution, data and files | Proposed |
| [0027](0027-template-function-for-gate-targets.md) | A template function writes the generate and check targets of each language | Proposed |
| [0028](0028-one-test-over-the-catalog.md) | One test over the catalog renders every producer | Proposed |
| [0029](0029-golangci-lint-module-plugins.md) | ergon tool run builds golangci-lint with the module plugins of go.lint.plugins | Accepted, superseded in part by ADR-0035 |
| [0030](0030-command-of-ergon.md) | The command that runs ergon is an option of the section common | Accepted |
| [0031](0031-restore-the-newest-tool-cache.md) | The tool cache of a job has its key without the digest as its restore key | Accepted |
| [0032](0032-prune-the-replaced-tool-caches.md) | nightly.yml deletes the tool caches that newer caches replaced | Accepted |
| [0033](0033-prune-the-tools-before-the-cache-saves.md) | A job removes the tools that the options do not name before its tool cache saves | Accepted |
| [0034](0034-version-files-in-the-tool-cache-key.md) | The key of a tool cache covers the files that set the version of the toolchain | Accepted |
| [0035](0035-lint-go-analyzers-as-module-plugins.md) | golangci-lint runs the analyzers of lint-go as module plugins in place of ergon-go-vet | Accepted |
| [0036](0036-codeql-of-go-fetches-own-modules-from-origin.md) | The CodeQL analysis of Go fetches the modules of its repository from their origin | Accepted |
| [0037](0037-sboms-keep-release-modules-from-proxy.md) | syft does not read a module of a release from the module proxy | Accepted |
| [0038](0038-binaries-build-outside-the-workspace.md) | The binaries of a release build outside the workspace against a proxy of the released modules | Accepted |
| [0039](0039-setup-steps-of-each-toolchain.md) | The key ci.steps of each toolchain adds steps to the setup of its jobs | Accepted |
| [0040](0040-releases-include-the-notice.md) | The archives and the packages of a release include the NOTICE of a repository under Apache-2.0 | Accepted |
| [0041](0041-package-manager-of-the-js-toolchain.md) | The key package-manager of the section js selects npm or pnpm | Accepted |
| [0042](0042-vite-plus-toolkit-of-the-js-toolchain.md) | The toolkit vite-plus runs the gates of JavaScript and TypeScript through vp | Accepted |
| [0043](0043-lint-of-markdown-and-workflows.md) | The targets lint-common and lint-github lint the Markdown files and the workflows | Accepted |
| [0044](0044-security-workflows-of-a-private-repository.md) | The key code-security renders the security workflows, and the Scorecard runs in a public repository alone | Accepted |
| [0045](0045-command-line-on-go-cli.md) | The command line of ergon runs on go.stealthscale.io/cli | Accepted |
| [0046](0046-merge-of-a-local-toml-file.md) | A local TOML file is a TOML document that merges into its managed file | Accepted |
| [0047](0047-cargo-llvm-cov-without-a-coverage-gate.md) | rust.tools pins cargo-llvm-cov, and the gate of Rust does not measure coverage | Accepted |
| [0048](0048-python-audit-skip.md) | The key audit.skip of the section python leaves packages without a release on PyPI out of the scan | Accepted |
| [0049](0049-hooks-exclude-of-the-common-files.md) | The key common.hooks.exclude keeps paths out of the hooks that change or refuse a file | Accepted |
| [0050](0050-busl-parameters-from-init-new-flags.md) | ergon init new sets the parameters of BUSL-1.1 from its flags | Accepted |
| [0051](0051-empty-busl-parameters.md) | A parameter of BUSL-1.1 may be empty | Accepted |
| [0052](0052-ergon-yaml-decode-on-go-cli.md) | ergon decodes the sections of .ergon.yaml with go.stealthscale.io/cli | Accepted |
| [0053](0053-one-line-of-updated-dependencies.md) | A changelog entry has one line of updated dependencies | Accepted |
| [0054](0054-empty-release-plans-in-ci.md) | CI treats an empty release plan as a plan without changesets | Accepted |
| [0055](0055-proposals-on-a-staging-branch.md) | A proposal is committed on a staging branch, and its branch moves once | Accepted |
| [0056](0056-cgo-binaries-on-linux-with-zig.md) | Commands with cgo build every platform on Linux with zig | Accepted |
| [0057](0057-own-go-modules-out-of-the-license-check.md) | The dependency review leaves the repository's own Go modules out of its license check | Accepted |
| [0058](0058-spec-roles-in-ergon-yaml.md) | The key spec.roles runs spec in the gate and the proposals of a repository | Accepted |
