// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/python/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the configuration of ruff and the fragments of Python", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				baseline.Ruff, language.EditorConfig, language.GitAttributes, language.GitIgnore,
				language.Makefile, language.CI, language.Security, language.Dependabot,
			}, "the paths of the files")
		})

		t.Run("renders the configuration of ruff as a managed file", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.Ruff)
			first, _, _ := strings.Cut(config, "\n")
			assert.Equal(t, first, "# Managed by ergon init. Add repository settings to .ergon/local/"+
				baseline.Ruff+" and run ergon init sync.", "the first line of the configuration")
		})

		t.Run("selects every stable rule but the one that conflicts with the formatter", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.Ruff)
			assert.Contains(t, config, "\n[lint]\nselect = [\"ALL\"]\n", "the rules")
			assert.Contains(t, config, "\nignore = [\"COM812\"]\n", "the rule that the formatter conflicts with")
		})

		t.Run("ends with the table of the ignores of a file", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.Ruff)
			_, last, found := strings.Cut(config, "\n[lint.per-file-ignores]\n")
			assert.True(t, found, "the table [lint.per-file-ignores]")
			assert.NotContains(t, last, "\n[", "a table after [lint.per-file-ignores]")
		})

		t.Run("lints with ruff and with mypy in its strictest mode", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "\tuvx ruff@0.16.10 check --no-fix .\n", "the lint of ruff")
			assert.Contains(t, makefile, "\tuvx ruff@0.16.10 format --check .\n", "the format check of ruff")
			assert.Contains(t, makefile,
				"\tuv run --with mypy==2.4.0 mypy --strict --warn-unreachable --strict-equality-for-none $(MYPY_CODES)",
				"the type check of mypy")
		})

		t.Run("adds the gate of Python to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-python\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-python: lint-python test-python audit-python ##",
				"the Makefile fragment")
		})

		t.Run("adds pip-audit over the packages of uv.lock to the vulnerability scans", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "audit: audit-python\n", "the Makefile fragment")
			assert.Contains(t, makefile, "uv export --frozen --all-packages", "the export of uv.lock")
			assert.Contains(t, makefile, "uvx pip-audit@2.10.1 --locked --strict", "the scan of the export")
		})

		t.Run("adds the job check-python to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-python:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-python\n", "the gate of the job")
		})

		t.Run("adds the CodeQL analysis of Python to the security checks", func(t *testing.T) {
			t.Parallel()
			security := fragment(t, baseline.Initializer(), language.Security)
			assert.Contains(t, security, "\n  codeql-python:\n", "the job of the fragment")
			assert.Contains(t, security, "language: python\n", "the CodeQL language")
		})

		t.Run("adds uv to the updates of Dependabot", func(t *testing.T) {
			t.Parallel()
			dependabot := fragment(t, baseline.Initializer(), language.Dependabot)
			assert.Contains(t, dependabot, "package-ecosystem: uv\n", "the ecosystem of the fragment")
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
