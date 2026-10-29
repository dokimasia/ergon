# go.dokimi.dev/ergon/lang/python

## 0.7.3

### Patch Changes

- Updated dependencies [199d84c, 679728c]
  - go.dokimi.dev/ergon/core@0.19.0
  - go.dokimi.dev/ergon/service@0.24.0

## 0.7.2

### Patch Changes

- 66acd85: Move the pins of the baseline to their newest releases.
  
  - `common.tools.rumdl` from 0.2.79 to 0.3.0
  - `github.ci.actions.codeql` from v4.38.2 to v4.39.1
  - `github.ci.actions.scorecard` from v2.4.4 to v2.5.0
  - `github.ci.actions.create-github-app-token` from v3.2.0 to v3.3.0
  - `jvm.tools.osv-scanner` from 2.6.0 to 2.7.0
  - `jvm.ci.actions.setup-java` from v6.0.1 to v6.1.0
  - `php.tools.phpstan` from 2.2.17 to 2.3.1
  - `php.ci.actions.setup-php` from 2.37.2 to 2.40.0
  - `js.tools.biome` from 2.5.15 to 2.6.2
  - `js.ci.actions.setup-node` from v7.0.0 to v7.1.0
  - `go.tools.golangci-lint` from v2.14.0 to v2.15.0
  - `go.tools.govulncheck` from v1.8.0 to v1.9.0
  - `go.tools.benchstat` from v0.0.0-20260929162123-406019bb8b68 to v0.0.0-20261009192801-be2c69fb417e
  - `go.tools.goreleaser` from 2.18.2 to 2.19.0
  - `go.tools.cosign` from 3.1.3 to 3.1.4
  - `go.tools.syft` from 1.54.1 to 1.55.0
  - `python.tools.uv` from 0.12.23 to 0.13.5
  - `python.tools.ruff` from 0.16.10 to 0.17.1
  - `terraform.tools.uv` from 0.12.23 to 0.13.5
  - `terraform.tools.checkov` from 3.3.23 to 3.3.30
- Updated dependencies [e64915e, aca8fdb, d60c2e1, 66acd85]
  - go.dokimi.dev/ergon/core@0.18.0
  - go.dokimi.dev/ergon/service@0.23.0

## 0.7.1

### Patch Changes

- Updated dependencies [2bccfb2, efca217, c960170, f61f91e, 0488606, 6da7d0b]
  - go.dokimi.dev/ergon/service@0.22.0
  - go.dokimi.dev/ergon/core@0.17.0

## 0.7.0

### Minor Changes

- 477d633: **Breaking:** Run each command line of the key `generate.commands` in `generate-python` and `verify-generate-python` in place of the keys `generate.command` and `generate.args`.

### Patch Changes

- Updated dependencies [2e44067, 3af4e3d, 43a3f4c]
  - go.dokimi.dev/ergon/core@0.16.0
  - go.dokimi.dev/ergon/service@0.21.0

## 0.6.1

### Patch Changes

- Updated dependencies [a0544e1]
  - go.dokimi.dev/ergon/service@0.20.0

## 0.6.0

### Minor Changes

- 4ffdc5b: Build the distributions of a release with `uv build` and upload them to PyPI with `uv publish`.
- 4ffdc5b: **Breaking:** Take the client of PyPI in `python.Register`.

### Patch Changes

- Updated dependencies [cce748c, a902417, 926de92]
  - go.dokimi.dev/ergon/core@0.15.0
  - go.dokimi.dev/ergon/service@0.19.0

## 0.5.6

### Patch Changes

- Updated dependencies [2ac18fe, 4bab617, 5bb672d]
  - go.dokimi.dev/ergon/core@0.14.0
  - go.dokimi.dev/ergon/service@0.18.0

## 0.5.5

### Patch Changes

- Updated dependencies [0292353, 3c4519d, b23672b]
  - go.dokimi.dev/ergon/service@0.17.0
  - go.dokimi.dev/ergon/core@0.13.0

