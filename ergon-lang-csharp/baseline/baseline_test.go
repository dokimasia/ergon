// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/csharp/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the analyzer configuration and the fragments of C#", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				baseline.GlobalConfig, language.EditorConfig, language.GitAttributes, language.GitIgnore,
				language.Makefile, language.CI, language.Security, language.Dependabot,
			}, "the paths of the files")
		})

		t.Run("renders the analyzer configuration as a managed file", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.GlobalConfig)
			first, _, _ := strings.Cut(config, "\n")
			assert.Equal(t, first, "# Managed by ergon init. Add repository settings to .ergon/local/"+
				baseline.GlobalConfig+" and run ergon init sync.", "the first line of the configuration")
			assert.Contains(t, config, "\nis_global = true\n", "the global configuration")
		})

		t.Run("raises every analyzer diagnostic to an error", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.GlobalConfig)
			assert.Contains(t, config, "\ndotnet_analyzer_diagnostic.severity = error\n", "the severity")
		})

		t.Run("lints with every rule of the latest analysis level as an error", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "\tdotnet format --verify-no-changes --severity info\n", "the format check")
			build := "\tdotnet build --no-incremental -warnaserror -p:TreatWarningsAsErrors=true " +
				"-p:AnalysisLevel=latest-all"
			assert.Contains(t, makefile, build, "the build of lint-csharp")
			assert.Contains(t, makefile, "-p:GenerateDocumentationFile=true -p:WarningLevel=9999\n",
				"the documentation and the warning level of the build")
		})

		t.Run("adds the gate of C# to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-csharp\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-csharp: lint-csharp test-csharp audit-csharp ##",
				"the Makefile fragment")
		})

		t.Run("adds the NuGet audit of a forced restore to the vulnerability scans", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "audit: audit-csharp\n", "the Makefile fragment")
			assert.Contains(t, makefile, "dotnet restore --force -p:NuGetAudit=true -p:NuGetAuditMode=all",
				"the restore of audit-csharp")
			assert.Contains(t, makefile, "-p:WarningsAsErrors=NU1900%3BNU1901%3BNU1902%3BNU1903%3BNU1904\n",
				"the audit warnings that fail the restore")
		})

		t.Run("adds the job check-csharp to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-csharp:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-csharp\n", "the gate of the job")
		})

		t.Run("adds the CodeQL analysis of C# to the security checks", func(t *testing.T) {
			t.Parallel()
			security := fragment(t, baseline.Initializer(), language.Security)
			assert.Contains(t, security, "\n  codeql-csharp:\n", "the job of the fragment")
			assert.Contains(t, security, "language: csharp\n", "the CodeQL language")
		})

		t.Run("adds nuget to the updates of Dependabot", func(t *testing.T) {
			t.Parallel()
			dependabot := fragment(t, baseline.Initializer(), language.Dependabot)
			assert.Contains(t, dependabot, "package-ecosystem: nuget\n", "the ecosystem of the fragment")
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
