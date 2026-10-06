// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/bash/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the configuration of shellcheck and the fragments of Bash", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				baseline.ShellCheckRC, language.EditorConfig, language.GitAttributes, language.Makefile, language.CI,
			}, "the paths of the files")
		})

		t.Run("renders the configuration of shellcheck as a managed file", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.ShellCheckRC)
			first, _, _ := strings.Cut(config, "\n")
			assert.Equal(t, first, "# Managed by ergon init. Add repository settings to .ergon/local/"+
				baseline.ShellCheckRC+" and run ergon init sync.", "the first line of the configuration")
			assert.Contains(t, config, "\nexternal-sources=true\n", "the sourced scripts")
		})

		t.Run("enables the optional checks of the baseline", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.ShellCheckRC)
			assert.Contains(t, config, "\nenable=check-extra-masked-returns,check-set-e-suppressed,"+
				"add-default-case,avoid-nullary-conditions,deprecate-which,require-double-brackets,"+
				"require-variable-braces,useless-use-of-cat\n", "the optional checks")
		})

		t.Run("adds the gate of Bash to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-bash\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-bash: lint-bash ##", "the Makefile fragment")
		})

		t.Run("runs the shellcheck that it pins", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "uvx --from shellcheck-py==0.11.0.1 shellcheck", "the shellcheck of the gate")
		})

		t.Run("checks every Bash source that git does not ignore", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			listing := "git ls-files -z --cached --others --exclude-standard -- '*.sh' '*.bash' | \\\n"
			assert.Contains(t, makefile, listing+"\t\txargs -0 -r uvx", "the sources of the check")
		})

		t.Run("adds the job check-bash to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-bash:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-bash\n", "the gate of the job")
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
