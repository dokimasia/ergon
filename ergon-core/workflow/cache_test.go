// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workflow"
)

func TestCache(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give func(*workflow.Cache)
		}{
			{name: "returns nil for a cache with files and an interval", give: func(*workflow.Cache) {}},
			{
				name: "returns nil for a cache without files and with an input",
				give: func(c *workflow.Cache) { c.Files, c.Days, c.Input = nil, 0, "history" },
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := lintCache()
				tt.give(c)
				assert.NoError(t, c.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give func(*workflow.Cache)
		}{
			{name: "returns ErrInvalidJob for an empty id", give: func(c *workflow.Cache) { c.ID = "" }},
			{
				name: "returns ErrInvalidJob for an id with an uppercase letter",
				give: func(c *workflow.Cache) { c.ID = "golangci-Lint" },
			},
			{
				name: "returns ErrInvalidJob for an id that starts with a digit",
				give: func(c *workflow.Cache) { c.ID = "1lint" },
			},
			{name: "returns ErrInvalidJob for an empty name", give: func(c *workflow.Cache) { c.Name = "" }},
			{
				name: "returns ErrInvalidJob for a name that spans lines",
				give: func(c *workflow.Cache) { c.Name = "the cache\nof golangci-lint" },
			},
			{name: "returns ErrInvalidJob for an empty path", give: func(c *workflow.Cache) { c.Path = "" }},
			{
				name: "returns ErrInvalidJob for a path that spans lines",
				give: func(c *workflow.Cache) { c.Path = "~/.cache\n~/go" },
			},
			{
				name: "returns ErrInvalidJob for an empty pattern of files",
				give: func(c *workflow.Cache) { c.Files = []string{"go.work", ""} },
			},
			{
				name: "returns ErrInvalidJob for a pattern of files with a single quote",
				give: func(c *workflow.Cache) { c.Files = []string{"go'work"} },
			},
			{
				name: "returns ErrInvalidJob for a pattern of files that spans lines",
				give: func(c *workflow.Cache) { c.Files = []string{"go.work\ngo.mod"} },
			},
			{name: "returns ErrInvalidJob for days below 0", give: func(c *workflow.Cache) { c.Days = -1 }},
			{
				name: "returns ErrInvalidJob for an input with a space",
				give: func(c *workflow.Cache) { c.Input = "the history" },
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := lintCache()
				tt.give(c)
				assert.ErrorIs(t, c.Validate(), workflow.ErrInvalidJob, "Validate")
			})
		}
	})
}

// lintCache returns a new valid cache of the cases: the cache of golangci-lint under the user cache
// directory of Linux, with the files of its key and an interval of 7 days.
func lintCache() *workflow.Cache {
	return &workflow.Cache{
		ID:    "golangci-lint",
		Name:  "the cache of golangci-lint",
		Path:  "~/.cache/golangci-lint",
		Files: []string{"**/go.sum", "go.work"},
		Days:  7,
	}
}
