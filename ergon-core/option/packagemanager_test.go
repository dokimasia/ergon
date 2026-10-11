// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

func TestPackageManager(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for every package manager", func(t *testing.T) {
			t.Parallel()
			assert.Total(t, option.PackageManager.Validate, []option.PackageManager{"npm", "pnpm"},
				"Validate of the spelling of each package manager")
		})

		invalid := []struct {
			name string
			give option.PackageManager
		}{
			{name: "returns ErrInvalid for the empty package manager", give: ""},
			{name: "returns ErrInvalid for a package manager that ergon does not run", give: "yarn"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})
}
