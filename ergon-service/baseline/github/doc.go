// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package github is the producer of the GitHub files of ergon init: the workflows, the actions
// that install ergon and GNU make, the configuration of Dependabot, the configuration of
// actionlint, the issue forms, the template of a pull request, and CODEOWNERS.
//
// [Producer] renders them from the templates under templates/, whose tree mirrors the paths in the
// repository: templates/managed/.github/ for the managed files, templates/seeded/.github/ for
// CODEOWNERS, and templates/shared/ for the fragment of the Makefile with the target lint-github.
// Each template ends in .tmpl, so GitHub takes no template for a workflow.
//
// # Workflows
//
// The GitHub files render the contributions of every producer, so no toolchain appears in them:
//
//   - ci.yml is the gate on every pull request, on the merge queue and on main. It renders each
//     job of the contributions from one skeleton: the checkout, the installation of GNU make, the
//     setup steps, the installation of ergon, the cache of the tools of ergon for a job that runs
//     tools, and the steps of the job, as [Options.Jobs] states.
//   - nightly.yml runs the nightly jobs of the contributions on the schedule of the section github
//     and on demand, each from the same skeleton, on the Linux runner unless its setup lists
//     runners, as [Options.NightlyJobs] states. A job with App creates the token of the GitHub App
//     of ERGON_APP_CLIENT_ID and ERGON_APP_PRIVATE_KEY before its setup steps. It creates the token
//     only when the repository has the variable. When a job keeps the tools of ergon in the cache,
//     the job prune-tools deletes each cache of the tools that a newer cache of the same job
//     replaced, through ergon tool ci prune. A repository without either job has no nightly.yml.
//   - version.yml opens or updates the version pull request when the run of ci.yml for a push to
//     main succeeds, from the commit of that run, through ergon release ci version. With the
//     variable ERGON_APP_CLIENT_ID and the secret ERGON_APP_PRIVATE_KEY of a GitHub App, it opens
//     the pull request with a token of the App, whose events start the runs of the checks.
//   - release.yml publishes on every push to main. The job select-mode chooses the next job. The
//     job verify lets the publish run only when a run of ci.yml passed on the content of the commit,
//     through ergon release ci verify. The jobs pack and publish then release each package of the
//     publish plan. When a contribution has assets, the job pack attests them with actions/attest,
//     with the permissions id-token and attestations, which cosign also signs with. When a
//     contribution uploads packages to a registry, the job publish runs in the environment release
//     with the permission id-token, whose OIDC token the trusted publishing of the registry
//     exchanges. When the assets have a tap, the job homebrew commits their casks to it after the
//     publish, with a token of the GitHub App of ERGON_APP_CLIENT_ID and ERGON_APP_PRIVATE_KEY,
//     through ergon release ci homebrew.
//   - The job version of version.yml and the job pack of release.yml run the release steps of the
//     contributions before the installation of ergon. The job publish of release.yml runs them too
//     when a contribution uploads packages to a registry.
//   - security.yml runs the dependency review on a pull request, the OpenSSF Scorecard weekly, and
//     a job codeql-<language> for each CodeQL analysis of the contributions, which calls
//     codeql.yml. codeql.yml runs the steps of each analysis before CodeQL, in the runs of the
//     language of that analysis alone. The job dependency-review runs the review steps of the
//     contributions before the review. When a contribution has such steps, the review leaves the
//     packages of [workflow.OwnPackages] out of its license check. A private repository needs GitHub
//     Code Security for the analyses and the review, so ergon init renders neither workflow when the
//     key code-security is false. The job visibility queries the API of GitHub for the visibility of
//     the repository, because a scheduled run has no event payload. The Scorecard runs only in a
//     public repository.
//   - baseline.yml runs ergon init ci upgrade weekly, which moves the managed files to the newest
//     release of ergon and opens or updates the pull request of the change.
//   - dependabot.yml updates the package manager of each update of the contributions weekly, with
//     the minor and patch updates of each directory in one pull request and each major update in a
//     pull request of its own. A repository without an update has no dependabot.yml.
//
// The setup steps of a job come before the installation of ergon, so a repository that builds ergon
// from its own source builds it with the toolchain of the job. A job whose steps run tools restores
// the tool directory of ergon from the cache of GitHub Actions, and saves it after a run that
// succeeds, under a key of the system, the architecture, the job, the runtime version of its matrix,
// and the digest of .ergon.yaml, the lock and the version files of its setup. [ToolsCache] starts
// the key. When the key misses, the job restores the newest cache of the same system, architecture,
// job and runtime version, and its last step runs ergon tool prune, so the cache saves only the
// tools that the options name.
//
// Every workflow grants no permission at its top level, and each job grants only the scopes that
// it needs. Every action is pinned to the commit of a release, with the release in a comment.
// actions/checkout runs with persist-credentials set to false. The concurrency groups of the
// workflows are these:
//
//   - ci.yml cancels a superseded run of a pull request, and gives each run of a push or a merge
//     group a group of its own, so the run of a push never waits for the run of an earlier push.
//   - security.yml groups its runs by workflow and ref, and cancels a superseded run of a pull
//     request.
//   - release.yml and baseline.yml group their runs by workflow and ref, and version.yml by the
//     branch of the run of ci.yml that triggered it. These three cancel no run.
//   - nightly.yml groups its runs by workflow, and cancels no run.
//
// # Options
//
// The section github of .ergon.yaml, [Options], names the runners of the jobs, the Linux runner of
// the checks of text and of CodeQL, the release of GNU make for Windows, the release of actionlint,
// the pins of the actions of the GitHub files, the limit of their jobs, the schedule of nightly.yml,
// and whether the repository has GitHub Code Security. It configures the platform alone: the
// section of each toolchain configures the setup of its jobs.
//
// # Lint of the workflows
//
// The target lint-github of the Makefile runs [Actionlint] on the workflows, and ergon tool run
// installs it as a release binary. The managed .github/actionlint.yaml contains the runners of the
// section as labels, so actionlint accepts a runner image that is newer than its release.
// lint-github turns off shellcheck and pyflakes. ergon pins neither program, so actionlint would
// take them from the machine. The targets lint and check depend on lint-github.
//
// # Contribution
//
// [Producer.Contribution] returns the jobs baseline and workflows of ci.yml, which run on the Linux
// runner as checks of text. baseline runs ergon init check, so a managed file that differs from its
// rendering fails the gate. workflows runs make lint-github.
//
// # Local files
//
// [Producer.CheckLocal] refuses a local file of dependabot.yml that adds an update of the ecosystem
// github-actions or pre-commit. ergon init renders the files of both ecosystems, so a pull request
// of Dependabot would edit a managed file, and ergon init sync would write the pins of .ergon.yaml
// back.
//
// # Errors
//
// [Producer.Data] returns an error that wraps [go.dokimi.dev/ergon/core/option.ErrInvalid] for a
// job of ci.yml or of nightly.yml whose runners the section github does not list, which names the
// job and the runner.
// [Producer.CheckLocal] returns an error that wraps [language.ErrInvalidLocal].
//
// # Dependency position
//
// Imports the standard library, go.yaml.in/yaml/v3, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. internal/cli of the
// root module imports it.
package github
