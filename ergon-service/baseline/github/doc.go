// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package github is the producer of the GitHub files of ergon init: the workflows, the actions
// that install ergon and GNU make, the configuration of Dependabot, the issue forms, the template
// of a pull request, and CODEOWNERS.
//
// [Producer] renders them from the templates under templates/, whose tree mirrors the paths in the
// repository: templates/managed/.github/ for the managed files and templates/seeded/.github/ for
// CODEOWNERS. Each template ends in .tmpl, so GitHub takes no template for a workflow.
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
//     runners, as [Options.NightlyJobs] states. A repository without a nightly job has no
//     nightly.yml.
//   - version.yml opens or updates the version pull request when the run of ci.yml for a push to
//     main succeeds, from the commit of that run, through ergon release ci version. With the
//     variable ERGON_APP_CLIENT_ID and the secret ERGON_APP_PRIVATE_KEY of a GitHub App, it opens
//     the pull request with a token of the App, whose events start the runs of the checks.
//   - release.yml publishes on every push to main. The job select-mode chooses the next job. The
//     job verify lets the publish run only when a run of ci.yml passed on the content of the commit,
//     through ergon release ci verify. The jobs pack and publish then release each package of the
//     publish plan.
//   - The job version of version.yml and the job pack of release.yml run the release steps of the
//     contributions before the installation of ergon.
//   - security.yml runs the dependency review on a pull request, the OpenSSF Scorecard weekly, and
//     a job codeql-<language> for each CodeQL analysis of the contributions, which calls
//     codeql.yml.
//   - baseline.yml runs ergon init ci upgrade weekly, which moves the managed files to the newest
//     release of ergon and opens or updates the pull request of the change.
//   - dependabot.yml updates the package manager of each update of the contributions weekly, with
//     the minor and patch updates of each directory in one pull request and each major update in a
//     pull request of its own. A repository without an update has no dependabot.yml.
//
// The setup steps of a job come before the installation of ergon, so a repository that builds ergon
// from its own source builds it with the toolchain of the job. A job whose steps run tools restores
// the tool directory of ergon from the cache of GitHub Actions, and saves it after a run that
// succeeds, under a key of the system, the architecture, the job, the runtime version of its matrix
// and the digest of .ergon.yaml and the lock.
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
// the checks of text and of CodeQL, the release of GNU make for Windows, the pins of the actions of
// the GitHub files, the limit of their jobs, and the schedule of nightly.yml. It configures the
// platform alone: the section of each toolchain configures the setup of its jobs.
//
// # Contribution
//
// [Producer.Contribution] returns the job baseline of ci.yml, which runs ergon init check on the
// Linux runner as a check of text, so a managed file that differs from its rendering fails the
// gate.
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
