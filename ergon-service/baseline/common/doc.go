// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package common renders the common files of ergon init: the files of a repository that do not
// depend on its languages or its forge.
//
// [Initializer] renders them from the templates under templates/, whose tree mirrors the paths in
// the repository: templates/managed/ for the managed files, templates/seeded/ for the seeded
// files, and templates/licenses/ for the license texts. Each template ends in .tmpl, so neither
// git nor an editor takes a template such as .gitignore for its own configuration.
//
// # Shared files
//
// .editorconfig, .gitattributes, .gitignore and the Makefile are shared: Initializer renders
// their first fragment, and each language appends its own.
//
// # Answers
//
// A template names an answer as {{name}}, {{owner}}, {{license}}, {{year}}, {{repository}} or
// {{security-contact}}, and the release of commitlint as {{commitlint}}. [Initializer.Files]
// returns an error that wraps [ErrInvalidAnswer] for an answer that a file cannot render.
//
// # Commit messages
//
// .commitlint.yaml configures [Commitlint], which the commit-msg hook of .pre-commit-config.yaml
// runs on each commit message. The workflow of the gate runs the same release on the commits of a
// pull request.
//
// # Dependency position
//
// Imports the standard library and [go.dokimi.dev/ergon/core/language]. internal/cli of the root
// module imports it.
package common
