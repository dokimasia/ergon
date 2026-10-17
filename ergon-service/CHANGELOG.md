# go.dokimi.dev/ergon/service

## 0.13.0

### Minor Changes

- 64b7050: Add `baselinetest.Keys` to check that go.stealthscale.io/cli decodes every key of the section of a producer.
- 67ba789: **Breaking:** Keep the baseline value of an option that `.ergon.yaml` states without a value.
- 67ba789: **Breaking:** Refuse a key of `.ergon.yaml` in another case than the key of its option, such as `Lint:` for `lint`.
- 67ba789: **Breaking:** Return `options.ErrDefect` from `options.Fields` for an exported field tagged `yaml:"-"`.

### Patch Changes

- 67ba789: Decode the sections of `.ergon.yaml` with go.stealthscale.io/cli in place of viper.
- 67ba789: Fix a string option of `.ergon.yaml` to take an unquoted date or number as written, such as `change-date: 2030-01-01`.
- 67ba789: Fix each key of a map in `.ergon.yaml` to keep its case, such as `Dockerfile` in `license.styles`.

## 0.12.0

### Minor Changes

- 27a850e: Accept an empty parameter of BUSL-1.1, and write its line of LICENSE with the label alone.
- 27a850e: Add `baseline.Options.Settings` to resolve the options with the values that a command sets.
- 0a55b7d: Add `common.hooks.exclude` for the paths whose content the hooks of file hygiene keep as it is.
- 5394d3c: Add `github.code-security`, whose value false leaves `security.yml` and `codeql.yml` out of a private repository without GitHub Code Security.
- 0a819fb: Add `licenses.FixContent` to return the content that `ergon license fix` writes into one file.
- 27a850e: Add `licenses.Parameters.All` to iterate over the parameters of BUSL-1.1 by their keys.
- 27a850e: Add `options.Set` to write values into the content of `.ergon.yaml` as strings.
- 0a55b7d: **Breaking:** Read a local TOML file as a valid TOML document, and merge its tables into the managed TOML file.
- 0a819fb: Write the license header into each seeded file and into a new `.ergon.yaml`.

### Patch Changes

- 5394d3c: Grant `actions: read` to the CodeQL analysis, and run the Scorecard in a public repository alone.
- 0a55b7d: Skip the Markdown files in `trailing-whitespace` by their type in place of the pattern `\.md$`.
- Updated dependencies [a2520a3, 0a819fb, 8b78d7a]
  - go.dokimi.dev/ergon/core@0.11.0

## 0.11.0

### Minor Changes

- c82c878: Replace the action `common.ci.actions.markdownlint` with `common.tools.rumdl` in the target `lint-common` and the job `docs`.
- c82c878: Lint the workflows with `github.tools.actionlint` in the target `lint-github` and the job `workflows`.

### Patch Changes

- c82c878: Leave out a group of `.ergon.yaml` whose options the producer no longer has at the values of the lock.

## 0.10.0

### Minor Changes

- 9f8ee46: Add the template data .Toolchain and .Managed for the toolchain of a language and the managed files of a rendering.
- 9f8ee46: Start the npm packages of ergon tool run with pnpm dlx when the package manager is pnpm.

### Patch Changes

- 9f8ee46: Leave pnpm-lock.yaml without a license header.
- Updated dependencies [9f8ee46]
  - go.dokimi.dev/ergon/core@0.10.0

## 0.9.1

### Patch Changes

- Updated dependencies [ca0ca90]
  - go.dokimi.dev/ergon/core@0.9.0

## 0.9.0

### Minor Changes

- c17340d: Add `workflow.CodeQL.Steps`, which `codeql.yml` runs before the analysis of their language.

### Patch Changes

- Updated dependencies [c17340d]
  - go.dokimi.dev/ergon/core@0.8.0

## 0.8.0

### Minor Changes

