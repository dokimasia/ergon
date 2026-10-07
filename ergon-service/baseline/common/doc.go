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
// The section common of .ergon.yaml, [Options], names the release of commitlint, the release of
// pre-commit-hooks, the pin of the action of markdownlint, and the limit of the jobs of the common
// files. commitlint is a release binary: ergon tool run installs it and checks its archive against
// the digest of the platform.
//
// # Workflows
//
// [Producer.Contribution] returns the jobs docs and commits of ci.yml. Both check text, so they run
// on the Linux runner of the section github. commits checks the commit messages of a pull request
// with the commitlint of the section, the release that the commit-msg hook of
// .pre-commit-config.yaml runs.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. internal/cli of the
// root module imports it.
package common
