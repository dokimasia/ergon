// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

func TestTags(t *testing.T) {
	t.Parallel()

	t.Run("Tags", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the commit of an annotated and of a lightweight tag", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.go": files.Text("package a\n")})
			first := vcstest.Commit(t, dir, "first")
			second := vcstest.Commit(t, dir, "second")
			assert.NoError(t, vcs.Tag(t.Context(), dir, "core/v1.0.0", first, "Release core 1.0.0."), "Tag")
			vcstest.Git(t, dir, "tag", "v2.0.0", second)
			got, err := vcs.Tags(t.Context(), dir)
			assert.NoError(t, err, "Tags")
			assert.Equal(t, got, map[string]string{"core/v1.0.0": first, "v2.0.0": second}, "the tags")
		})

		t.Run("returns an empty map for a repository without tags", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			vcstest.Commit(t, dir, "first")
			got, err := vcs.Tags(t.Context(), dir)
			assert.NoError(t, err, "Tags")
			assert.Empty(t, got, "the tags")
		})

		t.Run("returns ErrGit for a directory outside a working tree", func(t *testing.T) {
			t.Parallel()
			_, err := vcs.Tags(t.Context(), t.TempDir())
			assert.ErrorIs(t, err, vcs.ErrGit, "Tags")
		})
	})

	t.Run("Tag", func(t *testing.T) {
		t.Parallel()

		t.Run("creates an annotated tag with the message", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			message := "## 1.0.0\n\n### Minor Changes\n\n- Add the Unit type."
			assert.NoError(t, vcs.Tag(t.Context(), dir, "v1.0.0", commit, message), "Tag")
			assert.Equal(t, strings.TrimSpace(vcstest.Git(t, dir, "cat-file", "-t", "v1.0.0")), "tag", "the object")
			assert.Equal(t, vcstest.Git(t, dir, "tag", "--list", "--format=%(contents)", "v1.0.0"), message+"\n",
				"the annotation, with the headings that start with #")
		})

		t.Run("returns ErrGit for a tag that exists", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.Tag(t.Context(), dir, "v1.0.0", commit, "First."), "the first Tag")
			assert.ErrorIs(t, vcs.Tag(t.Context(), dir, "v1.0.0", commit, "Again."), vcs.ErrGit, "the second Tag")
		})
	})

	t.Run("Push", func(t *testing.T) {
		t.Parallel()

		t.Run("pushes the refs to the remote", func(t *testing.T) {
			t.Parallel()
			remote := t.TempDir()
			vcstest.Git(t, remote, "init", "--quiet", "--bare")
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.Tag(t.Context(), dir, "v1.0.0", commit, "First."), "Tag")
			assert.NoError(t, vcs.Push(t.Context(), dir, remote, "refs/tags/v1.0.0"), "Push")
			got, err := vcs.Tags(t.Context(), remote)
			assert.NoError(t, err, "Tags of the remote")
			assert.Equal(t, got, map[string]string{"v1.0.0": commit}, "the tags of the remote")
		})

		t.Run("returns ErrGit and pushes no ref when one ref does not exist", func(t *testing.T) {
			t.Parallel()
			remote := t.TempDir()
			vcstest.Git(t, remote, "init", "--quiet", "--bare")
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.Tag(t.Context(), dir, "v1.0.0", commit, "First."), "Tag")
			err := vcs.Push(t.Context(), dir, remote, "refs/tags/v1.0.0", "refs/tags/v9.9.9")
			assert.ErrorIs(t, err, vcs.ErrGit, "Push")
			got, err := vcs.Tags(t.Context(), remote)
			assert.NoError(t, err, "Tags of the remote")
			assert.Empty(t, got, "the tags of the remote")
		})
	})
}
