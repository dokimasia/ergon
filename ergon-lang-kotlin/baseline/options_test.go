// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/kotlin/baseline"
)

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Tools", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for the tools at the baseline", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, kotlinOptions().Tools.Validate(), "Validate")
			})

			t.Run("returns nil for another version of ktlint-cli", func(t *testing.T) {
				t.Parallel()
				tools := baseline.Tools{Ktlint: "com.pinterest.ktlint:ktlint-cli@1.9.0"}
				assert.NoError(t, tools.Validate(), "Validate")
			})

			t.Run("returns ErrInvalid for another artifact of ktlint", func(t *testing.T) {
				t.Parallel()
				tools := baseline.Tools{Ktlint: "com.pinterest:ktlint@0.50.0"}
				err := tools.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.HasSuffix(
					t,
					err.Error(),
					`ktlint "com.pinterest:ktlint@0.50.0", which is not a version of com.pinterest.ktlint:ktlint-cli`,
					"the error",
				)
			})
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for the options at the baseline", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, baseline.Producer{}.Options().Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for a step that Kotlin does not have", func(t *testing.T) {
			t.Parallel()
			o := kotlinOptions()
			o.Check = option.Check{option.StepFuzz}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})

		t.Run("returns nil for the step generate with a command", func(t *testing.T) {
			t.Parallel()
			o := kotlinOptions()
			o.Check = option.Check{option.StepGenerate}
			o.Generate.Command = []string{"./gradlew", "openApiGenerate"}
			assert.NoError(t, o.Validate(), "Validate")
		})

		t.Run("returns ErrInvalid for the step generate without a command", func(t *testing.T) {
			t.Parallel()
			o := kotlinOptions()
			o.Check = option.Check{option.StepGenerate}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job of Kotlin, which runs the setup of the jvm toolchain", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, kotlinOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-kotlin",
					Name:        "Kotlin",
					Toolchain:   "jvm",
					Permissions: map[string]string{"contents": "read"},
					Tools:       true,
					Steps:       []workflow.Step{{Name: "Check Kotlin", Run: []string{"make check-kotlin"}}},
				}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := kotlinOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})
	})
}
