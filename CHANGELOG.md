# go.dokimi.dev/ergon

## 0.14.3

### Patch Changes

- Updated dependencies [7b12cbc]
  - go.dokimi.dev/ergon/lang/typescript@0.4.1
  - go.dokimi.dev/ergon/lang/javascript@0.7.1

## 0.14.2

### Patch Changes

- 3af4e3d: Update go.stealthscale.io/cli to v0.7.0 to decode the command lines of `generate.commands`.
- Updated dependencies [2e44067, 8281533, 477d633, 52edc26, 0b95e6a, cc6847b, e59b9e1, 1cd83e5, 27980dd, 993f78e, 0332c9e, f928381, 3af4e3d, 43a3f4c]
  - go.dokimi.dev/ergon/core@0.16.0
  - go.dokimi.dev/ergon/lang/php@0.5.0
  - go.dokimi.dev/ergon/lang/python@0.7.0
  - go.dokimi.dev/ergon/lang/rust@0.7.0
  - go.dokimi.dev/ergon/lang/terraform@0.5.0
  - go.dokimi.dev/ergon/lang/typescript@0.4.0
  - go.dokimi.dev/ergon/lang/bash@0.5.0
  - go.dokimi.dev/ergon/lang/csharp@0.4.0
  - go.dokimi.dev/ergon/lang/go@0.10.0
  - go.dokimi.dev/ergon/lang/java@0.6.0
  - go.dokimi.dev/ergon/lang/javascript@0.7.0
  - go.dokimi.dev/ergon/lang/kotlin@0.4.0
  - go.dokimi.dev/ergon/service@0.21.0

## 0.14.1

### Patch Changes

- Updated dependencies [30797a4, a0544e1]
  - go.dokimi.dev/ergon/lang/typescript@0.3.10
  - go.dokimi.dev/ergon/lang/javascript@0.6.0
  - go.dokimi.dev/ergon/service@0.20.0
  - go.dokimi.dev/ergon/lang/bash@0.4.11
  - go.dokimi.dev/ergon/lang/csharp@0.3.11
  - go.dokimi.dev/ergon/lang/go@0.9.3
  - go.dokimi.dev/ergon/lang/java@0.5.1
  - go.dokimi.dev/ergon/lang/kotlin@0.3.18
  - go.dokimi.dev/ergon/lang/php@0.4.12
  - go.dokimi.dev/ergon/lang/python@0.6.1
  - go.dokimi.dev/ergon/lang/rust@0.6.1
  - go.dokimi.dev/ergon/lang/terraform@0.4.11

## 0.14.0

### Minor Changes

- 926de92: Publish the crates of Cargo workspaces to crates.io through trusted publishing.
- 926de92: Publish the packages of npm and pnpm workspaces to the npm registry through trusted publishing.
- 926de92: Publish the releases of Python projects to PyPI through trusted publishing.

### Patch Changes

- 926de92: Resolve `--out-dir` and `--from-pack-dir` of `ergon release` against the working directory.
- Updated dependencies [005bb4a, 5fcc594, 539a599, cce748c, a902417, 4ffdc5b, 926de92]
  - go.dokimi.dev/ergon/lang/java@0.5.0
  - go.dokimi.dev/ergon/lang/kotlin@0.3.17
  - go.dokimi.dev/ergon/lang/javascript@0.5.0
  - go.dokimi.dev/ergon/lang/rust@0.6.0
  - go.dokimi.dev/ergon/core@0.15.0
  - go.dokimi.dev/ergon/service@0.19.0
  - go.dokimi.dev/ergon/lang/python@0.6.0
  - go.dokimi.dev/ergon/lang/typescript@0.3.9
  - go.dokimi.dev/ergon/lang/bash@0.4.10
  - go.dokimi.dev/ergon/lang/csharp@0.3.10
  - go.dokimi.dev/ergon/lang/go@0.9.2
  - go.dokimi.dev/ergon/lang/php@0.4.11
  - go.dokimi.dev/ergon/lang/terraform@0.4.10

## 0.13.0

