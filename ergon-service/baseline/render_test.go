// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/baseline"
)

func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("New", func(t *testing.T) {
			t.Parallel()

			text := []byte("text\n")
			tests := []struct {
				name string
				file language.File
			}{
				{
					name: "returns ErrInvalidFile for a file with Content and Fragment",
					file: language.File{Path: "a.txt", Class: language.Managed, Content: text, Fragment: text},
				},
				{
					name: "returns ErrInvalidFile for a file without Content and Fragment",
					file: language.File{Path: "a.txt", Class: language.Managed},
				},
				{
					name: "returns ErrInvalidFile for a path that is not clean",
					file: language.File{Path: "a/../b.txt", Class: language.Managed, Content: text},
				},
				{
					name: "returns ErrInvalidFile for an absolute path",
					file: language.File{Path: "/etc/passwd", Class: language.Managed, Content: text},
				},
				{
					name: "returns ErrInvalidFile for the path of the directory itself",
					file: language.File{Path: ".", Class: language.Managed, Content: text},
				},
				{
					name: "returns ErrInvalidFile for a path under .ergon",
					file: language.File{Path: ".ergon/local/a.txt", Class: language.Managed, Content: text},
				},
				{
					name: "returns ErrInvalidFile for the path .ergon",
					file: language.File{Path: ".ergon", Class: language.Managed, Content: text},
				},
				{
					name: "returns ErrInvalidFile for an invalid class",
					file: language.File{Path: "a.txt", Content: text},
				},
				{
					name: "returns ErrInvalidFile for a whole file that another producer renders",
					file: language.File{Path: license, Class: language.Managed, Content: text},
				},
				{
					name: "returns ErrInvalidFile for a fragment of a file that another producer renders whole",
					file: language.File{Path: license, Class: language.Managed, Fragment: text},
				},
				{
					name: "returns ErrInvalidFile for a whole file that another producer renders in fragments",
					file: language.File{Path: ignore, Class: language.Managed, Content: text},
				},
				{
					name: "returns ErrInvalidFile for a fragment of another class",
					file: language.File{Path: ignore, Class: language.Seeded, Fragment: text},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					r, err := baseline.Open(directory(t), catalog(t), version, common, renders(tt.file))
					assert.NoError(t, err, "Open")
					_, err = r.New(answers(), baseline.Options{})
					assert.ErrorIs(t, err, baseline.ErrInvalidFile, "New")
				})
			}

			t.Run("renders a file in a directory that another file of the producer shares", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				r, err := baseline.Open(root, catalog(t), version, common, renders(
					language.File{Path: "docs/a.md", Class: language.Seeded, Content: text},
					language.File{Path: "docs/b.md", Class: language.Seeded, Content: text},
				))
				assert.NoError(t, err, "Open")
				_, err = r.New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Equal(t, content(t, root, "docs/b.md"), "text\n", "the second file")
			})
		})
	})
}

// renders returns a base producer named extra that renders files.
func renders(files ...language.File) baseline.Producer {
	return baseline.Producer{Name: "extra", Initializer: renderer(func(*language.Answers) ([]language.File, error) {
		return files, nil
	})}
}
