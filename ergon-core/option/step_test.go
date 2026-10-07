// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

func TestStep(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for every step", func(t *testing.T) {
			t.Parallel()
			assert.Total(t, option.Step.Validate, []option.Step{
				"fmt", "lint", "test", "race", "fuzz", "bench", "mutate", "generate", "audit",
			}, "Validate of the spelling of each step")
		})

		invalid := []struct {
			name string
			give option.Step
		}{
			{name: "returns ErrInvalid for the empty step", give: ""},
			{name: "returns ErrInvalid for a step that ergon does not have", give: "deploy"},
			{name: "returns ErrInvalid for a step in uppercase", give: "Lint"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give option.Check
			}{
				{name: "returns nil for steps in any order", give: option.Check{option.StepTest, option.StepLint}},
				{name: "returns nil for no step", give: option.Check{}},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give option.Check
			}{
				{
					name: "returns ErrInvalid for a step that ergon does not have",
					give: option.Check{option.StepLint, "deploy"},
				},
				{
					name: "returns ErrInvalid for a step named twice",
					give: option.Check{option.StepLint, option.StepLint},
				},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})

		t.Run("Only", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for steps that the producer has", func(t *testing.T) {
				t.Parallel()
				check := option.Check{option.StepLint, option.StepAudit}
				assert.NoError(t, check.Only(option.StepLint, option.StepTest, option.StepAudit), "Only")
			})

			t.Run("returns ErrInvalid for a step that the producer does not have", func(t *testing.T) {
				t.Parallel()
				check := option.Check{option.StepLint, option.StepRace}
				assert.ErrorIs(t, check.Only(option.StepLint, option.StepTest), option.ErrInvalid, "Only")
			})
		})
	})
}