### Minor Changes

- 5bb672d: Propose the updates of the copies of spec in the job `propose-spec` of `nightly.yml` in place of `spec.yml`.
- 5bb672d: **Breaking:** Remove the key `spec.ci.actions.propose` from `.ergon.yaml`.

### Patch Changes

- Updated dependencies [2ac18fe, 4bab617, 5bb672d]
  - go.dokimi.dev/ergon/core@0.14.0
  - go.dokimi.dev/ergon/service@0.18.0
  - go.dokimi.dev/ergon/lang/bash@0.4.9
  - go.dokimi.dev/ergon/lang/csharp@0.3.9
  - go.dokimi.dev/ergon/lang/go@0.9.1
  - go.dokimi.dev/ergon/lang/java@0.4.9
  - go.dokimi.dev/ergon/lang/javascript@0.4.8
  - go.dokimi.dev/ergon/lang/kotlin@0.3.16
  - go.dokimi.dev/ergon/lang/php@0.4.10
  - go.dokimi.dev/ergon/lang/python@0.5.6
  - go.dokimi.dev/ergon/lang/rust@0.5.6
  - go.dokimi.dev/ergon/lang/terraform@0.4.9
  - go.dokimi.dev/ergon/lang/typescript@0.3.8

## 0.12.0

### Minor Changes

- 04511bf: Render the files of spec in a repository with a role in `spec.roles`.

### Patch Changes

- Updated dependencies [0292353, 3c4519d, b23672b]
  - go.dokimi.dev/ergon/service@0.17.0
  - go.dokimi.dev/ergon/core@0.13.0
  - go.dokimi.dev/ergon/lang/go@0.9.0
  - go.dokimi.dev/ergon/lang/csharp@0.3.8
  - go.dokimi.dev/ergon/lang/java@0.4.8
  - go.dokimi.dev/ergon/lang/javascript@0.4.7
  - go.dokimi.dev/ergon/lang/kotlin@0.3.15
  - go.dokimi.dev/ergon/lang/python@0.5.5
  - go.dokimi.dev/ergon/lang/rust@0.5.5
  - go.dokimi.dev/ergon/lang/terraform@0.4.8
  - go.dokimi.dev/ergon/lang/typescript@0.3.7
  - go.dokimi.dev/ergon/lang/bash@0.4.8
  - go.dokimi.dev/ergon/lang/php@0.4.9

## 0.11.0

### Minor Changes

- 730a376: Add `ergon tool path` to install a tool and print its program.

### Patch Changes

- fc0dc4d: Require Go 1.27.3.
- Updated dependencies [b452c56, 786739b, e73fbba, 0f5df78, fc0dc4d]
  - go.dokimi.dev/ergon/core@0.12.0
  - go.dokimi.dev/ergon/service@0.16.0
  - go.dokimi.dev/ergon/lang/go@0.8.0
  - go.dokimi.dev/ergon/lang/bash@0.4.7
  - go.dokimi.dev/ergon/lang/csharp@0.3.7
  - go.dokimi.dev/ergon/lang/java@0.4.7
  - go.dokimi.dev/ergon/lang/javascript@0.4.6
  - go.dokimi.dev/ergon/lang/kotlin@0.3.14
  - go.dokimi.dev/ergon/lang/php@0.4.8
  - go.dokimi.dev/ergon/lang/python@0.5.4
  - go.dokimi.dev/ergon/lang/rust@0.5.4
  - go.dokimi.dev/ergon/lang/terraform@0.4.7
  - go.dokimi.dev/ergon/lang/typescript@0.3.6

## 0.10.4

### Patch Changes

