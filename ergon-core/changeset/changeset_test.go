// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package changeset_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/version"
)

// id is the ID of the changesets of the cases.
const id = "add-the-unit-type-3f2a"

// written is a changeset as changesets writes it, with two packages.
const written = `---
"go.dokimi.dev/ergon/core": minor
"dokimi-assert": patch
---

Add the Unit type to the vocabulary.
`

func TestChangeset(t *testing.T) {
	t.Parallel()

	t.Run("Changeset", func(t *testing.T) {
		t.Parallel()

		t.Run("encodes to the JSON of changesets without the line of a release", func(t *testing.T) {
			t.Parallel()
			c := changeset.Changeset{ID: id, Summary: "Fix it.", Releases: []changeset.Release{
				{Name: "a", Bump: version.BumpPatch, Line: 2},
			}}
			got, err := json.Marshal(c)
			assert.NoError(t, err, "Marshal")
			assert.Equal(t, string(got), `{"id":"`+id+`","summary":"Fix it.","releases":[{"name":"a","type":"patch"}]}`,
				"the JSON")
		})
	})

	t.Run("Parse", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give string
			want changeset.Changeset
		}{
			{
				name: "returns the releases and the summary of a file that changesets writes",
				give: written,
				want: changeset.Changeset{ID: id, Releases: []changeset.Release{
					{Name: "go.dokimi.dev/ergon/core", Bump: version.BumpMinor, Line: 2},
					{Name: "dokimi-assert", Bump: version.BumpPatch, Line: 3},
				}, Summary: "Add the Unit type to the vocabulary."},
			},
			{
				name: "returns a name in single quotes without the quotes",
				give: "---\n'dokimi-assert': major\n---\n\nBreak it.\n",
				want: changeset.Changeset{ID: id, Releases: []changeset.Release{
					{Name: "dokimi-assert", Bump: version.BumpMajor, Line: 2},
				}, Summary: "Break it."},
			},
			{
				name: "returns an unquoted name up to the last colon",
				give: "---\ndev.dokimi:assert-core: none\n---\n",
				want: changeset.Changeset{ID: id, Releases: []changeset.Release{
					{Name: "dev.dokimi:assert-core", Bump: version.BumpNone, Line: 2},
				}},
			},
			{
				name: "returns a quoted name with the prefix of a toolchain",
				give: "---\n\"python:dokimi-assert\": patch\n---\nFix it.",
				want: changeset.Changeset{ID: id, Releases: []changeset.Release{
					{Name: "python:dokimi-assert", Bump: version.BumpPatch, Line: 2},
				}, Summary: "Fix it."},
			},
			{
				name: "returns no release for an empty front matter",
				give: "---\n---\n\nDocument the release.\n",
				want: changeset.Changeset{ID: id, Summary: "Document the release."},
			},
			{
				name: "counts the empty lines of the front matter in the line of a release",
				give: "---\n\n\"a\": patch\n\n---\n",
				want: changeset.Changeset{ID: id, Releases: []changeset.Release{
					{Name: "a", Bump: version.BumpPatch, Line: 3},
				}},
			},
			{
				name: "reads the line endings of Windows",
				give: "---\r\n\"a\": minor\r\n---\r\n\r\nFirst.\r\nSecond.\r\n",
				want: changeset.Changeset{
					ID: id, Releases: []changeset.Release{{Name: "a", Bump: version.BumpMinor, Line: 2}},
					Summary: "First.\nSecond.",
				},
			},
			{
				name: "returns a summary of several paragraphs",
				give: "---\n\"a\": patch\n---\n\nFirst.\n\n- second\n",
				want: changeset.Changeset{
					ID: id, Releases: []changeset.Release{{Name: "a", Bump: version.BumpPatch, Line: 2}},
					Summary: "First.\n\n- second",
				},
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := changeset.Parse(id, []byte(tt.give))
				assert.NoError(t, err, "Parse")
				assert.Equal(t, got, tt.want, "the changeset")
			})
		}

		invalid := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns ErrInvalid for a file without front matter",
				give: "Add the Unit type.\n",
				want: id + ".md:1: the file does not open with the line ---",
			},
			{
				name: "returns ErrInvalid for front matter that does not close",
				give: "---\n\"a\": patch\n",
				want: id + ".md:3: the front matter does not close",
			},
			{
				name: "returns ErrInvalid for a line without a colon",
				give: "---\n\"a\" patch\n---\n",
				want: id + `.md:2: the line "\"a\" patch", which is no name and level`,
			},
			{
				name: "returns ErrInvalid for a line without a name",
				give: "---\n\"\": patch\n---\n",
				want: id + `.md:2: the line "\"\": patch", which names no package`,
			},
			{
				name: "returns ErrInvalid for a level that is none of the four",
				give: "---\n\"a\": huge\n---\n",
				want: id + `.md:2: version: invalid version: level "huge"`,
			},
			{
				name: "returns ErrInvalid for a package that the front matter names twice",
				give: "---\n\"a\": patch\n\"b\": patch\n'a': minor\n---\n",
				want: id + `.md:4: the package "a", which the front matter names twice`,
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := changeset.Parse(id, []byte(tt.give))
				assert.ErrorIs(t, err, changeset.ErrInvalid, "Parse")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}
	})

	t.Run("Format", func(t *testing.T) {
		t.Parallel()

		t.Run("writes a changeset as changesets writes it", func(t *testing.T) {
			t.Parallel()
			c := changeset.Changeset{ID: id, Releases: []changeset.Release{
				{Name: "go.dokimi.dev/ergon/core", Bump: version.BumpMinor},
				{Name: "dokimi-assert", Bump: version.BumpPatch},
			}, Summary: "Add the Unit type to the vocabulary."}
			assert.Equal(t, string(changeset.Format(&c)), written, "the file")
		})

		t.Run("writes an empty front matter for a changeset without releases", func(t *testing.T) {
			t.Parallel()
			c := changeset.Changeset{ID: id, Summary: "Document the release."}
			assert.Equal(t, string(changeset.Format(&c)), "---\n---\n\nDocument the release.\n", "the file")
		})

		t.Run("writes what Parse reads back", func(t *testing.T) {
			t.Parallel()
			c := changeset.Changeset{ID: id, Releases: []changeset.Release{
				{Name: "dev.dokimi:assert-core", Bump: version.BumpMajor, Line: 2},
				{Name: "rust:dokimi-assert", Bump: version.BumpNone, Line: 3},
			}, Summary: "First.\n\nSecond."}
			assert.RoundTrip(t, func(c changeset.Changeset) ([]byte, error) { return changeset.Format(&c), nil },
				func(data []byte) (changeset.Changeset, error) { return changeset.Parse(id, data) }, c,
				"Parse undoes Format")
		})
	})
}
