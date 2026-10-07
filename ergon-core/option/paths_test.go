// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

func TestPaths(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give option.Paths
		}{
			{name: "returns nil for a pattern of every package", give: option.Paths{"./..."}},
			{name: "returns nil for pathspecs with a space", give: option.Paths{"*.sh", "my scripts/*.bash"}},
			{name: "returns nil for no path", give: option.Paths{}},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, tt.give.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give option.Paths
		}{
			{name: "returns ErrInvalid for an empty path", give: option.Paths{""}},
			{name: "returns ErrInvalid for a path named twice", give: option.Paths{".", "."}},
			{name: "returns ErrInvalid for a path that spans lines", give: option.Paths{"a\nb"}},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})
}
