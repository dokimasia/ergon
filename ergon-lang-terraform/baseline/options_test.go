// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/terraform/baseline"
)

// release is the address of the assets of tflint 0.64.0.
const release = "https://github.com/terraform-linters/tflint/releases/download/v0.64.0/"

// setupTerraform is the pin of setup-terraform at the baseline.
var setupTerraform = workflow.Action{
	Uses:    "hashicorp/setup-terraform",
	Commit:  "dfe3c3f87815947d99a8997f908cb6525fc44e9e",
	Release: "v4.0.1",
}

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

	t.Run("TFLint", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the .zip of Linux on x86-64",
					give: option.LinuxAMD64,
					want: option.Asset{URL: release + "tflint_linux_amd64.zip", Program: "tflint"},
				},
				{
					name: "returns the .zip of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{URL: release + "tflint_darwin_arm64.zip", Program: "tflint"},
				},
				{
					name: "returns the .zip of Windows with the program of the suffix exe",
					give: option.WindowsAMD64,
					want: option.Asset{URL: release + "tflint_windows_amd64.zip", Program: "tflint.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := terraformOptions().Tools.TFLint.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for an invalid platform", func(t *testing.T) {
				t.Parallel()
				_, err := terraformOptions().Tools.TFLint.Asset("plan9/amd64")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
			})
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the repository of the releases of tflint", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, terraformOptions().Tools.TFLint.Repository(), "terraform-linters/tflint", "Repository")
			})
		})
	})
}

// terraformOptions returns new options of the section terraform at the baseline.
func terraformOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
