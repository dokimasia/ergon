// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/java/baseline"
)

// version is the version of the release binaries of the cases.
const version = "1.2.3"

// release is the address of the assets of osv-scanner at the version of the cases.
const release = "https://github.com/google/osv-scanner/releases/download/v1.2.3/"

// setupJava is the pin of setup-java at the baseline.
var setupJava = jvmOptions().CI.Actions.SetupJava

// installTypeScript is a step of ci.steps, which installs a compiler that the tests run.
var installTypeScript = workflow.Step{Name: "Install TypeScript", Run: []string{"npm install --global typescript"}}

func TestToolchainOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Toolchain{}.Options().Validate(), "Validate")
		})

		t.Run("returns nil for options without a value", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, new(baseline.ToolchainOptions).Validate(), "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the setup, the analysis and the updates of the jvm toolchain at the baseline",
			func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, jvmOptions().Contribution(), workflow.Contribution{
					Setup: &workflow.Setup{
						Files:    ".java-version",
						Runners:  option.Runners{},
						Versions: []string{},
						Steps: []workflow.Step{{
							Name: "Set up Java",
							Uses: setupJava,
							With: map[string]string{"distribution": "temurin", "java-version-file": ".java-version"},
						}},
						Timeout: 30,
					},
					CodeQL: []workflow.CodeQL{{
						Language:  "java-kotlin",
						Name:      "Java and Kotlin",
						BuildMode: "autobuild",
						Files:     ".java-version",
						Timeout:   30,
					}},
					Updates: []workflow.Update{{Ecosystem: "gradle", Directories: []string{"/"}}},
				}, "the contribution")
			})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := jvmOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})

		t.Run("adds the steps of ci.steps to the end of the setup", func(t *testing.T) {
			t.Parallel()
			o := jvmOptions()
			o.CI.Steps = []workflow.Step{installTypeScript}
			want := slices.Concat(jvmOptions().Contribution().Setup.Steps, o.CI.Steps)
			assert.Equal(t, o.Contribution().Setup.Steps, want, "the steps of the setup")
		})

		t.Run("installs the Java of the matrix for versions of the options", func(t *testing.T) {
			t.Parallel()
			o := jvmOptions()
			o.CI.Versions = []string{"21", "25"}
			setup := o.Contribution().Setup
			assert.Equal(t, setup.Steps[0].With,
				map[string]string{"distribution": "temurin", "java-version": "${{ matrix.version }}"},
				"the inputs of setup-java")
		})
	})

	t.Run("OSVScanner", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the program of Linux on x86-64",
					give: option.LinuxAMD64,
					want: option.Asset{URL: release + "osv-scanner_linux_amd64"},
				},
				{
					name: "returns the program of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{URL: release + "osv-scanner_darwin_arm64"},
				},
				{
					name: "returns the program of Windows with the suffix exe",
					give: option.WindowsAMD64,
					want: option.Asset{URL: release + "osv-scanner_windows_amd64.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := baseline.OSVScanner{Version: version}.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for an invalid platform", func(t *testing.T) {
				t.Parallel()
				_, err := baseline.OSVScanner{Version: version}.Asset("plan9/amd64")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
			})
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the repository of the releases of osv-scanner", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, jvmOptions().Tools.OSVScanner.Repository(), "google/osv-scanner", "Repository")
			})
		})
	})
}
