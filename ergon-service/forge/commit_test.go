// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

func TestCommit(t *testing.T) {
	t.Parallel()

	t.Run("Commit", func(t *testing.T) {
		t.Parallel()

		t.Run("commits the files and the deletions after the head that the branch must have", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{
				"POST " + graphqlPath: {
					{body: `{"data":{"createCommitOnBranch":{"commit":{"oid":"` + commitB + `"}}}}`},
				},
			})
			files := map[string][]byte{"b/go.mod": []byte("module b\n"), "a/CHANGELOG.md": []byte("# a\n")}
			got, err := c.Commit(t.Context(), repo, "ergon-release/main", commitA, "Version Packages\n\nThe body.",
				files, []string{".changeset/strange-words-combine.md"})
			assert.NoError(t, err, "Commit")
			assert.Equal(t, got, commitB, "the commit")
			assert.Equal(t, variablesOf(t, fake.recorded()[0].body), map[string]any{"input": map[string]any{
				"branch":          map[string]any{"repositoryNameWithOwner": repo, "branchName": "ergon-release/main"},
				"message":         map[string]any{"headline": "Version Packages", "body": "The body."},
				"expectedHeadOid": commitA,
				"fileChanges": map[string]any{
					"additions": []any{
						map[string]any{"path": "a/CHANGELOG.md", "contents": "IyBhCg=="},
						map[string]any{"path": "b/go.mod", "contents": "bW9kdWxlIGIK"},
					},
					"deletions": []any{map[string]any{"path": ".changeset/strange-words-combine.md"}},
				},
			}}, "the variables")
		})

		t.Run("returns ErrGitHub for a branch whose head is another commit", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				"POST " + graphqlPath: {
					{body: `{"errors":[{"message":"Expected branch to point to ` + commitA + `"}]}`},
				},
			})
			_, err := c.Commit(t.Context(), repo, "ergon-release/main", commitA, "Version Packages", nil, nil)
			assert.ErrorIs(t, err, forge.ErrGitHub, "Commit")
			assert.Contains(t, err.Error(), "Expected branch to point to", "the error")
		})
	})

	t.Run("Tree", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the tree of the commit", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				"GET /repos/" + repo + "/git/commits/" + commitA: {
					{body: `{"sha":"` + commitA + `","tree":{"sha":"` + treeA + `"},"parents":[]}`},
				},
			})
			tree, err := c.Tree(t.Context(), repo, commitA)
			assert.NoError(t, err, "Tree")
			assert.Equal(t, tree, treeA, "the tree")
		})

		t.Run("returns ErrGitHub for a commit that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			_, err := c.Tree(t.Context(), repo, commitA)
			assert.ErrorIs(t, err, forge.ErrGitHub, "Tree")
		})
	})
}
