// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// windows is the system of Windows, which stores no execute bit of a file.
const windows = "windows"

func TestSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("Snapshot", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a tree of the tracked and untracked files of the working tree", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{".gitignore": files.Text("ignored\n"), "a.txt": files.Text("a\n")})
			vcstest.Commit(t, dir, "first")
			write(t, dir, "a.txt", "changed\n")
			write(t, dir, "new.txt", "new\n")
			write(t, dir, "ignored", "secret\n")
			tree, err := vcs.Snapshot(t.Context(), dir)
			assert.NoError(t, err, "Snapshot")
			assert.Equal(t, vcstest.Git(t, dir, "ls-tree", "--name-only", tree), ".gitignore\na.txt\nnew.txt\n",
				"the files of the snapshot")
			assert.Equal(t, vcstest.Git(t, dir, "show", tree+":a.txt"), "changed\n", "the content of a.txt")
		})

		t.Run("returns the tree of HEAD for a working tree without a change", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			vcstest.Commit(t, dir, "first")
			tree, err := vcs.Snapshot(t.Context(), dir)
			assert.NoError(t, err, "Snapshot")
			assert.Equal(t, tree, strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD^{tree}")), "the tree")
		})

		t.Run("returns a tree in a repository without a commit", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			tree, err := vcs.Snapshot(t.Context(), dir)
			assert.NoError(t, err, "Snapshot")
			assert.Equal(t, vcstest.Git(t, dir, "ls-tree", "--name-only", tree), "a.txt\n", "the files of the snapshot")
		})

		t.Run("leaves HEAD and the index unchanged", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			head := vcstest.Commit(t, dir, "first")
			write(t, dir, "a.txt", "changed\n")
			write(t, dir, "new.txt", "new\n")
			var err error
			assert.Pure(t, func() string { return vcstest.Git(t, dir, "status", "--porcelain") },
				func() { _, err = vcs.Snapshot(t.Context(), dir) }, "the status")
			assert.NoError(t, err, "Snapshot")
			assert.Equal(t, strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD")), head, "HEAD")
		})

		t.Run("returns ErrGit for a directory outside a working tree", func(t *testing.T) {
			t.Parallel()
			_, err := vcs.Snapshot(t.Context(), t.TempDir())
			assert.ErrorIs(t, err, vcs.ErrGit, "Snapshot")
		})
	})

	t.Run("Restore", func(t *testing.T) {
		t.Parallel()

		t.Run("gives each path its content in the commit and removes a path that the commit does not have",
			func(t *testing.T) {
				t.Parallel()
				dir := vcstest.Repository(t, files.Tree{
					"keep.txt": files.Text(
						"keep\n",
					),
					"changed.txt": files.Text("old\n"),
					"gone/b.txt":  files.Text("b\n"),
				})
				vcstest.Commit(t, dir, "first")
				commit, err := vcs.Snapshot(t.Context(), dir)
				assert.NoError(t, err, "Snapshot")
				write(t, dir, "changed.txt", "new\n")
				assert.NoError(t, os.RemoveAll(filepath.Join(dir, "gone")), "RemoveAll of gone")
				write(t, dir, "added.txt", "added\n")
				err = vcs.Restore(
					t.Context(),
					dir,
					commit,
					[]string{"changed.txt", "gone/b.txt", "added.txt", "absent.txt"},
				)
				assert.NoError(t, err, "Restore")
				files.Contains(t, os.DirFS(dir), files.Tree{
					"keep.txt":    files.Text("keep\n"),
					"changed.txt": files.Text("old\n"),
					"gone/b.txt":  files.Text("b\n"),
				}, "the working tree")
				files.Absent(t, filepath.Join(dir, "added.txt"), "the file that the commit does not have")
			})

		t.Run("restores an executable file and a symbolic link", func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == windows {
				t.Skip("Windows stores no execute bit of a file")
			}
			dir := vcstest.Repository(
				t,
				files.Tree{"run.sh": files.Executable("#!/bin/sh\n"), "link": files.Link("run.sh")},
			)
			vcstest.Commit(t, dir, "first")
			commit, err := vcs.Snapshot(t.Context(), dir)
			assert.NoError(t, err, "Snapshot")
			assert.NoError(t, os.Remove(filepath.Join(dir, "run.sh")), "Remove of run.sh")
			assert.NoError(t, os.Remove(filepath.Join(dir, "link")), "Remove of link")
			assert.NoError(t, vcs.Restore(t.Context(), dir, commit, []string{"run.sh", "link"}), "Restore")
			files.HasContent(t, filepath.Join(dir, "run.sh"), "#!/bin/sh\n", "the content of run.sh")
			info, err := os.Stat(filepath.Join(dir, "run.sh"))
			assert.NoError(t, err, "Stat of run.sh")
			assert.NotEqual(t, info.Mode().Perm()&0o100, fs.FileMode(0), "the executable bit of run.sh")
			files.LinksTo(t, filepath.Join(dir, "link"), "run.sh", "the link")
		})

		t.Run("returns nil for no path", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, vcs.Restore(t.Context(), t.TempDir(), "HEAD", nil), "Restore")
		})

		t.Run("returns ErrGit for a commit that the repository does not have", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			vcstest.Commit(t, dir, "first")
			err := vcs.Restore(t.Context(), dir, strings.Repeat("0", 40), []string{"a.txt"})
			assert.ErrorIs(t, err, vcs.ErrGit, "Restore")
		})

		t.Run("returns ErrGit for a path that is a directory in the commit", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"dir/a.txt": files.Text("a\n")})
			commit := vcstest.Commit(t, dir, "first")
			err := vcs.Restore(t.Context(), dir, commit, []string{"dir"})
			assert.ErrorIs(t, err, vcs.ErrGit, "Restore")
		})

		t.Run("returns the error of a directory that a file is in the way of", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"dir/a.txt": files.Text("a\n")})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, os.RemoveAll(filepath.Join(dir, "dir")), "RemoveAll of dir")
			write(t, dir, "dir", "a file\n")
			err := vcs.Restore(t.Context(), dir, commit, []string{"dir/a.txt"})
			assert.HasError(t, err, "Restore")
		})

		t.Run("returns the error of a removal of a path that the commit does not have", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, os.MkdirAll(filepath.Join(dir, "full", "inner"), 0o755), "MkdirAll of full")
			err := vcs.Restore(t.Context(), dir, commit, []string{"full"})
			assert.HasError(t, err, "Restore")
		})

		t.Run("returns the error of a removal before a symbolic link", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"target": files.Text("t\n"), "link": files.Link("target")})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, os.Remove(filepath.Join(dir, "link")), "Remove of link")
			assert.NoError(t, os.MkdirAll(filepath.Join(dir, "link", "inner"), 0o755), "MkdirAll of link")
			err := vcs.Restore(t.Context(), dir, commit, []string{"link"})
			assert.HasError(t, err, "Restore")
		})
	})
}

// TestSnapshotEnv runs the cases of Snapshot that change the environment of the process, one at a
// time.
func TestSnapshotEnv(t *testing.T) {
	t.Run("Snapshot", func(t *testing.T) {
		t.Run("returns the error of a temporary directory that it cannot create", func(t *testing.T) {
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			// os.TempDir reads TMPDIR on Linux and macOS, and TMP and TEMP on Windows.
			for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
				t.Setenv(name, filepath.Join(dir, "absent"))
			}
			_, err := vcs.Snapshot(t.Context(), dir)
			assert.ErrorIs(t, err, fs.ErrNotExist, "Snapshot")
		})
	})
}
