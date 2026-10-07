// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/release"
)

// The client of package forge reads the links of the changelog of GitHub for ergon release.
var _ release.Forge = (*forge.Client)(nil)

func TestForge(t *testing.T) {
	t.Parallel()

	t.Run("Host", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			server string
			want   string
		}{
			{
				name: "links the references of a changelog to github.com for an empty server",
				want: "https://github.com/emotion-js/emotion/issues/42",
			},
			{
				name:   "links the references of a changelog to the server that it names",
				server: "https://git.example.com/",
				want:   "https://git.example.com/emotion-js/emotion/issues/42",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := blankState(t)
				s.add("pkg", "1.0.0")
				s.sets = []changeset.Changeset{{
					ID: "some-id", Summary: "something\nfixes #42",
					Releases: []changeset.Release{{Name: "pkg", Bump: minor}},
				}}
				cfg := defaultConfig()
				cfg.Changelog = release.Changelog{Format: release.ChangelogGitHub, Repo: emotion}
				got := entries(t, s, &cfg, nil, release.Host{Forge: gitHub{}, Server: tt.server})
				want := []release.Entry{{
					Name: "pkg", Text: "## 1.1.0\n\n### Minor Changes\n\n- something\n  fixes [#42](" + tt.want + ")",
				}}
				assert.Equal(t, got, want, "the entries")
			})
		}
	})
}
