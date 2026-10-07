// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"context"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	golang "go.dokimi.dev/ergon/lang/go"
	"go.dokimi.dev/ergon/lang/go/baseline"
	service "go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// name pins the name of the producer of Go, which is the name of its section.
const name = "go"

// git is the access to a repository that the catalog of the cases does not read.
var git = golang.Git{
	Tags:     func(context.Context, string) (map[string]string, error) { return nil, nil },
	Snapshot: func(context.Context, string) (string, error) { return "", nil },
}

// workflows are the files of the GitHub files that the contribution of Go changes.
var workflows = []string{
	".github/workflows/ci.yml", ".github/workflows/release.yml", ".github/workflows/security.yml",
	".github/dependabot.yml",
}

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

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is go", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the files of Go at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(golang.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the job, the analysis and the updates of Go into the GitHub files", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(golang.Language),
					service.Producer{Name: github.Name, Producer: github.Producer{}})
				baselinetest.Hygiene(t, dir)
				for _, file := range workflows {
					got, err := os.ReadFile(path.Join(dir, file))
					assert.NoError(t, err, "ReadFile of "+file)
					golden.Match(t, path.Join("workflows", path.Base(file)), got, golden.ShouldUpdate())
				}
			})

			t.Run("enables every linter of the baseline and no other", func(t *testing.T) {
				t.Parallel()
				config := rendered(t, baseline.Producer{}.Options(), ".golangci.yml")
				assert.Equal(t, enabled(t, config), linters, "the enabled linters")
			})

			t.Run("runs the steps that check of the section go names", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Check = option.Check{option.StepLint, option.StepFuzz}
				makefile := rendered(t, o, "Makefile")
				assert.Contains(t, makefile, "\ncheck-go: lint-go fuzz-go ## Run the gate of Go\n", "the Makefile")
			})
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first, _ := baseline.Producer{}.Options().(*baseline.Options)
				second, _ := baseline.Producer{}.Options().(*baseline.Options)
				first.Paths[0] = "./changed/..."
				assert.Equal(t, second.Paths, option.Paths{"./..."}, "the paths of the second value")
			})
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the contribution of the options", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.CI.Timeout = 45
				assert.Equal(t, baseline.Producer{}.Contribution(o), o.Contribution(), "the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				assert.Equal(t, baseline.Producer{}.Contribution(nil), o.Contribution(), "the contribution")
			})
		})
	})
}

// catalog returns a catalog with Go.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, golang.Register(c, git), "Register of Go")
	return c
}

// rendered returns the file name that the producer of Go renders for the options o and the answers
// of the cases, and stops the test when it renders none.
func rendered(t *testing.T, o language.Options, name string) string {
	t.Helper()
	files, err := render.Render([]render.Unit{{Name: baseline.Name, Producer: baseline.Producer{}, Options: o}},
		baselinetest.Answers(golang.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	i := slices.IndexFunc(files, func(f render.File) bool { return f.Path == name })
	assert.True(t, i >= 0, "Go renders "+name)
	return string(files[i].Content)
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