- c47195d: Fix `ergon release ci version` and `ergon init ci upgrade` to keep their open pull request open when they update it.
- Updated dependencies [c47195d]
  - go.dokimi.dev/ergon/service@0.15.0
  - go.dokimi.dev/ergon/lang/bash@0.4.6
  - go.dokimi.dev/ergon/lang/csharp@0.3.6
  - go.dokimi.dev/ergon/lang/go@0.7.7
  - go.dokimi.dev/ergon/lang/java@0.4.6
  - go.dokimi.dev/ergon/lang/javascript@0.4.5
  - go.dokimi.dev/ergon/lang/kotlin@0.3.13
  - go.dokimi.dev/ergon/lang/php@0.4.7
  - go.dokimi.dev/ergon/lang/python@0.5.3
  - go.dokimi.dev/ergon/lang/rust@0.5.3
  - go.dokimi.dev/ergon/lang/terraform@0.4.6
  - go.dokimi.dev/ergon/lang/typescript@0.3.5

## 0.10.3

### Patch Changes

- Updated dependencies [1e2b2f4]
  - go.dokimi.dev/ergon/lang/php@0.4.6

## 0.10.2

### Patch Changes

- 87ed939: Fix `ergon release ci select-mode` to count only the changesets that release a package.
- 87ed939: Fix `ergon release ci version` to open a version pull request only for changesets that release a package.
- Updated dependencies [87ed939, 60f5c35]
  - go.dokimi.dev/ergon/service@0.14.0
  - go.dokimi.dev/ergon/lang/bash@0.4.5
  - go.dokimi.dev/ergon/lang/csharp@0.3.5
  - go.dokimi.dev/ergon/lang/go@0.7.6
  - go.dokimi.dev/ergon/lang/java@0.4.5
  - go.dokimi.dev/ergon/lang/javascript@0.4.4
  - go.dokimi.dev/ergon/lang/kotlin@0.3.12
  - go.dokimi.dev/ergon/lang/php@0.4.5
  - go.dokimi.dev/ergon/lang/python@0.5.2
  - go.dokimi.dev/ergon/lang/rust@0.5.2
  - go.dokimi.dev/ergon/lang/terraform@0.4.5
  - go.dokimi.dev/ergon/lang/typescript@0.3.4

## 0.10.1

### Patch Changes

- Updated dependencies [64b7050, 67ba789, 1a63ff6, 452f07d]
  - go.dokimi.dev/ergon/service@0.13.0
  - go.dokimi.dev/ergon/lang/bash@0.4.4
  - go.dokimi.dev/ergon/lang/csharp@0.3.4
  - go.dokimi.dev/ergon/lang/java@0.4.4
  - go.dokimi.dev/ergon/lang/javascript@0.4.3
  - go.dokimi.dev/ergon/lang/kotlin@0.3.11
  - go.dokimi.dev/ergon/lang/php@0.4.4
  - go.dokimi.dev/ergon/lang/python@0.5.1
  - go.dokimi.dev/ergon/lang/rust@0.5.1
  - go.dokimi.dev/ergon/lang/terraform@0.4.4
  - go.dokimi.dev/ergon/lang/typescript@0.3.3
  - go.dokimi.dev/ergon/lang/go@0.7.5

## 0.10.0

### Minor Changes

- 27a850e: Add `--licensed-work`, `--additional-use-grant`, `--change-date` and `--change-license` to `ergon init new` for the parameters of BUSL-1.1.
- c62ac29: **Breaking:** Exit with 2 for `ergon` without a command, after the help on the standard error.
- c62ac29: **Breaking:** Remove the flags `--config` and `-v` and the command `ergon help`.
- a2520a3: Release the members of a uv workspace with `ergon release`.
- c62ac29: Run the command line on go.stealthscale.io/cli, and read each option from its variable of the environment, such as `ERGON_INIT_NEW_OWNER` for `--owner` of `ergon init new`.

### Patch Changes

- Updated dependencies [27a850e, a2520a3, 0a55b7d, 5394d3c, 0a819fb, ee75001, 4295f7e, c62ac29, 5260d20, 8b78d7a]
  - go.dokimi.dev/ergon/service@0.12.0
  - go.dokimi.dev/ergon/core@0.11.0
  - go.dokimi.dev/ergon/lang/python@0.5.0
  - go.dokimi.dev/ergon/lang/rust@0.5.0
  - go.dokimi.dev/ergon/lang/go@0.7.4
  - go.dokimi.dev/ergon/lang/csharp@0.3.3
  - go.dokimi.dev/ergon/lang/java@0.4.3
  - go.dokimi.dev/ergon/lang/javascript@0.4.2
  - go.dokimi.dev/ergon/lang/kotlin@0.3.10
  - go.dokimi.dev/ergon/lang/terraform@0.4.3
  - go.dokimi.dev/ergon/lang/typescript@0.3.2
  - go.dokimi.dev/ergon/lang/bash@0.4.3
  - go.dokimi.dev/ergon/lang/php@0.4.3

