// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"os"
	"path"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
	service "go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// name pins the name of the producer of JavaScript, which is the name of its section.
const name = "javascript"

// workflows are the files of the GitHub files that the contributions of JavaScript and the js
// toolchain change.
var workflows = []string{".github/workflows/ci.yml", ".github/workflows/security.yml", ".github/dependabot.yml"}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is javascript", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the files of JavaScript and the js toolchain at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(javascript.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the jobs, the analysis and the updates of JavaScript into the GitHub files",
				func(t *testing.T) {
					t.Parallel()
					dir := baselinetest.New(t, catalog(t), baselinetest.Answers(javascript.Language),
						service.Producer{Name: github.Name, Producer: github.Producer{}})
					baselinetest.Hygiene(t, dir)
					for _, file := range workflows {
						got, err := os.ReadFile(path.Join(dir, file))
						assert.NoError(t, err, "ReadFile of "+file)
						golden.Match(t, path.Join("workflows", path.Base(file)), got, golden.ShouldUpdate())
					}
				})

			t.Run("runs the steps that check of the section javascript names", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Check = option.Check{option.StepAudit}
				makefile := rendered(t, "Makefile", jsOptions(),
					render.Unit{Name: baseline.Name, Producer: baseline.Producer{}, Options: o})
				assert.Contains(t, makefile, "\ncheck-javascript: audit-javascript ## Run the gate of JavaScript\n",
					"the Makefile")
			})

			t.Run("runs the tests of the section js with the arguments of the section javascript", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Test.Args = []string{"test/unit", "test/e2e"}
				makefile := rendered(t, "Makefile", jsOptions(),
					render.Unit{Name: baseline.Name, Producer: baseline.Producer{}, Options: o})
				expect.That(t, makefile).
					Contains("\nJAVASCRIPT_TEST_ARGS ?= test/unit test/e2e\n", "the arguments of the tests").
					Contains("\ntest-javascript: ## Run the JavaScript tests\n"+
						"\t$(JS_TEST)$(if $(JAVASCRIPT_TEST_ARGS), $(JAVASCRIPT_TEST_ARGS))\n", "the target of the tests")
			})

			t.Run("names Biome in the help of lint-javascript at the baseline", func(t *testing.T) {
				t.Parallel()
				unit := render.Unit{
					Name:     baseline.Name,
					Producer: baseline.Producer{},
					Options:  baseline.Producer{}.Options(),
				}
				assert.Contains(t, rendered(t, "Makefile", jsOptions(), unit),
					"\nlint-javascript: lint-js ## Check the JavaScript sources with Biome\n", "the Makefile")
			})

			t.Run("names vp check in the help of lint-javascript for the toolkit vite-plus", func(t *testing.T) {
				t.Parallel()
				js := jsOptions()
				js.Toolkit = baseline.ToolkitVitePlus
				unit := render.Unit{
					Name:     baseline.Name,
					Producer: baseline.Producer{},
					Options:  baseline.Producer{}.Options(),
				}
				assert.Contains(t, rendered(t, "Makefile", js, unit),
					"\nlint-javascript: lint-js ## Check the JavaScript sources with vp check\n", "the Makefile")
			})

			t.Run("runs the command of the generators and checks their files in the step generate", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Generate = option.Generate{Command: []string{"npm", "run", "generate"}, Args: []string{}}
				o.Check = option.Check{option.StepGenerate}
				makefile := rendered(t, "Makefile", jsOptions(),
					render.Unit{Name: baseline.Name, Producer: baseline.Producer{}, Options: o})
				expect.That(t, makefile).
					Contains("\nJAVASCRIPT_GENERATE ?= npm run generate\n", "the command of the generators").
					Contains("\ngenerate: generate-javascript\n", "the aggregate generate").
					Contains("\ngenerate-javascript: ## Run the generators of JavaScript\n\t$(JAVASCRIPT_GENERATE)\n",
						"the target that runs the generators").
					Contains("\n\t@$(call verify-generated,JAVASCRIPT_GENERATE,generate-javascript)\n",
						"the target that checks them").
					Contains("\ncheck-javascript: verify-generate-javascript ## Run the gate of JavaScript\n", "the gate")
			})

			t.Run("renders no target of the generators without a command", func(t *testing.T) {
				t.Parallel()
				unit := render.Unit{
					Name:     baseline.Name,
					Producer: baseline.Producer{},
					Options:  baseline.Producer{}.Options(),
				}
				assert.NotContains(t, rendered(t, "Makefile", jsOptions(), unit), "generate", "the Makefile")
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

// catalog returns a catalog with the js toolchain and JavaScript.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, javascript.Register(c), "Register of JavaScript")
	return c
}

// rendered returns the file name that the js toolchain with the options o and units, whose toolchain
// it is, render for the answers of the cases. It stops the test when they render none.
func rendered(t *testing.T, name string, o *baseline.ToolchainOptions, units ...render.Unit) string {
	t.Helper()
	all := make([]render.Unit, 0, 1+len(units))
	all = append(all, render.Unit{Name: baseline.ToolchainName, Producer: baseline.Toolchain{}, Options: o})
	for _, u := range units {
		u.Toolchain = baseline.ToolchainName
		all = append(all, u)
	}
	files, err := render.Render(all, baselinetest.Answers(javascript.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	i := slices.IndexFunc(files, func(f render.File) bool { return f.Path == name })
	assert.InRange(t, i, 0, float64(len(files)-1), "the units render "+name)
	return string(files[i].Content)
}
