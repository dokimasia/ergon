// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"net/http"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// The commits, the tag object and the tree of the cases.
const (
	commitA   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	commitB   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	commitC   = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	tagObject = "cccccccccccccccccccccccccccccccccccccccc"
	treeA     = "dddddddddddddddddddddddddddddddddddddddd"
)

// The routes of the refs of the cases.
const (
	branchRoute = "GET /repos/" + repo + "/git/ref/heads/ergon-release/main"
	tagRoute    = "GET /repos/" + repo + "/git/ref/tags/ergon-core/v0.1.0"
	refsRoute   = "POST /repos/" + repo + "/git/refs"
)

// failure is the response of a request that fails on GitHub.
var failure = response{status: http.StatusInternalServerError, body: `{"message":"Server Error"}`}

func TestRefs(t *testing.T) {
	t.Parallel()

	t.Run("Branch", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the commit of a branch", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(
				t,
				map[string][]response{branchRoute: {{body: `{"object":{"type":"commit","sha":"` + commitA + `"}}`}}},
			)
			sha, found, err := c.Branch(t.Context(), repo, "ergon-release/main")
			assert.NoError(t, err, "Branch")
			assert.True(t, found, "whether the repository has the branch")
			assert.Equal(t, sha, commitA, "the commit")
		})

		t.Run("reports false for a branch that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, nil)
			_, found, err := c.Branch(t.Context(), repo, "ergon-release/main")
			assert.NoError(t, err, "Branch")
			assert.False(t, found, "whether the repository has the branch")
		})
	})

	t.Run("SetBranch", func(t *testing.T) {
		t.Parallel()

		t.Run("moves a branch that the repository has", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{
				branchRoute: {{body: `{"object":{"type":"commit","sha":"` + commitA + `"}}`}},
				"PATCH /repos/" + repo + "/git/refs/heads/ergon-release/main": {{}},
			})
			assert.NoError(t, c.SetBranch(t.Context(), repo, "ergon-release/main", commitB), "SetBranch")
			assert.Equal(t, fake.recorded()[1].body, `{"sha":"`+commitB+`","force":true}`, "the update")
		})

		t.Run("creates a branch that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{refsRoute: {{status: http.StatusCreated}}})
			assert.NoError(t, c.SetBranch(t.Context(), repo, "ergon-release/main", commitB), "SetBranch")
			assert.Equal(t, fake.recorded()[1].body, `{"ref":"refs/heads/ergon-release/main","sha":"`+commitB+`"}`,
				"the new ref")
		})

		t.Run("returns the error of the lookup of the branch", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{branchRoute: {failure}})
			err := c.SetBranch(t.Context(), repo, "ergon-release/main", commitB)
			assert.ErrorIs(t, err, forge.ErrGitHub, "SetBranch")
		})
	})

	t.Run("Tag", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the commit of a lightweight tag", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(
				t,
				map[string][]response{tagRoute: {{body: `{"object":{"type":"commit","sha":"` + commitA + `"}}`}}},
			)
			sha, found, err := c.Tag(t.Context(), repo, "ergon-core/v0.1.0")
			assert.NoError(t, err, "Tag")
			assert.True(t, found, "whether the repository has the tag")
			assert.Equal(t, sha, commitA, "the commit")
		})

		t.Run("returns the commit of an annotated tag", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				tagRoute: {{body: `{"object":{"type":"tag","sha":"` + tagObject + `"}}`}},
				"GET /repos/" + repo + "/git/tags/" + tagObject: {
					{body: `{"object":{"type":"commit","sha":"` + commitB + `"}}`},
				},
			})
			sha, found, err := c.Tag(t.Context(), repo, "ergon-core/v0.1.0")
			assert.NoError(t, err, "Tag")
			assert.True(t, found, "whether the repository has the tag")
			assert.Equal(t, sha, commitB, "the commit")
		})

		t.Run("reports false for a tag that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, nil)
			_, found, err := c.Tag(t.Context(), repo, "ergon-core/v0.1.0")
			assert.NoError(t, err, "Tag")
			assert.False(t, found, "whether the repository has the tag")
		})

		t.Run("returns the error of the tag object of an annotated tag", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				tagRoute: {{body: `{"object":{"type":"tag","sha":"` + tagObject + `"}}`}},
				"GET /repos/" + repo + "/git/tags/" + tagObject: {failure},
			})
			_, _, err := c.Tag(t.Context(), repo, "ergon-core/v0.1.0")
			assert.ErrorIs(t, err, forge.ErrGitHub, "Tag")
		})

		t.Run("returns the error of the lookup of the tag", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{tagRoute: {failure}})
			_, _, err := c.Tag(t.Context(), repo, "ergon-core/v0.1.0")
			assert.ErrorIs(t, err, forge.ErrGitHub, "Tag")
		})
	})

	t.Run("CreateTag", func(t *testing.T) {
		t.Parallel()

		t.Run("creates the ref of the tag at the commit", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{refsRoute: {{status: http.StatusCreated}}})
			assert.NoError(t, c.CreateTag(t.Context(), repo, "ergon-core/v0.1.0", commitA), "CreateTag")
			assert.Equal(t, fake.recorded()[0].body, `{"ref":"refs/tags/ergon-core/v0.1.0","sha":"`+commitA+`"}`,
				"the new ref")
		})

		t.Run("returns ErrGitHub for a tag that the repository has", func(t *testing.T) {
			t.Parallel()
			exists := response{status: http.StatusUnprocessableEntity, body: `{"message":"Reference already exists"}`}
			c, _ := serve(t, map[string][]response{refsRoute: {exists}})
			err := c.CreateTag(t.Context(), repo, "ergon-core/v0.1.0", commitA)
			assert.ErrorIs(t, err, forge.ErrGitHub, "CreateTag")
			assert.Contains(t, err.Error(), "Reference already exists", "the error")
		})
	})
}