## 0.9.1

### Patch Changes

- Updated dependencies [c82c878]
  - go.dokimi.dev/ergon/service@0.11.0
  - go.dokimi.dev/ergon/lang/bash@0.4.2
  - go.dokimi.dev/ergon/lang/csharp@0.3.2
  - go.dokimi.dev/ergon/lang/go@0.7.3
  - go.dokimi.dev/ergon/lang/java@0.4.2
  - go.dokimi.dev/ergon/lang/javascript@0.4.1
  - go.dokimi.dev/ergon/lang/kotlin@0.3.9
  - go.dokimi.dev/ergon/lang/php@0.4.2
  - go.dokimi.dev/ergon/lang/python@0.4.2
  - go.dokimi.dev/ergon/lang/rust@0.4.2
  - go.dokimi.dev/ergon/lang/terraform@0.4.2
  - go.dokimi.dev/ergon/lang/typescript@0.3.1

## 0.9.0

### Minor Changes

- 9f8ee46: Start the npm packages of ergon tool run with the package manager of the section js.

### Patch Changes

- Updated dependencies [9f8ee46]
  - go.dokimi.dev/ergon/lang/javascript@0.4.0
  - go.dokimi.dev/ergon/lang/typescript@0.3.0
  - go.dokimi.dev/ergon/core@0.10.0
  - go.dokimi.dev/ergon/service@0.10.0
  - go.dokimi.dev/ergon/lang/bash@0.4.1
  - go.dokimi.dev/ergon/lang/csharp@0.3.1
  - go.dokimi.dev/ergon/lang/go@0.7.2
  - go.dokimi.dev/ergon/lang/java@0.4.1
  - go.dokimi.dev/ergon/lang/kotlin@0.3.8
  - go.dokimi.dev/ergon/lang/php@0.4.1
  - go.dokimi.dev/ergon/lang/python@0.4.1
  - go.dokimi.dev/ergon/lang/rust@0.4.1
  - go.dokimi.dev/ergon/lang/terraform@0.4.1

## 0.8.4

### Patch Changes

- Updated dependencies [a07f0e1]
  - go.dokimi.dev/ergon/lang/go@0.7.1

## 0.8.3

### Patch Changes

- Updated dependencies [ca0ca90]
  - go.dokimi.dev/ergon/core@0.9.0
  - go.dokimi.dev/ergon/lang/bash@0.4.0
  - go.dokimi.dev/ergon/lang/csharp@0.3.0
  - go.dokimi.dev/ergon/lang/go@0.7.0
  - go.dokimi.dev/ergon/lang/java@0.4.0
  - go.dokimi.dev/ergon/lang/javascript@0.3.0
  - go.dokimi.dev/ergon/lang/php@0.4.0
  - go.dokimi.dev/ergon/lang/python@0.4.0
  - go.dokimi.dev/ergon/lang/rust@0.4.0
  - go.dokimi.dev/ergon/lang/terraform@0.4.0
  - go.dokimi.dev/ergon/lang/kotlin@0.3.7
  - go.dokimi.dev/ergon/lang/typescript@0.2.7
  - go.dokimi.dev/ergon/service@0.9.1

## 0.8.2

### Patch Changes

- Updated dependencies [e854655, 7a2dbd0]
  - go.dokimi.dev/ergon/lang/go@0.6.2

## 0.8.1

### Patch Changes

