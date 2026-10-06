// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/typescript/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the fragments of TypeScript for the shared files", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				language.EditorConfig, language.GitAttributes, language.GitIgnore, language.Makefile, language.CI,
			}, "the paths of the fragments")
		})

		t.Run("adds the gate of TypeScript to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-typescript\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-typescript: lint-typescript test-typescript audit-js ##",
				"the Makefile fragment")
		})

		t.Run("lints with Biome and with the type-safety options of tsc", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "\nlint-typescript: lint-js ##", "the lint of Biome")
			assert.Contains(t, makefile,
				"\tnpx --yes -p typescript@7.0.2 tsc --noEmit --strict --noUncheckedIndexedAccess --noImplicitOverride",
				"the type check of tsc")
			assert.Contains(t, makefile, "--exactOptionalPropertyTypes\n", "the last option of tsc")
		})

		t.Run("adds the job check-typescript to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-typescript:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: npm ci\n", "the installation of the dependencies")
			assert.Contains(t, ci, "run: make check-typescript\n", "the gate of the job")
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
