// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/java/baseline"
)

// formatRelease is the address of the assets of google-java-format 1.37.0.
const formatRelease = "https://github.com/google/google-java-format/releases/download/v1.37.0/"

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Tools", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for the tools at the baseline", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, javaOptions().Tools.Validate(), "Validate")
			})

			t.Run("returns nil for another version of pmd-java", func(t *testing.T) {
				t.Parallel()
				tools := baseline.Tools{PMD: "net.sourceforge.pmd:pmd-java@7.29.0"}
				assert.NoError(t, tools.Validate(), "Validate")
			})

			t.Run("returns ErrInvalid for another artifact of PMD", func(t *testing.T) {
				t.Parallel()
				tools := baseline.Tools{PMD: "net.sourceforge.pmd:pmd-kotlin@7.28.0"}
				err := tools.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.HasSuffix(
					t,
					err.Error(),
					`pmd "net.sourceforge.pmd:pmd-kotlin@7.28.0", which is not a version of net.sourceforge.pmd:pmd-java`,
					"the error",
				)
			})
		})
	})

	t.Run("GoogleJavaFormat", func(t *testing.T) {
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
					want: option.Asset{URL: formatRelease + "google-java-format_linux-x86-64"},
				},
				{
					name: "returns the program of Linux on 64-bit ARM",
					give: option.LinuxARM64,
					want: option.Asset{URL: formatRelease + "google-java-format_linux-arm64"},
				},
				{
					name: "returns the program of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{URL: formatRelease + "google-java-format_darwin-arm64"},
				},
				{
					name: "returns the program of Windows with the suffix exe",
					give: option.WindowsAMD64,
					want: option.Asset{URL: formatRelease + "google-java-format_windows-x86-64.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := javaOptions().Tools.GoogleJavaFormat.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			absent := []struct {
				name string
				give option.Platform
			}{
				{name: "returns ErrNoAsset for macOS on x86-64", give: option.DarwinAMD64},
				{name: "returns ErrNoAsset for Windows on 64-bit ARM", give: option.WindowsARM64},
				{name: "returns ErrNoAsset for an invalid platform", give: "plan9/amd64"},
			}
			for _, tt := range absent {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					_, err := javaOptions().Tools.GoogleJavaFormat.Asset(tt.give)
					assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
				})
			}
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the repository of the releases of google-java-format", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, javaOptions().Tools.GoogleJavaFormat.Repository(), "google/google-java-format",
					"Repository")
			})
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step that Java does not have", func(t *testing.T) {
			t.Parallel()
			o := javaOptions()
			o.Check = option.Check{option.StepFmt}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns nil for the step generate with a command", func(t *testing.T) {
			t.Parallel()
			o := javaOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate.Command = []string{"./gradlew", "generateProto"}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for the step generate without a command", func(t *testing.T) {
			t.Parallel()
			o := javaOptions()
			o.Check = option.Check{option.StepGenerate}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job of Java, which runs the setup of the jvm toolchain", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, javaOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-java",
					Name:        "Java",
					Toolchain:   "jvm",
					Permissions: map[string]string{"contents": "read"},
					Tools:       true,
					Steps:       []workflow.Step{{Name: "Check Java", Run: []string{"make check-java"}}},
				}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := javaOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})
	})
}