- Updated dependencies [c17340d, ce53c4b]
  - go.dokimi.dev/ergon/core@0.8.0
  - go.dokimi.dev/ergon/service@0.9.0
  - go.dokimi.dev/ergon/lang/go@0.6.1
  - go.dokimi.dev/ergon/lang/bash@0.3.6
  - go.dokimi.dev/ergon/lang/csharp@0.2.6
  - go.dokimi.dev/ergon/lang/java@0.3.6
  - go.dokimi.dev/ergon/lang/javascript@0.2.6
  - go.dokimi.dev/ergon/lang/kotlin@0.3.6
  - go.dokimi.dev/ergon/lang/php@0.3.6
  - go.dokimi.dev/ergon/lang/python@0.3.6
  - go.dokimi.dev/ergon/lang/rust@0.3.6
  - go.dokimi.dev/ergon/lang/terraform@0.3.6
  - go.dokimi.dev/ergon/lang/typescript@0.2.6

## 0.8.0

### Minor Changes

- 170d883: Add `common.ergon`, the command that runs ergon in the targets of the Makefile and in the commit-msg hook.
- 876696f: Add `ergon tool ci prune`, which deletes each tool cache of GitHub Actions that a newer cache of the same job replaced.
- 876696f: Add `ergon tool prune` and `tool.Runner.Prune`, which remove each file of the tool directory that is not the install of a tool of the options.

### Patch Changes

- Updated dependencies [170d883, 876696f, c6386e5, 74613ae, e93df52]
  - go.dokimi.dev/ergon/service@0.8.0
  - go.dokimi.dev/ergon/core@0.7.0
  - go.dokimi.dev/ergon/lang/go@0.6.0
  - go.dokimi.dev/ergon/lang/csharp@0.2.5
  - go.dokimi.dev/ergon/lang/javascript@0.2.5
  - go.dokimi.dev/ergon/lang/typescript@0.2.5
  - go.dokimi.dev/ergon/lang/bash@0.3.5
  - go.dokimi.dev/ergon/lang/java@0.3.5
  - go.dokimi.dev/ergon/lang/kotlin@0.3.5
  - go.dokimi.dev/ergon/lang/php@0.3.5
  - go.dokimi.dev/ergon/lang/python@0.3.5
  - go.dokimi.dev/ergon/lang/rust@0.3.5
  - go.dokimi.dev/ergon/lang/terraform@0.3.5

## 0.7.0

### Minor Changes

- c80cbcd: Add `go.lint.plugins`, the module plugins of golangci-lint that `ergon tool run` builds into golangci-lint and `.golangci.yml` enables.

### Patch Changes

- 8b59127: Run lint-go, test-go, race-go and audit-go in every module, and fail after the last module when one of them failed. The job check-go of ci.yml runs make --keep-going check-go, so a failed step does not skip the steps after it. The managed .golangci.yml does not run wrapcheck on test files and on the files of generators.
- Updated dependencies [c80cbcd, ccbcc8f, ddcc0a1, 8b59127]
  - go.dokimi.dev/ergon/lang/go@0.5.0
  - go.dokimi.dev/ergon/core@0.6.0
  - go.dokimi.dev/ergon/service@0.7.0
  - go.dokimi.dev/ergon/lang/bash@0.3.4
  - go.dokimi.dev/ergon/lang/csharp@0.2.4
  - go.dokimi.dev/ergon/lang/java@0.3.4
  - go.dokimi.dev/ergon/lang/javascript@0.2.4
  - go.dokimi.dev/ergon/lang/kotlin@0.3.4
  - go.dokimi.dev/ergon/lang/php@0.3.4
  - go.dokimi.dev/ergon/lang/python@0.3.4
  - go.dokimi.dev/ergon/lang/rust@0.3.4
  - go.dokimi.dev/ergon/lang/terraform@0.3.4
  - go.dokimi.dev/ergon/lang/typescript@0.2.4

## 0.6.1

### Patch Changes

