// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workflow"
)

func TestCodeQL(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give func(*workflow.CodeQL)
		}{
			{name: "returns nil for an analysis of Go", give: func(*workflow.CodeQL) {}},
			{
				name: "returns nil for a language of two names",
				give: func(c *workflow.CodeQL) { c.Language, c.Name = "javascript-typescript", "JavaScript and TypeScript" },
			},
			{name: "returns nil for the build mode none", give: func(c *workflow.CodeQL) { c.BuildMode = "none" }},
			{
				name: "returns nil for an analysis with a step before it",
				give: func(c *workflow.CodeQL) {
					c.Steps = []workflow.Step{{Name: "Name the modules", Run: []string{"go list -m"}}}
				},
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := goAnalysis()
				tt.give(&c)
				assert.NoError(t, c.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give func(*workflow.CodeQL)
		}{
			{
				name: "returns ErrInvalidCodeQL for a language with a sign",
				give: func(c *workflow.CodeQL) { c.Language = "c#" },
			},
			{name: "returns ErrInvalidCodeQL for an empty name", give: func(c *workflow.CodeQL) { c.Name = "" }},
			{
				name: "returns ErrInvalidCodeQL for a name that spans lines",
				give: func(c *workflow.CodeQL) { c.Name = "Go\n" },
			},
			{
				name: "returns ErrInvalidCodeQL for an unknown build mode",
				give: func(c *workflow.CodeQL) { c.BuildMode = "auto" },
			},
			{
				name: "returns ErrInvalidCodeQL for the build mode manual",
				give: func(c *workflow.CodeQL) { c.BuildMode = "manual" },
			},
			{name: "returns ErrInvalidCodeQL for empty files", give: func(c *workflow.CodeQL) { c.Files = "" }},
			{
				name: "returns ErrInvalidCodeQL for files with a single quote",
				give: func(c *workflow.CodeQL) { c.Files = "go'work" },
			},
			{name: "returns ErrInvalidCodeQL for a timeout of 0", give: func(c *workflow.CodeQL) { c.Timeout = 0 }},
			{
				name: "returns ErrInvalidCodeQL for a step without an action and without a command",
				give: func(c *workflow.CodeQL) { c.Steps = []workflow.Step{{Name: "nothing"}} },
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := goAnalysis()
				tt.give(&c)
				assert.ErrorIs(t, c.Validate(), workflow.ErrInvalidCodeQL, "Validate")
			})
		}
	})
}

// goAnalysis returns a valid analysis of the cases: Go, built automatically once go.work exists.
func goAnalysis() workflow.CodeQL {
	return workflow.CodeQL{Language: "go", Name: "Go", BuildMode: "autobuild", Files: "go.work", Timeout: 30}
}
