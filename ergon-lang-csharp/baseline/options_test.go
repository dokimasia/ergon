// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/csharp/baseline"
)

// setupDotnet is the pin of setup-dotnet at the baseline.
var setupDotnet = workflow.Action{
	Uses:    "actions/setup-dotnet",
	Commit:  "a98b56852c35b8e3190ac28c8c2271da59106c68",
	Release: "v6.0.0",
}

// installTypeScript is a step of ci.steps, which installs a compiler that the tests run.
var installTypeScript = workflow.Step{Name: "Install TypeScript", Run: []string{"npm install --global typescript"}}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step that C# does not have", func(t *testing.T) {
			t.Parallel()
			o := csharpOptions()
			o.Check = option.Check{option.StepFuzz}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns nil for the step generate with a command", func(t *testing.T) {
			t.Parallel()
			o := csharpOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate.Command = []string{"dotnet", "tool", "run", "nswag"}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for the step generate without a command", func(t *testing.T) {
			t.Parallel()
			o := csharpOptions()
			o.Check = option.Check{option.StepGenerate}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job, the analysis and the updates of C# at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, csharpOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-csharp",
					Name:        "C#",
					Permissions: map[string]string{"contents": "read"},
					Setup: &workflow.Setup{
						Files:    "global.json",
						Runners:  option.Runners{},
						Versions: []string{},
						Steps: []workflow.Step{{
							Name: "Set up .NET",
							Uses: setupDotnet,
							With: map[string]string{"global-json-file": "global.json"},
						}},
						Timeout: 30,
					},
					Steps: []workflow.Step{{Name: "Check C#", Run: []string{"make check-csharp"}}},
				}},
				CodeQL: []workflow.CodeQL{
					{Language: "csharp", Name: "C#", BuildMode: "none", Files: "global.json", Timeout: 30},
				},
				Updates: []workflow.Update{{Ecosystem: "nuget", Directories: []string{"/"}}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := csharpOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("adds the steps of ci.steps to the end of the setup", func(t *testing.T) {
			t.Parallel()
			o := csharpOptions()
			o.CI.Steps = []workflow.Step{installTypeScript}
			want := slices.Concat(csharpOptions().Contribution().Jobs[0].Setup.Steps, o.CI.Steps)
			assert.Equal(t, o.Contribution().Jobs[0].Setup.Steps, want, "the steps of the setup")
		})

		t.Run("installs the SDK of the matrix for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := csharpOptions()
			o.CI.Versions = []string{"10.0.x", "11.0.x"}
			setup := o.Contribution().Jobs[0].Setup
			assert.Equal(t, setup.Steps[0].With, map[string]string{"dotnet-version": "${{ matrix.version }}"},
				"the inputs of setup-dotnet")
		})
	})
}

// csharpOptions returns new options of the section csharp at the baseline.
func csharpOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
