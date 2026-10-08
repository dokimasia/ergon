# go.dokimi.dev/ergon/service

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
