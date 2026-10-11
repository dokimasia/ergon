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
	jsbaseline "go.dokimi.dev/ergon/lang/javascript/baseline"
	"go.dokimi.dev/ergon/lang/typescript"
	"go.dokimi.dev/ergon/lang/typescript/baseline"
	service "go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// name pins the name of the producer of TypeScript, which is the name of its section.
const name = "typescript"

// workflows are the files of the GitHub files that the contributions of TypeScript and the js
// toolchain change.
var workflows = []string{".github/workflows/ci.yml", ".github/workflows/security.yml", ".github/dependabot.yml"}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is typescript", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the files of TypeScript and the js toolchain at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(typescript.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the files of the js toolchain once beside JavaScript", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(javascript.Language, typescript.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "javascript", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the jobs, the analysis and the updates of TypeScript into the GitHub files",
				func(t *testing.T) {
					t.Parallel()
					dir := baselinetest.New(t, catalog(t), baselinetest.Answers(typescript.Language),
						service.Producer{Name: github.Name, Producer: github.Producer{}})
					baselinetest.Hygiene(t, dir)
					for _, file := range workflows {
						got, err := os.ReadFile(path.Join(dir, file))
						assert.NoError(t, err, "ReadFile of "+file)
						golden.Match(t, path.Join("workflows", path.Base(file)), got, golden.ShouldUpdate())
					}
				})

			t.Run("renders the Makefile and .prettierignore of TypeScript for pnpm and vite-plus", func(t *testing.T) {
				t.Parallel()
				js := jsOptions()
				js.PackageManager = option.PackageManagerPNPM
				js.Toolkit = jsbaseline.ToolkitVitePlus
				for _, file := range []string{"Makefile", ".prettierignore"} {
					got := rendered(t, file, js, baseline.Producer{}.Options())
					golden.Match(t, path.Join("vite-plus", file), []byte(got), golden.ShouldUpdate())
				}
			})

			t.Run("runs the steps that check of the section typescript names", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Check = option.Check{option.StepAudit}
				makefile := rendered(t, "Makefile", jsOptions(), o)
				assert.Contains(t, makefile, "\ncheck-typescript: audit-typescript ## Run the gate of TypeScript\n",
					"the Makefile")
			})

			t.Run("runs the tests of the section js with the arguments of the section typescript", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Test.Args = []string{"test/unit"}
				expect.That(t, rendered(t, "Makefile", jsOptions(), o)).
					Contains("\nTYPESCRIPT_TEST_ARGS ?= test/unit\n", "the arguments of the tests").
					Contains("\ntest-typescript: ## Run the TypeScript tests\n"+
						"\t$(JS_TEST)$(if $(TYPESCRIPT_TEST_ARGS), $(TYPESCRIPT_TEST_ARGS))\n", "the target of the tests")
			})

			t.Run("runs tsc in its strictest mode for the toolkit biome", func(t *testing.T) {
				t.Parallel()
				expect.That(t, rendered(t, "Makefile", jsOptions(), baseline.Producer{}.Options())).
					Contains("\nTSC := $(ERGON) tool run typescript.typescript --\n", "the tsc of the section").
					Contains("\nlint-typescript: lint-js ## Check the TypeScript sources with Biome, and with tsc in its "+
						"strictest mode\n\t$(TSC) --noEmit --strict ", "the target of the lint")
			})

			t.Run("runs no tsc for the toolkit vite-plus", func(t *testing.T) {
				t.Parallel()
				js := jsOptions()
				js.Toolkit = jsbaseline.ToolkitVitePlus
				expect.That(t, rendered(t, "Makefile", js, baseline.Producer{}.Options())).
					Contains("\nlint-typescript: lint-js ## Check the TypeScript sources with vp check\n"+
						"test-typescript: ", "the target of the lint").
					NotContains("TSC", "the tsc of the section")
			})

			t.Run("runs the command of the generators and checks their files in the step generate", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Generate = option.Generate{Command: []string{"npm", "run", "generate"}, Args: []string{}}
				o.Check = option.Check{option.StepGenerate}
				expect.That(t, rendered(t, "Makefile", jsOptions(), o)).
					Contains("\nTYPESCRIPT_GENERATE ?= npm run generate\n", "the command of the generators").
					Contains("\ngenerate: generate-typescript\n", "the aggregate generate").
					Contains("\ngenerate-typescript: ## Run the generators of TypeScript\n\t$(TYPESCRIPT_GENERATE)\n",
						"the target that runs the generators").
					Contains("\n\t@$(call verify-generated,TYPESCRIPT_GENERATE,generate-typescript)\n",
						"the target that checks them").
					Contains("\ncheck-typescript: verify-generate-typescript ## Run the gate of TypeScript\n", "the gate")
			})

			t.Run("renders no target of the generators without a command", func(t *testing.T) {
				t.Parallel()
				makefile := rendered(t, "Makefile", jsOptions(), baseline.Producer{}.Options())
				assert.NotContains(t, makefile, "generate", "the Makefile")
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

// catalog returns a catalog with the js toolchain, JavaScript and TypeScript.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, javascript.Register(c), "Register of JavaScript")
	assert.NoError(t, typescript.Register(c), "Register of TypeScript")
	return c
}

// rendered returns the file name that the js toolchain with the options js and the producer of
// TypeScript with the options o render for the answers of the cases. It stops the test when they
// render none.
func rendered(t *testing.T, name string, js *jsbaseline.ToolchainOptions, o language.Options) string {
	t.Helper()
	units := []render.Unit{
		{Name: jsbaseline.ToolchainName, Producer: jsbaseline.Toolchain{}, Options: js},
		{Name: baseline.Name, Producer: baseline.Producer{}, Options: o, Toolchain: jsbaseline.ToolchainName},
	}
	files, err := render.Render(units, baselinetest.Answers(typescript.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	i := slices.IndexFunc(files, func(f render.File) bool { return f.Path == name })
	assert.InRange(t, i, 0, float64(len(files)-1), "the units render "+name)
	return string(files[i].Content)
}

// jsOptions returns new options of the section js at the baseline.
func jsOptions() *jsbaseline.ToolchainOptions {
	o, _ := jsbaseline.Toolchain{}.Options().(*jsbaseline.ToolchainOptions)
	return o
}
