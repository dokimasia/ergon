// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// TestMain runs the tests without the variables and the configuration of git of the environment,
// so git reads the working trees of the tests alone.
func TestMain(m *testing.M) {
	vcstest.Isolate()
	os.Exit(m.Run())
}

func TestFiles(t *testing.T) {
	t.Parallel()

	t.Run("Files", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the tracked and the untracked files, sorted", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{
				".gitignore":      files.Text("bin/\n"),
				"main.go":         files.Text("package main\n"),
				"docs/README.md":  files.Text("# docs\n"),
				"bin/tool":        files.Text("binary\n"),
				"cmd/a b/main.go": files.Text("package main\n"),
			}, "main.go", ".gitignore")
			got, err := vcs.Files(t.Context(), dir)
			assert.NoError(t, err, "Files")
			assert.Equal(t, got, []string{".gitignore", "cmd/a b/main.go", "docs/README.md", "main.go"}, "the files")
		})

		t.Run("returns a tracked file that the working tree deleted", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"main.go": files.Text("package main\n")}, "main.go")
			assert.NoError(t, os.Remove(dir+"/main.go"), "Remove")
			got, err := vcs.Files(t.Context(), dir)
			assert.NoError(t, err, "Files")
			assert.Equal(t, got, []string{"main.go"}, "the files")
		})

		t.Run("returns no file for a working tree without files", func(t *testing.T) {
			t.Parallel()
			got, err := vcs.Files(t.Context(), vcstest.Repository(t, files.Tree{}))
			assert.NoError(t, err, "Files")
			assert.Empty(t, got, "the files")
		})

		t.Run("returns ErrGit for a directory outside a working tree", func(t *testing.T) {
			t.Parallel()
			_, err := vcs.Files(t.Context(), t.TempDir())
			assert.ErrorIs(t, err, vcs.ErrGit, "Files")
		})

		t.Run("returns ErrGit for a context that ended", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"main.go": files.Text("package main\n")})
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, err := vcs.Files(ctx, dir)
			assert.ErrorIs(t, err, vcs.ErrGit, "Files")
		})
	})

	t.Run("Changed", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the files that the branch and the working tree change since the merge base",
			func(t *testing.T) {
				t.Parallel()
				dir := vcstest.Repository(t, files.Tree{
					"kept.go":    files.Text("package main\n"),
					"edited.go":  files.Text("package main\n"),
					"removed.go": files.Text("package main\n"),
					"old.go":     files.Text("package main\n\nfunc renamed() {}\n"),
				})
				vcstest.Commit(t, dir, "base")
				vcstest.Git(t, dir, "branch", "base")
				write(t, dir, "committed.go", "package main\n")
				vcstest.Commit(t, dir, "add a file")
				vcstest.Git(t, dir, "mv", "old.go", "new.go")
				vcstest.Git(t, dir, "rm", "--quiet", "removed.go")
				write(t, dir, "edited.go", "package main\n\n// Edited.\n")
				write(t, dir, "untracked.go", "package main\n")
				got, err := vcs.Changed(t.Context(), dir, "base")
				assert.NoError(t, err, "Changed")
				assert.Equal(t, got, []string{
					"committed.go", "edited.go", "new.go", "old.go", "removed.go", "untracked.go",
				}, "the changed files")
			})

		t.Run("returns the changes since the merge base and not the changes of the base after it",
			func(t *testing.T) {
				t.Parallel()
				dir := vcstest.Repository(t, files.Tree{"a.go": files.Text("package main\n")})
				vcstest.Commit(t, dir, "base")
				vcstest.Git(t, dir, "branch", "main-line")
				write(t, dir, "branch.go", "package main\n")
				vcstest.Commit(t, dir, "on the branch")
				vcstest.Git(t, dir, "switch", "--quiet", "main-line")
				write(t, dir, "base.go", "package main\n")
				vcstest.Commit(t, dir, "on the base")
				vcstest.Git(t, dir, "switch", "--quiet", "-")
				got, err := vcs.Changed(t.Context(), dir, "main-line")
				assert.NoError(t, err, "Changed")
				assert.Equal(t, got, []string{"branch.go"}, "the changed files")
			})

		t.Run("returns no file for a branch at its base", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.go": files.Text("package main\n")})
			vcstest.Commit(t, dir, "base")
			got, err := vcs.Changed(t.Context(), dir, "HEAD")
			assert.NoError(t, err, "Changed")
			assert.Empty(t, got, "the changed files")
		})

		t.Run("returns ErrGit for a base that names no commit", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.go": files.Text("package main\n")})
			vcstest.Commit(t, dir, "base")
			_, err := vcs.Changed(t.Context(), dir, "origin/main")
			assert.ErrorIs(t, err, vcs.ErrGit, "Changed")
		})

		t.Run("returns ErrGit for a context that ended", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.go": files.Text("package main\n")})
			vcstest.Commit(t, dir, "base")
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, err := vcs.Changed(ctx, dir, "HEAD")
			assert.ErrorIs(t, err, vcs.ErrGit, "Changed")
		})
	})
}

// write writes content to the file at path, relative to dir, and stops the test when the write
// fails.
func write(tb testing.TB, dir, path, content string) {
	tb.Helper()
	assert.NoError(tb, os.WriteFile(filepath.Join(dir, path), []byte(content), 0o644), "WriteFile of "+path)
}
