// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"os"
	"path"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/csharp"
	"go.dokimi.dev/ergon/lang/csharp/baseline"
	service "go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// name pins the name of the producer of C#, which is the name of its section.
const name = "csharp"

// workflows are the files of the GitHub files that the contribution of C# changes.
var workflows = []string{".github/workflows/ci.yml", ".github/workflows/security.yml", ".github/dependabot.yml"}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is csharp", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the files of C# at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(csharp.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the job, the analysis and the updates of C# into the GitHub files", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(csharp.Language),
					service.Producer{Name: github.Name, Producer: github.Producer{}})
				baselinetest.Hygiene(t, dir)
				for _, file := range workflows {
					got, err := os.ReadFile(path.Join(dir, file))
					assert.NoError(t, err, "ReadFile of "+file)
					golden.Match(t, path.Join("workflows", path.Base(file)), got, golden.ShouldUpdate())
				}
			})

			t.Run("runs the steps that check of the section csharp names", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Check = option.Check{option.StepAudit}
				makefile := rendered(t, o, "Makefile")
				assert.Contains(t, makefile, "\ncheck-csharp: audit-csharp ## Run the gate of C#\n", "the Makefile")
			})

			t.Run("runs dotnet test with the arguments of the section csharp", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Test.Args = []string{"--no-build", "--logger", "trx"}
				makefile := rendered(t, o, "Makefile")
				assert.Contains(t, makefile, "\nCSHARP_TEST_ARGS ?= --no-build --logger trx\n", "the Makefile")
			})
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first, _ := baseline.Producer{}.Options().(*baseline.Options)
				second, _ := baseline.Producer{}.Options().(*baseline.Options)
				first.Check[0] = option.StepAudit
				assert.Equal(t, second.Check, option.Check{option.StepLint, option.StepTest, option.StepAudit},
					"the steps of the second value")
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

// catalog returns a catalog with C#.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, csharp.Register(c), "Register of C#")
	return c
}

// rendered returns the file name that the producer of C# renders for the options o and the answers
// of the cases, and stops the test when it renders none.
func rendered(t *testing.T, o language.Options, name string) string {
	t.Helper()
	files, err := render.Render([]render.Unit{{Name: baseline.Name, Producer: baseline.Producer{}, Options: o}},
		baselinetest.Answers(csharp.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	i := slices.IndexFunc(files, func(f render.File) bool { return f.Path == name })
	assert.True(t, i >= 0, "C# renders "+name)
	return string(files[i].Content)
}
