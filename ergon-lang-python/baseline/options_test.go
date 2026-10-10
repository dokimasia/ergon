// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/python/baseline"
)

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

		t.Run("returns ErrInvalid for a step that Python does not have", func(t *testing.T) {
			t.Parallel()
			o := pythonOptions()
			o.Check = option.Check{option.StepRace}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns nil for the step generate with a command", func(t *testing.T) {
			t.Parallel()
			o := pythonOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate.Command = []string{"datamodel-codegen"}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for the step generate without a command", func(t *testing.T) {
			t.Parallel()
			o := pythonOptions()
			o.Check = option.Check{option.StepGenerate}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job, the analysis and the updates of Python at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, pythonOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-python",
					Name:        "Python",
					Permissions: map[string]string{"contents": "read"},
					Setup: &workflow.Setup{
						Files:    "pyproject.toml",
						Runners:  option.Runners{},
						Versions: []string{},
						Steps:    []workflow.Step{},
						Timeout:  30,
					},
					Tools: true,
					Steps: []workflow.Step{{Name: "Check Python", Run: []string{"make check-python"}}},
				}},
				CodeQL: []workflow.CodeQL{
					{Language: "python", Name: "Python", BuildMode: "none", Files: "pyproject.toml", Timeout: 30},
				},
				Updates: []workflow.Update{{Ecosystem: "uv", Directories: []string{"/"}}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := pythonOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("adds the steps of ci.steps to the setup", func(t *testing.T) {
			t.Parallel()
			o := pythonOptions()
			o.CI.Steps = []workflow.Step{installTypeScript}
			assert.Equal(t, o.Contribution().Jobs[0].Setup.Steps, []workflow.Step{installTypeScript},
				"the steps of the setup")
		})

		t.Run("installs the version of the matrix through UV_PYTHON for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := pythonOptions()
			o.CI.Versions = []string{"3.13", "3.14"}
			setup := o.Contribution().Jobs[0].Setup
			assert.Equal(t, setup.Versions, []string{"3.13", "3.14"}, "the versions")
			assert.Equal(t, setup.Env, map[string]string{"UV_PYTHON": "${{ matrix.version }}"}, "the environment")
		})
	})
}

// pythonOptions returns new options of the section python at the baseline.
func pythonOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
