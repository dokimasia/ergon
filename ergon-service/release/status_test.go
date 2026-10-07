// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The files that a branch of the cases changes.
var (
	// indexA and indexB are a source file of pkg-a and of pkg-b.
	indexA = path.Join(packagesDir, "pkg-a", "index.js")
	indexB = path.Join(packagesDir, "pkg-b", "index.js")

	// fixA is a changeset that releases pkg-a at patch.
	fixA = path.Join(changeset.Dir, "fix-a"+changeset.Ext)
)

// fixAText is the text of fixA.
const fixAText = "---\n\"pkg-a\": patch\n---\n\nFix a.\n"

func TestStatus(t *testing.T) {
	t.Parallel()

	t.Run("NewStatus", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the packages whose files the branch changed", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			root, base := branch(t, s, map[string]string{indexB: "b\n"})
			got := status(t, s, root, base, nil)
			assert.Equal(t, got.Changed, []string{"pkg-b"}, "the changed packages")
		})

		t.Run("reports a changed package that no changeset of the branch names", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			root, base := branch(t, s, map[string]string{indexA: "a\n", indexB: "b\n", fixA: fixAText})
			got := status(t, s, root, base, nil)
			assert.Equal(t, got.Changed, []string{"pkg-a", "pkg-b"}, "the changed packages")
			assert.Equal(t, got.Uncovered, []string{"pkg-b"}, "the packages without a changeset")
		})

		t.Run("plans the changesets that the branch added", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			root, base := branch(t, s, map[string]string{fixA: fixAText})
			got := status(t, s, root, base, nil)
			assert.Equal(t, got.Plan.Releases, []release.Release{rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "fix-a")},
				"the releases")
		})

		t.Run("reports no package without a changeset after an empty changeset", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			docs := path.Join(changeset.Dir, "docs"+changeset.Ext)
			root, base := branch(t, s, map[string]string{indexB: "b\n", docs: "---\n---\n\nDocs.\n"})
			got := status(t, s, root, base, nil)
			assert.Empty(t, got.Uncovered, "the packages without a changeset")
		})

		t.Run("counts a file only when changedFilePatterns admit it", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			readme := path.Join(packagesDir, "pkg-a", "README.md")
			root, base := branch(t, s, map[string]string{readme: "# a\n", indexB: "b\n"})
			got := status(t, s, root, base, func(c *release.Config) { c.ChangedFilePatterns = []string{"**/*.js"} })
			assert.Equal(t, got.Changed, []string{"pkg-b"}, "the changed packages")
		})

		t.Run("counts a file for the package with the longest directory that contains it", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			s.pkgs[0].Dir = "."
			notes := path.Join(changeset.Dir, "notes.txt")
			root, base := branch(t, s, map[string]string{indexB: "b\n", notes: "x\n"})
			got := status(t, s, root, base, nil)
			assert.Equal(t, got.Changed, []string{"pkg-b"}, "the changed packages")
		})

		t.Run("leaves out a skipped package", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			root, base := branch(t, s, map[string]string{indexB: "b\n"})
			got := status(t, s, root, base, func(c *release.Config) { c.Ignore = []string{"pkg-b"} })
			assert.Empty(t, got.Changed, "the changed packages")
		})

		t.Run("reports a requirement that excludes the current version of its package", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^2.0.0")
			s.require("pkg-b", "left-pad", workspace.KindRuntime, "^9.0.0")
			root, base := branch(t, s, nil)
			got := status(t, s, root, base, nil)
			assert.Equal(t, got.Broken, []string{"pkg-b requires pkg-a ^2.0.0, which excludes its version 1.0.0"},
				"the broken requirements")
		})

		t.Run("returns ErrGit for a base that names no commit", func(t *testing.T) {
			t.Parallel()
			s := statusState(t)
			root, _ := branch(t, s, nil)
			cfg := defaultConfig()
			_, err := release.NewStatus(t.Context(), root, s.graph(t), &cfg, nil, "absent")
			assert.ErrorIs(t, err, vcs.ErrGit, "NewStatus")
		})

		t.Run(
			"returns ErrChangeset for a changeset of a package that the repository does not have",
			func(t *testing.T) {
				t.Parallel()
				s := statusState(t)
				fixZ := path.Join(changeset.Dir, "fix-z"+changeset.Ext)
				root, base := branch(t, s, map[string]string{fixZ: "---\n\"pkg-z\": patch\n---\n\nFix z.\n"})
				sets, err := release.ReadChangesets(root)
				assert.NoError(t, err, "ReadChangesets")
				cfg := defaultConfig()
				_, err = release.NewStatus(t.Context(), root, s.graph(t), &cfg, sets, base)
				assert.ErrorIs(t, err, release.ErrChangeset, "NewStatus")
			},
		)
	})
}

// statusState returns a state of pkg-a and pkg-b at 1.0.0 without changesets, for the test tb.
func statusState(tb testing.TB) *state {
	tb.Helper()
	s := blankState(tb)
	s.add("pkg-a", "1.0.0")
	s.add("pkg-b", "1.0.0")
	return s
}

// branch returns a working tree of git with a file in the directory of each package of s and a
// directory .changeset, all committed, and that commit, with the files of changes written after it
// as the changes of a branch, for the test tb.
func branch(tb testing.TB, s *state, changes map[string]string) (string, string) {
	tb.Helper()
	tree := files.Tree{path.Join(changeset.Dir, changeset.Readme): files.Text("# Changesets\n")}
	for _, p := range s.pkgs {
		tree[path.Join(p.Dir, manifestFile)] = files.Text(p.Name + "\n")
	}
	root := vcstest.Repository(tb, tree)
	base := vcstest.Commit(tb, root, "the base")
	for name, text := range changes {
		full := filepath.Join(root, filepath.FromSlash(name))
		assert.NoError(tb, os.MkdirAll(filepath.Dir(full), dirPerm), "the directory of "+name)
		assert.NoError(tb, os.WriteFile(full, []byte(text), versionPerm), "the write of "+name)
	}
	return root, base
}

// status returns the status of the branch of root since base, with the changesets of root, under
// the default configuration that change changes when it is not nil, for the test tb.
func status(tb testing.TB, s *state, root, base string, change func(c *release.Config)) release.Status {
	tb.Helper()
	sets, err := release.ReadChangesets(root)
	assert.NoError(tb, err, "ReadChangesets")
	cfg := defaultConfig()
	if change != nil {
		change(&cfg)
	}
	got, err := release.NewStatus(tb.Context(), root, s.graph(tb), &cfg, sets, base)
	assert.NoError(tb, err, "NewStatus")
	return got
}
