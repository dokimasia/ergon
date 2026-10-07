// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/service/release"
)

// The modes of the release workflow, which the jobs of release.yml compare against.
const (
	pinnedVersion = "version"
	pinnedPublish = "publish"
	pinnedNone    = "none"
)

func TestMode(t *testing.T) {
	t.Parallel()

	t.Run("SelectMode", func(t *testing.T) {
		t.Parallel()

		withEntry := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{{{Name: "pkg-a"}}}}
		tests := []struct {
			plan *release.PublishPlan
			name string
			want string
			sets []changeset.Changeset
		}{
			{
				name: "returns version for a repository with changesets",
				sets: []changeset.Changeset{{ID: "strange-words-combine"}},
				plan: withEntry,
				want: pinnedVersion,
			},
			{
				name: "returns publish for a plan with an entry",
				plan: withEntry,
				want: pinnedPublish,
			},
			{
				name: "returns none for a plan without an entry",
				plan: &release.PublishPlan{Version: 1},
				want: pinnedNone,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, release.SelectMode(tt.sets, tt.plan), tt.want, "SelectMode")
			})
		}
	})
}
