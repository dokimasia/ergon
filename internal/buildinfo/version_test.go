// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package buildinfo_test

import (
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
