# go.dokimi.dev/ergon/lang/bash

## 0.4.4

### Patch Changes

- Updated dependencies [64b7050, 67ba789]
  - go.dokimi.dev/ergon/service@0.13.0

## 0.4.3

### Patch Changes

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

- 94c3bd1: State the repository on GitHub whose releases publish each release binary, which a baseline update reads.
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
