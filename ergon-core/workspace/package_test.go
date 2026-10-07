// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workspace"
)

func TestPackage(t *testing.T) {
	t.Parallel()

	t.Run("Kind", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give workspace.Kind
				want bool
			}{
				{name: "reports true for runtime", give: workspace.KindRuntime, want: true},
				{name: "reports true for optional", give: workspace.KindOptional, want: true},
				{name: "reports true for peer", give: workspace.KindPeer, want: true},
				{name: "reports true for dev", give: workspace.KindDev, want: true},
				{name: "reports false for the zero value", give: "", want: false},
				{name: "reports false for an unknown kind", give: "build", want: false},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, tt.give.Valid(), tt.want, "Valid")
				})
			}
		})
	})

	t.Run("Package", func(t *testing.T) {
		t.Parallel()

		t.Run("states a package that has never been released with its zero value", func(t *testing.T) {
			t.Parallel()
			var p workspace.Package
			assert.True(t, p.Version.IsZero(), "the version of the zero value")
			assert.Empty(t, p.Deps, "the requirements of the zero value")
		})
	})
}
