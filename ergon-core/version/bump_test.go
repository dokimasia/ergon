// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package version_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/version"
)

// levels are the four levels, from the lowest to the highest.
var levels = []version.Bump{version.BumpNone, version.BumpPatch, version.BumpMinor, version.BumpMajor}

func TestBump(t *testing.T) {
	t.Parallel()

	t.Run("ParseBump", func(t *testing.T) {
		t.Parallel()

		for _, b := range levels {
			t.Run("returns the level "+string(b), func(t *testing.T) {
				t.Parallel()
				got, err := version.ParseBump(string(b))
				assert.NoError(t, err, "ParseBump")
				assert.Equal(t, got, b, "the level")
			})
		}

		invalid := []struct {
			name string
			give string
		}{
			{name: "returns ErrInvalid for the empty level", give: ""},
			{name: "returns ErrInvalid for a level in capitals", give: "Major"},
			{name: "returns ErrInvalid for an unknown level", give: "huge"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := version.ParseBump(tt.give)
				assert.ErrorIs(t, err, version.ErrInvalid, "ParseBump")
			})
		}
	})

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for each level", func(t *testing.T) {
			t.Parallel()
			assert.Total(t, func(b version.Bump) error {
				if !b.Valid() {
					return version.ErrInvalid
				}
				return nil
			}, levels, "Valid of each level")
		})

		t.Run("reports false for the empty level", func(t *testing.T) {
			t.Parallel()
			assert.False(t, version.Bump("").Valid(), "Valid")
		})
	})

	t.Run("Max", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			a, b version.Bump
			want version.Bump
		}{
			{
				name: "returns the second level when it is higher", a: version.BumpNone, b: version.BumpPatch,
				want: version.BumpPatch,
			},
			{
				name: "returns the first level when it is higher", a: version.BumpMajor, b: version.BumpMinor,
				want: version.BumpMajor,
			},
			{
				name: "returns the level for two equal levels", a: version.BumpMinor, b: version.BumpMinor,
				want: version.BumpMinor,
			},
			{
				name: "returns the valid level over one that is not valid", a: "", b: version.BumpNone,
				want: version.BumpNone,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.a.Max(tt.b), tt.want, "Max")
			})
		}
	})

	t.Run("AtLeast", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			a, b version.Bump
			want bool
		}{
			{name: "reports true for a higher level", a: version.BumpMinor, b: version.BumpPatch, want: true},
			{name: "reports true for the same level", a: version.BumpPatch, b: version.BumpPatch, want: true},
			{name: "reports false for a lower level", a: version.BumpPatch, b: version.BumpMinor, want: false},
			{name: "reports false for a level that is not valid", a: "", b: version.BumpNone, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.a.AtLeast(tt.b), tt.want, "AtLeast")
			})
		}
	})
}
