// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/baseline/common"
)

func TestHooks(t *testing.T) {
	t.Parallel()

	t.Run("Target", func(t *testing.T) {
		t.Parallel()

		t.Run("has the spelling of its aggregate target of the Makefile", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, common.TargetFmt, "fmt", "TargetFmt")
			expect.Equal(t, common.TargetLint, "lint", "TargetLint")
			expect.Equal(t, common.TargetTest, "test", "TargetTest")
			expect.Equal(t, common.TargetAudit, "audit", "TargetAudit")
			expect.Equal(t, common.TargetCheck, "check", "TargetCheck")
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for each aggregate target", func(t *testing.T) {
				t.Parallel()
				all := []common.Target{
					common.TargetFmt, common.TargetLint, common.TargetTest, common.TargetAudit, common.TargetCheck,
				}
				assert.Total(t, func(target common.Target) error { return target.Validate() }, all,
					"Validate of each aggregate target")
			})

			tests := []struct {
				name string
				give common.Target
				want string
			}{
				{
					name: "returns ErrInvalid for a step that is no aggregate target",
					give: "race",
					want: `option: invalid value: target "race", which is none of fmt, lint, test, audit and check`,
				},
				{
					name: "returns ErrInvalid for an empty target",
					give: "",
					want: `option: invalid value: target "", which is none of fmt, lint, test, audit and check`,
				},
				{
					name: "returns ErrInvalid for a target in another case",
					give: "Lint",
					want: `option: invalid value: target "Lint", which is none of fmt, lint, test, audit and check`,
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					err := tt.give.Validate()
					assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
					assert.Equal(t, err.Error(), tt.want, "the text of the error")
				})
			}
		})
	})

	t.Run("Targets", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			valid := []struct {
				name string
				give common.Targets
			}{
				{name: "returns nil for no target", give: common.Targets{}},
				{name: "returns nil for a nil list", give: nil},
				{
					name: "returns nil for targets in any order",
					give: common.Targets{common.TargetCheck, common.TargetFmt},
				},
			}
			for _, tt := range valid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.NoError(t, tt.give.Validate(), "Validate")
				})
			}

			invalid := []struct {
				name string
				give common.Targets
				want string
			}{
				{
					name: "returns ErrInvalid for a target that is no aggregate target",
					give: common.Targets{common.TargetLint, "race"},
					want: `option: invalid value: target "race", which is none of fmt, lint, test, audit and check`,
				},
				{
					name: "returns ErrInvalid for a target that the list names twice",
					give: common.Targets{common.TargetLint, common.TargetTest, common.TargetLint},
					want: "option: invalid value: target lint, which the stage names twice",
				},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					err := tt.give.Validate()
					assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
					assert.Equal(t, err.Error(), tt.want, "the text of the error")
				})
			}
		})
	})
}