- b68662b: Install a Go tool of ergon tool run once for each version of the go command. A build of dokimi-mutate-go by an older Go refuses the packages of a newer one.
- da88eb5: Describe the cask of ergon without the full stop at its end, which brew audit refuses.
- Updated dependencies [b68662b, dfc0ced]
  - go.dokimi.dev/ergon/service@0.6.1
  - go.dokimi.dev/ergon/lang/go@0.4.1
  - go.dokimi.dev/ergon/lang/bash@0.3.3
  - go.dokimi.dev/ergon/lang/csharp@0.2.3
  - go.dokimi.dev/ergon/lang/java@0.3.3
  - go.dokimi.dev/ergon/lang/javascript@0.2.3
  - go.dokimi.dev/ergon/lang/kotlin@0.3.3
  - go.dokimi.dev/ergon/lang/php@0.3.3
  - go.dokimi.dev/ergon/lang/python@0.3.3
  - go.dokimi.dev/ergon/lang/rust@0.3.3
  - go.dokimi.dev/ergon/lang/terraform@0.3.3
  - go.dokimi.dev/ergon/lang/typescript@0.2.3

## 0.6.0

### Minor Changes

- b8f1c6a: Build, sign and attest the binaries of the commands of a Go module in each release of the module. The key binaries of the section go lists the commands. ergon init renders the configuration of GoReleaser of each module with a command. The job pack of release.yml builds the archives, the deb, rpm and apk packages, the casks, the checksums with their cosign signature and the SBOMs, and attests them. UPX packs each Linux binary with LZMA, and ergon tool run unpacks the .tar.xz of a release binary such as UPX. ergon release publish attaches the assets to a draft release before it publishes the release. ergon release ci homebrew then commits the casks to the tap of the key homebrew. The managed .gitignore ignores publish-plan.json and /dist/, so that Go writes the version of the release into each binary.
- d78232a: Read the version of ergon from the build information that Go writes from the tag of its module, without flags of the linker, and report a build of a working tree with changes as dev.
- d616a74: Refuse to write a new lock, or a lock that a release wrote, in a build of ergon without a release, whose version dev no release has. Such a build keeps a lock that such a build wrote.
- 326b07e: Render nightly.yml, which runs the long steps of each language on a schedule, each step in a job of its own. The key nightly of the section go maps fuzz, bench and mutate to the limit of each job in minutes, and the key nightly.schedule of the section github states when the workflow runs.

### Patch Changes

- 14e7a3a: Move golang.org/x/net to v0.60.0, which fixes GO-2026-6603, GO-2026-6610, GO-2026-6611, GO-2026-6612 and GO-2026-6617.
- babadd5: Require Go 1.27.2, whose standard library fixes the nine vulnerabilities that govulncheck reports for Go 1.27.1, such as GO-2026-6604 of os.Root on Windows.
- 2fdb4d1: Return an error on Windows for a file in place of the directory of the assets or the casks of a release, as on Linux and macOS. ergon release publish and ergon release ci homebrew read the directory.
- Updated dependencies [b8f1c6a, 14e7a3a, d616a74, 326b07e, babadd5, 2fdb4d1]
  - go.dokimi.dev/ergon/core@0.5.0
  - go.dokimi.dev/ergon/service@0.6.0
  - go.dokimi.dev/ergon/lang/go@0.4.0
  - go.dokimi.dev/ergon/lang/bash@0.3.2
  - go.dokimi.dev/ergon/lang/csharp@0.2.2
  - go.dokimi.dev/ergon/lang/java@0.3.2
  - go.dokimi.dev/ergon/lang/javascript@0.2.2
  - go.dokimi.dev/ergon/lang/kotlin@0.3.2
  - go.dokimi.dev/ergon/lang/php@0.3.2
  - go.dokimi.dev/ergon/lang/python@0.3.2
  - go.dokimi.dev/ergon/lang/rust@0.3.2
  - go.dokimi.dev/ergon/lang/terraform@0.3.2
  - go.dokimi.dev/ergon/lang/typescript@0.2.2

## 0.5.0

### Minor Changes

- 4044f32: Skip the jobs of a run of ci.yml when a passed run covers its content: a push whose content passed, and a version commit whose parent passed. The new first job skip runs ergon release ci skip, ergon release ci version marks the commit of the version pull request with the status ergon/version, and the job result is the one check that a branch requires.

