// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs_test

import (
	"context"
	"os"
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
}
