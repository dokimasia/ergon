// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/version"
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

		sets := []changeset.Changeset{{ID: "strange-words-combine"}}
		releasing := &release.Plan{
			Changesets: sets,
			Releases:   []release.Release{{Name: "pkg-a", Bump: version.BumpPatch}},
		}
		atNone := &release.Plan{Changesets: sets, Releases: []release.Release{{Name: "pkg-a", Bump: version.BumpNone}}}
		withEntry := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{{{Name: "pkg-a"}}}}
		tests := []struct {
			givePlan    *release.Plan
			givePublish *release.PublishPlan
			name        string
			want        string
			giveStale   []string
		}{
			{
				name:        "returns version for a plan that releases a package",
				givePlan:    releasing,
				givePublish: withEntry,
				want:        pinnedVersion,
			},
			{
				name:        "returns version for stale lockfiles",
				givePlan:    &release.Plan{},
				givePublish: withEntry,
				giveStale:   []string{"b/go.sum"},
				want:        pinnedVersion,
			},
			{
				name:        "returns publish for a plan of a changeset without releases",
				givePlan:    &release.Plan{Changesets: sets},
				givePublish: withEntry,
				want:        pinnedPublish,
			},
			{
				name:        "returns publish for a plan of releases at none",
				givePlan:    atNone,
				givePublish: withEntry,
				want:        pinnedPublish,
			},
			{
				name:        "returns publish for a publish plan with an entry",
				givePlan:    &release.Plan{},
				givePublish: withEntry,
				want:        pinnedPublish,
			},
			{
				name:        "returns none for a publish plan without an entry",
				givePlan:    atNone,
				givePublish: &release.PublishPlan{Version: 1},
				want:        pinnedNone,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, release.SelectMode(tt.givePlan, tt.givePublish, tt.giveStale), tt.want, "SelectMode")
			})
		}
	})
}
