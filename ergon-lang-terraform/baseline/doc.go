// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Terraform of ergon init: the configuration of tflint, the
// fragments of Terraform of the shared files, the section terraform of .ergon.yaml, and the part of
// Terraform of the workflows.
//
// [Producer] renders .tflint.hcl as a managed file from templates/managed/, and its fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile from templates/shared/.
//
// # Makefile
//
// The fragment of the Makefile runs fmt-terraform, lint-terraform, test-terraform and
// audit-terraform over the directories of the section, and check-terraform, which requires the
// targets of the steps that the key check of the section names. tflint and uv are release
// binaries of the section, and checkov runs through uv on Python 3.13. audit-terraform fails on a
// file that checkov cannot parse.
//
// # Options
//
// [Options] is the section terraform: the tools, the paths, the steps of the gate, the options of
// test-terraform and audit-terraform, and the key ci of the job check-terraform. [TFLint] states
// where the release of tflint publishes the asset of each platform. [Options.Contribution] returns
// the job check-terraform and the updates of the providers and the modules.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
