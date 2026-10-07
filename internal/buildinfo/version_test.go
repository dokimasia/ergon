// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package buildinfo_test

import (
	"runtime/debug"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/internal/buildinfo"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("Format", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			version string
			commit  string
			date    string
			want    string
		}{
			{
				name:    "returns dev for an empty version",
				version: "", commit: "abc123", date: "2026-01-01",
				want: "dev",
			},
			{
				name:    "returns the version for an empty commit",
				version: "1.2.3", commit: "", date: "2026-01-01",
				want: "1.2.3",
			},
			{
				name:    "returns the version with the commit for an empty date",
				version: "1.2.3", commit: "abc123", date: "",
				want: "1.2.3 (abc123)",
			},
			{
				name:    "returns the version with the commit and the date",
				version: "1.2.3", commit: "abc123", date: "2026-01-01",
				want: "1.2.3 (abc123, built 2026-01-01)",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, buildinfo.Format(tt.version, tt.commit, tt.date), tt.want, "the version string")
			})
		}
	})

	t.Run("Release", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			linked string
			main   string
			want   string
		}{
			{
				name: "returns the version of the linker over the version of the module", linked: "1.2.3",
				main: "v1.0.0", want: "1.2.3",
			},
			{name: "returns the release of the module without its v", main: "v1.2.3", want: "1.2.3"},
			{name: "returns a pre-release of the module", main: "v1.2.3-rc.1", want: "1.2.3-rc.1"},
			{name: "returns nothing for a build of a working tree", main: "(devel)", want: ""},
			{
				name: "returns nothing for a build of go install at a commit",
				main: "v0.0.0-20261007120000-0123456789ab", want: "",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				info := &debug.BuildInfo{Main: debug.Module{Path: "go.dokimi.dev/ergon", Version: tt.main}}
				assert.Equal(t, buildinfo.Release(tt.linked, info), tt.want, "the release")
			})
		}

		t.Run("returns nothing for a binary without build information", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, buildinfo.Release("", nil), "the release")
		})
	})

	t.Run("Version", func(t *testing.T) {
		t.Parallel()

		t.Run("returns dev for a build without the flags", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, buildinfo.Version(), "dev", "the release of the test binary")
		})
	})

	t.Run("Full", func(t *testing.T) {
		t.Parallel()

		t.Run("returns dev for a build without the flags", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, buildinfo.Full(), "dev", "the version of the test binary")
		})
	})
}
