// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/go/baseline"
)

// setupGo is the pin of setup-go at the baseline.
var setupGo = workflow.Action{
	Uses:    "actions/setup-go",
	Commit:  "b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
	Release: "v7.0.0",
}

// requireWork is the step of the job of Go that fails a repository without go.work.
var requireWork = workflow.Step{
	Name: "Require go.work",
	If:   "hashFiles('go.work') == ''",
	Run: []string{
		`echo "::error::The targets of Go run in the modules of go.work, which the repository lacks."`,
		"exit 1",
	},
}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns nil for every step of Go", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Check = option.Check{
				option.StepLint, option.StepTest, option.StepRace, option.StepFuzz, option.StepBench,
				option.StepMutate, option.StepGenerate, option.StepAudit,
			}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for the step fmt, which formats the sources", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Check = option.Check{option.StepFmt}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns ErrInvalid for the step generate without a command", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate = option.Generate{}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run(
			"returns the job, the release steps, the analysis and the updates of Go at the baseline",
			func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, goOptions().Contribution(), workflow.Contribution{
					Jobs: []workflow.Job{{
						ID:          "check-go",
						Name:        "Go",
						Permissions: map[string]string{"contents": "read"},
						Setup: &workflow.Setup{
							Files:    "**/go.mod",
							Runners:  option.Runners{},
							Versions: []string{},
							Steps: []workflow.Step{
								requireWork,
								{
									Name: "Set up Go",
									Uses: setupGo,
									With: map[string]string{
										"go-version-file":       "go.work",
										"cache-dependency-path": "**/go.sum",
									},
								},
							},
							Timeout: 30,
						},
						Tools: true,
						Steps: []workflow.Step{{Name: "Check Go", Run: []string{"make check-go"}}},
					}},
					Release: []workflow.Step{{
						Name: "Set up Go",
						If:   "hashFiles('go.work') != ''",
						Uses: setupGo,
						With: map[string]string{"go-version-file": "go.work", "cache-dependency-path": "**/go.sum"},
					}},
					CodeQL: []workflow.CodeQL{
						{Language: "go", Name: "Go", BuildMode: "autobuild", Files: "go.work", Timeout: 30},
					},
					Updates: []workflow.Update{{Ecosystem: "gomod", Directories: []string{"/", "/**/*"}}},
				}, "the contribution")
			},
		)

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := goOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("sets up the version of the matrix for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.CI.Runners = option.Runners{"ubuntu-26.04"}
			o.CI.Versions = []string{"1.27", "1.26"}
			setup := o.Contribution().Jobs[0].Setup
			assert.Equal(t, setup.Runners, []string{"ubuntu-26.04"}, "the runners")
			assert.Equal(t, setup.Versions, []string{"1.27", "1.26"}, "the versions")
			assert.Equal(t, setup.Steps[1].With, map[string]string{
				"go-version":            "${{ matrix.version }}",
				"cache-dependency-path": "**/go.sum",
			}, "the inputs of setup-go")
		})

		t.Run("sets up the version of go.work in a release for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := goOptions()
			o.CI.Versions = []string{"1.27", "1.26"}
			release := o.Contribution().Release
			assert.Length(t, release, 1, "the release steps")
			assert.Equal(t, release[0].With, map[string]string{
				"go-version-file":       "go.work",
				"cache-dependency-path": "**/go.sum",
			}, "the inputs of setup-go")
		})
	})
}

// goOptions returns new options of the section go at the baseline.
func goOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
