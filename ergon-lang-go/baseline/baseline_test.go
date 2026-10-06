// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/go/baseline"
)

// linters are the linters that the configuration of golangci-lint enables, pinned because each
// is a decision of the baseline: every linter that a Go repository of the baseline enables.
var linters = []string{
	"asasalint", "bodyclose", "containedctx", "contextcheck", "copyloopvar", "decorder", "depguard", "dupl",
	"dupword", "durationcheck", "errcheck", "errname", "errorlint", "exhaustive", "fatcontext", "forbidigo",
	"forcetypeassert", "funcorder", "goconst", "gocritic", "gosec", "govet", "ineffassign", "interfacebloat",
	"makezero", "mirror", "misspell", "modernize", "musttag", "nilerr", "nilnil", "noctx", "nolintlint",
	"paralleltest", "perfsprint", "prealloc", "predeclared", "reassign", "revive", "sloglint", "spancheck",
	"staticcheck", "testifylint", "testpackage", "thelper", "tparallel", "unconvert", "unparam", "unused",
	"usestdlibvars", "usetesting", "wastedassign", "whitespace", "wrapcheck",
}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Initializer", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the configuration of golangci-lint and the fragments of Go", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, paths(t, baseline.Initializer()), []string{
				baseline.GolangCI, language.EditorConfig, language.GitAttributes, language.GitIgnore,
				language.Makefile, language.CI, language.Security, language.Dependabot,
			}, "the paths of the files")
		})

		t.Run("renders the configuration of golangci-lint as a managed file", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.GolangCI)
			first, _, _ := strings.Cut(config, "\n")
			assert.Equal(t, first, "# Managed by ergon init. Add repository settings to .ergon/local/"+
				baseline.GolangCI+" and run ergon init sync.", "the first line of the configuration")
			assert.Contains(t, config, "\nversion: \"2\"\n", "the version of the configuration")
		})

		t.Run("enables every linter of the baseline and no other", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, enabled(t, content(t, baseline.Initializer(), baseline.GolangCI)), linters,
				"the enabled linters")
		})

		t.Run("denies the replaced packages in a depguard rule that allows every other", func(t *testing.T) {
			t.Parallel()
			config := content(t, baseline.Initializer(), baseline.GolangCI)
			rule := "        replaced:\n          list-mode: lax\n          files:\n            - $all\n"
			assert.Contains(t, config, rule, "the depguard rule")
			assert.Contains(t, config, "            - pkg: io/ioutil$\n", "the denial of io/ioutil")
		})

		t.Run("adds the gate of Go to the gate of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "check: check-go\n", "the Makefile fragment")
			assert.Contains(t, makefile, "check-go: lint-go test-go audit-go ##", "the Makefile fragment")
		})

		t.Run("lints, tests and scans every module of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "go list -m -f '{{.Dir}}' | while IFS= read -r dir;",
				"the loop over the modules")
			assert.Contains(t, makefile,
				"GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0\n",
				"the pinned golangci-lint")
			lint := `(cd "$$dir" && $(GOLANGCI_LINT) run ./... && $(GOLANGCI_LINT) fmt --diff ./...)`
			assert.Contains(t, makefile, lint, "the lint of a module")
			assert.Contains(t, makefile, `go -C "$$dir" test ./...`, "the tests of a module")
			assert.Contains(t, makefile, `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 -C "$$dir" ./...`,
				"the vulnerability scan of a module")
		})

		t.Run("adds govulncheck to the vulnerability scans of the repository", func(t *testing.T) {
			t.Parallel()
			makefile := fragment(t, baseline.Initializer(), language.Makefile)
			assert.Contains(t, makefile, "audit: audit-go\n", "the Makefile fragment")
		})

		t.Run("adds the job check-go to the workflow of the gate", func(t *testing.T) {
			t.Parallel()
			ci := fragment(t, baseline.Initializer(), language.CI)
			assert.Contains(t, ci, "\n  check-go:\n", "the job of the fragment")
			assert.Contains(t, ci, "- uses: "+language.Checkout+"\n", "the checkout of the job")
			assert.Contains(t, ci, "run: make check-go\n", "the gate of the job")
		})

		t.Run("adds the CodeQL analysis of Go to the security checks", func(t *testing.T) {
			t.Parallel()
			security := fragment(t, baseline.Initializer(), language.Security)
			assert.Contains(t, security, "\n  codeql-go:\n", "the job of the fragment")
			assert.Contains(t, security, "language: go\n", "the CodeQL language")
		})

		t.Run("adds gomod to the updates of Dependabot", func(t *testing.T) {
			t.Parallel()
			dependabot := fragment(t, baseline.Initializer(), language.Dependabot)
			assert.Contains(t, dependabot, "package-ecosystem: gomod\n", "the ecosystem of the fragment")
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

// enabled returns the linters of the list linters.enable of config, sorted. It reads the list
// as the lines of the form "    - name" that follow the line "  enable:", up to the first line
// that is neither such an item nor a comment.
func enabled(t *testing.T, config string) []string {
	t.Helper()
	_, list, found := strings.Cut(config, "\n  enable:\n")
	assert.True(t, found, "the list linters.enable")
	var names []string
	for line := range strings.Lines(list) {
		item, isItem := strings.CutPrefix(strings.TrimSuffix(line, "\n"), "    - ")
		if isItem {
			names = append(names, item)
			continue
		}
		if !strings.HasPrefix(line, "    #") {
			break
		}
	}
	slices.Sort(names)
	return names
}
