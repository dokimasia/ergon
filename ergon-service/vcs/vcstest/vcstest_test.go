// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcstest_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// TestMain runs the tests without the variables and the configuration of git of the environment.
func TestMain(m *testing.M) {
	vcstest.Isolate()
	os.Exit(m.Run())
}

// TestVcstestEnv runs serially, because it changes the environment of the process.
func TestVcstestEnv(t *testing.T) {
	t.Run("Isolate", func(t *testing.T) {
		t.Run("removes the variables of git and sets the configuration to none", func(t *testing.T) {
			t.Setenv("GIT_DIR", "/elsewhere/.git")
			t.Setenv("GIT_INDEX_FILE", "/elsewhere/index")
			vcstest.Isolate()
			_, dir := os.LookupEnv("GIT_DIR")
			assert.False(t, dir, "GIT_DIR is set")
			_, index := os.LookupEnv("GIT_INDEX_FILE")
			assert.False(t, index, "GIT_INDEX_FILE is set")
			assert.Equal(t, os.Getenv("GIT_CONFIG_GLOBAL"), os.DevNull, "GIT_CONFIG_GLOBAL")
			assert.Equal(t, os.Getenv("GIT_CONFIG_NOSYSTEM"), "1", "GIT_CONFIG_NOSYSTEM")
		})

		t.Run("turns off the automatic maintenance that a commit starts in the background", func(t *testing.T) {
			vcstest.Isolate()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			assert.Equal(t, strings.TrimSpace(vcstest.Git(t, dir, "config", "maintenance.auto")), "false",
				"the configuration of the automatic maintenance")
			t.Setenv("GIT_TRACE", "1")
			trace := vcstest.Git(t, dir, "commit", "--quiet", "--allow-empty", "--message", "first")
			assert.NotContains(t, trace, "maintenance", "the commands that the commit starts")
		})
	})
}

func TestVcstest(t *testing.T) {
	t.Parallel()

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a working tree that tracks the files of track", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{
				"a.txt": files.Text("a\n"),
				"b.txt": files.Text("b\n"),
			}, "a.txt")
			out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "ls-files").Output()
			assert.NoError(t, err, "git ls-files")
			assert.Equal(t, strings.TrimSpace(string(out)), "a.txt", "the tracked files")
		})

		t.Run("returns a working tree without a tracked file", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "ls-files").Output()
			assert.NoError(t, err, "git ls-files")
			assert.Empty(t, string(out), "the tracked files")
		})
	})

	t.Run("Git", func(t *testing.T) {
		t.Parallel()

		t.Run("runs git in the working tree", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			vcstest.Git(t, dir, "add", "a.txt")
			out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "ls-files").Output()
			assert.NoError(t, err, "git ls-files")
			assert.Equal(t, strings.TrimSpace(string(out)), "a.txt", "the tracked files")
		})

		t.Run("returns the output of git", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")}, "a.txt")
			assert.Equal(t, vcstest.Git(t, dir, "ls-files"), "a.txt\n", "the output")
		})
	})

	t.Run("Commit", func(t *testing.T) {
		t.Parallel()

		t.Run("commits every change and returns the commit", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.txt": files.Text("a\n")})
			commit := vcstest.Commit(t, dir, "add a")
			assert.Matches(t, commit, "^[0-9a-f]{40}$", "the commit")
			assert.Equal(t, vcstest.Git(t, dir, "log", "--format=%H %an %s"), commit+" Test add a\n", "the log")
			assert.Empty(t, vcstest.Git(t, dir, "status", "--porcelain"), "the changes after the commit")
		})

		t.Run("commits a working tree without changes", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			first := vcstest.Commit(t, dir, "first")
			assert.NotEqual(t, vcstest.Commit(t, dir, "second"), first, "the second commit")
		})
	})
}
