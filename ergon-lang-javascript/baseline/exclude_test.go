// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
)

func TestExclude(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give baseline.Exclude
		}{
			{name: "returns nil for an empty list", give: baseline.Exclude{}},
			{name: "returns nil for a directory at the root", give: baseline.Exclude{"vendor"}},
			{name: "returns nil for a glob of files below a directory", give: baseline.Exclude{"src/**/*.gen.ts"}},
			{name: "returns nil for a directory at any depth", give: baseline.Exclude{"**/dist"}},
			{name: "returns nil for a path with the characters of a scope", give: baseline.Exclude{"@scope/a_b+c~d-e"}},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, tt.give.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give baseline.Exclude
		}{
			{name: "returns ErrInvalid for an empty glob", give: baseline.Exclude{""}},
			{name: "returns ErrInvalid for a glob that starts with a slash", give: baseline.Exclude{"/vendor"}},
			{name: "returns ErrInvalid for a glob that ends with a slash", give: baseline.Exclude{"vendor/"}},
			{name: "returns ErrInvalid for an empty segment", give: baseline.Exclude{"a//b"}},
			{name: "returns ErrInvalid for the segment of the current directory", give: baseline.Exclude{"./a"}},
			{name: "returns ErrInvalid for the segment of the parent directory", give: baseline.Exclude{"a/../b"}},
			{name: "returns ErrInvalid for a double star beside other characters", give: baseline.Exclude{"a**"}},
			{name: "returns ErrInvalid for two double stars in a row", give: baseline.Exclude{"**/**"}},
			{name: "returns ErrInvalid for a question mark", give: baseline.Exclude{"a?b"}},
			{name: "returns ErrInvalid for a class of characters", give: baseline.Exclude{"a[bc]"}},
			{name: "returns ErrInvalid for alternatives", give: baseline.Exclude{"{a,b}"}},
			{name: "returns ErrInvalid for a negation", give: baseline.Exclude{"!a"}},
			{name: "returns ErrInvalid for a space", give: baseline.Exclude{"a b"}},
			{name: "returns ErrInvalid for a glob that appears twice", give: baseline.Exclude{"a", "b", "a"}},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})
}
