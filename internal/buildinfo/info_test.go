// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package buildinfo_test

import (
	"runtime/debug"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/internal/buildinfo"
)

// The keys of the build settings that the go command records for a build of a git checkout.
const (
	vcsKey      = "vcs"
	revisionKey = "vcs.revision"
	timeKey     = "vcs.time"
	modifiedKey = "vcs.modified"
)

// The build settings of a build of a git checkout.
const (
	revision = "a5eb82a5bf341cc25678344462e16a994dbd1f31"
	stamp    = "2026-10-08T23:05:58Z"
)

// commit is the time of the commit of stamp.
var commit = time.Date(2026, time.October, 8, 23, 5, 58, 0, time.UTC)

func TestInfo(t *testing.T) {
	t.Parallel()

	t.Run("From", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			version  string
			settings []debug.BuildSetting
			want     buildinfo.Info
		}{
			{
				name:    "returns the release of a build at a tag with its commit",
				version: "v0.6.0",
				settings: []debug.BuildSetting{
					{Key: vcsKey, Value: "git"},
					{Key: revisionKey, Value: revision},
					{Key: timeKey, Value: stamp},
					{Key: modifiedKey, Value: "false"},
				},
				want: buildinfo.Info{Version: "0.6.0", Revision: revision, Time: commit},
			},
			{
				name:    "returns the release of go install of a version",
				version: "v1.2.3-rc.1",
				want:    buildinfo.Info{Version: "1.2.3-rc.1"},
			},
			{
				name:    "returns dev for a build that records (devel)",
				version: "(devel)",
				want:    buildinfo.Info{Version: buildinfo.Development},
			},
			{
				name:    "returns dev for a build at a commit after the last tag",
				version: "v0.5.1-0.20261008230558-a5eb82a5bf34",
				settings: []debug.BuildSetting{
					{Key: revisionKey, Value: revision},
					{Key: modifiedKey, Value: "false"},
				},
				want: buildinfo.Info{Version: buildinfo.Development, Revision: revision},
			},
			{
				name:     "returns dev for a build at a tag of a working tree with changes",
				version:  "v0.5.0+dirty",
				settings: []debug.BuildSetting{{Key: modifiedKey, Value: "true"}},
				want:     buildinfo.Info{Version: buildinfo.Development, Modified: true},
			},
			{
				name:    "returns dev for a version with +dirty without the setting vcs.modified",
				version: "v2.0.0+incompatible.dirty",
				want:    buildinfo.Info{Version: buildinfo.Development, Modified: true},
			},
			{
				name:     "returns dev for a release version of a working tree with changes",
				version:  "v0.6.0",
				settings: []debug.BuildSetting{{Key: modifiedKey, Value: "true"}},
				want:     buildinfo.Info{Version: buildinfo.Development, Modified: true},
			},
			{
				name:     "returns the zero time for a time that is not RFC 3339",
				version:  "v0.6.0",
				settings: []debug.BuildSetting{{Key: timeKey, Value: "yesterday"}},
				want:     buildinfo.Info{Version: "0.6.0"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				info := &debug.BuildInfo{
					Main:     debug.Module{Path: "go.dokimi.dev/ergon", Version: tt.version},
					Settings: tt.settings,
				}
				assert.Equal(t, buildinfo.From(info), tt.want, "the Info of the build information")
			})
		}

		t.Run("returns dev for a binary without build information", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, buildinfo.From(nil), buildinfo.Info{Version: buildinfo.Development}, "the Info")
		})
	})

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the Info of the build information of the running binary", func(t *testing.T) {
			t.Parallel()
			info, ok := debug.ReadBuildInfo()
			assert.True(t, ok, "the test binary has build information")
			assert.Equal(t, buildinfo.Read(), buildinfo.From(info), "the Info of the test binary")
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give buildinfo.Info
			want string
		}{
			{
				name: "returns the version with the short commit and the date",
				give: buildinfo.Info{Version: "0.6.0", Revision: revision, Time: commit},
				want: "0.6.0 (a5eb82a, 2026-10-08)",
			},
			{
				name: "returns dev with the note modified for a working tree with changes",
				give: buildinfo.Info{Version: buildinfo.Development, Revision: revision, Time: commit, Modified: true},
				want: "dev (a5eb82a, 2026-10-08, modified)",
			},
			{
				name: "returns the version alone for a build without a commit",
				give: buildinfo.Info{Version: "1.2.3"},
				want: "1.2.3",
			},
			{
				name: "returns a commit shorter than seven digits whole",
				give: buildinfo.Info{Version: "1.2.3", Revision: "abc"},
				want: "1.2.3 (abc)",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the version string")
			})
		}
	})
}