- 170d883: Add `common.ergon`, the command that runs ergon in the targets of the Makefile and in the commit-msg hook.
- 876696f: Add `ergon tool prune` and `tool.Runner.Prune`, which remove each file of the tool directory that is not the install of a tool of the options.
- c6386e5: Add `option.Linters` for a group of options whose module plugins `ergon tool run` builds into golangci-lint.
- 74613ae: Add `baseline.Repository.Sections`, which returns the resolved options of each section.
- 74613ae: Leave out an option of `.ergon.yaml` that its producer no longer has when the lock records it with the same value.
- 74613ae: Add the job `prune-tools` to `nightly.yml` of each repository with a job that runs tools, which runs `ergon tool ci prune` each night.
- 74613ae: End each job that runs tools with `ergon tool prune` when the key of its tool cache missed.
- 74613ae: Restore the newest tool cache of a job of `ci.yml` and `nightly.yml` when the key of its cache misses.
- c6386e5: Key the tool cache of a job by the files of `workflow.Setup.VersionFiles`, which the jobs of Go set to `go.work`.

### Patch Changes

- Updated dependencies [c6386e5]
  - go.dokimi.dev/ergon/core@0.7.0

## 0.7.0

### Minor Changes

- ddcc0a1: Build golangci-lint with the module plugins of the option that its tag `plugins` names in `tool.Runner.Run`.

### Patch Changes

- Updated dependencies [ccbcc8f]
  - go.dokimi.dev/ergon/core@0.6.0

## 0.6.1

### Patch Changes

- b68662b: Install a Go tool of ergon tool run once for each version of the go command. A build of dokimi-mutate-go by an older Go refuses the packages of a newer one.

## 0.6.0

### Minor Changes

- b8f1c6a: Build, sign and attest the binaries of the commands of a Go module in each release of the module. The key binaries of the section go lists the commands. ergon init renders the configuration of GoReleaser of each module with a command. The job pack of release.yml builds the archives, the deb, rpm and apk packages, the casks, the checksums with their cosign signature and the SBOMs, and attests them. UPX packs each Linux binary with LZMA, and ergon tool run unpacks the .tar.xz of a release binary such as UPX. ergon release publish attaches the assets to a draft release before it publishes the release. ergon release ci homebrew then commits the casks to the tap of the key homebrew. The managed .gitignore ignores publish-plan.json and /dist/, so that Go writes the version of the release into each binary.
- d616a74: Refuse to write a new lock, or a lock that a release wrote, in a build of ergon without a release, whose version dev no release has. Such a build keeps a lock that such a build wrote.
- 326b07e: Render nightly.yml, which runs the long steps of each language on a schedule, each step in a job of its own. The key nightly of the section go maps fuzz, bench and mutate to the limit of each job in minutes, and the key nightly.schedule of the section github states when the workflow runs.

### Patch Changes

- 14e7a3a: Move golang.org/x/net to v0.60.0, which fixes GO-2026-6603, GO-2026-6610, GO-2026-6611, GO-2026-6612 and GO-2026-6617.
- babadd5: Require Go 1.27.2, whose standard library fixes the nine vulnerabilities that govulncheck reports for Go 1.27.1, such as GO-2026-6604 of os.Root on Windows.
- 2fdb4d1: Return an error on Windows for a file in place of the directory of the assets or the casks of a release, as on Linux and macOS. ergon release publish and ergon release ci homebrew read the directory.
- Updated dependencies [b8f1c6a, 326b07e, babadd5]
  - go.dokimi.dev/ergon/core@0.5.0

## 0.5.0

### Minor Changes

- 4044f32: Skip the jobs of a run of ci.yml when a passed run covers its content: a push whose content passed, and a version commit whose parent passed. The new first job skip runs ergon release ci skip, ergon release ci version marks the commit of the version pull request with the status ergon/version, and the job result is the one check that a branch requires.

## 0.4.0

### Minor Changes

