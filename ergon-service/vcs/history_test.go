// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

func TestHistory(t *testing.T) {
	t.Parallel()

	t.Run("Head", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the commit of HEAD", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			vcstest.Commit(t, dir, "first")
			want := vcstest.Commit(t, dir, "second")
			got, err := vcs.Head(t.Context(), dir)
			assert.NoError(t, err, "Head")
			assert.Equal(t, got, want, "the commit")
		})

		t.Run("returns ErrGit for a repository without a commit", func(t *testing.T) {
			t.Parallel()
			_, err := vcs.Head(t.Context(), vcstest.Repository(t, files.Tree{}))
			assert.ErrorIs(t, err, vcs.ErrGit, "Head")
		})
	})

	t.Run("HeadTree", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the commit of HEAD and its tree", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.md": files.Text("a\n")})
			vcstest.Commit(t, dir, "first")
			write(t, dir, "a.md", "b\n")
			want := vcstest.Commit(t, dir, "second")
			commit, tree, err := vcs.HeadTree(t.Context(), dir)
			assert.NoError(t, err, "HeadTree")
			expect.That(t, commit).Equal(want, "the commit")
			expect.That(t, tree).Equal(strings.TrimSpace(vcstest.Git(t, dir, "log", "-1", "--format=%T", want)),
				"the tree")
		})

		t.Run("returns ErrGit for a repository without a commit", func(t *testing.T) {
			t.Parallel()
			_, _, err := vcs.HeadTree(t.Context(), vcstest.Repository(t, files.Tree{}))
			assert.ErrorIs(t, err, vcs.ErrGit, "HeadTree")
		})
	})

	t.Run("AddedBy", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the commit that added the file", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{".changeset/add-unit.md": files.Text("---\n---\n")})
			want := vcstest.Commit(t, dir, "add the changeset")
			write(t, dir, ".changeset/add-unit.md", "---\n\"a\": patch\n---\n")
			vcstest.Commit(t, dir, "edit the changeset")
			got, err := vcs.AddedBy(t.Context(), dir, ".changeset/add-unit.md")
			assert.NoError(t, err, "AddedBy")
			assert.Equal(t, got, want, "the commit")
		})

		t.Run("returns the empty string for a file that no commit added", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			vcstest.Commit(t, dir, "first")
			write(t, dir, "new.md", "new\n")
			got, err := vcs.AddedBy(t.Context(), dir, "new.md")
			assert.NoError(t, err, "AddedBy")
			assert.Empty(t, got, "the commit")
		})

		t.Run("returns ErrGit for a directory outside a working tree", func(t *testing.T) {
			t.Parallel()
			_, err := vcs.AddedBy(t.Context(), t.TempDir(), "a.md")
			assert.ErrorIs(t, err, vcs.ErrGit, "AddedBy")
		})
	})
}
