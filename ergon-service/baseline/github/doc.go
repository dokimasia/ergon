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
//   - ci.yml is the gate on every pull request, on the merge queue and on main, and the workflow
//     that release.yml calls before a publish. It renders each job of the contributions from one
//     skeleton: the checkout, the installation of GNU make, the setup steps, the installation of
//     ergon, the cache of the tools of ergon for a job that runs tools, and the steps of the job,
//     as [Options.Jobs] states.
//   - release.yml runs the release flow of ergon release ci on every push to main. The job
//     select-mode chooses the next job. The job version opens or updates the version pull request.
//     After its merge, the jobs ci, pack and publish release each package of the publish plan. The
//     jobs version and pack run the release steps of the contributions before the installation of
//     ergon.
//   - security.yml runs the dependency review on a pull request, the OpenSSF Scorecard weekly, and
//     a job codeql-<language> for each CodeQL analysis of the contributions, which calls
//     codeql.yml.
//   - baseline.yml checks the managed files against the newest release of ergon weekly, and opens
//     an issue when they are outdated.
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
// actions/checkout runs with persist-credentials set to false. A workflow that a push or a pull
// request triggers groups its runs by workflow and ref, and cancels a superseded run on a pull
// request only. The group of ci.yml starts with ci, so a run that release.yml calls, whose workflow
// is the caller's, does not share the group of the caller.
//
// # Options
//
// The section github of .ergon.yaml, [Options], names the runners of the jobs, the Linux runner of
// the checks of text and of CodeQL, the release of GNU make for Windows, the pins of the actions of
// the GitHub files, and the limit of their jobs. It configures the platform alone: the section of
// each toolchain configures the setup of its jobs.
//
// # Contribution
//
// [Producer.Contribution] returns the job baseline of ci.yml, which runs ergon init check on the
// Linux runner as a check of text, so a managed file that differs from its rendering fails the
// gate.
//
// # Errors
//
// [Producer.Data] returns an error that wraps [go.dokimi.dev/ergon/core/option.ErrInvalid] for a
// job whose runners the section github does not list, which names the job and the runner.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. internal/cli of the
// root module imports it.
package github
