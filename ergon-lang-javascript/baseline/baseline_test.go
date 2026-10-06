// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the fragments of JavaScript for the shared files", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				language.EditorConfig, language.GitAttributes, language.GitIgnore, language.Makefile, language.CI,
			}, "the paths of the fragments")
		})

		t.Run("adds the gate of JavaScript to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-javascript\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-javascript: lint-javascript test-javascript audit-js ##",
				"the Makefile fragment")
			assert.Contains(t, makefile, "\nlint-javascript: lint-js ##", "the lint of JavaScript")
		})

		t.Run("adds the job check-javascript to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-javascript:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: npm ci\n", "the installation of the dependencies")
			assert.Contains(t, ci, "run: make check-javascript\n", "the gate of the job")
		})
	})

	t.Run("Toolchain", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the configuration of Biome and the fragments of the js toolchain", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Toolchain()),
				[]string{baseline.Biome, language.Makefile, language.Security, language.Dependabot},
				"the paths of the files")
		})

		t.Run("raises every stable group of rules of Biome to an error", func(t *testing.T) {
			t.Parallel()
			var config struct {
				Linter struct {
					Rules map[string]any `json:"rules"`
				} `json:"linter"`
			}
			assert.NoError(t, json.Unmarshal([]byte(content(t, baseline.Toolchain(), baseline.Biome)), &config),
				"Unmarshal of the configuration")
			assert.Equal(t, config.Linter.Rules, map[string]any{
				"a11y": "error", "complexity": "error", "correctness": "error", "performance": "error",
				"security": "error", "style": "error", "suspicious": "error",
			}, "the groups of rules")
		})

		t.Run("formats and lints with the Biome that it pins", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Toolchain(), language.Makefile)
			assert.Contains(t, makefile, "\nBIOME := npx --yes @biomejs/biome@2.5.15\n", "the release of Biome")
			assert.Contains(t, makefile, "\t$(BIOME) ci --error-on-warnings --skip=correctness/noNodejsModules .\n",
				"the lint of lint-js")
			assert.Contains(t, makefile, "\t$(BIOME) check --write --linter-enabled=false .\n", "the format of fmt-js")
		})

		t.Run("adds the npm audit to the vulnerability scans of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Toolchain(), language.Makefile)
			assert.Contains(t, makefile, "audit: audit-js\n", "the Makefile fragment")
			assert.Contains(t, makefile, "\tnpm audit --audit-level=low --include=dev\n", "the recipe of audit-js")
		})

		t.Run("adds the CodeQL analysis of javascript-typescript to the security checks", func(t *testing.T) {
			t.Parallel()
			security := fragment(t, baseline.Toolchain(), language.Security)
			assert.Contains(t, security, "\n  codeql-javascript-typescript:\n", "the job of the fragment")
			assert.Contains(t, security, "language: javascript-typescript\n", "the CodeQL language")
		})

		t.Run("adds npm to the updates of Dependabot", func(t *testing.T) {
			t.Parallel()
			dependabot := fragment(t, baseline.Toolchain(), language.Dependabot)
			assert.Contains(t, dependabot, "package-ecosystem: npm\n", "the ecosystem of the fragment")
		})
	})
}

// paths returns the paths of the files of initializer in their order. It fails the test for a
// file that is not managed, or that has both or neither of a content and a fragment.
func paths(t *testing.T, initializer language.Initializer) []string {
	t.Helper()
	files, err := initializer.Files(&language.Answers{})
	assert.NoError(t, err, "Files")
	out := make([]string, 0, len(files))
	for _, f := range files {
		assert.True(t, f.Class == language.Managed && (f.Content == nil) != (f.Fragment == nil),
			"the class and the text of "+f.Path)
		out = append(out, f.Path)
	}
	return out
}

// content returns the content of the file of initializer at path, and fails the test when
// initializer has none.
func content(t *testing.T, initializer language.Initializer, path string) string {
	t.Helper()
	files, err := initializer.Files(&language.Answers{})
	assert.NoError(t, err, "Files")
	for _, f := range files {
		if f.Path == path && f.Content != nil {
			return string(f.Content)
		}
	}
	t.Fatalf("no file %s", path)
	return ""
}

// fragment returns the fragment of initializer for the shared file path, and fails the test when
// initializer has none.
func fragment(t *testing.T, initializer language.Initializer, path string) string {
	t.Helper()
	files, err := initializer.Files(&language.Answers{})
	assert.NoError(t, err, "Files")
	for _, f := range files {
		if f.Path == path && f.Fragment != nil {
			return string(f.Fragment)
		}
	}
	t.Fatalf("no fragment of %s", path)
	return ""
}
