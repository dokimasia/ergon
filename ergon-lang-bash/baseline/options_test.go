// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/bash/baseline"
)

// release is the address of the assets of shellcheck 0.11.0, before the name of the asset.
const release = "https://github.com/koalaman/shellcheck/releases/download/v0.11.0/shellcheck-v0.11.0"

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step other than lint", func(t *testing.T) {
			t.Parallel()
			o := bashOptions()
			o.Check = option.Check{option.StepTest}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job of Bash on every runner at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, bashOptions().Contribution(), workflow.Contribution{Jobs: []workflow.Job{{
				ID:          "check-bash",
				Name:        "Bash",
				Permissions: map[string]string{"contents": "read"},
				Setup:       &workflow.Setup{Runners: option.Runners{}, Timeout: 30},
				Steps:       []workflow.Step{{Name: "Check Bash", Run: []string{"make check-bash"}}},
			}}}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := bashOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})
	})

	t.Run("Shellcheck", func(t *testing.T) {
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
					want: option.Asset{URL: release + ".linux.x86_64.tar.gz", Program: "shellcheck-v0.11.0/shellcheck"},
				},
				{
					name: "returns the .tar.gz of Linux on ARM",
					give: option.LinuxARM64,
					want: option.Asset{
						URL:     release + ".linux.aarch64.tar.gz",
						Program: "shellcheck-v0.11.0/shellcheck",
					},
				},
				{
					name: "returns the .tar.gz of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{
						URL:     release + ".darwin.aarch64.tar.gz",
						Program: "shellcheck-v0.11.0/shellcheck",
					},
				},
				{
					name: "returns the .zip of Windows on x86-64 with the program at its root",
					give: option.WindowsAMD64,
					want: option.Asset{URL: release + ".zip", Program: "shellcheck.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := bashOptions().Tools.Shellcheck.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			invalid := []struct {
				name string
				give option.Platform
			}{
				{
					name: "returns ErrNoAsset for Windows on ARM, which the release does not publish",
					give: option.WindowsARM64,
				},
				{name: "returns ErrNoAsset for an invalid platform", give: "plan9/amd64"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					_, err := bashOptions().Tools.Shellcheck.Asset(tt.give)
					assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
				})
			}
		})
	})
}

// bashOptions returns new options of the section bash at the baseline.
func bashOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
