// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline/github"
)

// The runners of the section github at the baseline.
var runners = option.Runners{"ubuntu-26.04", "macos-26", "windows-2025"}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, github.Producer{}.Options().Validate(), "Validate")
		})

		invalid := []struct {
			name string
			give func(*github.Options)
		}{
			{name: "returns ErrInvalid for no runner", give: func(o *github.Options) { o.Runners = nil }},
			{
				name: "returns ErrInvalid for a Linux runner that runners does not list",
				give: func(o *github.Options) { o.Linux = "ubuntu-24.04" },
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o := baselineOptions()
				tt.give(o)
				assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})

	t.Run("Jobs", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a check of text without runners", func(t *testing.T) {
			t.Parallel()
			steps := []workflow.Step{{Run: []string{"make docs"}}}
			c := workflow.Contribution{Jobs: []workflow.Job{
				{ID: "docs", Name: "Docs", Text: true, Timeout: 3, History: true, Ergon: true, Steps: steps},
			}}
			got, err := baselineOptions().Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Equal(t, got, []github.Job{
				{ID: "docs", Name: "Docs", Steps: steps, Timeout: 3, History: true, Ergon: true},
			}, "the jobs")
		})

		t.Run("returns a job without a setup on every runner", func(t *testing.T) {
			t.Parallel()
			steps := []workflow.Step{{Run: []string{"ergon init check"}}}
			c := workflow.Contribution{Jobs: []workflow.Job{
				{
					ID:          "baseline",
					Name:        "Baseline",
					If:          "always()",
					Timeout:     15,
					Permissions: read,
					Ergon:       true,
					Steps:       steps,
				},
			}}
			got, err := baselineOptions().Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Equal(t, got, []github.Job{{
				ID:          "baseline",
				Name:        "Baseline (${{ matrix.os }})",
				If:          "always()",
				Permissions: read,
				Runners:     runners,
				Steps:       steps,
				Timeout:     15,
				Ergon:       true,
			}}, "the jobs")
		})

		t.Run("returns a job with a setup on the runners and the versions of the setup", func(t *testing.T) {
			t.Parallel()
			setup := workflow.Step{Name: "Set up alpha", Uses: setupAlpha}
			check := workflow.Step{Run: []string{"make check-alpha"}}
			c := workflow.Contribution{Jobs: []workflow.Job{{
				ID:   "check-alpha",
				Name: "Alpha",
				Setup: &workflow.Setup{
					Runners:  []string{"macos-26"},
					Versions: []string{"1.0"},
					Env:      map[string]string{"A": "1"},
					Steps:    []workflow.Step{setup},
					Timeout:  20,
				},
				Steps: []workflow.Step{check},
			}}}
			got, err := baselineOptions().Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Equal(t, got, []github.Job{{
				ID:       "check-alpha",
				Name:     "Alpha (${{ matrix.os }}, ${{ matrix.version }})",
				Env:      map[string]string{"A": "1"},
				Runners:  []string{"macos-26"},
				Versions: []string{"1.0"},
				Steps:    []workflow.Step{setup, check},
				Timeout:  20,
				Make:     true,
				Ergon:    true,
			}}, "the jobs")
		})

		t.Run("returns a job whose setup lists no runner on every runner", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Jobs: []workflow.Job{{
				ID:    "check-alpha",
				Name:  "Alpha",
				Setup: &workflow.Setup{Timeout: 20},
				Steps: []workflow.Step{{Run: []string{"make check-alpha"}}},
			}}}
			got, err := baselineOptions().Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Length(t, got, 1, "the jobs")
			assert.Equal(t, got[0].Runners, []string(runners), "the runners")
			assert.Equal(t, got[0].Name, "Alpha (${{ matrix.os }})", "the name")
		})

		t.Run("guards each step of a setup that states files", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Jobs: []workflow.Job{{
				ID:   "check-alpha",
				Name: "Alpha",
				Setup: &workflow.Setup{
					Files:   "alpha.lock",
					Steps:   []workflow.Step{{Uses: setupAlpha}},
					Timeout: 20,
				},
				Steps: []workflow.Step{{If: "success()", Run: []string{"make check-alpha"}}},
			}}}
			got, err := baselineOptions().Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Length(t, got, 1, "the jobs")
			assert.Equal(t, got[0].Guard, "hashFiles('alpha.lock') != ''", "the guard")
			assert.Equal(t, got[0].Steps, []workflow.Step{
				{If: "hashFiles('alpha.lock') != ''", Uses: setupAlpha},
				{If: "hashFiles('alpha.lock') != '' && (success())", Run: []string{"make check-alpha"}},
			}, "the steps")
		})

		t.Run("leaves the contribution unchanged", func(t *testing.T) {
			t.Parallel()
			c := guarded()
			_, err := baselineOptions().Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Equal(t, c, guarded(), "the contribution")
		})

		t.Run("returns no job for a contribution without one", func(t *testing.T) {
			t.Parallel()
			got, err := baselineOptions().Jobs(&workflow.Contribution{})
			assert.NoError(t, err, "Jobs")
			assert.Empty(t, got, "the jobs")
		})

		t.Run("returns ErrInvalid for a job on a runner that runners does not list", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Jobs: []workflow.Job{{
				ID:    "check-alpha",
				Name:  "Alpha",
				Setup: &workflow.Setup{Runners: []string{"macos-26", "ubuntu-24.04"}, Timeout: 20},
				Steps: []workflow.Step{{Run: []string{"make check-alpha"}}},
			}}}
			_, err := baselineOptions().Jobs(&c)
			assert.ErrorIs(t, err, option.ErrInvalid, "Jobs")
		})
	})
}

// baselineOptions returns new options of the section github at the baseline.
func baselineOptions() *github.Options {
	o, _ := github.Producer{}.Options().(*github.Options)
	return o
}

// guarded returns a new contribution of a job whose setup states files, and whose steps have a
// condition and none.
func guarded() workflow.Contribution {
	return workflow.Contribution{Jobs: []workflow.Job{{
		ID:   "check-alpha",
		Name: "Alpha",
		Setup: &workflow.Setup{
			Files:   "alpha.lock",
			Steps:   []workflow.Step{{Uses: setupAlpha}},
			Timeout: 20,
		},
		Steps: []workflow.Step{{If: "success()", Run: []string{"make check-alpha"}}, {Run: []string{"true"}}},
	}}}
}
