// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package language_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("Job", func(t *testing.T) {
		t.Parallel()

		t.Run("replaces each placeholder of the checkout with the pinned action", func(t *testing.T) {
			t.Parallel()
			got := language.Job("- uses: {{checkout}}\n- uses: {{checkout}}\n")
			assert.Equal(t, got, "- uses: "+language.Checkout+"\n- uses: "+language.Checkout+"\n", "the job")
		})

		t.Run("replaces the placeholder of the runners with the three systems", func(t *testing.T) {
			t.Parallel()
			got := language.Job("os: {{runners}}\n")
			assert.Equal(t, got, "os: [ubuntu-26.04, macos-26, windows-2025]\n", "the matrix of the job")
		})

		t.Run("replaces the placeholder of the Go release with Go", func(t *testing.T) {
			t.Parallel()
			got := language.Job("go-version: \"{{go-version}}\"\n")
			assert.Equal(t, got, "go-version: \"1.27.1\"\n", "the Go release of the job")
		})
	})

	t.Run("Class", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give language.Class
				want bool
			}{
				{name: "reports false for the zero value", give: 0, want: false},
				{name: "reports true for Managed", give: language.Managed, want: true},
				{name: "reports true for Configured", give: language.Configured, want: true},
				{name: "reports true for Seeded", give: language.Seeded, want: true},
				{name: "reports false for the value after Seeded", give: language.Seeded + 1, want: false},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, tt.give.Valid(), tt.want, "Valid")
				})
			}
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

	t.Run("Fixed", func(t *testing.T) {
		t.Parallel()

		t.Run("Files", func(t *testing.T) {
			t.Parallel()

			t.Run("returns its files for any answers", func(t *testing.T) {
				t.Parallel()
				fixed := language.Fixed{{Path: language.GitIgnore, Class: language.Managed, Fragment: []byte("bin/\n")}}
				files, err := fixed.Files(&language.Answers{Name: "any"})
				assert.NoError(t, err, "Files")
				assert.Equal(t, files, []language.File(fixed), "the files")
			})

			t.Run("returns a copy of the list", func(t *testing.T) {
				t.Parallel()
				fixed := language.Fixed{{Path: language.GitIgnore, Class: language.Managed, Fragment: []byte("bin/\n")}}
				files, err := fixed.Files(nil)
				assert.NoError(t, err, "Files")
				files[0].Path = language.Makefile
				assert.Equal(t, fixed[0].Path, language.GitIgnore, "the path of the fixed file")
			})
		})
	})
}