- c61862c: Add make generate, which runs the generators of every language, and the options generate.command and generate.args to every language section of .ergon.yaml. generate-<language> runs the generators, and verify-generate-<language>, the step generate of check, fails when they change a file. make help lists the targets in groups: Common, each language, and the targets of the repository.
- 806100c: Make the targets of the hooks of .pre-commit-config.yaml options of .ergon.yaml: common.hooks.pre-commit and common.hooks.pre-push.
- 354fe1c: Open the version pull request in version.yml when the run of ci.yml for a push to main passes, and publish in release.yml after ergon release ci verify finds a run of ci.yml that passed on the content of the commit, such as the run of the version pull request. ergon release ci verify replaces ergon release ci wait, so no job waits for another workflow. ergon release ci version skips a commit that is no longer the head of main. A GitHub App in the variable ERGON_APP_CLIENT_ID and the secret ERGON_APP_PRIVATE_KEY opens the version pull request, so its checks run without an approval. Each push to main gets a concurrency group of its own in ci.yml.

### Patch Changes

- 12066ae: Drop the license header of a local file, such as .ergon/local/Makefile, when ergon init appends the file to its managed file.
- 7c5311d: Turn off the automatic maintenance of git in vcstest.Isolate, which a commit of git 2.48 and later starts in the background, so no process of git writes into the working tree of a test after the test ends.
- Updated dependencies [c61862c]
  - go.dokimi.dev/ergon/core@0.4.0

## 0.3.0

### Minor Changes

- 94c3bd1: Add ergon init upgrade and ergon init ci upgrade, which move a repository to the baseline of the newest release of ergon. The job of baseline.yml opens the pull request of the upgrade weekly, in place of an issue. ergon refuses a lock that a newer release of ergon wrote, and a local dependabot.yml that updates github-actions or pre-commit.
- 8bfa588: Make ergon release version print removed for each changeset that it removes. release.Version returns the removed paths apart from the written paths.
- 096abbd: Run make lint and make test in the pre-commit hook of .pre-commit-config.yaml, and make check in its pre-push hook. pre-commit install also installs the pre-push hook.
- 6f01723: Seed .changeset/config.json with the changelog format @changesets/cli/changelog, whose entries thank no author.
- 77ee1c0: Wait for the CI run of a commit before release.yml versions or publishes it. The job wait runs a new command for this, ergon release ci wait. The workflow release.yml no longer calls ci.yml, so each push to main runs the gate once.

### Patch Changes

- Updated dependencies [94c3bd1]
  - go.dokimi.dev/ergon/core@0.3.0

## 0.2.0

### Minor Changes

- [`afd76a1`](https://github.com/dokimasia/ergon/commit/afd76a177a970930daaf431bb6f6266b9c5028a3) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Give each directory that license.directories lists a license of its own: the header of its files, and its LICENSE, with a NOTICE for Apache-2.0.

- [`768f7d7`](https://github.com/dokimasia/ergon/commit/768f7d7cf70a5b4a0268a61d07dfb30d4e246cc7) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Keep the tools of ergon in the cache of GitHub Actions in each job of ci.yml that runs tools, and set up the toolchain of a job before ergon.

- [`6b1c2b5`](https://github.com/dokimasia/ergon/commit/6b1c2b5508b876007cff6ebab3dae22ac07d278f) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Run the job baseline of ci.yml on the Linux runner alone, and group the minor and patch updates of each package manager into one pull request of Dependabot.

### Patch Changes

- Updated dependencies [[`afd76a1`](https://github.com/dokimasia/ergon/commit/afd76a177a970930daaf431bb6f6266b9c5028a3), [`768f7d7`](https://github.com/dokimasia/ergon/commit/768f7d7cf70a5b4a0268a61d07dfb30d4e246cc7)]:
  - go.dokimi.dev/ergon/core@0.2.0

## 0.1.0

### Minor Changes

- [`0fa50eb`](https://github.com/dokimasia/ergon/commit/0fa50eb601a69e3602fcb2a442ee54a70a08ba21) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Release the first version.

### Patch Changes

- Updated dependencies [[`0fa50eb`](https://github.com/dokimasia/ergon/commit/0fa50eb601a69e3602fcb2a442ee54a70a08ba21)]:
  - go.dokimi.dev/ergon/core@0.1.0
