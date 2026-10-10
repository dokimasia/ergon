// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"slices"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// sdk is the file that pins the .NET SDK of a repository.
const sdk = "global.json"

// steps are the steps of the gate of C#, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepGenerate, option.StepAudit}

// Options are the options of the section csharp of .ergon.yaml.
type Options struct {
	// Check are the steps of check-csharp.
	Check option.Check `yaml:"check" doc:"The steps that check-csharp runs, in order: lint, test, generate or audit."`

	// Test are the options of test-csharp.
	Test option.Run `yaml:"test" doc:"test-csharp runs dotnet test with args."`

	// Generate are the options of generate-csharp and verify-generate-csharp.
	Generate option.Generate `yaml:"generate" doc:"generate-csharp runs command with args at the root of the repository. verify-generate-csharp, the step generate of check, runs the same and fails when the run changes a file of the repository. Without a command C# has neither target."`

	// Audit are the options of audit-csharp.
	Audit option.Threshold `yaml:"audit" doc:"audit-csharp fails on a known vulnerability of a NuGet package of severity or above: low, moderate, high or critical."`

	// CI are the pin of the setup of .NET, the runners, the versions and the limit of the job
	// check-csharp.
	CI option.MatrixCI[Actions] `yaml:"ci" doc:"The pin of setup-dotnet, the runners and the versions of the .NET SDK of the job check-csharp, and its limit in minutes. Empty runners select every runner of the section github, and empty versions the SDK of global.json."`
}

// Actions are the pins of the actions of the job check-csharp.
type Actions struct {
	// SetupDotnet installs the .NET SDK.
	SetupDotnet workflow.Action `yaml:"setup-dotnet" doc:"The action that installs the .NET SDK of global.json or of the matrix."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that C# does not
// have, and for the step generate without a command. The command checks each option by the
// Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	if err := o.Check.Only(steps...); err != nil {
		return err
	}
	return o.Generate.CheckStep(o.Check)
}

// Contribution returns the part of C# of the workflows for o:
//
//   - The job check-csharp runs make check-csharp on the runners of o, once the repository has
//     global.json. setup-dotnet installs the SDK of global.json, or the version of the matrix
//     where o lists versions. The steps of ci.steps run after setup-dotnet.
//   - The CodeQL analysis of csharp reads the sources without a build.
//   - Dependabot updates the NuGet packages.
func (o *Options) Contribution() workflow.Contribution {
	with := map[string]string{"global-json-file": sdk}
	if len(o.CI.Versions) > 0 {
		with = map[string]string{"dotnet-version": "${{ matrix.version }}"}
	}
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-csharp",
			Name:        "C#",
			Permissions: map[string]string{"contents": "read"},
			Setup: &workflow.Setup{
				Files:    sdk,
				Runners:  o.CI.Runners,
				Versions: o.CI.Versions,
				Steps: slices.Concat([]workflow.Step{{Name: "Set up .NET", Uses: o.CI.Actions.SetupDotnet, With: with}},
					o.CI.Steps),
				Timeout: o.CI.Timeout,
			},
			Steps: []workflow.Step{{Name: "Check C#", Run: []string{"make check-csharp"}}},
		}},
		CodeQL: []workflow.CodeQL{
			{Language: "csharp", Name: "C#", BuildMode: "none", Files: sdk, Timeout: o.CI.Timeout},
		},
		Updates: []workflow.Update{{Ecosystem: "nuget", Directories: []string{"/"}}},
	}
}
