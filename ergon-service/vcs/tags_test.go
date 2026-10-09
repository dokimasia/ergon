// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The arguments with which git starts its ssh signing program: ssh-keygen -Y sign -n git -f <key>
// <file>.
const (
	signFlag    = "-Y"
	signCommand = "sign"
)

// The signatures of the cases that sign a tag.
const (
	// pin is the PIN that the signing program of the cases reads from its standard input.
	pin = "123456"

	// signature is the signature that the signing program of the cases writes.
	signature = "-----BEGIN SSH SIGNATURE-----\nU1NIU0lH\n-----END SSH SIGNATURE-----\n"

	// noPIN is the error of the signing program of the cases for a standard input without pin.
	noPIN = "sign: no PIN on the standard input"
)

// newTag is the mark of git push for a tag that the remote did not have.
const newTag = "[new tag]"

// signFailed is the end of the error of a git tag whose signing program fails.
const signFailed = "exit status 128"

func TestTags(t *testing.T) {
	t.Parallel()

	t.Run("Tags", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the commit of an annotated and of a lightweight tag", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{"a.go": files.Text("package a\n")})
			first := vcstest.Commit(t, dir, "first")
			second := vcstest.Commit(t, dir, "second")
			assert.NoError(t, vcs.Tag(t.Context(), dir, vcs.Terminal{}, "core/v1.0.0", first, "Release core 1.0.0."),
				"Tag")
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
			assert.NoError(t, vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, message), "Tag")
			assert.Equal(t, strings.TrimSpace(vcstest.Git(t, dir, "cat-file", "-t", "v1.0.0")), "tag", "the object")
			assert.Equal(t, vcstest.Git(t, dir, "tag", "--list", "--format=%(contents)", "v1.0.0"), message+"\n\n",
				"the annotation with its headings and its newline, and the newline of git tag --list")
		})

		t.Run("ends the annotation with a newline", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, "One."), "Tag")
			assert.HasSuffix(t, vcstest.Git(t, dir, "cat-file", "-p", "v1.0.0"), "\n\nOne.\n", "the tag object")
		})

		t.Run("keeps the newline at the end of a message", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, "Two.\n"), "Tag")
			assert.HasSuffix(t, vcstest.Git(t, dir, "cat-file", "-p", "v1.0.0"), "\n\nTwo.\n", "the tag object")
		})

		t.Run("returns ErrGit for a tag that exists", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, "First."), "the first Tag")
			err := vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, "Again.")
			assert.ErrorIs(t, err, vcs.ErrGit, "the second Tag")
		})
	})

	t.Run("LightTag", func(t *testing.T) {
		t.Parallel()

		t.Run("creates a lightweight tag at the commit", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			first := vcstest.Commit(t, dir, "first")
			vcstest.Commit(t, dir, "second")
			assert.NoError(t, vcs.LightTag(t.Context(), dir, "lint/v0.1.0", first), "LightTag")
			assert.Equal(t, strings.TrimSpace(vcstest.Git(t, dir, "cat-file", "-t", "lint/v0.1.0")), "commit",
				"the object of the tag")
			assert.Equal(t, strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "lint/v0.1.0")), first, "the commit")
		})

		t.Run("creates an unsigned tag in a repository that signs its tags", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			vcstest.Git(t, dir, "config", "tag.gpgSign", "true")
			assert.NoError(t, vcs.LightTag(t.Context(), dir, "v0.1.0", commit), "LightTag")
			assert.Equal(t, strings.TrimSpace(vcstest.Git(t, dir, "cat-file", "-t", "v0.1.0")), "commit",
				"the object of the tag")
		})

		t.Run("returns ErrGit for a tag that exists", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.LightTag(t.Context(), dir, "v0.1.0", commit), "the first LightTag")
			assert.ErrorIs(t, vcs.LightTag(t.Context(), dir, "v0.1.0", commit), vcs.ErrGit, "the second LightTag")
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
			assert.NoError(t, vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, "First."), "Tag")
			assert.NoError(t, vcs.Push(t.Context(), dir, vcs.Terminal{}, remote, "refs/tags/v1.0.0"), "Push")
			got, err := vcs.Tags(t.Context(), remote)
			assert.NoError(t, err, "Tags of the remote")
			assert.Equal(t, got, map[string]string{"v1.0.0": commit}, "the tags of the remote")
		})

		t.Run("writes the standard error of git to the Stderr of the terminal", func(t *testing.T) {
			t.Parallel()
			remote := t.TempDir()
			vcstest.Git(t, remote, "init", "--quiet", "--bare")
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, "First."), "Tag")
			var stderr strings.Builder
			err := vcs.Push(t.Context(), dir, vcs.Terminal{Stderr: &stderr}, remote, "refs/tags/v1.0.0")
			assert.NoError(t, err, "Push")
			assert.Contains(t, stderr.String(), newTag, "the standard error of git push")
		})

		t.Run("returns ErrGit and pushes no ref when one ref does not exist", func(t *testing.T) {
			t.Parallel()
			remote := t.TempDir()
			vcstest.Git(t, remote, "init", "--quiet", "--bare")
			dir := vcstest.Repository(t, files.Tree{})
			commit := vcstest.Commit(t, dir, "first")
			assert.NoError(t, vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, "First."), "Tag")
			err := vcs.Push(t.Context(), dir, vcs.Terminal{}, remote, "refs/tags/v1.0.0", "refs/tags/v9.9.9")
			assert.ErrorIs(t, err, vcs.ErrGit, "Push")
			got, err := vcs.Tags(t.Context(), remote)
			assert.NoError(t, err, "Tags of the remote")
			assert.Empty(t, got, "the tags of the remote")
		})
	})
}

