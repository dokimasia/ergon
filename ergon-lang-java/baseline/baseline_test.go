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
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/java/baseline"
	service "go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// name pins the name of the producer of Java, which is the name of its section.
const name = "java"

// workflows are the files of the GitHub files that the contributions of Java and the jvm toolchain
// change.
var workflows = []string{".github/workflows/ci.yml", ".github/workflows/security.yml", ".github/dependabot.yml"}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is java", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the files of Java and the jvm toolchain at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(java.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the jobs, the analysis and the updates of Java into the GitHub files", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(java.Language),
					service.Producer{Name: github.Name, Producer: github.Producer{}})
				baselinetest.Hygiene(t, dir)
				for _, file := range workflows {
					got, err := os.ReadFile(path.Join(dir, file))
					assert.NoError(t, err, "ReadFile of "+file)
					golden.Match(t, path.Join("workflows", path.Base(file)), got, golden.ShouldUpdate())
				}
			})

			t.Run("runs the steps that check of the section java names", func(t *testing.T) {
				t.Parallel()
				o := javaOptions()
				o.Check = option.Check{option.StepAudit}
				makefile := rendered(t, o, "Makefile")
				assert.Contains(t, makefile, "\ncheck-java: audit-java ## Run the gate of Java\n", "the Makefile")
			})

			t.Run("runs the PMD of the section java", func(t *testing.T) {
				t.Parallel()
				o := javaOptions()
				o.Tools.PMD = "net.sourceforge.pmd:pmd-java@7.29.0"
				script := rendered(t, o, "gradle/ergon-java.init.gradle.kts")
				assert.Contains(t, script, "\n            toolVersion = \"7.29.0\"\n", "the init script")
			})
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first := javaOptions()
				second := javaOptions()
				first.Check[0] = option.StepAudit
				assert.Equal(t, second.Check, option.Check{option.StepLint, option.StepTest, option.StepAudit},
					"the steps of the second value")
			})
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the contribution of the options", func(t *testing.T) {
				t.Parallel()
				o := javaOptions()
				assert.Equal(t, baseline.Producer{}.Contribution(o), o.Contribution(), "the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.Producer{}.Contribution(nil), javaOptions().Contribution(), "the contribution")
			})
		})
	})
}

// catalog returns a catalog with the jvm toolchain and Java.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, java.Register(c), "Register of Java")
	return c
}

// rendered returns the file name that the producer of Java renders for the options o and the
// answers of the cases, and stops the test when it renders none.
func rendered(t *testing.T, o language.Options, name string) string {
	t.Helper()
	files, err := render.Render([]render.Unit{{Name: baseline.Name, Producer: baseline.Producer{}, Options: o}},
		baselinetest.Answers(java.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	i := slices.IndexFunc(files, func(f render.File) bool { return f.Path == name })
	assert.True(t, i >= 0, "Java renders "+name)
	return string(files[i].Content)
}

// javaOptions returns new options of the section java at the baseline.
func javaOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