## 0.5.4

### Patch Changes

- fc0dc4d: Require Go 1.27.3.
- Updated dependencies [b452c56, 786739b, 0f5df78, fc0dc4d]
  - go.dokimi.dev/ergon/core@0.12.0
  - go.dokimi.dev/ergon/service@0.16.0

## 0.5.3

### Patch Changes

- Updated dependencies [c47195d]
  - go.dokimi.dev/ergon/service@0.15.0

## 0.5.2

### Patch Changes

- Updated dependencies [87ed939, 60f5c35]
  - go.dokimi.dev/ergon/service@0.14.0

## 0.5.1

### Patch Changes

- Updated dependencies [64b7050, 67ba789]
  - go.dokimi.dev/ergon/service@0.13.0

## 0.5.0

### Minor Changes

- ee75001: Add `python.audit.skip` for the packages of `uv.lock` without a release on PyPI.
- a2520a3: **Breaking:** Take the tools of the section python in `python.Register`.
- a2520a3: Discover the members of a uv workspace as the packages of a release.
- a2520a3: Write the version and the requirements of a project into its `pyproject.toml` in a release, and refresh `uv.lock`.

### Patch Changes

- ee75001: Set `extend-exclude = [".ergon"]` in `ruff.toml`, and describe the merge of a local `ruff.toml`.
- Updated dependencies [27a850e, a2520a3, 0a55b7d, 5394d3c, 0a819fb, 8b78d7a]
  - go.dokimi.dev/ergon/service@0.12.0
  - go.dokimi.dev/ergon/core@0.11.0

## 0.4.2

### Patch Changes

- Updated dependencies [c82c878]
  - go.dokimi.dev/ergon/service@0.11.0

## 0.4.1

### Patch Changes

- Updated dependencies [9f8ee46]
  - go.dokimi.dev/ergon/core@0.10.0
  - go.dokimi.dev/ergon/service@0.10.0

## 0.4.0

### Minor Changes

- ca0ca90: Add the key `ci.steps` of each toolchain, whose steps its jobs run after the setup of the toolchain.

### Patch Changes

- Updated dependencies [ca0ca90]
  - go.dokimi.dev/ergon/core@0.9.0
  - go.dokimi.dev/ergon/service@0.9.1

## 0.3.6

### Patch Changes

- Updated dependencies [c17340d]
  - go.dokimi.dev/ergon/core@0.8.0
  - go.dokimi.dev/ergon/service@0.9.0

## 0.3.5

### Patch Changes

- Updated dependencies [170d883, 876696f, c6386e5, 74613ae]
  - go.dokimi.dev/ergon/service@0.8.0
  - go.dokimi.dev/ergon/core@0.7.0

## 0.3.4

### Patch Changes

- Updated dependencies [ccbcc8f, ddcc0a1]
  - go.dokimi.dev/ergon/core@0.6.0
  - go.dokimi.dev/ergon/service@0.7.0

## 0.3.3

### Patch Changes

- Updated dependencies [b68662b]
  - go.dokimi.dev/ergon/service@0.6.1

## 0.3.2

### Patch Changes

- babadd5: Require Go 1.27.2, whose standard library fixes the nine vulnerabilities that govulncheck reports for Go 1.27.1, such as GO-2026-6604 of os.Root on Windows.
- Updated dependencies [b8f1c6a, 14e7a3a, d616a74, 326b07e, babadd5, 2fdb4d1]
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

- Updated dependencies [c61862c, 12066ae, 806100c, 354fe1c, 7c5311d]
  - go.dokimi.dev/ergon/core@0.4.0
  - go.dokimi.dev/ergon/service@0.4.0

## 0.2.1

### Patch Changes

- Updated dependencies [94c3bd1, 8bfa588, 096abbd, 6f01723, 77ee1c0]
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
