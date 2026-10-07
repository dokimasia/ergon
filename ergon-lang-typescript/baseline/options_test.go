// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/typescript/baseline"
)

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step that TypeScript does not have", func(t *testing.T) {
			t.Parallel()
			o := typescriptOptions()
			o.Check = option.Check{option.StepFmt}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job of TypeScript, which runs the setup of the js toolchain", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, typescriptOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-typescript",
					Name:        "TypeScript",
					Toolchain:   "js",
					Permissions: map[string]string{"contents": "read"},
					Steps:       []workflow.Step{{Name: "Check TypeScript", Run: []string{"make check-typescript"}}},
				}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := typescriptOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})
	})
}

// typescriptOptions returns new options of the section typescript at the baseline.
func typescriptOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
