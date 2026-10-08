// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// pin is the file that pins the version of Terraform of a repository.
const pin = ".terraform-version"

// windows is the system of Windows, whose programs end in .exe.
const windows = "windows"

// steps are the steps of the gate of Terraform, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepGenerate, option.StepAudit}

// Options are the options of the section terraform of .ergon.yaml.
type Options struct {
	// Tools are the tools of the targets of Terraform.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Terraform: the release binaries of tflint and uv, and checkov, a package of PyPI that ergon tool run runs through uv."`

	// Paths are the directories of the targets.
	Paths option.Paths `yaml:"paths" doc:"The directories that the targets format, lint and scan, each with its subdirectories."`

	// Check are the steps of check-terraform.
	Check option.Check `yaml:"check" doc:"The steps that check-terraform runs, in order: lint, test, generate or audit."`

	// Test are the options of test-terraform.
	Test option.Run `yaml:"test" doc:"test-terraform runs terraform test with args."`

	// Generate are the options of generate-terraform and verify-generate-terraform.
	Generate option.Generate `yaml:"generate" doc:"generate-terraform runs command with args at the root of the repository, such as terraform-docs. verify-generate-terraform, the step generate of check, runs the same and fails when the run changes a file of the repository. Without a command Terraform has neither target."`

	// Audit are the options of audit-terraform.
	Audit option.Audit `yaml:"audit" doc:"audit-terraform scans the configuration with checkov, and accepts each check of ignore, by its identifier, such as CKV_AWS_20."`

	// CI are the pin of the setup of Terraform, the runners, the versions and the limit of the job
	// check-terraform.
	CI option.MatrixCI[Actions] `yaml:"ci" doc:"The pin of setup-terraform, the runners and the versions of Terraform of the job check-terraform, and its limit in minutes. Empty runners select every runner of the section github, and empty versions the version of .terraform-version."`
}

// Tools are the tools of the targets of Terraform.
type Tools struct {
	// TFLint lints the modules.
	TFLint TFLint `yaml:"tflint" doc:"The release of tflint, the linter of lint-terraform, with the rules of .tflint.hcl."`

	// UV runs checkov.
	UV option.UV `yaml:"uv" doc:"The release of uv, which runs checkov on Python 3.13."`

	// Checkov scans the configuration.
	Checkov option.PyPI `yaml:"checkov" doc:"The scan of audit-terraform, over the configuration."`
}

// TFLint is the release binary of tflint, terraform-linters/tflint on GitHub, whose release
// publishes a .zip for every platform.
type TFLint struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = TFLint{}

// tflintRepository is the repository on GitHub whose releases publish tflint.
const tflintRepository = "terraform-linters/tflint"

// Actions are the pins of the actions of the job check-terraform.
type Actions struct {
	// SetupTerraform installs Terraform.
	SetupTerraform workflow.Action `yaml:"setup-terraform" doc:"The action that installs Terraform, at the version of .terraform-version or of the matrix."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that Terraform does
// not have, and for the step generate without a command. The command checks each option by the
// Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	if err := o.Check.Only(steps...); err != nil {
		return err
	}
	return o.Generate.CheckStep(o.Check)
}

// Contribution returns the part of Terraform of the workflows for o:
//
//   - The job check-terraform runs make check-terraform on the runners of o, once the repository has
//     .terraform-version. setup-terraform installs the version of .terraform-version, which a step
//     reads, or the version of the matrix where o lists versions, without its wrapper of the
//     commands. The job keeps tflint and uv, which ergon tool run installs, in the cache of GitHub
//     Actions.
//   - Dependabot updates the providers and the modules of every directory.
//
// CodeQL analyzes no Terraform.
func (o *Options) Contribution() workflow.Contribution {
	install := []workflow.Step{
		{Name: "Read " + pin, ID: "pin", Run: []string{`echo "version=$(cat ` + pin + `)" >> "$GITHUB_OUTPUT"`}},
		setup(o.CI.Actions.SetupTerraform, "${{ steps.pin.outputs.version }}"),
	}
	if len(o.CI.Versions) > 0 {
		install = []workflow.Step{setup(o.CI.Actions.SetupTerraform, "${{ matrix.version }}")}
	}
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-terraform",
			Name:        "Terraform",
			Permissions: map[string]string{"contents": "read"},
			Setup: &workflow.Setup{
				Files:    pin,
				Runners:  o.CI.Runners,
				Versions: o.CI.Versions,
				Steps:    install,
				Timeout:  o.CI.Timeout,
			},
			Tools: true,
			Steps: []workflow.Step{{Name: "Check Terraform", Run: []string{"make check-terraform"}}},
		}},
		Updates: []workflow.Update{{Ecosystem: "terraform", Directories: []string{"/", "/**/*"}}},
	}
}

// Asset returns the archive of tflint for p from the release of the version of t on GitHub:
// tflint_<os>_<arch>.zip, with the program at its root, and tflint.exe on Windows. It returns an
// error that wraps [option.ErrNoAsset] for a platform that is not valid.
func (t TFLint) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil {
		return option.Asset{}, fmt.Errorf("%w: tflint has no asset for %q", option.ErrNoAsset, p)
	}
	program := "tflint"
	if p.OS() == windows {
		program += ".exe"
	}
	return option.Asset{
		URL: "https://github.com/" + tflintRepository + "/releases/download/v" + t.Version + "/tflint_" + p.OS() +
			"_" + p.Arch() + ".zip",
		Program: program,
	}, nil
}

// Repository returns terraform-linters/tflint, the repository on GitHub whose releases publish
// tflint.
func (TFLint) Repository() string {
	return tflintRepository
}

// setup returns the step that installs the version of Terraform with the action, without the
// wrapper that the action puts around the commands of terraform.
func setup(action workflow.Action, version string) workflow.Step {
	return workflow.Step{
		Name: "Set up Terraform",
		Uses: action,
		With: map[string]string{"terraform_version": version, "terraform_wrapper": "false"},
	}
}
