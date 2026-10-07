// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workflow"
)

func TestStep(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give workflow.Step
		}{
			{
				name: "returns nil for an action with inputs",
				give: workflow.Step{Uses: checkout, With: map[string]string{"persist-credentials": "false"}},
			},
			{
				name: "returns nil for an input that spans lines",
				give: workflow.Step{Uses: checkout, With: map[string]string{"globs": "**/*.md\n#node_modules"}},
			},
			{
				name: "returns nil for a command whose later line is indented",
				give: workflow.Step{Run: []string{"for f in *; do", "  echo \"$f\"", "done"}},
			},
			{
				name: "returns nil for a command with a name, an id, a condition and variables",
				give: workflow.Step{
					Name: "Read the pin",
					ID:   "pin",
					If:   "hashFiles('.terraform-version') != ''",
					Env:  map[string]string{"BASE_REF": "${{ github.base_ref }}"},
					Run:  []string{`echo "version=$(cat .terraform-version)" >> "$GITHUB_OUTPUT"`},
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
			give workflow.Step
		}{
			{
				name: "returns ErrInvalidStep for a name that spans lines",
				give: workflow.Step{Name: "a\nb", Run: []string{"true"}},
			},
			{
				name: "returns ErrInvalidStep for a condition that spans lines",
				give: workflow.Step{If: "a\r\nb", Run: []string{"true"}},
			},
			{
				name: "returns ErrInvalidStep for an id that starts with a digit",
				give: workflow.Step{ID: "1pin", Run: []string{"true"}},
			},
			{name: "returns ErrInvalidStep for neither an action nor a command", give: workflow.Step{Name: "nothing"}},
			{
				name: "returns ErrInvalidStep for both an action and a command",
				give: workflow.Step{Uses: checkout, Run: []string{"true"}},
			},
			{
				name: "returns ErrInvalidStep for an action that is not valid",
				give: workflow.Step{Uses: workflow.Action{Uses: "a/b"}},
			},
			{
				name: "returns ErrInvalidStep for inputs of a command",
				give: workflow.Step{Run: []string{"true"}, With: map[string]string{"a": "b"}},
			},
			{
				name: "returns ErrInvalidStep for an input whose name has a space",
				give: workflow.Step{Uses: checkout, With: map[string]string{"fetch depth": "0"}},
			},
			{
				name: "returns ErrInvalidStep for a variable whose name starts with a digit",
				give: workflow.Step{Run: []string{"true"}, Env: map[string]string{"1A": "b"}},
			},
			{name: "returns ErrInvalidStep for an empty line", give: workflow.Step{Run: []string{"true", ""}}},
			{
				name: "returns ErrInvalidStep for a line that spans lines",
				give: workflow.Step{Run: []string{"true\nfalse"}},
			},
			{
				name: "returns ErrInvalidStep for a line that ends in a space",
				give: workflow.Step{Run: []string{"true "}},
			},
			{
				name: "returns ErrInvalidStep for a line that ends in a tab",
				give: workflow.Step{Run: []string{"true\t"}},
			},
			{
				name: "returns ErrInvalidStep for a first line that starts with a space",
				give: workflow.Step{Run: []string{" true"}},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), workflow.ErrInvalidStep, "Validate")
			})
		}

		t.Run("returns ErrInvalidAction for an action that is not valid", func(t *testing.T) {
			t.Parallel()
			step := workflow.Step{Uses: workflow.Action{Uses: "a/b"}}
			assert.ErrorIs(t, step.Validate(), workflow.ErrInvalidAction, "Validate")
		})
	})
}
