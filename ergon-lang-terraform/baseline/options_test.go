// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/terraform/baseline"
)

// The spellings of the rulesets that the section pins. A repository writes them in lint.rulesets,
// so the test pins them.
const (
	pinnedAWS     baseline.Ruleset = "aws"
	pinnedAzureRM baseline.Ruleset = "azurerm"
	pinnedGoogle  baseline.Ruleset = "google"
)

// setupTerraform is the pin of setup-terraform at the baseline.
var setupTerraform = terraformOptions().CI.Actions.SetupTerraform

// installTypeScript is a step of ci.steps, which installs a compiler that the tests run.
var installTypeScript = workflow.Step{Name: "Install TypeScript", Run: []string{"npm install --global typescript"}}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step that Terraform does not have", func(t *testing.T) {
			t.Parallel()
			o := terraformOptions()
			o.Check = option.Check{option.StepBench}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns nil for the step generate with a command line", func(t *testing.T) {
			t.Parallel()
			o := terraformOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate.Commands = [][]string{{"terraform-docs", "markdown", "table"}}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns nil for the step generate with docs", func(t *testing.T) {
			t.Parallel()
			o := terraformOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate.Docs = true
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for the step generate without a generator", func(t *testing.T) {
			t.Parallel()
			o := terraformOptions()
			o.Check = option.Check{option.StepGenerate}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job and the updates of Terraform at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, terraformOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-terraform",
					Name:        "Terraform",
					Permissions: map[string]string{"contents": "read"},
					Setup: &workflow.Setup{
						Files:    ".terraform-version",
						Runners:  option.Runners{},
						Versions: []string{},
						Steps: []workflow.Step{
							{
								Name: "Read .terraform-version",
								ID:   "pin",
								Run:  []string{`echo "version=$(cat .terraform-version)" >> "$GITHUB_OUTPUT"`},
							},
							{
								Name: "Set up Terraform",
								Uses: setupTerraform,
								With: map[string]string{
									"terraform_version": "${{ steps.pin.outputs.version }}",
									"terraform_wrapper": "false",
								},
							},
						},
						Timeout: 30,
					},
					Tools: true,
					Steps: []workflow.Step{{Name: "Check Terraform", Run: []string{"make check-terraform"}}},
				}},
				Updates: []workflow.Update{{Ecosystem: "terraform", Directories: []string{"/", "/**/*"}}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := terraformOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("adds the steps of ci.steps to the end of the setup", func(t *testing.T) {
			t.Parallel()
			o := terraformOptions()
			o.CI.Steps = []workflow.Step{installTypeScript}
			want := slices.Concat(terraformOptions().Contribution().Jobs[0].Setup.Steps, o.CI.Steps)
			assert.Equal(t, o.Contribution().Jobs[0].Setup.Steps, want, "the steps of the setup")
		})

		t.Run("installs the version of the matrix for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := terraformOptions()
			o.CI.Versions = []string{"1.14.0"}
			setup := o.Contribution().Jobs[0].Setup
			assert.Equal(t, setup.Steps, []workflow.Step{{
				Name: "Set up Terraform",
				Uses: setupTerraform,
				With: map[string]string{"terraform_version": "${{ matrix.version }}", "terraform_wrapper": "false"},
			}}, "the setup steps")
		})
	})

	t.Run("Ruleset", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for each ruleset that the section pins", func(t *testing.T) {
				t.Parallel()
				assert.Total(t, func(r baseline.Ruleset) error { return r.Validate() },
					[]baseline.Ruleset{pinnedAWS, pinnedAzureRM, pinnedGoogle}, "Validate of each ruleset")
			})

			t.Run("returns ErrInvalid for a ruleset that the section does not pin", func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, baseline.Ruleset("opa").Validate(), option.ErrInvalid, "Validate")
			})
		})
	})

	t.Run("Lint", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for the rulesets that the section pins", func(t *testing.T) {
				t.Parallel()
				l := baseline.Lint{Rulesets: []baseline.Ruleset{pinnedGoogle, pinnedAWS}}
				assert.NoError(t, l.Validate(), "Validate")
			})

			t.Run("returns ErrInvalid for a ruleset that the section does not pin", func(t *testing.T) {
				t.Parallel()
				l := baseline.Lint{Rulesets: []baseline.Ruleset{pinnedAWS, "opa"}}
				assert.ErrorIs(t, l.Validate(), option.ErrInvalid, "Validate")
			})

			t.Run("returns ErrInvalid for a ruleset that rulesets lists twice", func(t *testing.T) {
				t.Parallel()
				l := baseline.Lint{Rulesets: []baseline.Ruleset{pinnedAWS, pinnedAzureRM, pinnedAWS}}
				assert.ErrorIs(t, l.Validate(), option.ErrInvalid, "Validate")
			})
		})
	})

	t.Run("Generate", func(t *testing.T) {
		t.Parallel()

		t.Run("CheckStep", func(t *testing.T) {
			t.Parallel()

			check := option.Check{option.StepLint, option.StepGenerate}

			t.Run("returns nil for docs without a command line", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, baseline.Generate{Docs: true}.CheckStep(check), "CheckStep")
			})

			t.Run("returns nil for a command line without docs", func(t *testing.T) {
				t.Parallel()
				g := baseline.Generate{Commands: [][]string{{"terraform-docs", "."}}}
				assert.NoError(t, g.CheckStep(check), "CheckStep")
			})

			t.Run("returns ErrInvalid for the step generate without a generator", func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, baseline.Generate{}.CheckStep(check), option.ErrInvalid, "CheckStep")
			})

			t.Run("returns nil for a check without the step generate", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, baseline.Generate{}.CheckStep(option.Check{option.StepLint}), "CheckStep")
			})
		})
	})
}

// terraformOptions returns new options of the section terraform at the baseline.
func terraformOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
