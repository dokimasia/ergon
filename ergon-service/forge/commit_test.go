// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"net/http"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// The routes of the default branch of the repository of the cases, and of its cask at commitA.
const (
	repoRoute = "GET /repos/" + repo
	mainRoute = "GET /repos/" + repo + "/git/ref/heads/main"
	caskRoute = "GET /repos/" + repo + "/contents/Casks/ergon.rb?ref=" + commitA
)

// The content of the cask of the cases, and the name that git gives the blob of that content.
const (
	cask     = "hello\n"
	caskBlob = "ce013625030ba8dba906f756967f9e9ca394464a"
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

	t.Run("PutFile", func(t *testing.T) {
		t.Parallel()

		t.Run("commits the file on the default branch after its head", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{
				repoRoute: {{body: `{"default_branch":"main"}`}},
				mainRoute: {{body: `{"object":{"type":"commit","sha":"` + commitA + `"}}`}},
				"POST " + graphqlPath: {
					{body: `{"data":{"createCommitOnBranch":{"commit":{"oid":"` + commitB + `"}}}}`},
				},
			})
			err := c.PutFile(t.Context(), repo, "Casks/ergon.rb", "ergon 0.6.0", []byte(cask))
			assert.NoError(t, err, "PutFile")
			requests := fake.recorded()
			assert.Length(t, requests, 4, "the requests")
			assert.Equal(t, requests[2].target, "/repos/"+repo+"/contents/Casks/ergon.rb?ref="+commitA,
				"the request of the file at the head")
			assert.Equal(t, variablesOf(t, requests[3].body), map[string]any{"input": map[string]any{
				"branch":          map[string]any{"repositoryNameWithOwner": repo, "branchName": "main"},
				"message":         map[string]any{"headline": "ergon 0.6.0", "body": ""},
				"expectedHeadOid": commitA,
				"fileChanges": map[string]any{
					"additions": []any{map[string]any{"path": "Casks/ergon.rb", "contents": "aGVsbG8K"}},
					"deletions": []any{},
				},
			}}, "the variables")
		})

		t.Run("commits nothing for a file that already has the content", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{
				repoRoute: {{body: `{"default_branch":"main"}`}},
				mainRoute: {{body: `{"object":{"type":"commit","sha":"` + commitA + `"}}`}},
				caskRoute: {{body: `{"sha":"` + caskBlob + `"}`}},
			})
			err := c.PutFile(t.Context(), repo, "Casks/ergon.rb", "ergon 0.6.0", []byte(cask))
			assert.NoError(t, err, "PutFile")
			assert.Length(t, fake.recorded(), 3, "the requests")
		})

		t.Run("commits a file whose content differs", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{
				repoRoute: {{body: `{"default_branch":"main"}`}},
				mainRoute: {{body: `{"object":{"type":"commit","sha":"` + commitA + `"}}`}},
				caskRoute: {{body: `{"sha":"` + treeA + `"}`}},
				"POST " + graphqlPath: {
					{body: `{"data":{"createCommitOnBranch":{"commit":{"oid":"` + commitB + `"}}}}`},
				},
			})
			err := c.PutFile(t.Context(), repo, "Casks/ergon.rb", "ergon 0.6.0", []byte(cask))
			assert.NoError(t, err, "PutFile")
			assert.Length(t, fake.recorded(), 4, "the requests")
		})

		t.Run("returns ErrGitHub for a repository without its default branch", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{repoRoute: {{body: `{"default_branch":"main"}`}}})
			err := c.PutFile(t.Context(), repo, "Casks/ergon.rb", "ergon 0.6.0", []byte(cask))
			assert.ErrorIs(t, err, forge.ErrGitHub, "PutFile")
			assert.Equal(t, err.Error(), `forge: GitHub failed: `+repo+` has no default branch "main"`, "the error")
		})

		t.Run("returns ErrGitHub for a repository that does not exist", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			err := c.PutFile(t.Context(), repo, "Casks/ergon.rb", "ergon 0.6.0", []byte(cask))
			assert.ErrorIs(t, err, forge.ErrGitHub, "PutFile")
		})

		t.Run("returns ErrGitHub for a branch that the host does not read", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				repoRoute: {{body: `{"default_branch":"main"}`}},
				mainRoute: {{status: http.StatusBadGateway}},
			})
			err := c.PutFile(t.Context(), repo, "Casks/ergon.rb", "ergon 0.6.0", []byte(cask))
			assert.ErrorIs(t, err, forge.ErrGitHub, "PutFile")
		})

		t.Run("returns ErrGitHub for a file that the host does not read", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				repoRoute: {{body: `{"default_branch":"main"}`}},
				mainRoute: {{body: `{"object":{"type":"commit","sha":"` + commitA + `"}}`}},
				caskRoute: {{status: http.StatusBadGateway}},
			})
			err := c.PutFile(t.Context(), repo, "Casks/ergon.rb", "ergon 0.6.0", []byte(cask))
			assert.ErrorIs(t, err, forge.ErrGitHub, "PutFile")
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

	t.Run("Parent", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the first parent of a merge commit", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				"GET /repos/" + repo + "/git/commits/" + commitC: {{
					body: `{"sha":"` + commitC + `","tree":{"sha":"` + treeA + `"},` +
						`"parents":[{"sha":"` + commitA + `"},{"sha":"` + commitB + `"}]}`,
				}},
			})
			parent, ok, err := c.Parent(t.Context(), repo, commitC)
			assert.NoError(t, err, "Parent")
			assert.True(t, ok, "whether the commit has a parent")
			assert.Equal(t, parent, commitA, "the parent")
		})

		t.Run("reports false for a commit without a parent", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				"GET /repos/" + repo + "/git/commits/" + commitA: {
					{body: `{"sha":"` + commitA + `","tree":{"sha":"` + treeA + `"},"parents":[]}`},
				},
			})
			parent, ok, err := c.Parent(t.Context(), repo, commitA)
			assert.NoError(t, err, "Parent")
			assert.False(t, ok, "whether the commit has a parent")
			assert.Empty(t, parent, "the parent")
		})

		t.Run("returns ErrGitHub for a commit that the repository does not have", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			_, _, err := c.Parent(t.Context(), repo, commitA)
			assert.ErrorIs(t, err, forge.ErrGitHub, "Parent")
		})
	})
}