// TestTagsProcess runs the cases whose git starts the test binary as its signing program, one at a
// time, because two processes of a coverage run that exit in the same nanosecond write one
// coverage file.
func TestTagsProcess(t *testing.T) {
	t.Run("Tag", func(t *testing.T) {
		t.Run("signs the tag with the PIN that the signing program reads from Stdin", func(t *testing.T) {
			dir, commit := signing(t)
			terminal := vcs.Terminal{Stdin: strings.NewReader(pin + "\n")}
			assert.NoError(t, vcs.Tag(t.Context(), dir, terminal, "v1.0.0", commit, "First."), "Tag")
			assert.HasSuffix(t, vcstest.Git(t, dir, "cat-file", "-p", "v1.0.0"), "\n\nFirst.\n"+signature,
				"the tag object, with the signature on a line of its own")
		})

		t.Run("returns ErrGit with the standard error of git without a terminal", func(t *testing.T) {
			dir, commit := signing(t)
			err := vcs.Tag(t.Context(), dir, vcs.Terminal{}, "v1.0.0", commit, "First.")
			assert.ErrorIs(t, err, vcs.ErrGit, "Tag")
			assert.Contains(t, err.Error(), noPIN, "the error")
		})

		t.Run("writes the standard error of git to the Stderr of the terminal", func(t *testing.T) {
			dir, commit := signing(t)
			var stderr strings.Builder
			err := vcs.Tag(t.Context(), dir, vcs.Terminal{Stderr: &stderr}, "v1.0.0", commit, "First.")
			assert.ErrorIs(t, err, vcs.ErrGit, "Tag")
			assert.Contains(t, stderr.String(), noPIN, "the standard error of git tag")
			assert.HasSuffix(t, err.Error(), signFailed, "the error, which states no standard error")
		})
	})
}

// sign is the signing program of the cases, which git starts as ssh-keygen -Y sign -n git -f <key>
// <file>. It reads a line from its standard input, as ssh-keygen reads the PIN of a security key,
// and writes signature to <file>.sig when the line is pin. It returns 1 with noPIN on its standard
// error for another line, and 1 when it cannot write the signature.
func sign(args []string) int {
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.TrimSpace(line) != pin {
		fmt.Fprintln(os.Stderr, noPIN)
		return 1
	}
	if err := os.WriteFile(args[len(args)-1]+".sig", []byte(signature), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "sign:", err)
		return 1
	}
	return 0
}

// signing returns a working tree of [vcstest.Repository] with one commit, whose git signs each tag
// with the test binary as its ssh signing program, as [sign] signs, and the commit. It stops the
// test when git fails.
func signing(tb testing.TB) (string, string) {
	tb.Helper()
	self, err := os.Executable()
	assert.NoError(tb, err, "Executable")
	dir := vcstest.Repository(tb, files.Tree{})
	vcstest.Git(tb, dir, "config", "gpg.format", "ssh")
	vcstest.Git(tb, dir, "config", "gpg.ssh.program", self)
	vcstest.Git(tb, dir, "config", "user.signingKey", filepath.Join(dir, "key"))
	vcstest.Git(tb, dir, "config", "tag.gpgSign", "true")
	return dir, vcstest.Commit(tb, dir, "first")
}
