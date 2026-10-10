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

	t.Run("Command", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give common.Command
			}{
				{name: "returns nil for the ergon on the PATH", give: common.Command{"ergon"}},
				{
					name: "returns nil for go run of the package of ergon",
					give: common.Command{"go", "run", "go.dokimi.dev/ergon/cmd/ergon"},
				},
				{
					name: "returns nil for go run of a version of ergon",
					give: common.Command{"go", "run", "go.dokimi.dev/ergon/cmd/ergon@v0.7.0"},
				},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give common.Command
			}{
				{name: "returns ErrInvalid for a command without a word", give: common.Command{}},
				{name: "returns ErrInvalid for an empty word", give: common.Command{"go", ""}},
				{name: "returns ErrInvalid for a word with a space", give: common.Command{"go run"}},
				{name: "returns ErrInvalid for a word with a dollar sign", give: common.Command{"$(ERGON)"}},
				{name: "returns ErrInvalid for a word with a number sign", give: common.Command{"ergon#1"}},
				{name: "returns ErrInvalid for a word with a single quote", give: common.Command{"'ergon'"}},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
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

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the repository of the releases of commitlint", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, common.Commitlint{}.Repository(), "conventionalcommit/commitlint", "Repository")
			})
		})
	})
}
