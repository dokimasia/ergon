// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// pin is the file that pins the version of Terraform of a repository.
const pin = ".terraform-version"

// steps are the steps of the gate of Terraform, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepGenerate, option.StepAudit}

// Options are the options of the section terraform of .ergon.yaml.
type Options struct {
	// Tools are the tools of the targets of Terraform.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Terraform: the release binaries of tflint, of its rulesets, of terraform-docs and of uv, and checkov, a package of PyPI that ergon tool run runs through uv."`

	// Paths are the directories of the targets.
	Paths option.Paths `yaml:"paths" doc:"The directories that the targets format, lint, test and scan, each with its subdirectories."`

	// Check are the steps of check-terraform.
	Check option.Check `yaml:"check" doc:"The steps that check-terraform runs, in order: lint, test, generate or audit."`

	// Lint are the options of lint-terraform.
	Lint Lint `yaml:"lint" doc:"lint-terraform loads the rulesets of the key rulesets beside the terraform ruleset, which tflint bundles."`

	// Test are the options of test-terraform.
	Test option.Run `yaml:"test" doc:"test-terraform runs terraform test with args in each directory that has test files in it or in its tests directory, and terraform validate in each other directory with a configuration. git ls-files lists the files of the paths, so the run skips each path that git ignores, such as .terraform."`

	// Generate are the options of generate-terraform and verify-generate-terraform.
	Generate Generate `yaml:"generate" doc:"generate-terraform runs terraform-docs when docs is true, and then each command line of commands in order at the root of the repository. verify-generate-terraform runs the same in the step generate of check, and fails when the run changes a file of the repository. Terraform renders both targets only for docs or a command line."`

	// Audit are the options of audit-terraform.
	Audit option.Audit `yaml:"audit" doc:"audit-terraform scans the configuration with checkov, and accepts each check of ignore, by its identifier, such as CKV_AWS_20."`

	// CI are the pin of the setup of Terraform, the runners, the versions and the limit of the job
	// check-terraform.
	CI option.MatrixCI[Actions] `yaml:"ci" doc:"The pin of setup-terraform, the runners and the versions of Terraform of the job check-terraform, and its limit in minutes. Empty runners select every runner of the section github, and empty versions the version of .terraform-version."`
}

// Tools are the tools of the targets of Terraform.
type Tools struct {
	// TFLint lints the modules.
	TFLint TFLint `yaml:"tflint" prefix:"tflint" doc:"The release of tflint, the linter of lint-terraform, with the rules of .tflint.hcl."`

	// AWSRuleset is the ruleset of tflint for the provider of AWS.
	AWSRuleset AWSRuleset `yaml:"tflint-ruleset-aws" prefix:"tflint-ruleset-aws" doc:"The release of tflint-ruleset-aws, which lint-terraform loads when lint.rulesets lists aws."`

	// AzureRMRuleset is the ruleset of tflint for the provider of Azure.
	AzureRMRuleset AzureRMRuleset `yaml:"tflint-ruleset-azurerm" prefix:"tflint-ruleset-azurerm" doc:"The release of tflint-ruleset-azurerm, which lint-terraform loads when lint.rulesets lists azurerm."`

	// GoogleRuleset is the ruleset of tflint for the provider of Google Cloud.
	GoogleRuleset GoogleRuleset `yaml:"tflint-ruleset-google" prefix:"tflint-ruleset-google" doc:"The release of tflint-ruleset-google, which lint-terraform loads when lint.rulesets lists google."`

	// TerraformDocs writes the tables of the inputs and the outputs of the modules.
	TerraformDocs TerraformDocs `yaml:"terraform-docs" doc:"The release of terraform-docs, which generate-terraform runs when generate.docs is true."`

	// UV runs checkov.
	UV option.UV `yaml:"uv" doc:"The release of uv, which runs checkov on Python 3.13."`

	// Checkov scans the configuration.
	Checkov option.PyPI `yaml:"checkov" doc:"The scan of audit-terraform, over the configuration."`
}

