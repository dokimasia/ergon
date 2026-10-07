// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workflow"
)

// checkout is a valid action of the cases: actions/checkout at the commit of its release v7.0.1.
var checkout = workflow.Action{
	Uses:    "actions/checkout",
	Commit:  "3d3c42e5aac5ba805825da76410c181273ba90b1",
	Release: "v7.0.1",
}

func TestAction(t *testing.T) {
	t.Parallel()

	t.Run("Path", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the action in a directory of its repository at the same commit", func(t *testing.T) {
			t.Parallel()
			codeql := workflow.Action{Uses: "github/codeql-action", Commit: checkout.Commit, Release: "v4.38.2"}
			assert.Equal(t, codeql.Path("init"), workflow.Action{
				Uses:    "github/codeql-action/init",
				Commit:  checkout.Commit,
				Release: "v4.38.2",
			}, "the action init of codeql-action")
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns uses at the commit with the release in a comment", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, checkout.String(), "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1",
				"the reference of the checkout")
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give workflow.Action
		}{
			{name: "returns nil for an action of owner and name", give: checkout},
			{
				name: "returns nil for an action in a directory of its repository",
				give: workflow.Action{
					Uses:    "github/codeql-action/upload-sarif",
					Commit:  checkout.Commit,
					Release: "v4.38.2",
				},
			},
			{
				name: "returns nil for a release without a v",
				give: workflow.Action{Uses: "shivammathur/setup-php", Commit: checkout.Commit, Release: "2.37.2"},
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
			give workflow.Action
		}{
			{
				name: "returns ErrInvalidAction for an empty uses",
				give: workflow.Action{Commit: checkout.Commit, Release: "v1"},
			},
			{
				name: "returns ErrInvalidAction for a uses without a name",
				give: workflow.Action{Uses: "actions", Commit: checkout.Commit, Release: "v1"},
			},
			{
				name: "returns ErrInvalidAction for an owner that starts with a hyphen",
				give: workflow.Action{Uses: "-a/b", Commit: checkout.Commit, Release: "v1"},
			},
			{
				name: "returns ErrInvalidAction for an empty directory",
				give: workflow.Action{Uses: "a/b/", Commit: checkout.Commit, Release: "v1"},
			},
			{
				name: "returns ErrInvalidAction for a local action",
				give: workflow.Action{Uses: "./.github/actions/setup-ergon", Commit: checkout.Commit, Release: "v1"},
			},
			{
				name: "returns ErrInvalidAction for a commit of 39 digits",
				give: workflow.Action{Uses: "a/b", Commit: checkout.Commit[1:], Release: "v1"},
			},
			{
				name: "returns ErrInvalidAction for a commit in uppercase",
				give: workflow.Action{Uses: "a/b", Commit: "3D3C42E5AAC5BA805825DA76410C181273BA90B1", Release: "v1"},
			},
			{
				name: "returns ErrInvalidAction for a tag in place of a commit",
				give: workflow.Action{Uses: "a/b", Commit: "v7", Release: "v7"},
			},
			{
				name: "returns ErrInvalidAction for an empty release",
				give: workflow.Action{Uses: "a/b", Commit: checkout.Commit},
			},
			{
				name: "returns ErrInvalidAction for a release with a space",
				give: workflow.Action{Uses: "a/b", Commit: checkout.Commit, Release: "v7 beta"},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.ErrorIs(t, tt.give.Validate(), workflow.ErrInvalidAction, "Validate")
			})
		}
	})
}
