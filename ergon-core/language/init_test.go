// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package language_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workspace"
)

// config pins the path of .ergon.yaml.
const config = ".ergon.yaml"

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("Config", func(t *testing.T) {
		t.Parallel()

		t.Run("is .ergon.yaml at the root of the repository", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, language.Config, config, "Config")
		})
	})

	t.Run("ErrInvalidLocal", func(t *testing.T) {
		t.Parallel()

		t.Run("starts its text with the name of the package", func(t *testing.T) {
			t.Parallel()
			assert.HasPrefix(t, language.ErrInvalidLocal.Error(), "language: ", "the text of ErrInvalidLocal")
		})
	})

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give language.Repository
				want bool
			}{
				{name: "reports true for an owner and a name", give: "dokimasia/ergon", want: true},
				{name: "reports true for an owner with a hyphen", give: "stealth-scale/agents", want: true},
				{name: "reports true for a name with a dot and an underscore", give: "a/go.dokimi_dev", want: true},
				{name: "reports false for the empty repository", give: "", want: false},
				{name: "reports false for a name without an owner", give: "ergon", want: false},
				{name: "reports false for an owner that starts with a hyphen", give: "-a/b", want: false},
				{name: "reports false for an owner with a dot", give: "a.b/c", want: false},
				{name: "reports false for an empty name", give: "a/", want: false},
				{name: "reports false for a name with a slash", give: "a/b/c", want: false},
				{name: "reports false for a name with a space", give: "a/b c", want: false},
				{name: "reports false for a name after a newline", give: "a/b\nc", want: false},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, tt.give.Valid(), tt.want, "Valid")
				})
			}
		})
	})

	t.Run("Answers", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give func(*language.Answers)
			}{
				{name: "returns nil for the answers of a repository", give: func(*language.Answers) {}},
				{
					name: "returns nil for the license with parameters",
					give: func(a *language.Answers) { a.License = spdx.BUSL11 },
				},
				{name: "returns nil for the first year", give: func(a *language.Answers) { a.Year = 1 }},
				{name: "returns nil for the last year", give: func(a *language.Answers) { a.Year = 9999 }},
				{name: "returns nil for no language", give: func(a *language.Answers) { a.Languages = nil }},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					a := answers()
					tt.give(&a)
					assert.NoError(t, a.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give func(*language.Answers)
			}{
				{name: "returns ErrInvalidAnswer for an empty name", give: func(a *language.Answers) { a.Name = "" }},
				{
					name: "returns ErrInvalidAnswer for a name of spaces",
					give: func(a *language.Answers) { a.Name = "  " },
				},
				{
					name: "returns ErrInvalidAnswer for an owner that spans lines",
					give: func(a *language.Answers) { a.Owner = "A\nB" },
				},
				{
					name: "returns ErrInvalidAnswer for a security contact that spans lines",
					give: func(a *language.Answers) { a.SecurityContact = "a@b.c\r" },
				},
				{
					name: "returns ErrInvalidAnswer for a license that ergon does not support",
					give: func(a *language.Answers) { a.License = "AMD-newlib" },
				},
				{name: "returns ErrInvalidAnswer for the year 0", give: func(a *language.Answers) { a.Year = 0 }},
				{
					name: "returns ErrInvalidAnswer for the year 10000",
					give: func(a *language.Answers) { a.Year = 10000 },
				},
				{
					name: "returns ErrInvalidAnswer for a repository without an owner",
					give: func(a *language.Answers) { a.Repository = "ergon" },
				},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					a := answers()
					tt.give(&a)
					assert.ErrorIs(t, a.Validate(), language.ErrInvalidAnswer, "Validate")
				})
			}
		})
	})
}

// answers returns new valid answers of the cases: a repository of Go under MIT.
func answers() language.Answers {
	return language.Answers{
		Name:            "ergon",
		Owner:           "Dokimasia B.V.",
		License:         spdx.MIT,
		Repository:      "dokimasia/ergon",
		SecurityContact: "security@dokimi.dev",
		Languages:       []workspace.Language{"go"},
		Year:            2026,
	}
}
