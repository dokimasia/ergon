// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"context"
	"net/http"
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
	"go.dokimi.dev/ergon/lang/rust"
	"go.dokimi.dev/ergon/lang/rust/baseline"
	service "go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/overlay"
	"go.dokimi.dev/ergon/service/baseline/render"
)

// name pins the name of the producer of Rust, which is the name of its section.
const name = "rust"

// workflows are the files of the GitHub files that the contribution of Rust changes.
var workflows = []string{
	".github/workflows/ci.yml", ".github/workflows/nightly.yml", ".github/workflows/security.yml",
	".github/workflows/version.yml", ".github/workflows/release.yml", ".github/dependabot.yml",
}

// rustTools are the tools of the toolchain of Rust of the cases. They do not run a program, and
// they read an environment without variables.
var rustTools = rust.Tools{
	Command: func(context.Context, string, string, []string, []string) error { return nil },
	Getenv:  func(string) string { return "" },
}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is rust", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the files of Rust at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(rust.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the contribution of Rust into the GitHub files", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(rust.Language),
					service.Producer{Name: github.Name, Producer: github.Producer{}})
				baselinetest.Hygiene(t, dir)
				for _, file := range workflows {
					got, err := os.ReadFile(path.Join(dir, file))
					assert.NoError(t, err, "ReadFile of "+file)
					golden.Match(t, path.Join("workflows", path.Base(file)), got, golden.ShouldUpdate())
				}
			})

			t.Run("runs the steps that check of the section rust names", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Check = option.Check{option.StepAudit}
				makefile := rendered(t, o, "Makefile")
				assert.Contains(t, makefile, "\ncheck-rust: audit-rust ## Run the gate of Rust\n", "the Makefile")
			})

			t.Run("runs the command of the generators and checks their files in the step generate", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Generate = option.Generate{
					Command: []string{"cargo", "run", "--package", "codegen"},
					Args:    []string{},
				}
				o.Check = option.Check{option.StepGenerate}
				expect.That(t, rendered(t, o, "Makefile")).
					Contains("\nRUST_GENERATE ?= cargo run --package codegen\n", "the command of the generators").
					Contains("\ngenerate: generate-rust\n", "the aggregate generate").
					Contains("\ngenerate-rust: ## Run the generators of Rust\n\t$(RUST_GENERATE)\n",
						"the target that runs the generators").
					Contains("\n\t@$(call verify-generated,RUST_GENERATE,generate-rust)\n", "the target that checks them").
					Contains("\ncheck-rust: verify-generate-rust ## Run the gate of Rust\n", "the gate")
			})

			t.Run("runs the doc tests of the workspace after the tests of every crate", func(t *testing.T) {
				t.Parallel()
				expect.That(t, rendered(t, baseline.Producer{}.Options(), "Makefile")).
					Contains("\nRUST_TEST_ARGS ?= --workspace --all-targets\n", "the tests of every crate").
					Contains("\n\tcargo test $(RUST_TEST_ARGS)\n\t@if cargo metadata --no-deps --format-version 1 |",
						"the doc tests after the tests").
					Contains(` grep -Eq '"kind":\[[^]]*"(lib|rlib|proc-macro)"'; then \`,
						"the check for a crate whose doc tests cargo runs").
					Contains("\n\t\tcargo test --workspace --doc; \\\n\tfi\n", "the doc tests of the workspace")
			})

			t.Run("checks each published crate on the Rust of its rust-version when a crate sets one",
				func(t *testing.T) {
					t.Parallel()
					expect.That(t, rendered(t, baseline.Producer{}.Options(), "Makefile")).
						Contains("\nCARGO_HACK := $(ERGON) tool run rust.cargo-hack --\n", "the cargo-hack of the section").
						Contains("\nmsrv-rust: ## Check each published crate on the Rust of its rust-version\n"+
							"\t@if cargo metadata --no-deps --format-version 1 | grep -q '\"rust_version\":\"'; then \\\n"+
							"\t\t$(CARGO_HACK) hack check --rust-version --workspace --ignore-private --all-targets "+
							"--all-features; \\\n\tfi\n", "the target of the check").
						Contains("\ncheck-rust: lint-rust test-rust msrv-rust audit-rust ## Run the gate of Rust\n",
							"the gate")
				})

			t.Run("pins cargo-llvm-cov for the targets of the repository", func(t *testing.T) {
				t.Parallel()
				makefile := rendered(t, baseline.Producer{}.Options(), "Makefile")
				assert.Contains(t, makefile, "\nCARGO_LLVM_COV := $(ERGON) tool run rust.cargo-llvm-cov --\n",
					"the Makefile")
			})

			t.Run("renders no target of the generators without a command", func(t *testing.T) {
				t.Parallel()
				makefile := rendered(t, baseline.Producer{}.Options(), "Makefile")
				assert.NotContains(t, makefile, "generate", "the Makefile")
			})

			t.Run("renders a clippy.toml that a local file merges into without other changes", func(t *testing.T) {
				t.Parallel()
				clippy := rendered(t, baseline.Producer{}.Options(), "clippy.toml")
				got, err := overlay.MergeTOML([]byte(clippy), []byte("too-many-lines-threshold = 80\n"))
				assert.NoError(t, err, "MergeTOML of the local clippy.toml")
				assert.Equal(t, string(got), clippy+"too-many-lines-threshold = 80\n", "the merged clippy.toml")
			})
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first, _ := baseline.Producer{}.Options().(*baseline.Options)
				second, _ := baseline.Producer{}.Options().(*baseline.Options)
				first.Test.Args[0] = "--lib"
				assert.Equal(t, second.Test.Args, []string{"--workspace", "--all-targets"},
					"the arguments of the second value")
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

// catalog returns a catalog with Rust.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, rust.Register(c, rustTools, &http.Client{}), "Register of Rust")
	return c
}

// rendered returns the file name that the producer of Rust renders for the options o and the
// answers of the cases, and stops the test when it renders none.
func rendered(t *testing.T, o language.Options, name string) string {
	t.Helper()
	files, err := render.Render([]render.Unit{{Name: baseline.Name, Producer: baseline.Producer{}, Options: o}},
		baselinetest.Answers(rust.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	i := slices.IndexFunc(files, func(f render.File) bool { return f.Path == name })
	assert.InRange(t, i, 0, float64(len(files)-1), "Rust renders "+name)
	return string(files[i].Content)
}