// Ruleset is a ruleset of tflint that the section pins: the name of its plugin, such as aws. The
// tool of the ruleset in the section is tflint-ruleset-<name>.
type Ruleset string

// The rulesets of tflint that the section pins.
const (
	// RulesetAWS is tflint-ruleset-aws, the ruleset of the provider of AWS.
	RulesetAWS Ruleset = "aws"

	// RulesetAzureRM is tflint-ruleset-azurerm, the ruleset of the provider of Azure.
	RulesetAzureRM Ruleset = "azurerm"

	// RulesetGoogle is tflint-ruleset-google, the ruleset of the provider of Google Cloud.
	RulesetGoogle Ruleset = "google"
)

// rulesets are the rulesets of tflint that the section pins.
var rulesets = []Ruleset{RulesetAWS, RulesetAzureRM, RulesetGoogle}

// Validate returns an error that wraps [option.ErrInvalid] for an r that is none of the rulesets
// that the section pins: aws, azurerm and google.
func (r Ruleset) Validate() error {
	if !slices.Contains(rulesets, r) {
		return fmt.Errorf("%w: ruleset %q, which is none of aws, azurerm and google", option.ErrInvalid, r)
	}
	return nil
}

// Lint are the options of lint-terraform.
type Lint struct {
	// Rulesets are the rulesets of tflint that lint-terraform loads beside the terraform ruleset.
	Rulesets []Ruleset `yaml:"rulesets" doc:"The rulesets of tflint that lint-terraform loads, from aws, azurerm and google. ergon tool run installs each from the release of its tool, such as tflint-ruleset-aws, and lint-terraform links its program into .tflint.d/plugins. tflint --init installs the plugin of each block of the local .tflint.hcl with a source and a version."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a ruleset that is not valid, as
// [Ruleset.Validate] states, and for a ruleset that rulesets lists twice.
func (l Lint) Validate() error {
	for i, r := range l.Rulesets {
		if err := r.Validate(); err != nil {
			return err
		}
		if slices.Contains(l.Rulesets[:i], r) {
			return fmt.Errorf("%w: ruleset %s, which rulesets lists twice", option.ErrInvalid, r)
		}
	}
	return nil
}

// Generate are the options of the step generate of Terraform: terraform-docs, and the command
// lines of the other generators.
type Generate struct {
	// Generate are the command lines of the other generators, which run after terraform-docs.
	option.Generate `yaml:",inline"`

	// Docs reports that generate-terraform runs terraform-docs before the command lines.
	Docs bool `yaml:"docs" doc:"When true, generate-terraform runs the terraform-docs of the section on each path before the command lines. terraform-docs reads the .terraform-docs.yml of the root of the repository, and fails without it."`
}

// CheckStep returns an error that wraps [option.ErrInvalid] when c names the step generate and g
// does not run a generator. Such a step has no target.
func (g Generate) CheckStep(c option.Check) error {
	if g.Docs {
		return nil
	}
	return g.Generate.CheckStep(c)
}

// Actions are the pins of the actions of the job check-terraform.
type Actions struct {
	// SetupTerraform installs Terraform.
	SetupTerraform workflow.Action `yaml:"setup-terraform" doc:"The action that installs Terraform, at the version of .terraform-version or of the matrix."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that Terraform does
// not have, and for the step generate without terraform-docs and without a command line. The
// command checks each option by the Validate method of its type before it calls Validate.
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
//     commands. The steps of ci.steps run after setup-terraform. The job keeps the release binaries
//     and uv, which ergon tool run installs, in the cache of GitHub Actions.
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
				Steps:    slices.Concat(install, o.CI.Steps),
				Timeout:  o.CI.Timeout,
			},
			Tools: true,
			Steps: []workflow.Step{{Name: "Check Terraform", Run: []string{"make check-terraform"}}},
		}},
		Updates: []workflow.Update{{Ecosystem: "terraform", Directories: []string{"/", "/**/*"}}},
	}
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
