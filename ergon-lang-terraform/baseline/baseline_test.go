// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/terraform/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the configuration of tflint and the fragments of Terraform", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				baseline.TFLint, language.EditorConfig, language.GitAttributes, language.GitIgnore,
				language.Makefile, language.CI, language.Dependabot,
			}, "the paths of the files")
		})

		t.Run("renders the configuration of tflint as a managed file", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.TFLint)
			first, _, _ := strings.Cut(config, "\n")
			assert.Equal(t, first, "# Managed by ergon init. Add repository settings to .ergon/local/"+
				baseline.TFLint+" and run ergon init sync.", "the first line of the configuration")
		})

		t.Run("enables every rule of the bundled terraform ruleset", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.TFLint)
			assert.Contains(t, config, "plugin \"terraform\" {\n  enabled = true\n  preset  = \"all\"\n}\n",
				"the terraform ruleset")
		})

		t.Run("requires the release of tflint that the job installs", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.TFLint)
			ci := fragment(t, baseline.Initializer(), language.CI)
			_, required, found := strings.Cut(config, "required_version = \"")
			assert.True(t, found, "the required version of tflint")
			release, _, _ := strings.Cut(required, "\"")
			assert.Contains(t, ci, "\n          tflint_version: v"+release+"\n", "the release that the job installs")
		})

		t.Run("lints every module with the configuration of the repository root", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "\ttflint --recursive --config=\"$(CURDIR)/"+baseline.TFLint+"\"\n",
				"the lint of lint-terraform")
		})

		t.Run("adds the gate of Terraform to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-terraform\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-terraform: lint-terraform test-terraform audit-terraform ##",
				"the Makefile fragment")
		})

		t.Run("adds checkov over the configuration to the vulnerability scans", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "audit: audit-terraform\n", "the Makefile fragment")
			assert.Contains(t, makefile,
				"CKV_PARSE_ERROR_FAIL=true uvx --python 3.13 --exclude-newer 2026-10-06T00:00:00Z checkov==3.3.23",
				"the scan of audit-terraform")
		})

		t.Run("installs uv, which runs checkov, in the job check-terraform", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "uses: astral-sh/setup-uv@", "the setup of uv in the job")
		})

		t.Run("adds the job check-terraform to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-terraform:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-terraform\n", "the gate of the job")
		})

		t.Run("adds terraform to the updates of Dependabot", func(t *testing.T) {
			t.Parallel()
			dependabot := fragment(t, baseline.Initializer(), language.Dependabot)
			assert.Contains(t, dependabot, "package-ecosystem: terraform\n", "the ecosystem of the fragment")
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
