// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workspace"
)

// names are the cases of Valid that Language and Toolchain share. The byte ranges a to z and 0 to
// 9 are bounded by the cases at both ends and by the bytes just outside them: ` before a, { after
// z, / before 0 and : after 9.
var names = []struct {
	name string
	give string
	want bool
}{
	{name: "reports true for one letter", give: "a", want: true},
	{name: "reports true for a word of letters", give: "kotlin", want: true},
	{name: "reports true for the letters a and z", give: "az", want: true},
	{name: "reports true for the digits 0 and 9 after a letter", give: "c09", want: true},
	{name: "reports false for the empty name", give: "", want: false},
	{name: "reports false for a leading digit", give: "9c", want: false},
	{name: "reports false for an uppercase letter", give: "Go", want: false},
	{name: "reports false for the byte before a", give: "a`", want: false},
	{name: "reports false for the byte after z", give: "a{", want: false},
	{name: "reports false for the byte before 0", give: "a/", want: false},
	{name: "reports false for the byte after 9", give: "a:", want: false},
	{name: "reports false for a hyphen", give: "c-sharp", want: false},
}

func TestName(t *testing.T) {
	t.Parallel()

	t.Run("Language", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			for _, tt := range names {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, workspace.Language(tt.give).Valid(), tt.want, "Valid of "+tt.give)
				})
			}
		})
	})

	t.Run("Toolchain", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			for _, tt := range names {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, workspace.Toolchain(tt.give).Valid(), tt.want, "Valid of "+tt.give)
				})
			}
		})
	})
}
