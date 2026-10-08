// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"net/http"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// The status of the cases.
const (
	statusContext = "ergon/version"
	statusState   = "success"
	statusPage    = "https://github.com/dokimasia/ergon/actions/runs/37838509558"
)

// The routes of the statuses of the cases.
const (
	setStatusRoute = "POST /repos/" + repo + "/statuses/" + commitA
	statusRoute    = "GET /repos/" + repo + "/commits/" + commitA + "/status?per_page=100"
)

func TestStatuses(t *testing.T) {
	t.Parallel()

	t.Run("SetStatus", func(t *testing.T) {
		t.Parallel()

		t.Run("sets the state, the context, the description and the page of the status", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{setStatusRoute: {{status: http.StatusCreated}}})
			err := c.SetStatus(t.Context(), repo, commitA, statusState, statusContext, "The version.", statusPage)
			assert.NoError(t, err, "SetStatus")
			assert.Equal(t, fake.recorded()[0].body, `{"state":"success","context":"ergon/version",`+
				`"description":"The version.","target_url":"`+statusPage+`"}`, "the status")
		})

		t.Run("leaves out an empty description and an empty page", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{setStatusRoute: {{status: http.StatusCreated}}})
			assert.NoError(t, c.SetStatus(t.Context(), repo, commitA, statusState, statusContext, "", ""), "SetStatus")
			assert.Equal(t, fake.recorded()[0].body, `{"state":"success","context":"ergon/version"}`, "the status")
		})

		t.Run("returns ErrGitHub for a token without the permission", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{setStatusRoute: {{
				status: http.StatusForbidden, body: `{"message":"Resource not accessible by integration"}`,
			}}})
			err := c.SetStatus(t.Context(), repo, commitA, statusState, statusContext, "", "")
			assert.ErrorIs(t, err, forge.ErrGitHub, "SetStatus")
		})
	})

	t.Run("Status", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the state of the status of the context", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{statusRoute: {{
				body: `{"state":"failure","statuses":[{"state":"failure","context":"lint"},` +
					`{"state":"success","context":"ergon/version"}]}`,
			}}})
			state, ok, err := c.Status(t.Context(), repo, commitA, statusContext)
			assert.NoError(t, err, "Status")
			assert.True(t, ok, "whether the commit has the status")
			assert.Equal(t, state, statusState, "the state")
		})

		t.Run("reports false for a commit without a status of the context", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{statusRoute: {{
				body: `{"state":"pending","statuses":[{"state":"success","context":"lint"}]}`,
			}}})
			state, ok, err := c.Status(t.Context(), repo, commitA, statusContext)
			assert.NoError(t, err, "Status")
			assert.False(t, ok, "whether the commit has the status")
			assert.Empty(t, state, "the state")
		})

		t.Run("returns ErrGitHub for a commit that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			_, _, err := c.Status(t.Context(), repo, commitA, statusContext)
			assert.ErrorIs(t, err, forge.ErrGitHub, "Status")
		})
	})
}
