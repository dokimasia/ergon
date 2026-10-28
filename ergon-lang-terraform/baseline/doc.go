// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Terraform of ergon init: the configuration of tflint, the
// fragments of Terraform of the shared files, the section terraform of .ergon.yaml, and the part of
// Terraform of the workflows.
//
// [Producer] renders .tflint.hcl as a managed file from templates/managed/, and its fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile from templates/shared/. .tflint.hcl
// enables the terraform ruleset, which tflint bundles, and each ruleset of the key lint.rulesets.
//
// # Makefile
//
// The fragment of the Makefile runs fmt-terraform, lint-terraform, test-terraform and
// audit-terraform over the directories of the section, and check-terraform, which requires the
// targets of the steps that the key check of the section names. The variable TERRAFORM is the
// terraform command. tflint, its rulesets, terraform-docs and uv are release binaries of the
// section, and checkov runs through uv on Python 3.13. lint-terraform links the program of each
// ruleset of lint.rulesets into .tflint.d/plugins, which the fragment of .gitignore ignores, and
// runs tflint --init and tflint with that plugin directory. test-terraform lists the files of the
// directories with git ls-files. It runs terraform test in each directory with test files in it or
// in its tests directory, and terraform validate in each other directory with a configuration.
// audit-terraform fails on a file that checkov cannot parse. With docs or command lines in the key
// generate, the fragment also renders generate-terraform, which runs terraform-docs and then the
// command lines in order, and verify-generate-terraform, which fails when they change a file. A
// line ##@ Terraform starts the group of Terraform in make help.
//
// # Options
//
// [Options] is the section terraform. It has the tools, the paths and the steps of the gate, the
// rulesets of lint-terraform in [Lint], the options of test-terraform, of the generators in
// [Generate] and of audit-terraform, and the key ci of the job check-terraform. The Asset methods
// of [TFLint], [AWSRuleset], [AzureRMRuleset], [GoogleRuleset] and [TerraformDocs] return where
// their releases publish the asset of each platform. [Options.Contribution] returns the job
// check-terraform and the updates of the providers and the modules.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
