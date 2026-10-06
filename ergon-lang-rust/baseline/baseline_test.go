// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/rust/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the configuration of clippy and the fragments of Rust", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				baseline.Clippy, language.EditorConfig, language.GitAttributes, language.GitIgnore,
				language.Makefile, language.CI, language.Security, language.Dependabot,
			}, "the paths of the files")
		})

		t.Run("renders the configuration of clippy as a managed file", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.Clippy)
			first, _, _ := strings.Cut(config, "\n")
			assert.Equal(t, first, "# Managed by ergon init. Add repository settings to .ergon/local/"+
				baseline.Clippy+" and run ergon init sync.", "the first line of the configuration")
			assert.Contains(t, config, "\navoid-breaking-exported-api = false\n", "a setting of clippy")
		})

		t.Run("denies the pedantic lints and the documentation lints", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			clippy := "\tcargo clippy --workspace --all-targets --all-features -- " +
				"-D warnings -D clippy::pedantic -D missing_docs"
			assert.Contains(t, makefile, clippy, "the lint of clippy")
			assert.Contains(t, makefile,
				"\tRUSTDOCFLAGS=\"-D warnings -D rustdoc::private_doc_tests -D rustdoc::unescaped_backticks\"",
				"the lints of rustdoc")
		})

		t.Run("adds the gate of Rust to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-rust\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-rust: lint-rust test-rust audit-rust ##", "the Makefile fragment")
		})

		t.Run("adds cargo-audit to the vulnerability scans of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "audit: audit-rust\n", "the Makefile fragment")
			assert.Contains(t, makefile, "\tcargo install --locked cargo-audit --version 0.22.2\n\tcargo audit\n",
				"the recipe of audit-rust")
		})

		t.Run("adds the job check-rust to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-rust:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-rust\n", "the gate of the job")
		})

		t.Run("adds the CodeQL analysis of Rust to the security checks", func(t *testing.T) {
			t.Parallel()
			security := fragment(t, baseline.Initializer(), language.Security)
			assert.Contains(t, security, "\n  codeql-rust:\n", "the job of the fragment")
			assert.Contains(t, security, "language: rust\n", "the CodeQL language")
		})

		t.Run("adds cargo to the updates of Dependabot", func(t *testing.T) {
			t.Parallel()
			dependabot := fragment(t, baseline.Initializer(), language.Dependabot)
			assert.Contains(t, dependabot, "package-ecosystem: cargo\n", "the ecosystem of the fragment")
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
