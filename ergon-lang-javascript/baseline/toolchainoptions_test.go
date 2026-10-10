// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
)

// setupNode is the pin of setup-node at the baseline.
var setupNode = workflow.Action{
	Uses:    "actions/setup-node",
	Commit:  "820762786026740c76f36085b0efc47a31fe5020",
	Release: "v7.0.0",
}

// installTypeScript is a step of ci.steps, which installs a compiler that the tests run.
var installTypeScript = workflow.Step{Name: "Install TypeScript", Run: []string{"npm install --global typescript"}}

func TestToolchainOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Toolchain{}.Options().Validate(), "Validate")
		})

		t.Run("returns nil for options without a value", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, new(baseline.ToolchainOptions).Validate(), "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the setup, the analysis and the updates of the js toolchain at the baseline",
			func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, jsOptions().Contribution(), workflow.Contribution{
					Setup: &workflow.Setup{
						Files:    "package.json",
						Runners:  option.Runners{},
						Versions: []string{},
						Steps: []workflow.Step{
							{
								Name: "Set up Node.js",
								Uses: setupNode,
								With: map[string]string{"node-version-file": "package.json"},
							},
							{Name: "Install the packages", Run: []string{"npm ci"}},
						},
						Timeout: 30,
					},
					CodeQL: []workflow.CodeQL{{
						Language:  "javascript-typescript",
						Name:      "JavaScript and TypeScript",
						BuildMode: "none",
						Files:     "package.json",
						Timeout:   30,
					}},
					Updates: []workflow.Update{{Ecosystem: "npm", Directories: []string{"/"}}},
				}, "the contribution")
			})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := jsOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("adds the steps of ci.steps to the end of the setup", func(t *testing.T) {
			t.Parallel()
			o := jsOptions()
			o.CI.Steps = []workflow.Step{installTypeScript}
			want := slices.Concat(jsOptions().Contribution().Setup.Steps, o.CI.Steps)
			assert.Equal(t, o.Contribution().Setup.Steps, want, "the steps of the setup")
		})

		t.Run("installs the Node.js of the matrix for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := jsOptions()
			o.CI.Versions = []string{"24", "26"}
			setup := o.Contribution().Setup
			assert.Equal(t, setup.Steps[0].With, map[string]string{"node-version": "${{ matrix.version }}"},
				"the inputs of setup-node")
		})
	})
}
