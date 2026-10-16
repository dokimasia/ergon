// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/lang/python/workspace"
)

func TestRequirement(t *testing.T) {
	t.Parallel()

	t.Run("ParseRequirement", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveText string
			want     workspace.Requirement
		}{
			{
				name:     "returns a name without a specifier",
				giveText: "probe-a",
				want:     workspace.Requirement{Name: "probe-a", Start: 7, End: 7},
			},
			{
				name:     "returns a name and its specifier",
				giveText: "probe-a>=0.1.0",
				want:     workspace.Requirement{Name: "probe-a", Specifier: ">=0.1.0", Start: 7, End: 14},
			},
			{
				name:     "returns the specifier between the extras and the marker",
				giveText: ` Probe_A [cli] >= 0.1, <1 ; python_version >= "3.12"`,
				want:     workspace.Requirement{Name: "Probe_A", Specifier: ">= 0.1, <1", Start: 15, End: 25},
			},
			{
				name:     "returns the specifier inside its parentheses",
				giveText: "tqdm ( >=4 )",
				want:     workspace.Requirement{Name: "tqdm", Specifier: ">=4", Start: 7, End: 10},
			},
			{
				name:     "returns the place of a specifier for a requirement with a marker alone",
				giveText: "probe-a[cli]; python_version > '3'",
				want:     workspace.Requirement{Name: "probe-a", Start: 12, End: 12},
			},
			{
				name:     "returns a requirement of a URL",
				giveText: "probe-a @ https://example.com/probe_a-1.0-py3-none-any.whl",
				want:     workspace.Requirement{Name: "probe-a", Start: 7, End: 7, URL: true},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := workspace.ParseRequirement(tt.giveText)
				assert.NoError(t, err, "ParseRequirement")
				assert.Equal(t, got, tt.want, "the requirement")
			})
		}

		invalid := []struct {
			name     string
			giveText string
		}{
			{name: "returns ErrRequirement for an empty text", giveText: ""},
			{name: "returns ErrRequirement for a text without a name", giveText: ">=1.0"},
			{name: "returns ErrRequirement for extras that do not close", giveText: "probe-a[cli>=1"},
			{name: "returns ErrRequirement for parentheses that do not close", giveText: "probe-a (>=1"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.ParseRequirement(tt.giveText)
				assert.ErrorIs(t, err, workspace.ErrRequirement, "ParseRequirement")
			})
		}
	})

	t.Run("Normalize", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveName string
			want     string
		}{
			{name: "returns a name in lowercase", giveName: "Probe-A", want: "probe-a"},
			{name: "returns each run of separators as one hyphen", giveName: "probe_.-a", want: "probe-a"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, workspace.Normalize(tt.giveName), tt.want, "the name")
			})
		}
	})
}
