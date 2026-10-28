// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/lang/terraform/baseline"
)

// version is the version of the release binaries of the cases.
const version = "1.2.3"

// The addresses of the assets of tflint and of terraform-docs at the version of the cases.
const (
	tflintRelease        = "https://github.com/terraform-linters/tflint/releases/download/v1.2.3/"
	terraformDocsRelease = "https://github.com/terraform-docs/terraform-docs/releases/download/v1.2.3/"
)

// pin is the pin of the release binaries of the cases: the version of the cases without digests.
var pin = option.Binary{Version: version}

func TestBinaries(t *testing.T) {
	t.Parallel()

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
					want: option.Asset{URL: tflintRelease + "tflint_linux_amd64.zip", Program: "tflint"},
				},
				{
					name: "returns the .zip of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{URL: tflintRelease + "tflint_darwin_arm64.zip", Program: "tflint"},
				},
				{
					name: "returns the .zip of Windows with the program of the suffix exe",
					give: option.WindowsAMD64,
					want: option.Asset{URL: tflintRelease + "tflint_windows_amd64.zip", Program: "tflint.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := baseline.TFLint{Binary: pin}.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for an invalid platform", func(t *testing.T) {
				t.Parallel()
				_, err := baseline.TFLint{Binary: pin}.Asset("plan9/amd64")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
			})
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the repository of the releases of tflint", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.TFLint{}.Repository(), "terraform-linters/tflint", "Repository")
			})
		})
	})

	t.Run("AWSRuleset", func(t *testing.T) {
		t.Parallel()
		ruleset(t, baseline.AWSRuleset{Binary: pin}, "aws")
	})

	t.Run("AzureRMRuleset", func(t *testing.T) {
		t.Parallel()
		ruleset(t, baseline.AzureRMRuleset{Binary: pin}, "azurerm")
	})

	t.Run("GoogleRuleset", func(t *testing.T) {
		t.Parallel()
		ruleset(t, baseline.GoogleRuleset{Binary: pin}, "google")
	})

	t.Run("TerraformDocs", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the .tar.gz of Linux on x86-64",
					give: option.LinuxAMD64,
					want: option.Asset{
						URL:     terraformDocsRelease + "terraform-docs-v1.2.3-linux-amd64.tar.gz",
						Program: "terraform-docs",
					},
				},
				{
					name: "returns the .tar.gz of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{
						URL:     terraformDocsRelease + "terraform-docs-v1.2.3-darwin-arm64.tar.gz",
						Program: "terraform-docs",
					},
				},
				{
					name: "returns the .zip of Windows with the program of the suffix exe",
					give: option.WindowsAMD64,
					want: option.Asset{
						URL:     terraformDocsRelease + "terraform-docs-v1.2.3-windows-amd64.zip",
						Program: "terraform-docs.exe",
					},
				},
				{
					name: "returns the .zip of Windows on 64-bit ARM",
					give: option.WindowsARM64,
					want: option.Asset{
						URL:     terraformDocsRelease + "terraform-docs-v1.2.3-windows-arm64.zip",
						Program: "terraform-docs.exe",
					},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := baseline.TerraformDocs{Binary: pin}.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for an invalid platform", func(t *testing.T) {
				t.Parallel()
				_, err := baseline.TerraformDocs{Binary: pin}.Asset("plan9/amd64")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
			})
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the repository of the releases of terraform-docs", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.TerraformDocs{}.Repository(), "terraform-docs/terraform-docs", "Repository")
			})
		})
	})
}

// ruleset runs the cases of the methods Asset and Repository of the release binary give of the
// ruleset of tflint named name, at the version of the cases.
func ruleset(t *testing.T, give option.Release, name string) {
	t.Helper()
	program := "tflint-ruleset-" + name
	release := "https://github.com/terraform-linters/" + program + "/releases/download/v1.2.3/" + program

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
				want: option.Asset{URL: release + "_linux_amd64.zip", Program: program},
			},
			{
				name: "returns the .zip of macOS on Apple silicon",
				give: option.DarwinARM64,
				want: option.Asset{URL: release + "_darwin_arm64.zip", Program: program},
			},
			{
				name: "returns the .zip of Windows with the program of the suffix exe",
				give: option.WindowsAMD64,
				want: option.Asset{URL: release + "_windows_amd64.zip", Program: program + ".exe"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := give.Asset(tt.give)
				assert.NoError(t, err, "Asset")
				assert.Equal(t, got, tt.want, "the asset")
			})
		}

		t.Run("returns ErrNoAsset for Windows on 64-bit ARM", func(t *testing.T) {
			t.Parallel()
			_, err := give.Asset(option.WindowsARM64)
			assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
		})

		t.Run("returns ErrNoAsset for an invalid platform", func(t *testing.T) {
			t.Parallel()
			_, err := give.Asset("plan9/amd64")
			assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
		})
	})

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the repository of the releases of the ruleset", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, give.Repository(), "terraform-linters/"+program, "Repository")
		})
	})
}
