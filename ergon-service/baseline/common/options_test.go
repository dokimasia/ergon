// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/baseline/common"
)

// release is the address of the assets of commitlint 0.12.0.
const release = "https://github.com/conventionalcommit/commitlint/releases/download/v0.12.0/"

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, common.Producer{}.Options().Validate(), "Validate")
		})
	})

	t.Run("Commitlint", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the archive of Linux on x86-64",
					give: option.LinuxAMD64,
					want: option.Asset{URL: release + "commitlint_v0.12.0_linux_amd64.tar.gz", Program: "commitlint"},
				},
				{
					name: "returns the archive of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{URL: release + "commitlint_v0.12.0_darwin_arm64.tar.gz", Program: "commitlint"},
				},
				{
					name: "returns the archive of Windows with the program of the suffix exe",
					give: option.WindowsAMD64,
					want: option.Asset{
						URL:     release + "commitlint_v0.12.0_windows_amd64.tar.gz",
						Program: "commitlint.exe",
					},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					o, _ := common.Producer{}.Options().(*common.Options)
					got, err := o.Tools.Commitlint.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for an invalid platform", func(t *testing.T) {
				t.Parallel()
				o, _ := common.Producer{}.Options().(*common.Options)
				_, err := o.Tools.Commitlint.Asset("plan9/amd64")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
			})
		})
	})
}
