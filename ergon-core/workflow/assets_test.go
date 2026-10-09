// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workflow"
)

func TestAssets(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give workflow.Assets
		}{
			{name: "returns nil for a tap as owner/name", give: workflow.Assets{Tap: "dokimasia/homebrew-tap"}},
			{name: "returns nil for assets without a tap", give: workflow.Assets{}},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, tt.give.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give string
		}{
			{name: "returns ErrInvalidAssets for a tap without an owner", give: "homebrew-tap"},
			{name: "returns ErrInvalidAssets for a tap with a path", give: "dokimasia/homebrew-tap/Casks"},
			{name: "returns ErrInvalidAssets for an owner that starts with a hyphen", give: "-dokimasia/homebrew-tap"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				a := workflow.Assets{Tap: tt.give}
				err := a.Validate()
				assert.ErrorIs(t, err, workflow.ErrInvalidAssets, "Validate")
				assert.Equal(t, err.Error(), `workflow: invalid assets: tap "`+tt.give+`", which is not owner/name`,
					"the error")
			})
		}
	})
}
