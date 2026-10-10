# go.dokimi.dev/ergon/lang/go

## 0.6.1

### Patch Changes

- ce53c4b: Set up Go and set GOPRIVATE to the modules of `go.work` before the CodeQL analysis of Go.
- Updated dependencies [c17340d]
  - go.dokimi.dev/ergon/core@0.8.0
  - go.dokimi.dev/ergon/service@0.9.0

## 0.6.0

### Minor Changes

- e93df52: **Breaking:** Replace `go.tools.ergon-go-vet` and `go.lint.exclude` with `go.lint.analyzers`, which builds the analyzers of `go.dokimi.dev/lint` into golangci-lint.
- e93df52: **Breaking:** Remove the package `analysis` and the command `ergon-go-vet`, which moved to `go.dokimi.dev/lint` as its package `analysis` and its command `dokimi-lint-go`.
- c6386e5: Key the tool cache of a job by the files of `workflow.Setup.VersionFiles`, which the jobs of Go set to `go.work`.

### Patch Changes

- Updated dependencies [170d883]
- Updated dependencies [876696f]
- Updated dependencies [c6386e5]
- Updated dependencies [74613ae]
- Updated dependencies [74613ae]
- Updated dependencies [74613ae]
- Updated dependencies [74613ae]
- Updated dependencies [74613ae]
- Updated dependencies [c6386e5]
  - go.dokimi.dev/ergon/service@0.8.0
  - go.dokimi.dev/ergon/core@0.7.0

## 0.5.0

### Minor Changes

- c80cbcd: Add `go.lint.plugins`, the module plugins of golangci-lint that `ergon tool run` builds into golangci-lint and `.golangci.yml` enables.

### Patch Changes

- 8b59127: Run lint-go, test-go, race-go and audit-go in every module, and fail after the last module when one of them failed. The job check-go of ci.yml runs make --keep-going check-go, so a failed step does not skip the steps after it. The managed .golangci.yml does not run wrapcheck on test files and on the files of generators.
- Updated dependencies [ccbcc8f]
- Updated dependencies [ddcc0a1]
  - go.dokimi.dev/ergon/core@0.6.0
  - go.dokimi.dev/ergon/service@0.7.0

## 0.4.1

### Patch Changes

- dfc0ced: Refuse a description of a command with a cask that brew audit refuses, such as a description with a full stop at its end.
- Updated dependencies [b68662b]
  - go.dokimi.dev/ergon/service@0.6.1

## 0.4.0

### Minor Changes

- b8f1c6a: Build, sign and attest the binaries of the commands of a Go module in each release of the module. The key binaries of the section go lists the commands. ergon init renders the configuration of GoReleaser of each module with a command. The job pack of release.yml builds the archives, the deb, rpm and apk packages, the casks, the checksums with their cosign signature and the SBOMs, and attests them. UPX packs each Linux binary with LZMA, and ergon tool run unpacks the .tar.xz of a release binary such as UPX. ergon release publish attaches the assets to a draft release before it publishes the release. ergon release ci homebrew then commits the casks to the tap of the key homebrew. The managed .gitignore ignores publish-plan.json and /dist/, so that Go writes the version of the release into each binary.
- 326b07e: Render nightly.yml, which runs the long steps of each language on a schedule, each step in a job of its own. The key nightly of the section go maps fuzz, bench and mutate to the limit of each job in minutes, and the key nightly.schedule of the section github states when the workflow runs.

### Patch Changes

- babadd5: Require Go 1.27.2, whose standard library fixes the nine vulnerabilities that govulncheck reports for Go 1.27.1, such as GO-2026-6604 of os.Root on Windows.
- Updated dependencies [b8f1c6a]
- Updated dependencies [14e7a3a]
- Updated dependencies [d616a74]
- Updated dependencies [326b07e]
- Updated dependencies [babadd5]
- Updated dependencies [2fdb4d1]
  - go.dokimi.dev/ergon/core@0.5.0
  - go.dokimi.dev/ergon/service@0.6.0

## 0.3.1

### Patch Changes

- Updated dependencies [4044f32]
  - go.dokimi.dev/ergon/service@0.5.0

## 0.3.0

### Minor Changes

- c61862c: Add make generate, which runs the generators of every language, and the options generate.command and generate.args to every language section of .ergon.yaml. generate-<language> runs the generators, and verify-generate-<language>, the step generate of check, fails when they change a file. make help lists the targets in groups: Common, each language, and the targets of the repository.

### Patch Changes

- f81caba: Move the pins of the baseline to their newest releases.
  
  - `go.tools.ergon-go-vet` from v0.1.0 to v0.2.1
- Updated dependencies [c61862c]
- Updated dependencies [12066ae]
- Updated dependencies [806100c]
- Updated dependencies [354fe1c]
- Updated dependencies [7c5311d]
  - go.dokimi.dev/ergon/core@0.4.0
  - go.dokimi.dev/ergon/service@0.4.0

## 0.2.1

### Patch Changes

- Updated dependencies [94c3bd1]
- Updated dependencies [8bfa588]
- Updated dependencies [096abbd]
- Updated dependencies [6f01723]
- Updated dependencies [77ee1c0]
  - go.dokimi.dev/ergon/service@0.3.0
  - go.dokimi.dev/ergon/core@0.3.0

## 0.2.0

### Minor Changes

- [`768f7d7`](https://github.com/dokimasia/ergon/commit/768f7d7cf70a5b4a0268a61d07dfb30d4e246cc7) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Keep the tools of ergon in the cache of GitHub Actions in each job of ci.yml that runs tools, and set up the toolchain of a job before ergon.

### Patch Changes

- Updated dependencies [[`afd76a1`](https://github.com/dokimasia/ergon/commit/afd76a177a970930daaf431bb6f6266b9c5028a3), [`768f7d7`](https://github.com/dokimasia/ergon/commit/768f7d7cf70a5b4a0268a61d07dfb30d4e246cc7), [`6b1c2b5`](https://github.com/dokimasia/ergon/commit/6b1c2b5508b876007cff6ebab3dae22ac07d278f)]:
  - go.dokimi.dev/ergon/core@0.2.0
  - go.dokimi.dev/ergon/service@0.2.0

## 0.1.0

### Minor Changes

- [`0fa50eb`](https://github.com/dokimasia/ergon/commit/0fa50eb601a69e3602fcb2a442ee54a70a08ba21) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Release the first version.

### Patch Changes

- Updated dependencies [[`0fa50eb`](https://github.com/dokimasia/ergon/commit/0fa50eb601a69e3602fcb2a442ee54a70a08ba21)]:
  - go.dokimi.dev/ergon/core@0.1.0
  - go.dokimi.dev/ergon/service@0.1.0
