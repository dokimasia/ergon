// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/baseline/spec"
)

// specRelease is the address of the assets of spec 0.2.0, the release binary at the baseline.
const specRelease = "https://github.com/stealth-scale/spec/releases/download/v0.2.0/"

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Role", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give spec.Role
				want bool
			}{
				{name: "reports true for definition", give: "definition", want: true},
				{name: "reports true for implementation", give: "implementation", want: true},
				{name: "reports false for implements, the section of the role", give: "implements", want: false},
				{name: "reports false for a role in another case", give: "Definition", want: false},
				{name: "reports false for the empty role", give: "", want: false},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, tt.give.Valid(), tt.want, "Valid")
				})
			}
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give []spec.Role
		}{
			{name: "returns nil without a role", give: nil},
			{name: "returns nil for an implementation", give: []spec.Role{spec.Implementation}},
			{name: "returns nil for both roles", give: []spec.Role{spec.Definition, spec.Implementation}},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o, _ := spec.Producer{}.Options().(*spec.Options)
				o.Roles = tt.give
				assert.NoError(t, o.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give []spec.Role
			want string
		}{
			{
				name: "returns ErrInvalid for a role that is not valid",
				give: []spec.Role{"implements"},
				want: `option: invalid value: the role "implements", which is neither definition nor implementation`,
			},
			{
				name: "returns ErrInvalid for a role that the list repeats",
				give: []spec.Role{spec.Definition, spec.Definition},
				want: "option: invalid value: roles lists the role definition twice",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o, _ := spec.Producer{}.Options().(*spec.Options)
				o.Roles = tt.give
				err := o.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.Equal(t, err.Error(), tt.want, "the error")
			})
		}
	})

	t.Run("Spec", func(t *testing.T) {
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
					want: option.Asset{URL: specRelease + "spec_0.2.0_linux_amd64.tar.gz", Program: "spec"},
				},
				{
					name: "returns the archive of Linux on ARM",
					give: option.LinuxARM64,
					want: option.Asset{URL: specRelease + "spec_0.2.0_linux_arm64.tar.gz", Program: "spec"},
				},
				{
					name: "returns the archive of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{URL: specRelease + "spec_0.2.0_darwin_arm64.tar.gz", Program: "spec"},
				},
				{
					name: "returns the archive of Windows with the program of the suffix exe",
					give: option.WindowsAMD64,
					want: option.Asset{URL: specRelease + "spec_0.2.0_windows_amd64.tar.gz", Program: "spec.exe"},
				},
				{
					name: "returns the archive of Windows on ARM with the program of the suffix exe",
					give: option.WindowsARM64,
					want: option.Asset{URL: specRelease + "spec_0.2.0_windows_arm64.tar.gz", Program: "spec.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					o, _ := spec.Producer{}.Options().(*spec.Options)
					got, err := o.Tools.Spec.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for an invalid platform", func(t *testing.T) {
				t.Parallel()
				o, _ := spec.Producer{}.Options().(*spec.Options)
				_, err := o.Tools.Spec.Asset("plan9/amd64")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
			})
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the repository of the releases of spec", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, spec.Spec{}.Repository(), "stealth-scale/spec", "Repository")
			})
		})
	})
}
