// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give option.Version
		}{
			{name: "returns nil for a version of three numbers", give: "4.4.1"},
			{name: "returns nil for a version that starts with a v", give: "v6.0.0"},
			{name: "returns nil for a version with a suffix of a build", give: "1.2.3-rc.1+build_7"},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, tt.give.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give option.Version
		}{
			{name: "returns ErrInvalid for the empty version", give: ""},
			{name: "returns ErrInvalid for a version with a space", give: "4.4 .1"},
			{name: "returns ErrInvalid for a version that starts with a hyphen", give: "-4.4.1"},
			{name: "returns ErrInvalid for a version with a semicolon", give: "4.4.1;exit"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})
}
