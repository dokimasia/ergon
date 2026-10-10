# go.dokimi.dev/ergon/lang/typescript

## 0.2.5

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
  - go.dokimi.dev/ergon/lang/javascript@0.2.5

## 0.2.4

### Patch Changes

- Updated dependencies [ccbcc8f]
- Updated dependencies [ddcc0a1]
  - go.dokimi.dev/ergon/core@0.6.0
  - go.dokimi.dev/ergon/service@0.7.0
  - go.dokimi.dev/ergon/lang/javascript@0.2.4

## 0.2.3

### Patch Changes

- Updated dependencies [b68662b]
  - go.dokimi.dev/ergon/service@0.6.1
  - go.dokimi.dev/ergon/lang/javascript@0.2.3

## 0.2.2

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
  - go.dokimi.dev/ergon/lang/javascript@0.2.2

## 0.2.1

### Patch Changes

- Updated dependencies [4044f32]
  - go.dokimi.dev/ergon/service@0.5.0
  - go.dokimi.dev/ergon/lang/javascript@0.2.1

## 0.2.0

### Minor Changes

- c61862c: Add make generate, which runs the generators of every language, and the options generate.command and generate.args to every language section of .ergon.yaml. generate-<language> runs the generators, and verify-generate-<language>, the step generate of check, fails when they change a file. make help lists the targets in groups: Common, each language, and the targets of the repository.

### Patch Changes

- Updated dependencies [c61862c]
- Updated dependencies [12066ae]
- Updated dependencies [806100c]
- Updated dependencies [354fe1c]
- Updated dependencies [7c5311d]
  - go.dokimi.dev/ergon/core@0.4.0
  - go.dokimi.dev/ergon/service@0.4.0
  - go.dokimi.dev/ergon/lang/javascript@0.2.0

## 0.1.2

### Patch Changes

- Updated dependencies [94c3bd1]
- Updated dependencies [8bfa588]
- Updated dependencies [096abbd]
- Updated dependencies [6f01723]
- Updated dependencies [77ee1c0]
  - go.dokimi.dev/ergon/service@0.3.0
  - go.dokimi.dev/ergon/core@0.3.0
  - go.dokimi.dev/ergon/lang/javascript@0.1.2

## 0.1.1

### Patch Changes

- Updated dependencies [[`afd76a1`](https://github.com/dokimasia/ergon/commit/afd76a177a970930daaf431bb6f6266b9c5028a3), [`768f7d7`](https://github.com/dokimasia/ergon/commit/768f7d7cf70a5b4a0268a61d07dfb30d4e246cc7), [`6b1c2b5`](https://github.com/dokimasia/ergon/commit/6b1c2b5508b876007cff6ebab3dae22ac07d278f)]:
  - go.dokimi.dev/ergon/core@0.2.0
  - go.dokimi.dev/ergon/service@0.2.0
  - go.dokimi.dev/ergon/lang/javascript@0.1.1

## 0.1.0

### Minor Changes

- [`0fa50eb`](https://github.com/dokimasia/ergon/commit/0fa50eb601a69e3602fcb2a442ee54a70a08ba21) Thanks [@stealth-rklopper](https://github.com/stealth-rklopper)! - Release the first version.

### Patch Changes

- Updated dependencies [[`0fa50eb`](https://github.com/dokimasia/ergon/commit/0fa50eb601a69e3602fcb2a442ee54a70a08ba21)]:
  - go.dokimi.dev/ergon/core@0.1.0
  - go.dokimi.dev/ergon/service@0.1.0
  - go.dokimi.dev/ergon/lang/javascript@0.1.0
