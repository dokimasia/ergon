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
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/kotlin"
	"go.dokimi.dev/ergon/lang/kotlin/baseline"
	service "go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// name pins the name of the producer of Kotlin, which is the name of its section.
const name = "kotlin"

// workflows are the files of the GitHub files that the contributions of Kotlin and the jvm
// toolchain change.
var workflows = []string{".github/workflows/ci.yml", ".github/workflows/security.yml", ".github/dependabot.yml"}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is kotlin", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the files of Kotlin and the jvm toolchain at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(kotlin.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the files of the jvm toolchain once beside Java", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(java.Language, kotlin.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "java", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the jobs, the analysis and the updates of Kotlin into the GitHub files", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(kotlin.Language),
					service.Producer{Name: github.Name, Producer: github.Producer{}})
				baselinetest.Hygiene(t, dir)
				for _, file := range workflows {
					got, err := os.ReadFile(path.Join(dir, file))
					assert.NoError(t, err, "ReadFile of "+file)
					golden.Match(t, path.Join("workflows", path.Base(file)), got, golden.ShouldUpdate())
				}
			})

			t.Run("runs the steps that check of the section kotlin names", func(t *testing.T) {
				t.Parallel()
				o := kotlinOptions()
				o.Check = option.Check{option.StepAudit}
				makefile := rendered(t, o, "Makefile")
				assert.Contains(t, makefile, "\ncheck-kotlin: audit-kotlin ## Run the gate of Kotlin\n", "the Makefile")
			})

			t.Run("runs the command of the generators and checks their files in the step generate", func(t *testing.T) {
				t.Parallel()
				o := kotlinOptions()
				o.Generate = option.Generate{
					Command: []string{"./gradlew", "openApiGenerate"},
					Args:    []string{"--info"},
				}
				o.Check = option.Check{option.StepGenerate}
				expect.That(t, rendered(t, o, "Makefile")).
					Contains("\nKOTLIN_GENERATE ?= ./gradlew openApiGenerate --info\n", "the command of the generators").
					Contains("\ngenerate: generate-kotlin\n", "the aggregate generate").
					Contains("\ngenerate-kotlin: ## Run the generators of Kotlin\n\t$(KOTLIN_GENERATE)\n",
						"the target that runs the generators").
					Contains("\n\t@$(call verify-generated,KOTLIN_GENERATE,generate-kotlin)\n", "the target that checks them").
					Contains("\ncheck-kotlin: verify-generate-kotlin ## Run the gate of Kotlin\n", "the gate")
			})

			t.Run("renders no target of the generators without a command", func(t *testing.T) {
				t.Parallel()
				assert.NotContains(t, rendered(t, kotlinOptions(), "Makefile"), "generate", "the Makefile")
			})
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first := kotlinOptions()
				second := kotlinOptions()
				first.Check[0] = option.StepAudit
				assert.Equal(t, second.Check, option.Check{option.StepLint, option.StepTest, option.StepAudit},
					"the steps of the second value")
			})
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the contribution of the options", func(t *testing.T) {
				t.Parallel()
				o := kotlinOptions()
				assert.Equal(t, baseline.Producer{}.Contribution(o), o.Contribution(), "the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.Producer{}.Contribution(nil), kotlinOptions().Contribution(),
					"the contribution")
			})
		})
	})
}

// catalog returns a catalog with the jvm toolchain, Java and Kotlin.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, java.Register(c), "Register of Java")
	assert.NoError(t, kotlin.Register(c), "Register of Kotlin")
	return c
}

// rendered returns the file name that the producer of Kotlin renders for the options o and the
// answers of the cases, and stops the test when it renders none.
func rendered(t *testing.T, o language.Options, name string) string {
	t.Helper()
	files, err := render.Render([]render.Unit{{Name: baseline.Name, Producer: baseline.Producer{}, Options: o}},
		baselinetest.Answers(kotlin.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	i := slices.IndexFunc(files, func(f render.File) bool { return f.Path == name })
	assert.InRange(t, i, 0, float64(len(files)-1), "Kotlin renders "+name)
	return string(files[i].Content)
}

// kotlinOptions returns new options of the section kotlin at the baseline.
func kotlinOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
