// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/java/baseline"
)

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("Tools", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for the tools at the baseline", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, javaOptions().Tools.Validate(), "Validate")
			})

			t.Run("returns nil for another version of pmd-java", func(t *testing.T) {
				t.Parallel()
				tools := baseline.Tools{PMD: "net.sourceforge.pmd:pmd-java@7.29.0"}
				assert.NoError(t, tools.Validate(), "Validate")
			})

			t.Run("returns ErrInvalid for another artifact of PMD", func(t *testing.T) {
				t.Parallel()
				tools := baseline.Tools{PMD: "net.sourceforge.pmd:pmd-kotlin@7.28.0"}
				err := tools.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.HasSuffix(
					t,
					err.Error(),
					`pmd "net.sourceforge.pmd:pmd-kotlin@7.28.0", which is not a version of net.sourceforge.pmd:pmd-java`,
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

		t.Run("returns ErrInvalid for a step that Java does not have", func(t *testing.T) {
			t.Parallel()
			o := javaOptions()
			o.Check = option.Check{option.StepFmt}
			assert.ErrorIs(t, o.Validate(), option.ErrInvalid, "Validate")
		})
	})

	t.Run("Contribution", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the job of Java, which runs the setup of the jvm toolchain", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, javaOptions().Contribution(), workflow.Contribution{
				Jobs: []workflow.Job{{
					ID:          "check-java",
					Name:        "Java",
					Toolchain:   "jvm",
					Permissions: map[string]string{"contents": "read"},
					Tools:       true,
					Steps:       []workflow.Step{{Name: "Check Java", Run: []string{"make check-java"}}},
				}},
			}, "the contribution")
		})

		t.Run("returns a valid contribution", func(t *testing.T) {
			t.Parallel()
			got := javaOptions().Contribution()
			assert.NoError(t, got.Validate(), "Validate of the contribution")
		})
	})
}
