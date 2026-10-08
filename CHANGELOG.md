# go.dokimi.dev/ergon

## 0.4.0

### Minor Changes

- 354fe1c: Open the version pull request in version.yml when the run of ci.yml for a push to main passes, and publish in release.yml after ergon release ci verify finds a run of ci.yml that passed on the content of the commit, such as the run of the version pull request. ergon release ci verify replaces ergon release ci wait, so no job waits for another workflow. ergon release ci version skips a commit that is no longer the head of main. A GitHub App in the variable ERGON_APP_CLIENT_ID and the secret ERGON_APP_PRIVATE_KEY opens the version pull request, so its checks run without an approval. Each push to main gets a concurrency group of its own in ci.yml.

### Patch Changes

- Updated dependencies [c61862c]
- Updated dependencies [12066ae]
- Updated dependencies [806100c]
- Updated dependencies [f81caba]
- Updated dependencies [354fe1c]
- Updated dependencies [7c5311d]
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
- Updated dependencies [94c3bd1]
- Updated dependencies [8bfa588]
- Updated dependencies [096abbd]
- Updated dependencies [6f01723]
- Updated dependencies [94c3bd1]
- Updated dependencies [77ee1c0]
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
