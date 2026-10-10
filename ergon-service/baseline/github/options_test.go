// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline/github"
)

// The runners of the section github at the baseline.
var runners = option.Runners{"ubuntu-26.04", "macos-26", "windows-2025"}

// The step that keeps the tools of ergon in the cache: its name, the tool directory of ergon on each
// system, and the key of a job without and with the runtime versions of a matrix.
const (
	toolsName       = "Keep the tools of ergon"
	toolsPath       = "${{ runner.os == 'Windows' && '~/AppData/Local/ergon/tools' || runner.os == 'macOS' && '~/Library/Caches/ergon/tools' || '~/.cache/ergon/tools' }}"
	toolsKey        = "ergon-tools-${{ runner.os }}-${{ runner.arch }}-${{ github.job }}-${{ hashFiles('.ergon.yaml', '.ergon/init.lock') }}"
	toolsVersionKey = "ergon-tools-${{ runner.os }}-${{ runner.arch }}-${{ github.job }}-${{ matrix.version }}-${{ hashFiles('.ergon.yaml', '.ergon/init.lock') }}"
)

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

	t.Run("Nightly", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give string
			}{
				{name: "returns nil for the schedule at the baseline", give: baselineOptions().Nightly.Schedule},
				{name: "returns nil for steps, ranges, lists and names", give: "*/30 1-5 1,15 * MON-FRI"},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, github.Nightly{Schedule: tt.give}.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give string
			}{
				{name: "returns ErrInvalid for four fields", give: "0 3 * *"},
				{name: "returns ErrInvalid for six fields", give: "0 0 3 * * *"},
				{name: "returns ErrInvalid for a field with a character of no cron field", give: "0 3 * * $DAY"},
				{name: "returns ErrInvalid for an empty schedule", give: ""},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					err := github.Nightly{Schedule: tt.give}.Validate()
					assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
					assert.Contains(t, err.Error(), "nightly.schedule", "the error")
				})
			}
		})
	})

	t.Run("NightlyJobs", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a job whose setup lists no runner on the Linux runner without a matrix", func(t *testing.T) {
			t.Parallel()
			fuzz := workflow.Step{Run: []string{"make fuzz-alpha"}}
			c := workflow.Contribution{Nightly: []workflow.Job{{
				ID:    "fuzz-alpha",
				Name:  "Fuzz alpha",
				Setup: &workflow.Setup{Timeout: 120},
				Steps: []workflow.Step{fuzz},
			}}}
			got, err := baselineOptions().NightlyJobs(&c)
			assert.NoError(t, err, "NightlyJobs")
			assert.Equal(t, got, []github.Job{{
				ID:      "fuzz-alpha",
				Name:    "Fuzz alpha",
				Steps:   []workflow.Step{fuzz},
				Timeout: 120,
				Make:    true,
				Ergon:   true,
			}}, "the jobs")
		})

		t.Run("returns a job without a setup on the Linux runner without a matrix", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Nightly: []workflow.Job{{
				ID:      "audit",
				Name:    "Audit",
				Timeout: 10,
				Ergon:   true,
				Steps:   []workflow.Step{{Run: []string{"ergon audit"}}},
			}}}
			got, err := baselineOptions().NightlyJobs(&c)
			assert.NoError(t, err, "NightlyJobs")
			assert.Length(t, got, 1, "the jobs")
			expect.Nil(t, got[0].Runners, "the runners")
			expect.Equal(t, got[0].Name, "Audit", "the name")
		})

		t.Run("returns a job whose setup lists versions on a matrix of the Linux runner", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Nightly: []workflow.Job{{
				ID:    "fuzz-alpha",
				Name:  "Fuzz alpha",
				Setup: &workflow.Setup{Versions: []string{"1.0", "2.0"}, Timeout: 120},
				Steps: []workflow.Step{{Run: []string{"make fuzz-alpha"}}},
			}}}
			got, err := baselineOptions().NightlyJobs(&c)
			assert.NoError(t, err, "NightlyJobs")
			assert.Length(t, got, 1, "the jobs")
			expect.Equal(t, got[0].Runners, []string{"ubuntu-26.04"}, "the runners")
			expect.Equal(t, got[0].Name, "Fuzz alpha (${{ matrix.os }}, ${{ matrix.version }})", "the name")
		})

		t.Run("returns a job whose setup lists runners on those runners", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Nightly: []workflow.Job{{
				ID:    "fuzz-alpha",
				Name:  "Fuzz alpha",
				Setup: &workflow.Setup{Runners: []string{"macos-26", "windows-2025"}, Timeout: 120},
				Steps: []workflow.Step{{Run: []string{"make fuzz-alpha"}}},
			}}}
			got, err := baselineOptions().NightlyJobs(&c)
			assert.NoError(t, err, "NightlyJobs")
			assert.Length(t, got, 1, "the jobs")
			expect.Equal(t, got[0].Runners, []string{"macos-26", "windows-2025"}, "the runners")
			expect.Equal(t, got[0].Name, "Fuzz alpha (${{ matrix.os }})", "the name")
		})

		t.Run("returns no job for a contribution without a nightly job", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Jobs: guarded().Jobs}
			got, err := baselineOptions().NightlyJobs(&c)
			assert.NoError(t, err, "NightlyJobs")
			assert.Empty(t, got, "the jobs")
		})

		t.Run("returns ErrInvalid for a job on a runner that runners does not list", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Nightly: []workflow.Job{{
				ID:    "fuzz-alpha",
				Name:  "Fuzz alpha",
				Setup: &workflow.Setup{Runners: []string{"ubuntu-24.04"}, Timeout: 120},
				Steps: []workflow.Step{{Run: []string{"make fuzz-alpha"}}},
			}}}
			_, err := baselineOptions().NightlyJobs(&c)
			assert.ErrorIs(t, err, option.ErrInvalid, "NightlyJobs")
		})
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
				Setup:    []workflow.Step{setup},
				Steps:    []workflow.Step{check},
				Timeout:  20,
				Make:     true,
				Ergon:    true,
			}}, "the jobs")
		})

		t.Run("keeps the tools of ergon in the cache before the steps of a job that runs tools", func(t *testing.T) {
			t.Parallel()
			check := workflow.Step{Run: []string{"make check-alpha"}}
			c := workflow.Contribution{Jobs: []workflow.Job{{
				ID:    "check-alpha",
				Name:  "Alpha",
				Setup: &workflow.Setup{Timeout: 20},
				Tools: true,
				Steps: []workflow.Step{check},
			}}}
			o := baselineOptions()
			got, err := o.Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Length(t, got, 1, "the jobs")
			assert.Equal(t, got[0].Steps, []workflow.Step{
				{
					Name: toolsName,
					Uses: o.CI.Actions.Cache,
					With: map[string]string{"path": toolsPath, "key": toolsKey},
				},
				check,
			}, "the steps")
		})

		t.Run("keys the cache of the tools of ergon by the runtime version of a matrix", func(t *testing.T) {
			t.Parallel()
			c := workflow.Contribution{Jobs: []workflow.Job{{
				ID:    "check-alpha",
				Name:  "Alpha",
				Setup: &workflow.Setup{Versions: []string{"1.0", "2.0"}, Timeout: 20},
				Tools: true,
				Steps: []workflow.Step{{Run: []string{"make check-alpha"}}},
			}}}
			got, err := baselineOptions().Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Length(t, got, 1, "the jobs")
			assert.NotEmpty(t, got[0].Steps, "the steps")
			assert.Equal(t, got[0].Steps[0].With["key"], toolsVersionKey, "the key of the cache")
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
			assert.Equal(t, got[0].Setup, []workflow.Step{{If: "hashFiles('alpha.lock') != ''", Uses: setupAlpha}},
				"the setup steps")
			assert.Equal(t, got[0].Steps, []workflow.Step{
				{If: "hashFiles('alpha.lock') != '' && (success())", Run: []string{"make check-alpha"}},
			}, "the steps")
		})

		t.Run("guards the setup that the jobs of two languages share once for each job", func(t *testing.T) {
			t.Parallel()
			shared := &workflow.Setup{Files: "alpha.lock", Steps: []workflow.Step{{Uses: setupAlpha}}, Timeout: 20}
			c := workflow.Contribution{Jobs: []workflow.Job{
				{ID: "check-alpha", Name: "Alpha", Setup: shared, Steps: []workflow.Step{{Run: []string{"make a"}}}},
				{ID: "check-beta", Name: "Beta", Setup: shared, Steps: []workflow.Step{{Run: []string{"make b"}}}},
			}}
			got, err := baselineOptions().Jobs(&c)
			assert.NoError(t, err, "Jobs")
			assert.Length(t, got, 2, "the jobs")
			guard := []workflow.Step{{If: "hashFiles('alpha.lock') != ''", Uses: setupAlpha}}
			expect.Equal(t, got[0].Setup, guard, "the setup steps of check-alpha")
			expect.Equal(t, got[1].Setup, guard, "the setup steps of check-beta")
			expect.Empty(t, shared.Steps[0].If, "the condition of the shared setup step")
		})

		t.Run("leaves the contribution unchanged", func(t *testing.T) {
			t.Parallel()
			c := guarded()
			var err error
			// The reading is the JSON of the contribution, a copy that includes the setup behind its
			// pointer.
			assert.Pure(t, func() string {
				encoded, _ := json.Marshal(c)
				return string(encoded)
			}, func() { _, err = baselineOptions().Jobs(&c) }, "the contribution")
			assert.NoError(t, err, "Jobs")
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

// guarded returns a new contribution of a job that runs tools, whose setup states files, and whose
// steps have a condition and none.
func guarded() workflow.Contribution {
	return workflow.Contribution{Jobs: []workflow.Job{{
		ID:   "check-alpha",
		Name: "Alpha",
		Setup: &workflow.Setup{
			Files:   "alpha.lock",
			Steps:   []workflow.Step{{Uses: setupAlpha}},
			Timeout: 20,
		},
		Tools: true,
		Steps: []workflow.Step{{If: "success()", Run: []string{"make check-alpha"}}, {Run: []string{"true"}}},
	}}}
}
