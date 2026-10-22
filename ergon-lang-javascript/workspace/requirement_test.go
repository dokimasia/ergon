// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"testing"

	"go.dokimi.dev/assert"
	jsworkspace "go.dokimi.dev/ergon/lang/javascript/workspace"
)

func TestRequirement(t *testing.T) {
	t.Parallel()

	t.Run("Shorthand", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want bool
		}{
			{name: "reports true for the workspace protocol alone", give: "workspace:", want: true},
			{name: "reports true for workspace:*", give: "workspace:*", want: true},
			{name: "reports true for workspace:^", give: "workspace:^", want: true},
			{name: "reports true for workspace:~", give: "workspace:~", want: true},
			{name: "reports true for a path in the directory of the package", give: "workspace:./a", want: true},
			{name: "reports true for a path in the parent directory", give: "workspace:../a", want: true},
			{name: "reports false for a range after the workspace protocol", give: "workspace:^1.2.0"},
			{name: "reports false for a path without a dot", give: "workspace:packages/a"},
			{name: "reports false for a range without the workspace protocol", give: "^1.2.0"},
			{name: "reports false for a star without the workspace protocol", give: "*"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, jsworkspace.Shorthand(tt.give), tt.want, "Shorthand")
			})
		}
	})
}
