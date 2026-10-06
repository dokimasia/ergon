// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package github renders the GitHub files of ergon init: the workflows, the actions that install
// ergon and GNU make, the configuration of Dependabot, the issue forms, the template of a pull
// request, and CODEOWNERS.
//
// [Initializer] renders them from the templates under templates/, whose tree mirrors the paths
// under .github in the repository: templates/managed/ for the managed files and templates/seeded/
// for CODEOWNERS. Each template ends in .tmpl, so GitHub does not take a template for a workflow.
//
// # Workflows
//
//   - ci.yml is the gate on every pull request, on the merge queue and on main: Markdown lint, the
//     commit messages of a pull request, ergon init check, and the job check-<language> of each
//     language.
//   - security.yml runs the dependency review on a pull request, the OpenSSF Scorecard weekly, and
//     the CodeQL job of each language that CodeQL analyzes, which calls codeql.yml.
//   - baseline.yml checks the managed files against the newest release of ergon weekly, and opens
//     an issue when they are outdated.
//
// A job that builds, tests or runs ergon runs on the matrix [language.Runners] of Linux, macOS and
// Windows. A job that checks text or analyzes the sources runs on [language.Linux]. Each runner
// is pinned to a version of its system. The action setup-make installs GNU make on Windows, whose
// image has none, so a job runs make in bash on every system.
//
// Every workflow grants no permission at its top level, and each job grants only the scopes that
// it needs. Every action is pinned to the commit of a release, with the release in a comment.
// actions/checkout runs with persist-credentials set to false. A workflow that an event triggers
// groups its runs by workflow and ref, and cancels a superseded run on a pull request only.
//
// # Shared files
//
// ci.yml, security.yml and dependabot.yml are shared: Initializer renders their first fragment,
// and each language appends its jobs or its package manager.
//
// # Errors
//
// [Initializer.Files] returns an error that wraps [ErrInvalidAnswer] for a repository that is not
// owner/name.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language] and
// [go.dokimi.dev/ergon/service/baseline/common], for the release of commitlint. internal/cli of
// the root module imports it.
package github
