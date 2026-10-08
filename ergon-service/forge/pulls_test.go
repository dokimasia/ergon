// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"net/http"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// The routes of the pull requests of the cases.
const (
	openRoute   = "GET /repos/" + repo + "/pulls?base=main&head=dokimasia%3Aergon-release%2Fmain&state=open"
	createRoute = "POST /repos/" + repo + "/pulls"
	updateRoute = "PATCH /repos/" + repo + "/pulls/7"
	headsRoute  = "GET /repos/" + repo + "/commits/" + commitA + "/pulls"
)

func TestPulls(t *testing.T) {
	t.Parallel()

	t.Run("PullRequest", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the number of the open pull request from a branch", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{openRoute: {{body: `[{"number":7},{"number":9}]`}}})
			number, found, err := c.PullRequest(t.Context(), repo, "ergon-release/main", "main")
			assert.NoError(t, err, "PullRequest")
			assert.True(t, found, "whether a pull request is open")
			assert.Equal(t, number, 7, "the number")
		})

		t.Run("reports false for a branch without an open pull request", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{openRoute: {{body: `[]`}}})
			_, found, err := c.PullRequest(t.Context(), repo, "ergon-release/main", "main")
			assert.NoError(t, err, "PullRequest")
			assert.False(t, found, "whether a pull request is open")
		})

		t.Run("returns the error of the request", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{openRoute: {failure}})
			_, _, err := c.PullRequest(t.Context(), repo, "ergon-release/main", "main")
			assert.ErrorIs(t, err, forge.ErrGitHub, "PullRequest")
		})
	})

	t.Run("CreatePullRequest", func(t *testing.T) {
		t.Parallel()

		t.Run("opens a pull request and returns its number", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(
				t,
				map[string][]response{createRoute: {{status: http.StatusCreated, body: `{"number":8}`}}},
			)
			number, err := c.CreatePullRequest(
				t.Context(),
				repo,
				"ergon-release/main",
				"main",
				"Version Packages",
				"Body.",
			)
			assert.NoError(t, err, "CreatePullRequest")
			assert.Equal(t, number, 8, "the number")
			assert.Equal(
				t,
				fake.recorded()[0].body,
				`{"title":"Version Packages","body":"Body.","head":"ergon-release/main","base":"main"}`,
				"the pull request",
			)
		})

		t.Run("returns the error of the request", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{createRoute: {failure}})
			_, err := c.CreatePullRequest(t.Context(), repo, "ergon-release/main", "main", "Version Packages", "Body.")
			assert.ErrorIs(t, err, forge.ErrGitHub, "CreatePullRequest")
		})
	})

	t.Run("PullHeads", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the head commit of each pull request of the commit in the order of GitHub", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{headsRoute: {{
				body: `[{"number":6,"head":{"ref":"ergon-release/main","sha":"` + commitB + `"}},` +
					`{"number":4,"head":{"ref":"fix","sha":"` + commitC + `"}}]`,
			}}})
			heads, err := c.PullHeads(t.Context(), repo, commitA)
			assert.NoError(t, err, "PullHeads")
			assert.Equal(t, heads, []string{commitB, commitC}, "the heads")
		})

		t.Run("returns no head for a commit without a pull request", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{headsRoute: {{body: `[]`}}})
			heads, err := c.PullHeads(t.Context(), repo, commitA)
			assert.NoError(t, err, "PullHeads")
			assert.Empty(t, heads, "the heads")
		})

		t.Run("returns the error of the request", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{headsRoute: {failure}})
			_, err := c.PullHeads(t.Context(), repo, commitA)
			assert.ErrorIs(t, err, forge.ErrGitHub, "PullHeads")
		})
	})

	t.Run("UpdatePullRequest", func(t *testing.T) {
		t.Parallel()

		t.Run("sets the title and the body of the pull request", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{updateRoute: {{}}})
			assert.NoError(
				t,
				c.UpdatePullRequest(t.Context(), repo, 7, "Version Packages", "Body."),
				"UpdatePullRequest",
			)
			assert.Equal(t, fake.recorded()[0].body, `{"title":"Version Packages","body":"Body."}`, "the update")
		})
	})
}
