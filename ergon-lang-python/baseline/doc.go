// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Python of ergon init: the configuration of ruff, the
// fragments of Python of the shared files, the section python of .ergon.yaml, and the part of
// Python of the workflows.
//
// [Producer] renders ruff.toml as a managed file from templates/managed/, and its fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile from templates/shared/.
//
// # Makefile
//
// The fragment of the Makefile runs fmt-python, lint-python, test-python and audit-python, and
// check-python, which requires the targets of the steps that the key check of the section names.
// Every tool runs through ergon tool run: uv is a release binary of the section, and the packages
// of PyPI run through it. mypy and pytest run in the environment of the project, and ruff and
// pip-audit in environments of their own. audit-python scans the packages of uv.lock, which uv
// exports to a pylock.toml, because uv audit is a preview command. With a command in the key
// generate, the fragment also renders generate-python, which runs it, and verify-generate-python,
// which fails when it changes a file. A line ##@ Python starts the group of Python in make help.
//
// # Options
//
// [Options] is the section python: the tools, the paths, the steps of the gate, the options of
// test-python, of the generators and of audit-python, and the key ci of the job check-python.
// [Options.Contribution] returns the job check-python, the CodeQL analysis of python and the
// updates of uv.lock.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
