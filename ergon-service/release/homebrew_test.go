// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"path"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/release"
)

// tapRepo is the tap of the cases.
const tapRepo = "dokimasia/homebrew-tap"

// errPut is the error of a tap that refuses a commit.
var errPut = errors.New("put failed")

// put is a commit of a cask that a tap received.
type put struct {
	// repo is the repository of the tap.
	repo string

	// path is the path of the cask in the tap.
	path string

	// message is the message of the commit.
	message string

	// content is the content of the cask.
	content string
}

// tap is a [release.TapForge] that records each commit, and returns fail.
type tap struct {
	// fail is the error of PutFile, or nil.
	fail error

	// puts are the commits that PutFile received, in their order.
	puts []put
}

var _ release.TapForge = (*tap)(nil)

// PutFile records the commit, and returns fail.
func (f *tap) PutFile(_ context.Context, repo, path, message string, content []byte) error {
	if f.fail != nil {
		return f.fail
	}
	f.puts = append(f.puts, put{repo: repo, path: path, message: message, content: string(content)})
	return nil
}

func TestHomebrew(t *testing.T) {
	t.Parallel()

	t.Run("Homebrew", func(t *testing.T) {
		t.Parallel()

		t.Run("commits each cask of the plan to the tap with its name and version", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{
				path.Join(language.CasksDir, "v0.6.0", "ergon.rb"):          files.Text("cask ergon\n"),
				path.Join(language.CasksDir, "lint", "v0.2.0", "lint.rb"):   files.Text("cask lint\n"),
				path.Join(language.CasksDir, "lint", "v0.2.0", "README.md"): files.Text("not a cask\n"),
				path.Join(language.CasksDir, "v0.5.0", "old.rb"):            files.Text("cask of no entry\n"),
			})
			plan := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{
				{tagOnly(t, "lint", "lint/v0.2.0", "0.2.0")},
				{tagOnly(t, "ergon", "v0.6.0", "0.6.0"), tagOnly(t, "core", "core/v0.4.1", "0.4.1")},
			}}
			f := &tap{}
			got, err := release.Homebrew(t.Context(), f, tapRepo, plan, dir)
			assert.NoError(t, err, "Homebrew")
			assert.Equal(t, got, []string{"Casks/lint.rb", "Casks/ergon.rb"}, "the casks")
			assert.Equal(t, f.puts, []put{
				{repo: tapRepo, path: "Casks/lint.rb", message: "lint 0.2.0", content: "cask lint\n"},
				{repo: tapRepo, path: "Casks/ergon.rb", message: "ergon 0.6.0", content: "cask ergon\n"},
			}, "the commits")
		})

		t.Run("commits nothing for a pack directory without casks", func(t *testing.T) {
			t.Parallel()
			plan := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{
				{tagOnly(t, "ergon", "v0.6.0", "0.6.0")},
			}}
			f := &tap{}
			got, err := release.Homebrew(t.Context(), f, tapRepo, plan, t.TempDir())
			assert.NoError(t, err, "Homebrew")
			assert.Empty(t, got, "the casks")
			assert.Empty(t, f.puts, "the commits")
		})

		t.Run("returns the error of the tap with the name of the cask", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{
				path.Join(language.CasksDir, "v0.6.0", "ergon.rb"): files.Text("cask ergon\n"),
			})
			plan := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{
				{tagOnly(t, "ergon", "v0.6.0", "0.6.0")},
			}}
			_, err := release.Homebrew(t.Context(), &tap{fail: errPut}, tapRepo, plan, dir)
			assert.ErrorIs(t, err, errPut, "Homebrew")
			assert.Contains(t, err.Error(), "commit the cask ergon to "+tapRepo, "the error")
		})

		t.Run("returns the error of a directory of casks that it cannot read", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{path.Join(language.CasksDir, "v0.6.0"): files.Text("a file\n")})
			plan := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{
				{tagOnly(t, "ergon", "v0.6.0", "0.6.0")},
			}}
			_, err := release.Homebrew(t.Context(), &tap{}, tapRepo, plan, dir)
			assert.HasError(t, err, "Homebrew")
			assert.Contains(t, err.Error(), "read the casks of v0.6.0", "the error")
		})
	})
}
