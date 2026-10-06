// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline/common"
	"go.yaml.in/yaml/v3"
)

// apacheDigest is the SHA-256 digest of the text of the Apache License 2.0 that
// https://www.apache.org/licenses/LICENSE-2.0.txt serves, pinned because the LICENSE must be that
// text byte for byte.
const apacheDigest = "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30"

// managed is the comment that opens a managed file with a comment syntax.
const managed = "Managed by ergon init. Add repository settings to .ergon/local/"

func TestCommon(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("Files", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the managed, seeded and configured common files", func(t *testing.T) {
				t.Parallel()
				classes := map[string]language.Class{}
				for path, f := range render(t, answers()) {
					classes[path] = f.Class
				}
				assert.Equal(t, classes, map[string]language.Class{
					".commitlint.yaml":            language.Managed,
					".editorconfig":               language.Managed,
					".gitattributes":              language.Managed,
					".gitignore":                  language.Managed,
					".markdownlint.yml":           language.Managed,
					".pre-commit-config.yaml":     language.Managed,
					"Makefile":                    language.Managed,
					"CODE_OF_CONDUCT.md":          language.Managed,
					"LICENSE":                     language.Managed,
					".changeset/config.json":      language.Seeded,
					".changeset/README.md":        language.Seeded,
					"README.md":                   language.Seeded,
					"CONTRIBUTING.md":             language.Seeded,
					"SECURITY.md":                 language.Seeded,
					"docs/README.md":              language.Seeded,
					"docs/adr/README.md":          language.Seeded,
					"docs/architecture/README.md": language.Seeded,
					"docs/rfc/README.md":          language.Seeded,
					"docs/roadmap/README.md":      language.Seeded,
					".ergon.yaml":                 language.Configured,
				}, "the classes of the common files")
			})

			t.Run("renders the shared files as fragments", func(t *testing.T) {
				t.Parallel()
				files := render(t, answers())
				for _, path := range []string{language.EditorConfig, language.GitAttributes, language.GitIgnore, language.Makefile} {
					assert.True(t, files[path].Content == nil && files[path].Fragment != nil, "the fragment of "+path)
				}
			})

			t.Run("declares the targets of the Makefile that each language extends", func(t *testing.T) {
				t.Parallel()
				makefile := text(render(t, answers())[language.Makefile])
				assert.Contains(t, makefile, ".PHONY: help fmt lint test audit check\n", "the phony targets")
				assert.Contains(t, makefile, "\naudit: ## Run the vulnerability scan of every language\n",
					"the target audit")
			})

			t.Run("opens each managed file that has a comment syntax with the managed comment", func(t *testing.T) {
				t.Parallel()
				files := render(t, answers())
				for path, f := range files {
					if f.Class != language.Managed || path == "LICENSE" {
						continue
					}
					first, _, _ := strings.Cut(text(f), "\n")
					assert.Contains(t, first, managed+path+" and run ergon init sync.", "the first line of "+path)
				}
			})

			t.Run("fills the answers into the files", func(t *testing.T) {
				t.Parallel()
				files := render(t, answers())
				assert.Contains(t, text(files["LICENSE"]), "Copyright (c) 2026 Dokimasia B.V.", "the LICENSE")
				assert.Contains(t, text(files["CODE_OF_CONDUCT.md"]), "for enforcement at security@example.com.",
					"the CODE_OF_CONDUCT.md")
				assert.Contains(t, text(files["SECURITY.md"]), "https://github.com/dokimasia/demo/security",
					"the SECURITY.md")
				assert.Contains(t, text(files["README.md"]), "# demo\n", "the README.md")
				assert.Contains(t, text(files["README.md"]), "demo is licensed under MIT.", "the README.md")
				assert.NotContains(t, text(files["README.md"])+text(files["SECURITY.md"]), "{{", "the filled files")
			})

			t.Run("renders the Apache License 2.0 and its NOTICE", func(t *testing.T) {
				t.Parallel()
				a := answers()
				a.License = common.Apache
				files := render(t, a)
				digest := sha256.Sum256(files["LICENSE"].Content)
				assert.Equal(t, hex.EncodeToString(digest[:]), apacheDigest, "the digest of the LICENSE")
				assert.Equal(t, text(files["NOTICE"]), "demo\nCopyright 2026 Dokimasia B.V.\n", "the NOTICE")
			})

			t.Run("renders no NOTICE for the MIT license", func(t *testing.T) {
				t.Parallel()
				_, ok := render(t, answers())["NOTICE"]
				assert.False(t, ok, "the NOTICE of MIT")
			})

			t.Run("renders the release configuration as JSON with the repository", func(t *testing.T) {
				t.Parallel()
				var config struct {
					Changelog []any `json:"changelog"`
				}
				assert.NoError(t, json.Unmarshal(render(t, answers())[".changeset/config.json"].Content, &config),
					"Unmarshal of .changeset/config.json")
				assert.Equal(t, config.Changelog[1], any(map[string]any{"repo": "dokimasia/demo"}), "the changelog")
			})

			t.Run("renders the release of commitlint in the hook and in its configuration", func(t *testing.T) {
				t.Parallel()
				files := render(t, answers())
				_, release, _ := strings.Cut(common.Commitlint, "@")
				assert.Contains(t, text(files[".pre-commit-config.yaml"]),
					`additional_dependencies: ["`+common.Commitlint+`"]`, "the .pre-commit-config.yaml")
				assert.Contains(t, text(files[".commitlint.yaml"]), "\nmin-version: "+release+"\n",
					"the .commitlint.yaml")
			})

			t.Run("renders the YAML files as YAML", func(t *testing.T) {
				t.Parallel()
				files := render(t, answers())
				documents := []string{".commitlint.yaml", ".markdownlint.yml", ".pre-commit-config.yaml", ".ergon.yaml"}
				for _, path := range documents {
					var doc map[string]any
					assert.NoError(t, yaml.Unmarshal([]byte(text(files[path])), &doc), "Unmarshal of "+path)
				}
			})

			t.Run("quotes the answers in .ergon.yaml", func(t *testing.T) {
				t.Parallel()
				a := answers()
				a.Name = "demo: two"
				a.Owner = `Acme "Inc"`
				var config struct {
					Name    string `yaml:"name"`
					License struct {
						Owner string `yaml:"owner"`
						SPDX  string `yaml:"spdx"`
					} `yaml:"license"`
				}
				assert.NoError(t, yaml.Unmarshal(render(t, a)[".ergon.yaml"].Content, &config),
					"Unmarshal of .ergon.yaml")
				assert.Equal(t, config.Name, "demo: two", "the name")
				assert.Equal(t, config.License.Owner, `Acme "Inc"`, "the owner")
				assert.Equal(t, config.License.SPDX, common.MIT, "the license")
			})

			tests := []struct {
				name   string
				change func(*language.Answers)
			}{
				{
					name:   "returns ErrInvalidAnswer for an empty name",
					change: func(a *language.Answers) { a.Name = " " },
				},
				{
					name:   "returns ErrInvalidAnswer for a name of two lines",
					change: func(a *language.Answers) { a.Name = "a\nb" },
				},
				{
					name:   "returns ErrInvalidAnswer for an empty owner",
					change: func(a *language.Answers) { a.Owner = "" },
				},
				{
					name:   "returns ErrInvalidAnswer for an owner with a carriage return",
					change: func(a *language.Answers) { a.Owner = "a\rb" },
				},
				{
					name:   "returns ErrInvalidAnswer for an empty security contact",
					change: func(a *language.Answers) { a.SecurityContact = "" },
				},
				{
					name:   "returns ErrInvalidAnswer for another license",
					change: func(a *language.Answers) { a.License = "GPL-3.0" },
				},
				{
					name:   "returns ErrInvalidAnswer for the year 0",
					change: func(a *language.Answers) { a.Year = 0 },
				},
				{
					name:   "returns ErrInvalidAnswer for the year 10000",
					change: func(a *language.Answers) { a.Year = 10000 },
				},
				{
					name:   "returns ErrInvalidAnswer for a repository without an owner",
					change: func(a *language.Answers) { a.Repository = "demo" },
				},
				{
					name:   "returns ErrInvalidAnswer for a repository with a space",
					change: func(a *language.Answers) { a.Repository = "a/b c" },
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					a := answers()
					tt.change(a)
					_, err := common.Initializer{}.Files(a)
					assert.ErrorIs(t, err, common.ErrInvalidAnswer, "Files")
				})
			}

			t.Run("returns the files for the years 1 and 9999", func(t *testing.T) {
				t.Parallel()
				for _, year := range []int{1, 9999} {
					a := answers()
					a.Year = year
					_, err := common.Initializer{}.Files(a)
					assert.NoError(t, err, "Files")
				}
			})
		})
	})
}

// answers returns new answers of the cases, which a case may change.
func answers() *language.Answers {
	return &language.Answers{
		Name:            "demo",
		Languages:       []workspace.Language{"go"},
		Owner:           "Dokimasia B.V.",
		License:         common.MIT,
		Year:            2026,
		Repository:      "dokimasia/demo",
		SecurityContact: "security@example.com",
	}
}

// render returns the files of the common files for a, by path.
func render(t *testing.T, a *language.Answers) map[string]language.File {
	t.Helper()
	files, err := common.Initializer{}.Files(a)
	assert.NoError(t, err, "Files")
	byPath := make(map[string]language.File, len(files))
	for _, f := range files {
		byPath[f.Path] = f
	}
	return byPath
}

// text returns the content or the fragment of f.
func text(f language.File) string {
	return string(f.Content) + string(f.Fragment)
}
