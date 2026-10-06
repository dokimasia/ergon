// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/java/baseline"
)

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the init script and the fragments of Java", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				baseline.InitScript, language.EditorConfig, language.GitAttributes, language.GitIgnore,
				language.Makefile, language.CI,
			}, "the paths of the files")
		})

		t.Run("renders the init script as a managed file", func(t *testing.T) {
			t.Parallel()
			script := content(t, baseline.Initializer(), baseline.InitScript)
			first, _, _ := strings.Cut(script, "\n")
			assert.Equal(t, first, "// Managed by ergon init. Add repository settings to .ergon/local/"+
				baseline.InitScript+" and run ergon init sync.", "the first line of the init script")
		})

		t.Run("applies PMD with its base ruleset and the warnings of javac as errors", func(t *testing.T) {
			t.Parallel()
			script := content(t, baseline.Initializer(), baseline.InitScript)
			assert.Contains(t, script, "\n            toolVersion = \"7.28.0\"\n", "the release of PMD")
			assert.Contains(t, script, "ruleSets = listOf(\"rulesets/java/quickstart.xml\")", "the ruleset of PMD")
			assert.Contains(t, script, "options.compilerArgs.addAll(listOf(\"-Xlint:all\", \"-Werror\"))",
				"the warnings of javac")
		})

		t.Run("lints with the init script", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "\t./gradlew --init-script "+baseline.InitScript+" check javadoc -x test\n",
				"the lint of lint-java")
		})

		t.Run("adds the gate of Java to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-java\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-java: lint-java test-java audit-jvm ##", "the Makefile fragment")
		})

		t.Run("adds the job check-java to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-java:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-java\n", "the gate of the job")
		})

		t.Run("installs the Go release that builds osv-scanner in the job check-java", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "go-version: \""+language.Go+"\"\n", "the Go release of the job")
		})
	})

	t.Run("Toolchain", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the fragments of the jvm toolchain for the shared files", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Toolchain()),
				[]string{language.Makefile, language.Security, language.Dependabot}, "the paths of the fragments")
		})

		t.Run("adds osv-scanner over the Gradle lockfiles to the vulnerability scans", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Toolchain(), language.Makefile)
			assert.Contains(t, makefile, "audit: audit-jvm\n", "the Makefile fragment")
			assert.Contains(t, makefile, "':(glob)**/gradle.lockfile' ':(glob)**/buildscript-gradle.lockfile'",
				"the lockfiles of audit-jvm")
			assert.Contains(t, makefile, "go run github.com/google/osv-scanner/v2/cmd/osv-scanner@v2.6.0 scan source",
				"the scan of audit-jvm")
		})

		t.Run("adds the CodeQL analysis of java-kotlin to the security checks", func(t *testing.T) {
			t.Parallel()
			security := fragment(t, baseline.Toolchain(), language.Security)
			assert.Contains(t, security, "\n  codeql-java-kotlin:\n", "the job of the fragment")
			assert.Contains(t, security, "language: java-kotlin\n", "the CodeQL language")
		})

		t.Run("adds gradle to the updates of Dependabot", func(t *testing.T) {
			t.Parallel()
			dependabot := fragment(t, baseline.Toolchain(), language.Dependabot)
			assert.Contains(t, dependabot, "package-ecosystem: gradle\n", "the ecosystem of the fragment")
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
