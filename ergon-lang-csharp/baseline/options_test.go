// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
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