### Patch Changes

- Updated dependencies [4044f32]
  - go.dokimi.dev/ergon/service@0.5.0
  - go.dokimi.dev/ergon/lang/bash@0.3.1
  - go.dokimi.dev/ergon/lang/csharp@0.2.1
  - go.dokimi.dev/ergon/lang/go@0.3.1
  - go.dokimi.dev/ergon/lang/java@0.3.1
  - go.dokimi.dev/ergon/lang/javascript@0.2.1
  - go.dokimi.dev/ergon/lang/kotlin@0.3.1
  - go.dokimi.dev/ergon/lang/php@0.3.1
  - go.dokimi.dev/ergon/lang/python@0.3.1
  - go.dokimi.dev/ergon/lang/rust@0.3.1
  - go.dokimi.dev/ergon/lang/terraform@0.3.1
  - go.dokimi.dev/ergon/lang/typescript@0.2.1

## 0.4.0

### Minor Changes

- 354fe1c: Open the version pull request in version.yml when the run of ci.yml for a push to main passes, and publish in release.yml after ergon release ci verify finds a run of ci.yml that passed on the content of the commit, such as the run of the version pull request. ergon release ci verify replaces ergon release ci wait, so no job waits for another workflow. ergon release ci version skips a commit that is no longer the head of main. A GitHub App in the variable ERGON_APP_CLIENT_ID and the secret ERGON_APP_PRIVATE_KEY opens the version pull request, so its checks run without an approval. Each push to main gets a concurrency group of its own in ci.yml.

### Patch Changes

- Updated dependencies [c61862c, 12066ae, 806100c, f81caba, 354fe1c, 7c5311d]
  - go.dokimi.dev/ergon/core@0.4.0
  - go.dokimi.dev/ergon/service@0.4.0
  - go.dokimi.dev/ergon/lang/bash@0.3.0
  - go.dokimi.dev/ergon/lang/csharp@0.2.0
  - go.dokimi.dev/ergon/lang/go@0.3.0
  - go.dokimi.dev/ergon/lang/java@0.3.0
  - go.dokimi.dev/ergon/lang/javascript@0.2.0
  - go.dokimi.dev/ergon/lang/kotlin@0.3.0
  - go.dokimi.dev/ergon/lang/php@0.3.0
  - go.dokimi.dev/ergon/lang/python@0.3.0
  - go.dokimi.dev/ergon/lang/rust@0.3.0
  - go.dokimi.dev/ergon/lang/terraform@0.3.0
  - go.dokimi.dev/ergon/lang/typescript@0.2.0

## 0.3.0

### Minor Changes

- 94c3bd1: Add ergon init upgrade and ergon init ci upgrade, which move a repository to the baseline of the newest release of ergon. The job of baseline.yml opens the pull request of the upgrade weekly, in place of an issue. ergon refuses a lock that a newer release of ergon wrote, and a local dependabot.yml that updates github-actions or pre-commit.
- 6f01723: Seed .changeset/config.json with the changelog format @changesets/cli/changelog, whose entries thank no author.
- 77ee1c0: Wait for the CI run of a commit before release.yml versions or publishes it. The job wait runs a new command for this, ergon release ci wait. The workflow release.yml no longer calls ci.yml, so each push to main runs the gate once.

### Patch Changes

- 8bfa588: Make ergon release version print removed for each changeset that it removes. release.Version returns the removed paths apart from the written paths.
- Updated dependencies [94c3bd1, 8bfa588, 096abbd, 6f01723, 77ee1c0]
  - go.dokimi.dev/ergon/service@0.3.0
  - go.dokimi.dev/ergon/core@0.3.0
  - go.dokimi.dev/ergon/lang/csharp@0.1.2
  - go.dokimi.dev/ergon/lang/go@0.2.1
  - go.dokimi.dev/ergon/lang/javascript@0.1.2
  - go.dokimi.dev/ergon/lang/kotlin@0.2.1
  - go.dokimi.dev/ergon/lang/php@0.2.1
  - go.dokimi.dev/ergon/lang/python@0.2.1
  - go.dokimi.dev/ergon/lang/rust@0.2.1
  - go.dokimi.dev/ergon/lang/typescript@0.1.2
  - go.dokimi.dev/ergon/lang/bash@0.2.1
  - go.dokimi.dev/ergon/lang/java@0.2.1
  - go.dokimi.dev/ergon/lang/terraform@0.2.1

