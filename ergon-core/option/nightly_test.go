// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

func TestNightly(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give option.Nightly
		}{
			{
				name: "returns nil for steps with a limit of a minute or more",
				give: option.Nightly{option.StepFuzz: 120, option.StepMutate: 1},
			},
			{name: "returns nil for no step", give: option.Nightly{}},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, tt.give.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give option.Nightly
			want string
		}{
			{
				name: "returns ErrInvalid for a step that ergon does not have",
				give: option.Nightly{"deploy": 30},
				want: `option: invalid value: step "deploy", which is none of`,
			},
			{
				name: "returns ErrInvalid for a limit below a minute",
				give: option.Nightly{option.StepFuzz: 120, option.StepBench: 0},
				want: "option: invalid value: the step bench has the limit 0, which is less than a minute",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.give.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.HasPrefix(t, err.Error(), tt.want, "the error")
			})
		}
	})

	t.Run("Only", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for steps that the producer has", func(t *testing.T) {
			t.Parallel()
			n := option.Nightly{option.StepFuzz: 120, option.StepMutate: 60}
			assert.NoError(t, n.Only(option.StepFuzz, option.StepBench, option.StepMutate), "Only")
		})

		t.Run("returns ErrInvalid that names the key nightly for a step that the producer does not have",
			func(t *testing.T) {
				t.Parallel()
				n := option.Nightly{option.StepFuzz: 120, option.StepTest: 10}
				err := n.Only(option.StepFuzz)
				assert.ErrorIs(t, err, option.ErrInvalid, "Only")
				want := "option: invalid value: nightly names test, which is no step of the section"
				assert.Equal(t, err.Error(), want, "the error")
			})
	})

	t.Run("Steps", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give option.Nightly
			want []option.Step
		}{
			{
				name: "returns the steps in the order of their targets",
				give: option.Nightly{option.StepMutate: 60, option.StepFuzz: 120, option.StepBench: 45},
				want: []option.Step{option.StepFuzz, option.StepBench, option.StepMutate},
			},
			{
				name: "returns the steps that ergon does not have last, in the order of their names",
				give: option.Nightly{"soak": 1, option.StepFuzz: 120, "deploy": 1},
				want: []option.Step{option.StepFuzz, "deploy", "soak"},
			},
			{name: "returns no step for an empty map", give: option.Nightly{}, want: []option.Step{}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Steps(), tt.want, "Steps")
			})
		}
	})
}
