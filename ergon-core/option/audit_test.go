// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

func TestAudit(t *testing.T) {
	t.Parallel()

	t.Run("Severity", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for every severity", func(t *testing.T) {
				t.Parallel()
				assert.Total(t, option.Severity.Validate, []option.Severity{"low", "moderate", "high", "critical"},
					"Validate of the spelling of each severity")
			})

			invalid := []struct {
				name string
				give option.Severity
			}{
				{name: "returns ErrInvalid for the empty severity", give: ""},
				{name: "returns ErrInvalid for a level of npm audit outside the scale", give: "info"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give option.Audit
		}{
			{name: "returns nil for no advisory", give: option.Audit{}},
			{
				name: "returns nil for advisories of PyPI, RustSec and GitHub, and a check of checkov",
				give: option.Audit{
					Ignore: []string{"PYSEC-2024-1", "RUSTSEC-2024-0001", "GHSA-4374-p667-p6c8", "CKV_AWS_20"},
				},
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
			give option.Audit
		}{
			{name: "returns ErrInvalid for an empty identifier", give: option.Audit{Ignore: []string{""}}},
			{
				name: "returns ErrInvalid for an identifier with a space",
				give: option.Audit{Ignore: []string{"PYSEC 1"}},
			},
			{
				name: "returns ErrInvalid for an identifier named twice",
				give: option.Audit{Ignore: []string{"A-1", "A-1"}},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})

	t.Run("Threshold", func(t *testing.T) {
		t.Parallel()

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for a severity of the scale", func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, option.Threshold{Severity: option.SeverityHigh}.Validate(), "Validate")
			})

			t.Run("returns ErrInvalid for a severity outside the scale", func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, option.Threshold{Severity: "info"}.Validate(), option.ErrInvalid, "Validate")
			})
		})
	})
}
