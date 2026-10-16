// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package common is the producer of the common files of ergon init: the files of a repository that
// do not depend on its languages or its forge.
//
// [Producer] renders them from the templates under templates/, whose tree mirrors the paths in the
// repository: templates/managed/ for the managed files, templates/seeded/ for the seeded files,
// and templates/shared/ for the first fragments of .editorconfig, .gitattributes, .gitignore and
// the Makefile, to which each language appends its own. Each template ends in .tmpl, so neither git
// nor an editor takes a template such as .gitignore for its own configuration.
//
// # Options
//
// The section common of .ergon.yaml, [Options], names the command of ergon, the releases of
// commitlint and rumdl, the release of pre-commit-hooks, the targets of the hooks, and the limit of
// the jobs of the common files. commitlint and rumdl are release binaries: ergon tool run installs
// each one and checks its archive against the digest of the platform.
//
// The Makefile and the commit-msg hook run ergon with the [Command] of the key ergon: the ergon on
// the PATH at the baseline. The repository of ergon runs the ergon of its own source with go run.
//
// # Markdown
//
// The target lint-common of the Makefile runs [Rumdl] with the rules of the managed .markdownlint.yml
// on the Markdown files of the repository. rumdl does not lint a file that .gitignore or the managed
// .markdownlintignore ignores. The managed .markdownlintignore ignores the test data. The targets
// lint and check depend on lint-common.
//
// # Hooks
//
// [Hooks] lists the aggregate targets of the Makefile, each a [Target], that the hooks of
// .pre-commit-config.yaml run before each commit and before each push. The template renders one
// hook for each target of each stage, whose id is the target, in the order of the list. A stage
// with no target has no hook.
//
// The hooks of pre-commit-hooks that change or refuse the content of a file are trailing-whitespace,
// end-of-file-fixer, mixed-line-ending and check-added-large-files. The template writes the regular
// expression of [Hooks.Exclude] as the key exclude of each of them, so a repository keeps files
// such as the copies of a model checkpoint byte for byte. trailing-whitespace skips the Markdown
// files by their type, so its key exclude has no pattern of its own to join.
//
// # Workflows
//
// [Producer.Contribution] returns the jobs docs, commits and changeset of ci.yml. They check text,
// so they run on the Linux runner of the section github. docs runs make lint-common, as make lint
// does in the hooks. commits checks the commit messages of a pull request with the commitlint of
// the section, the release that the commit-msg hook of .pre-commit-config.yaml runs. changeset runs
// ergon release status against the base of a pull request, with the configuration
// .changeset/config.json, which the common files seed. commits and changeset skip the pull requests
// of Dependabot, and changeset also skips the version pull request of the repository.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option], [go.dokimi.dev/ergon/core/workflow] and
// [go.dokimi.dev/ergon/service/release], for the branch of the version pull request. internal/cli
// of the root module imports it.
package common
