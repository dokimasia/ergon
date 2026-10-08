// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// The workflow, the commit and the page of the run of the cases.
const (
	runWorkflow = "ci.yml"
	runSHA      = "0749c74e25cc9171f23944f676f42168839ddc0e"
	runPage     = "https://github.com/dokimasia/ergon/actions/runs/37769836521"
)

// passedRoute is the route of the passed runs of the workflow for the commit of the cases.
const passedRoute = "GET /repos/" + repo + "/actions/workflows/" + runWorkflow + "/runs?head_sha=" + runSHA +
	"&per_page=1&status=success"

func TestRuns(t *testing.T) {
	t.Parallel()

	t.Run("Passed", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the page of the newest run that passed", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{passedRoute: {{
				body: `{"total_count":2,"workflow_runs":[` +
					`{"status":"completed","conclusion":"success","html_url":"` + runPage + `"},` +
					`{"status":"completed","conclusion":"success","html_url":"https://github.com/older"}]}`,
			}}})
			page, ok, err := c.Passed(t.Context(), repo, runWorkflow, runSHA)
			assert.NoError(t, err, "Passed")
			assert.True(t, ok, "whether a run passed")
			assert.Equal(t, page, runPage, "the page")
		})

		t.Run("reports false for a commit without a run that passed", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{passedRoute: {{body: `{"total_count":0,"workflow_runs":[]}`}}})
			page, ok, err := c.Passed(t.Context(), repo, runWorkflow, runSHA)
			assert.NoError(t, err, "Passed")
			assert.False(t, ok, "whether a run passed")
			assert.Empty(t, page, "the page")
		})

		t.Run("returns ErrGitHub for a workflow that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			_, _, err := c.Passed(t.Context(), repo, "missing.yml", runSHA)
			assert.ErrorIs(t, err, forge.ErrGitHub, "Passed")
		})
	})
}
