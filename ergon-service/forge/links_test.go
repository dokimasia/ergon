// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// The addresses that the fake GitHub of the links returns.
const (
	commitURL = "https://github.com/dokimasia/ergon/commit/a085003b1c"
	pullURL   = "https://github.com/dokimasia/ergon/pull/"
)

func TestLinks(t *testing.T) {
	t.Parallel()

	t.Run("CommitLinks", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the links of the first merged pull request and of its author", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{"POST " + graphqlPath: {{body: `{"data":{"repository":{"object":{
				"commitUrl":"` + commitURL + `",
				"associatedPullRequests":{"nodes":[
					{"number":2,"url":"` + pullURL + `2","mergedAt":null,"author":{"login":"cy","url":"https://github.com/cy"}},
					{"number":3,"url":"` + pullURL + `3","mergedAt":"2026-10-02T00:00:00Z","author":{"login":"bo","url":"https://github.com/bo"}},
					{"number":1,"url":"` + pullURL + `1","mergedAt":"2026-10-01T00:00:00Z","author":{"login":"an","url":"https://github.com/an"}}
				]},
				"author":{"user":{"login":"co","url":"https://github.com/co"}}}}}}`}}})
			commit, pull, author, err := c.CommitLinks(t.Context(), repo, "a085003b1c")
			assert.NoError(t, err, "CommitLinks")
			assert.Equal(t, commit, "[`a085003`]("+commitURL+")", "the link of the commit")
			assert.Equal(t, pull, "[#1]("+pullURL+"1)", "the link of the pull request")
			assert.Equal(t, author, "[@an](https://github.com/an)", "the link of the author")
		})

		t.Run("returns the author of a commit without a pull request", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{"POST " + graphqlPath: {{body: `{"data":{"repository":{"object":{
				"commitUrl":"` + commitURL + `","associatedPullRequests":{"nodes":[]},
				"author":{"user":{"login":"co","url":"https://github.com/co"}}}}}}`}}})
			_, pull, author, err := c.CommitLinks(t.Context(), repo, "a085003b1c")
			assert.NoError(t, err, "CommitLinks")
			assert.Equal(t, pull, "", "the link of the pull request")
			assert.Equal(t, author, "[@co](https://github.com/co)", "the link of the author")
		})

		t.Run("returns no author for a commit whose author has no account", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{"POST " + graphqlPath: {{body: `{"data":{"repository":{"object":{
				"commitUrl":"` + commitURL + `","associatedPullRequests":{"nodes":[]},"author":{"user":null}}}}}`}}})
			_, _, author, err := c.CommitLinks(t.Context(), repo, "a085003b1c")
			assert.NoError(t, err, "CommitLinks")
			assert.Equal(t, author, "", "the link of the author")
		})

		t.Run("returns empty links for a commit that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(
				t,
				map[string][]response{"POST " + graphqlPath: {{body: `{"data":{"repository":{"object":null}}}`}}},
			)
			commit, pull, author, err := c.CommitLinks(t.Context(), repo, "deadbee")
			assert.NoError(t, err, "CommitLinks")
			assert.Equal(t, commit+pull+author, "", "the links")
		})

		t.Run("sends the owner, the name of the repository and the commit", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(
				t,
				map[string][]response{"POST " + graphqlPath: {{body: `{"data":{"repository":{"object":null}}}`}}},
			)
			_, _, _, err := c.CommitLinks(t.Context(), repo, "deadbee")
			assert.NoError(t, err, "CommitLinks")
			assert.Equal(t, variablesOf(t, fake.recorded()[0].body),
				map[string]any{"owner": "dokimasia", "name": "ergon", "commit": "deadbee"}, "the variables")
		})
	})

	t.Run("PullLinks", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the links of a pull request, of its merge commit and of its author", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(
				t,
				map[string][]response{"POST " + graphqlPath: {{body: `{"data":{"repository":{"pullRequest":{
				"author":{"login":"an","url":"https://github.com/an"},
				"mergeCommit":{"commitUrl":"` + commitURL + `","abbreviatedOid":"a085003b1"}}}}}`}}},
			)
			pull, commit, author, err := c.PullLinks(t.Context(), repo, 1613)
			assert.NoError(t, err, "PullLinks")
			assert.Equal(
				t,
				pull,
				"[#1613](https://github.com/dokimasia/ergon/pull/1613)",
				"the link of the pull request",
			)
			assert.Equal(t, commit, "[`a085003`]("+commitURL+")", "the link of the commit")
			assert.Equal(t, author, "[@an](https://github.com/an)", "the link of the author")
			assert.Equal(t, variablesOf(t, fake.recorded()[0].body),
				map[string]any{"owner": "dokimasia", "name": "ergon", "number": float64(1613)}, "the variables")
		})

		t.Run("returns no commit and no author for a pull request without them", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(
				t,
				map[string][]response{"POST " + graphqlPath: {{body: `{"data":{"repository":{"pullRequest":{
				"author":null,"mergeCommit":null}}}}`}}},
			)
			_, commit, author, err := c.PullLinks(t.Context(), repo, 7)
			assert.NoError(t, err, "PullLinks")
			assert.Equal(t, commit+author, "", "the links")
		})

		t.Run("returns ErrGitHub for a pull request that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(
				t,
				map[string][]response{"POST " + graphqlPath: {{body: `{"errors":[{"message":"Could not resolve"}]}`}}},
			)
			_, _, _, err := c.PullLinks(t.Context(), repo, 99999)
			assert.ErrorIs(t, err, forge.ErrGitHub, "PullLinks")
		})
	})
}

// variablesOf returns the variables of body, the body of a GraphQL request, for the test tb.
func variablesOf(tb testing.TB, body string) map[string]any {
	tb.Helper()
	var query struct {
		Variables map[string]any `json:"variables"`
	}
	assert.NoError(tb, json.Unmarshal([]byte(body), &query), "the decode of the query")
	return query.Variables
}
