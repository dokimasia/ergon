// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
)

func TestToolkit(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for every toolkit", func(t *testing.T) {
			t.Parallel()
			assert.Total(t, baseline.Toolkit.Validate, []baseline.Toolkit{"biome", "vite-plus"},
				"Validate of the spelling of each toolkit")
		})

		invalid := []struct {
			name string
			give baseline.Toolkit
		}{
			{name: "returns ErrInvalid for the empty toolkit", give: ""},
			{name: "returns ErrInvalid for a toolkit that ergon does not run", give: "prettier"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})
}
