// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/kotlin/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the fragments of Kotlin for the shared files", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				language.EditorConfig, language.GitAttributes, language.GitIgnore, language.Makefile, language.CI,
			}, "the paths of the fragments")
		})

		t.Run("adds the gate of Kotlin to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-kotlin\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-kotlin: lint-kotlin test-kotlin audit-jvm ##", "the Makefile fragment")
		})

		t.Run("configures ktlint with the official code style and its experimental rules", func(t *testing.T) {
			t.Parallel()
			editorconfig := fragment(t, baseline.Initializer(), language.EditorConfig)
			assert.Contains(t, editorconfig, "\nktlint_code_style = ktlint_official\n", "the code style of ktlint")
			assert.Contains(t, editorconfig, "\nktlint_experimental = enabled\n", "the experimental rules")
		})

		t.Run("lints with the ktlint that it pins and checks", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "\nKTLINT_VERSION := 1.8.0\n", "the release of ktlint")
			assert.Contains(t, makefile,
				"\nKTLINT_SHA256 := 369ad2b789f95a011f807e1fcb690ccef80bd7cd014fd139e73ae82dcc0baeab\n",
				"the checksum of ktlint")
			lint := "java -jar $(KTLINT) --relative --patterns-from-stdin=\n"
			assert.Contains(t, makefile, lint, "the lint of ktlint")
		})

		t.Run("adds the job check-kotlin to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-kotlin:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-kotlin\n", "the gate of the job")
		})

		t.Run("installs the Go release that builds osv-scanner in the job check-kotlin", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "go-version: \""+language.Go+"\"\n", "the Go release of the job")
		})
	})
}

// paths returns the paths of the files of initializer in their order. It fails the test for a
// file that is not a managed fragment.
func paths(t *testing.T, initializer language.Initializer) []string {
	t.Helper()
	files, err := initializer.Files(&language.Answers{})
	assert.NoError(t, err, "Files")
	out := make([]string, 0, len(files))
	for _, f := range files {
		assert.True(t, f.Class == language.Managed && f.Content == nil && len(f.Fragment) > 0,
			"the fragment of "+f.Path)
		out = append(out, f.Path)
	}
	return out
}

// fragment returns the fragment of initializer for the shared file path, and fails the test when
// initializer has none.
func fragment(t *testing.T, initializer language.Initializer, path string) string {
	t.Helper()
	files, err := initializer.Files(&language.Answers{})
	assert.NoError(t, err, "Files")
	for _, f := range files {
		if f.Path == path {
			return string(f.Fragment)
		}
	}
	t.Fatalf("no fragment of %s", path)
	return ""
}
