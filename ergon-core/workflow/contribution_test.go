// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workflow"
)

func TestContribution(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the zero value", func(t *testing.T) {
			t.Parallel()
			var c workflow.Contribution
			assert.NoError(t, c.Validate(), "Validate")
		})

		t.Run("returns nil for a contribution of every part", func(t *testing.T) {
			t.Parallel()
			c := goContribution()
			assert.NoError(t, c.Validate(), "Validate")
		})

		invalid := []struct {
			name string
			give func(*workflow.Contribution)
			want error
		}{
			{
				name: "returns ErrInvalidJob for a setup that is not valid",
				give: func(c *workflow.Contribution) { c.Setup.Timeout = 0 },
				want: workflow.ErrInvalidJob,
			},
			{
				name: "returns ErrInvalidJob for a job that is not valid",
				give: func(c *workflow.Contribution) { c.Jobs[0].Name = "" },
				want: workflow.ErrInvalidJob,
			},
			{
				name: "returns ErrInvalidJob for a nightly job that is not valid",
				give: func(c *workflow.Contribution) { c.Nightly[0].Steps = nil },
				want: workflow.ErrInvalidJob,
			},
			{
				name: "returns ErrInvalidStep for a release step that is not valid",
				give: func(c *workflow.Contribution) { c.Release[0].Uses = workflow.Action{} },
				want: workflow.ErrInvalidStep,
			},
			{
				name: "returns ErrInvalidCodeQL for an analysis that is not valid",
				give: func(c *workflow.Contribution) { c.CodeQL[0].BuildMode = "" },
				want: workflow.ErrInvalidCodeQL,
			},
			{
				name: "returns ErrInvalidUpdate for an update that is not valid",
				give: func(c *workflow.Contribution) { c.Updates[0].Directories = nil },
				want: workflow.ErrInvalidUpdate,
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := goContribution()
				tt.give(&c)
				assert.ErrorIs(t, c.Validate(), tt.want, "Validate")
			})
		}
	})
}

// goContribution returns a new valid contribution of the cases with every part: a shared setup,
// the check of Go, the check of Go as a nightly job, the setup of Go in a release, its analysis and
// its updates.
func goContribution() workflow.Contribution {
	return workflow.Contribution{
		Setup:   goSetup(),
		Jobs:    []workflow.Job{*goJob()},
		Nightly: []workflow.Job{*goJob()},
		Release: []workflow.Step{{Uses: setupGo, With: map[string]string{"go-version-file": "go.work"}}},
		CodeQL:  []workflow.CodeQL{goAnalysis()},
		Updates: []workflow.Update{{Ecosystem: "gomod", Directories: []string{"/"}}},
	}
}
