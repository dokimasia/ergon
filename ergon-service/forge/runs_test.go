// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/ergon/service/forge"
)

// The workflow, the event, the commit and the page of the run of the cases.
const (
	runWorkflow = "ci.yml"
	runEvent    = "push"
	runSHA      = "0749c74e25cc9171f23944f676f42168839ddc0e"
	runPage     = "https://github.com/dokimasia/ergon/actions/runs/37769836521"
)

// The statuses and the conclusion of a run in the responses of the cases.
const (
	completed  = "completed"
	inProgress = "in_progress"
	success    = "success"
)

// runsRoute is the route of the runs of the workflow for the event and the commit of the cases.
const runsRoute = "GET /repos/" + repo + "/actions/workflows/" + runWorkflow + "/runs?event=" + runEvent +
	"&head_sha=" + runSHA + "&per_page=1"

func TestRuns(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the status, the conclusion and the page of the newest run", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{runsRoute: {{
				body: `{"total_count":2,"workflow_runs":[` +
					`{"status":"` + completed + `","conclusion":"` + success + `","html_url":"` + runPage + `"},` +
					`{"status":"` + completed + `","conclusion":"failure","html_url":"https://github.com/older"}]}`,
			}}})
			status, conclusion, page, err := c.Run(t.Context(), repo, runWorkflow, runSHA, runEvent)
			assert.NoError(t, err, "Run")
			expect.Equal(t, status, completed, "the status")
			expect.Equal(t, conclusion, success, "the conclusion")
			expect.Equal(t, page, runPage, "the page")
		})

		t.Run("returns an empty conclusion for a run that has not completed", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{runsRoute: {{
				body: `{"total_count":1,"workflow_runs":[` +
					`{"status":"` + inProgress + `","conclusion":null,"html_url":"` + runPage + `"}]}`,
			}}})
			status, conclusion, _, err := c.Run(t.Context(), repo, runWorkflow, runSHA, runEvent)
			assert.NoError(t, err, "Run")
			expect.Equal(t, status, inProgress, "the status")
			expect.Empty(t, conclusion, "the conclusion")
		})

		t.Run("returns empty results for a commit without a run", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{runsRoute: {{body: `{"total_count":0,"workflow_runs":[]}`}}})
			status, conclusion, page, err := c.Run(t.Context(), repo, runWorkflow, runSHA, runEvent)
			assert.NoError(t, err, "Run")
			expect.Empty(t, status, "the status")
			expect.Empty(t, conclusion, "the conclusion")
			expect.Empty(t, page, "the page")
		})

		t.Run("returns ErrGitHub for a workflow that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			_, _, _, err := c.Run(t.Context(), repo, "missing.yml", runSHA, runEvent)
			assert.ErrorIs(t, err, forge.ErrGitHub, "Run")
		})
	})
}
