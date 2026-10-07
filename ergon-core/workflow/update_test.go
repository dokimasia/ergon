// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workflow"
)

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give workflow.Update
		}{
			{
				name: "returns nil for the root directory",
				give: workflow.Update{Ecosystem: "npm", Directories: []string{"/"}},
			},
			{
				name: "returns nil for a glob of every directory",
				give: workflow.Update{Ecosystem: "gomod", Directories: []string{"/", "/**/*"}},
			},
			{
				name: "returns nil for an ecosystem with a hyphen",
				give: workflow.Update{Ecosystem: "github-actions", Directories: []string{"/"}},
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
			give workflow.Update
		}{
			{
				name: "returns ErrInvalidUpdate for an empty ecosystem",
				give: workflow.Update{Directories: []string{"/"}},
			},
			{
				name: "returns ErrInvalidUpdate for an ecosystem in uppercase",
				give: workflow.Update{Ecosystem: "NPM", Directories: []string{"/"}},
			},
			{name: "returns ErrInvalidUpdate for no directory", give: workflow.Update{Ecosystem: "npm"}},
			{
				name: "returns ErrInvalidUpdate for a relative directory",
				give: workflow.Update{Ecosystem: "npm", Directories: []string{"web"}},
			},
			{
				name: "returns ErrInvalidUpdate for a directory that spans lines",
				give: workflow.Update{Ecosystem: "npm", Directories: []string{"/\n/web"}},
			},
			{
				name: "returns ErrInvalidUpdate for a directory named twice",
				give: workflow.Update{Ecosystem: "npm", Directories: []string{"/", "/"}},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), workflow.ErrInvalidUpdate, "Validate")
			})
		}
	})
}
