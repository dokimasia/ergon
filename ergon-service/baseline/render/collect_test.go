// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package render_test

import (
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// setupJava is the action of the setup of Java in the cases.
var setupJava = workflow.Action{
	Uses:    "actions/setup-java",
	Commit:  "de7274f081f381c8f8158605e0321c36c376e2e6",
	Release: "v6.0.1",
}

// contributor is a producer of the cases with a part of the workflows.
type contributor struct {
	fixture

	// part is the contribution that Contribution returns.
	part workflow.Contribution
}

// Contribution returns c.part, whatever the options.
func (c contributor) Contribution(language.Options) workflow.Contribution {
	return c.part
}

func TestCollect(t *testing.T) {
	t.Parallel()

	t.Run("Collect", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the parts of the units in their order", func(t *testing.T) {
			t.Parallel()
			got, err := render.Collect([]render.Unit{
				contributing("common", workflow.Contribution{Jobs: []workflow.Job{text("docs")}}),
				unit("plain", fstest.MapFS{}),
				contributing("go", workflow.Contribution{
					Jobs:    []workflow.Job{text("check-go")},
					Release: []workflow.Step{{Run: []string{"go version"}}},
					CodeQL:  []workflow.CodeQL{analysis("go")},
					Updates: []workflow.Update{{Ecosystem: "gomod", Directories: []string{"/"}}},
				}),
				contributing("java", workflow.Contribution{Release: []workflow.Step{{Uses: setupJava}}}),
			})
			assert.NoError(t, err, "Collect")
			assert.Equal(t, got, workflow.Contribution{
				Jobs:    []workflow.Job{text("docs"), text("check-go")},
				Release: []workflow.Step{{Run: []string{"go version"}}, {Uses: setupJava}},
				CodeQL:  []workflow.CodeQL{analysis("go")},
				Updates: []workflow.Update{{Ecosystem: "gomod", Directories: []string{"/"}}},
			}, "the contributions")
		})

		t.Run("sets the setup of the toolchain that a job names", func(t *testing.T) {
			t.Parallel()
			setup := &workflow.Setup{Files: ".java-version", Timeout: 30, Steps: []workflow.Step{{Uses: setupJava}}}
			job := workflow.Job{
				ID:        "check-java",
				Name:      "Java",
				Toolchain: "jvm",
				Steps:     []workflow.Step{{Run: []string{"make check-java"}}},
			}
			got, err := render.Collect([]render.Unit{
				contributing("jvm", workflow.Contribution{Setup: setup}),
				contributing("java", workflow.Contribution{Jobs: []workflow.Job{job}}),
			})
			assert.NoError(t, err, "Collect")
			assert.Length(t, got.Jobs, 1, "the jobs")
			assert.Equal(t, got.Jobs[0].Setup, setup, "the setup of check-java", assert.ByIdentity())
			assert.Nil(t, got.Setup, "the setup of the contributions")
		})

		invalid := []struct {
			name  string
			units []render.Unit
		}{
			{
				name:  "returns ErrInvalidContribution for a part that is not valid",
				units: []render.Unit{contributing("go", workflow.Contribution{Jobs: []workflow.Job{{ID: "check-go"}}})},
			},
			{
				name: "returns ErrInvalidContribution for a toolchain without a setup",
				units: []render.Unit{contributing("java", workflow.Contribution{Jobs: []workflow.Job{
					{
						ID:        "check-java",
						Name:      "Java",
						Toolchain: "jvm",
						Steps:     []workflow.Step{{Run: []string{"make check-java"}}},
					},
				}})},
			},
			{
				name: "returns ErrInvalidContribution for a job that two parts declare",
				units: []render.Unit{
					contributing("common", workflow.Contribution{Jobs: []workflow.Job{text("docs")}}),
					contributing("other", workflow.Contribution{Jobs: []workflow.Job{text("docs")}}),
				},
			},
			{
				name: "returns ErrInvalidContribution for an analysis that two parts declare",
				units: []render.Unit{
					contributing("go", workflow.Contribution{CodeQL: []workflow.CodeQL{analysis("go")}}),
					contributing("other", workflow.Contribution{CodeQL: []workflow.CodeQL{analysis("go")}}),
				},
			},
			{
				name: "returns ErrInvalidContribution for the updates of a directory that two parts declare",
				units: []render.Unit{
					contributing(
						"go",
						workflow.Contribution{
							Updates: []workflow.Update{{Ecosystem: "gomod", Directories: []string{"/"}}},
						},
					),
					contributing("other", workflow.Contribution{Updates: []workflow.Update{
						{Ecosystem: "gomod", Directories: []string{"/tools", "/"}},
					}}),
				},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := render.Collect(tt.units)
				assert.ErrorIs(t, err, render.ErrInvalidContribution, "Collect")
			})
		}
	})
}

// contributing returns a unit of the cases that contributes part.
func contributing(name string, part workflow.Contribution) render.Unit {
	return render.Unit{Name: name, Producer: contributor{fixture: fixture{templates: fstest.MapFS{}}, part: part}}
}

// text returns a check of text of the cases with the id id.
func text(id string) workflow.Job {
	return workflow.Job{ID: id, Name: id, Text: true, Timeout: 10, Steps: []workflow.Step{{Run: []string{"true"}}}}
}

// analysis returns a CodeQL analysis of the cases of the language language.
func analysis(language string) workflow.CodeQL {
	return workflow.CodeQL{Language: language, Name: language, BuildMode: "none", Files: "go.work", Timeout: 30}
}
