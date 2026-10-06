// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/php/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the configuration of PHPStan and the fragments of PHP", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				baseline.PHPStan, language.EditorConfig, language.GitAttributes, language.GitIgnore,
				language.Makefile, language.CI, language.Dependabot,
			}, "the paths of the files")
		})

		t.Run("renders the configuration of PHPStan as a managed file", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.PHPStan)
			first, _, _ := strings.Cut(config, "\n")
			assert.Equal(t, first, "# Managed by ergon init. Add repository settings to .ergon/local/"+
				baseline.PHPStan+" and run ergon init sync.", "the first line of the configuration")
		})

		t.Run("checks at the strictest level with the strict rules", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.PHPStan)
			assert.Contains(t, config, "\n\tlevel: max\n", "the level of PHPStan")
			assert.Contains(t, config, "\n\t- .ergon/tools/php/vendor/phpstan/phpstan-strict-rules/rules.neon\n",
				"the strict rules")
		})

		t.Run("lints with the tools that it pins and installs apart from the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile,
				"\t\tphpstan/phpstan:2.2.17 phpstan/phpstan-strict-rules:2.0.12 php-cs-fixer/shim:3.95.27\n",
				"the releases of the tools")
			assert.Contains(t, makefile,
				"\tphp $(PHP_TOOLS)/vendor/bin/php-cs-fixer check --rules=@PER-CS --using-cache=no --diff .\n",
				"the format check")
			assert.Contains(t, makefile,
				"\tphp $(PHP_TOOLS)/vendor/bin/phpstan analyse --configuration="+baseline.PHPStan+" --no-progress\n",
				"the analysis of PHPStan")
		})

		t.Run("ignores the directory of the tools", func(t *testing.T) {
			t.Parallel()
			gitignore := fragment(t, baseline.Initializer(), language.GitIgnore)
			assert.Contains(t, gitignore, "\n/.ergon/tools/\n", "the ignored directory")
		})

		t.Run("adds the gate of PHP to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-php\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-php: lint-php test-php audit-php ##", "the Makefile fragment")
		})

		t.Run("adds the Composer audit of composer.lock to the vulnerability scans", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "audit: audit-php\n", "the Makefile fragment")
			assert.Contains(t, makefile, "\tcomposer audit --locked\n", "the recipe of audit-php")
		})

		t.Run("adds the job check-php to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-php:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-php\n", "the gate of the job")
		})

		t.Run("adds composer to the updates of Dependabot", func(t *testing.T) {
			t.Parallel()
			dependabot := fragment(t, baseline.Initializer(), language.Dependabot)
			assert.Contains(t, dependabot, "package-ecosystem: composer\n", "the ecosystem of the fragment")
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
