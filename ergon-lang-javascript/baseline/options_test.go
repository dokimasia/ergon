// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
)

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step that JavaScript does not have", func(t *testing.T) {
			t.Parallel()
			o := javascriptOptions()
			o.Check = option.Check{option.StepFmt}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job of JavaScript, which runs the setup of the js toolchain", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, javascriptOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-javascript",
					Name:        "JavaScript",
					Toolchain:   "js",
					Permissions: map[string]string{"contents": "read"},
					Steps:       []workflow.Step{{Name: "Check JavaScript", Run: []string{"make check-javascript"}}},
				}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := javascriptOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})
	})
}

// javascriptOptions returns new options of the section javascript at the baseline.
func javascriptOptions() *baseline.Options {
	o, _ := baseline.Producer{}.Options().(*baseline.Options)
	return o
}
