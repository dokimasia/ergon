// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"bytes"
	"io"
	"io/fs"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/service/release"
)

func TestChangesets(t *testing.T) {
	t.Parallel()

	t.Run("ReadChangesets", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the changesets in the order of their file names", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				".changeset/b-second.md": files.Text("---\n\"pkg-a\": minor\n---\n\nSecond.\n"),
				".changeset/a-first.md":  files.Text("---\n---\n\nFirst.\n"),
			})
			got, err := release.ReadChangesets(root)
			assert.NoError(t, err, "ReadChangesets")
			second := []changeset.Release{{Name: "pkg-a", Bump: minor, Line: 2}}
			assert.Equal(t, got, []changeset.Changeset{
				{ID: "a-first", Summary: "First."},
				{ID: "b-second", Summary: "Second.", Releases: second},
			}, "the changesets")
		})

		t.Run("skips the files that changesets reads as no changeset", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				".changeset/README.md":   files.Text("# Changesets\n"),
				".changeset/AGENTS.md":   files.Text("# Agents\n"),
				".changeset/CLAUDE.md":   files.Text("# Claude\n"),
				".changeset/GEMINI.md":   files.Text("# Gemini\n"),
				".changeset/.hidden.md":  files.Text("not a changeset\n"),
				".changeset/config.json": files.Text("{}\n"),
				".changeset/pre/x.md":    files.Text("not a changeset\n"),
			})
			got, err := release.ReadChangesets(root)
			assert.NoError(t, err, "ReadChangesets")
			assert.Empty(t, got, "the changesets")
		})

		t.Run("skips a README.md whose letters differ in case", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{".changeset/readme.MD": files.Text("# Changesets\n")})
			got, err := release.ReadChangesets(root)
			assert.NoError(t, err, "ReadChangesets")
			assert.Empty(t, got, "the changesets")
		})

		t.Run("returns an error that wraps fs.ErrNotExist for a repository without .changeset", func(t *testing.T) {
			t.Parallel()
			_, err := release.ReadChangesets(t.TempDir())
			assert.ErrorIs(t, err, fs.ErrNotExist, "ReadChangesets")
		})

		t.Run("returns the error of a changeset that it cannot read", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{".changeset/broken.md": files.Link("absent.md")})
			_, err := release.ReadChangesets(root)
			assert.ErrorIs(t, err, fs.ErrNotExist, "ReadChangesets")
		})

		t.Run("returns ErrInvalid with the file and the line of a changeset that does not parse", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{".changeset/bad.md": files.Text("---\n\"pkg-a\": huge\n---\n")})
			_, err := release.ReadChangesets(root)
			assert.ErrorIs(t, err, changeset.ErrInvalid, "ReadChangesets")
			assert.Contains(t, err.Error(), "bad.md:2", "the error")
		})
	})

	t.Run("AddChangeset", func(t *testing.T) {
		t.Parallel()

		added := changeset.Changeset{ID: "fix-a-3f9a", Summary: "Fix a.", Releases: []changeset.Release{
			{Name: "pkg-a", Bump: minor},
		}}

		t.Run("writes the changeset into the file of its ID", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			file, err := release.AddChangeset(root, &added)
			assert.NoError(t, err, "AddChangeset")
			assert.Equal(t, file, ".changeset/fix-a-3f9a.md", "the file")
			files.HasContent(t, filepath.Join(root, ".changeset", "fix-a-3f9a.md"), string(changeset.Format(&added)),
				"the changeset")
		})

		t.Run("returns an error that wraps fs.ErrExist for an ID that a file has", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{".changeset/fix-a-3f9a.md": files.Text("other\n")})
			_, err := release.AddChangeset(root, &added)
			assert.ErrorIs(t, err, fs.ErrExist, "AddChangeset")
			files.HasContent(t, filepath.Join(root, ".changeset", "fix-a-3f9a.md"), "other\n", "the file")
		})

		t.Run("returns an error for an ID that leaves the repository", func(t *testing.T) {
			t.Parallel()
			escaping := changeset.Changeset{ID: "../../escape"}
			_, err := release.AddChangeset(t.TempDir(), &escaping)
			assert.HasError(t, err, "AddChangeset")
			assert.Contains(t, err.Error(), "write ../escape.md", "the error")
		})

		t.Run("returns the error of a directory .changeset that it cannot create", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{".changeset": files.Text("a file\n")})
			_, err := release.AddChangeset(root, &added)
			assert.HasError(t, err, "AddChangeset")
			assert.Contains(t, err.Error(), "create .changeset", "the error")
		})

		t.Run("returns the error of a repository that does not exist", func(t *testing.T) {
			t.Parallel()
			_, err := release.AddChangeset(filepath.Join(t.TempDir(), "absent"), &added)
			assert.ErrorIs(t, err, fs.ErrNotExist, "AddChangeset")
		})
	})

	t.Run("ChangesetID", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			summary string
			want    string
		}{
			{
				name:    "returns the first words of the summary in lowercase and four hexadecimal digits",
				summary: "Add the Unit type to the vocabulary, and more.\n\nThe body.",
				want:    "add-the-unit-type-to-3f9a",
			},
			{name: "returns the digits alone for a summary without a letter", summary: "!!!", want: "3f9a"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := release.ChangesetID(tt.summary, bytes.NewReader([]byte{0x3f, 0x9a}))
				assert.NoError(t, err, "ChangesetID")
				assert.Equal(t, got, tt.want, "the ID")
			})
		}

		t.Run("returns the error of a source of random bytes that ends", func(t *testing.T) {
			t.Parallel()
			_, err := release.ChangesetID("Fix a.", bytes.NewReader([]byte{0x3f}))
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF, "ChangesetID")
		})
	})
}
