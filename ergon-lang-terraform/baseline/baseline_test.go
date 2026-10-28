// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"context"
	"os"
	"os/exec"
	"path"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/terraform"
	"go.dokimi.dev/ergon/lang/terraform/baseline"
	service "go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/render"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// name pins the name of the producer of Terraform, which is the name of its section.
const name = "terraform"

// repository is the access to a repository that the catalog of the cases does not read.
var repository = terraform.Repository{
	Tags:    func(context.Context, string) (map[string]string, error) { return map[string]string{}, nil },
	Answers: func(context.Context, string) (language.Answers, error) { return language.Answers{}, nil },
}

// workflows are the files of the GitHub files that the contribution of Terraform changes.
var workflows = []string{
	".github/workflows/ci.yml", ".github/workflows/nightly.yml", ".github/workflows/security.yml",
	".github/dependabot.yml",
}

// TestMain runs the tests without the variables and the configuration of git of the environment,
// so that git reads the repositories of the tests alone.
func TestMain(m *testing.M) {
	vcstest.Isolate()
	os.Exit(m.Run())
}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is terraform", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, baseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			t.Run("renders the files of Terraform at the baseline", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(terraform.Language))
				baselinetest.Hygiene(t, dir)
				golden.MatchTree(t, "baseline", os.DirFS(dir), golden.ShouldUpdate())
			})

			t.Run("renders the job and the updates of Terraform into the GitHub files", func(t *testing.T) {
				t.Parallel()
				dir := baselinetest.New(t, catalog(t), baselinetest.Answers(terraform.Language),
					service.Producer{Name: github.Name, Producer: github.Producer{}})
				baselinetest.Hygiene(t, dir)
				for _, file := range workflows {
					got, err := os.ReadFile(path.Join(dir, file))
					assert.NoError(t, err, "ReadFile of "+file)
					golden.Match(t, path.Join("workflows", path.Base(file)), got, golden.ShouldUpdate())
				}
			})

			t.Run("runs the steps that check of the section terraform names", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Check = option.Check{option.StepAudit}
				makefile := rendered(t, o, "Makefile")
				assert.Contains(t, makefile, "\ncheck-terraform: audit-terraform ## Run the gate of Terraform\n",
					"the Makefile")
			})

			t.Run("runs the command lines of the generators in the step generate", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Generate.Commands = [][]string{
					{"terraform-docs", "markdown", "."},
					{"terraform-docs", "markdown", "modules/vpc"},
				}
				o.Check = option.Check{option.StepGenerate}
				expect.That(t, rendered(t, o, "Makefile")).
					Contains("\nTERRAFORM_GENERATE ?= terraform-docs markdown . && terraform-docs markdown modules/vpc\n",
						"the command lines of the generators").
					Contains("\ngenerate: generate-terraform\n", "the aggregate generate").
					Contains("\ngenerate-terraform: ## Run the generators of Terraform\n\t$(TERRAFORM_GENERATE)\n",
						"the target that runs the generators").
					Contains("\n\t@$(call verify-generated,TERRAFORM_GENERATE,generate-terraform)\n",
						"the target that checks them").
					Contains("\ncheck-terraform: verify-generate-terraform ## Run the gate of Terraform\n", "the gate")
			})

			t.Run("runs terraform-docs before the command lines in the step generate", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Generate.Docs = true
				o.Generate.Commands = [][]string{{"go", "generate"}}
				want := "\nTERRAFORM_GENERATE ?= for path in $(TERRAFORM_PATHS); " +
					`do $(TERRAFORM_DOCS) --config .terraform-docs.yml "$$path" || exit 1; done && go generate` + "\n"
				assert.Contains(t, rendered(t, o, "Makefile"), want, "the Makefile")
			})

			t.Run("renders the targets of the step generate for docs without a command line", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Generate.Docs = true
				expect.That(t, rendered(t, o, "Makefile")).
					Contains("\nTERRAFORM_GENERATE ?= for path in $(TERRAFORM_PATHS); "+
						`do $(TERRAFORM_DOCS) --config .terraform-docs.yml "$$path" || exit 1; done`+"\n",
						"the run of terraform-docs").
					Contains("\ngenerate: generate-terraform\n", "the aggregate generate").
					Contains("\n\t@$(call verify-generated,TERRAFORM_GENERATE,generate-terraform)\n",
						"the target that checks the tables")
			})

			t.Run("renders no target of the step generate without a generator", func(t *testing.T) {
				t.Parallel()
				makefile := rendered(t, baseline.Producer{}.Options(), "Makefile")
				assert.NotContains(t, makefile, "generate", "the Makefile")
			})

			t.Run("runs terraform-docs with the configuration of the repository on each path", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Paths = option.Paths{"modules", "examples"}
				o.Generate.Docs = true
				dir := files.Workspace(t, files.Tree{"Makefile": files.Text(rendered(t, o, "Makefile"))})
				// echo writes the arguments of terraform-docs in place of a run.
				out, err := makeTarget(t, dir, "generate-terraform", "TERRAFORM_DOCS=echo")
				assert.NoError(t, err, "make generate-terraform:\n"+out)
				expect.That(t, out).
					Contains("\n--config .terraform-docs.yml modules\n", "the run on the path modules").
					Contains("\n--config .terraform-docs.yml examples\n", "the run on the path examples")
			})

			t.Run("enables each ruleset of lint.rulesets in .tflint.hcl", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Lint.Rulesets = []baseline.Ruleset{baseline.RulesetAWS, baseline.RulesetGoogle}
				assert.HasSuffix(t, rendered(t, o, ".tflint.hcl"), "\n  preset  = \"all\"\n}\n\n"+
					"plugin \"aws\" {\n  enabled = true\n}\n\nplugin \"google\" {\n  enabled = true\n}\n",
					".tflint.hcl")
			})

			t.Run("links each ruleset of lint.rulesets into the plugin directory of tflint", func(t *testing.T) {
				t.Parallel()
				o, _ := baseline.Producer{}.Options().(*baseline.Options)
				o.Lint.Rulesets = []baseline.Ruleset{baseline.RulesetAWS, baseline.RulesetGoogle}
				expect.That(t, rendered(t, o, "Makefile")).
					Contains("\nTERRAFORM_LINT_RULESETS ?= aws google\n", "the rulesets").
					Contains("\n\t\tprogram=\"$$($(ERGON) tool path terraform.tflint-ruleset-$$ruleset)\" && \\\n"+
						"\t\tln -sf \"$$program\" \"$(TFLINT_PLUGIN_DIR)/\" || exit 1; done\n", "the link of each ruleset").
					Contains("\n\tTFLINT_PLUGIN_DIR=\"$(TFLINT_PLUGIN_DIR)\" $(TFLINT) --init "+
						"--config=\"$(CURDIR)/.tflint.hcl\"\n", "the install of the plugins of the local file")
			})

			t.Run("runs terraform test in each directory with test files", func(t *testing.T) {
				t.Parallel()
				out, err := makeTarget(t, terraformRepository(t), "test-terraform", "TERRAFORM=echo")
				assert.NoError(t, err, "make test-terraform:\n"+out)
				expect.That(t, out).
					Contains("\n-chdir=modules/one init -backend=false -input=false\n-chdir=modules/one test\n",
						"the run of the module with a tests directory").
					Contains("\n-chdir=modules/two init -backend=false -input=false\n-chdir=modules/two test\n",
						"the run of the module with a test file beside its configuration").
					NotContains("-chdir=modules/one/tests", "the run of the tests directory")
			})

			t.Run("runs terraform validate in each other directory with a configuration", func(t *testing.T) {
				t.Parallel()
				out, err := makeTarget(t, terraformRepository(t), "test-terraform", "TERRAFORM=echo")
				assert.NoError(t, err, "make test-terraform:\n"+out)
				expect.That(t, out).
					Contains("\n-chdir=examples/complete init -backend=false -input=false\n"+
						"-chdir=examples/complete validate\n", "the run of the example").
					NotContains("-chdir=modules/one validate", "the run of the module with tests")
			})

			t.Run("skips the directories that git ignores", func(t *testing.T) {
				t.Parallel()
				out, err := makeTarget(t, terraformRepository(t), "test-terraform", "TERRAFORM=echo")
				assert.NoError(t, err, "make test-terraform:\n"+out)
				assert.NotContains(t, out, ".terraform/modules", "the output of make test-terraform")
			})

			t.Run("fails after it runs every directory when a directory fails", func(t *testing.T) {
				t.Parallel()
				out, err := makeTarget(t, terraformRepository(t), "test-terraform", "TERRAFORM=false")
				exit := assert.ErrorAs[*exec.ExitError](t, err, "the error of make test-terraform:\n"+out)
				assert.Equal(t, exit.ExitCode(), 2, "the exit status of make after a target fails")
				expect.That(t, out).
					Contains("terraform test modules/one\n", "the run of the first module").
					Contains("terraform test modules/two\n", "the run of the second module").
					Contains("terraform validate examples/complete\n", "the run of the example")
			})

			t.Run("reports a repository without a configuration", func(t *testing.T) {
				t.Parallel()
				dir := vcstest.Repository(t, files.Tree{
					"Makefile":  files.Text(rendered(t, baseline.Producer{}.Options(), "Makefile")),
					"README.md": files.Text("# demo\n"),
				})
				out, err := makeTarget(t, dir, "test-terraform", "TERRAFORM=false")
				assert.NoError(t, err, "make test-terraform:\n"+out)
				assert.Contains(t, out, "no configuration of Terraform in .\n", "the output of make test-terraform")
			})
		})

		t.Run("Options", func(t *testing.T) {
			t.Parallel()

			t.Run("returns a new value on each call", func(t *testing.T) {
				t.Parallel()
				first, _ := baseline.Producer{}.Options().(*baseline.Options)
				second, _ := baseline.Producer{}.Options().(*baseline.Options)
				first.Paths[0] = "modules"
				assert.Equal(t, second.Paths, option.Paths{"."}, "the paths of the second value")
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

// catalog returns a catalog with Terraform.
func catalog(t *testing.T) *language.Catalog {
	t.Helper()
	c := new(language.Catalog)
	assert.NoError(t, terraform.Register(c, repository), "Register of Terraform")
	return c
}

// rendered returns the file name that the producer of Terraform renders for the options o and the
// answers of the cases, and stops the test when it renders none.
func rendered(t *testing.T, o language.Options, name string) string {
	t.Helper()
	files, err := render.Render([]render.Unit{{Name: baseline.Name, Producer: baseline.Producer{}, Options: o}},
		baselinetest.Answers(terraform.Language), &workflow.Contribution{})
	assert.NoError(t, err, "Render")
	i := slices.IndexFunc(files, func(f render.File) bool { return f.Path == name })
	assert.InRange(t, i, 0, float64(len(files)-1), "Terraform renders "+name)
	return string(files[i].Content)
}

// terraformRepository returns a working tree of git with the Makefile of Terraform at the baseline
// and the configurations of a repository of modules: a module with a tests directory, a module with
// a test file beside its configuration, an example without tests, and a module of .terraform, which
// git ignores.
func terraformRepository(t *testing.T) string {
	t.Helper()
	return vcstest.Repository(t, files.Tree{
		"Makefile":                         files.Text(rendered(t, baseline.Producer{}.Options(), "Makefile")),
		".gitignore":                       files.Text(".terraform/\n"),
		"modules/one/main.tf":              files.Text("output \"value\" {\n  value = 1\n}\n"),
		"modules/one/tests/one.tftest.hcl": files.Text("run \"value\" {\n  command = plan\n}\n"),
		"modules/two/main.tf":              files.Text("output \"value\" {\n  value = 2\n}\n"),
		"modules/two/two.tftest.hcl":       files.Text("run \"value\" {\n  command = plan\n}\n"),
		"examples/complete/main.tf":        files.Text("module \"one\" {\n  source = \"../../modules/one\"\n}\n"),
		".terraform/modules/one/main.tf":   files.Text("output \"value\" {\n  value = 1\n}\n"),
	})
}

// makeTarget runs make with args in dir, and returns the output and the error of make.
func makeTarget(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "make", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