## 0.2.0

### Minor Changes

- [`afd76a1`](https://github.com/dokimasia/ergon/commit/afd76a177a970930daaf431bb6f6266b9c5028a3) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Give each directory that license.directories lists a license of its own: the header of its files, and its LICENSE, with a NOTICE for Apache-2.0.

- [`768f7d7`](https://github.com/dokimasia/ergon/commit/768f7d7cf70a5b4a0268a61d07dfb30d4e246cc7) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Keep the tools of ergon in the cache of GitHub Actions in each job of ci.yml that runs tools, and set up the toolchain of a job before ergon.

- [`6b1c2b5`](https://github.com/dokimasia/ergon/commit/6b1c2b5508b876007cff6ebab3dae22ac07d278f) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Run the job baseline of ci.yml on the Linux runner alone, and group the minor and patch updates of each package manager into one pull request of Dependabot.

### Patch Changes

- Updated dependencies [[`afd76a1`](https://github.com/dokimasia/ergon/commit/afd76a177a970930daaf431bb6f6266b9c5028a3), [`768f7d7`](https://github.com/dokimasia/ergon/commit/768f7d7cf70a5b4a0268a61d07dfb30d4e246cc7), [`6b1c2b5`](https://github.com/dokimasia/ergon/commit/6b1c2b5508b876007cff6ebab3dae22ac07d278f)]:
  - go.dokimi.dev/ergon/core@0.2.0
  - go.dokimi.dev/ergon/service@0.2.0
  - go.dokimi.dev/ergon/lang/bash@0.2.0
  - go.dokimi.dev/ergon/lang/go@0.2.0
  - go.dokimi.dev/ergon/lang/java@0.2.0
  - go.dokimi.dev/ergon/lang/kotlin@0.2.0
  - go.dokimi.dev/ergon/lang/php@0.2.0
  - go.dokimi.dev/ergon/lang/python@0.2.0
  - go.dokimi.dev/ergon/lang/rust@0.2.0
  - go.dokimi.dev/ergon/lang/terraform@0.2.0
  - go.dokimi.dev/ergon/lang/csharp@0.1.1
  - go.dokimi.dev/ergon/lang/javascript@0.1.1
  - go.dokimi.dev/ergon/lang/typescript@0.1.1

## 0.1.0

### Minor Changes

- [`0fa50eb`](https://github.com/dokimasia/ergon/commit/0fa50eb601a69e3602fcb2a442ee54a70a08ba21) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Release the first version.

### Patch Changes

- Updated dependencies [[`0fa50eb`](https://github.com/dokimasia/ergon/commit/0fa50eb601a69e3602fcb2a442ee54a70a08ba21)]:
  - go.dokimi.dev/ergon/core@0.1.0
  - go.dokimi.dev/ergon/service@0.1.0
  - go.dokimi.dev/ergon/lang/bash@0.1.0
  - go.dokimi.dev/ergon/lang/csharp@0.1.0
  - go.dokimi.dev/ergon/lang/go@0.1.0
  - go.dokimi.dev/ergon/lang/java@0.1.0
  - go.dokimi.dev/ergon/lang/javascript@0.1.0
  - go.dokimi.dev/ergon/lang/kotlin@0.1.0
  - go.dokimi.dev/ergon/lang/php@0.1.0
  - go.dokimi.dev/ergon/lang/python@0.1.0
  - go.dokimi.dev/ergon/lang/rust@0.1.0
  - go.dokimi.dev/ergon/lang/terraform@0.1.0
  - go.dokimi.dev/ergon/lang/typescript@0.1.0
