// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/lang/python/workspace"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("ParseVersion", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveText string
			want     string
		}{
			{name: "returns a release", giveText: "1.2.0", want: "1.2.0"},
			{name: "returns a release of one number", giveText: "1", want: "1"},
			{name: "returns the normal form of a pre-release", giveText: "1.0.0-Alpha.1", want: "1.0.0a1"},
			{name: "returns a pre-release without a number as number 0", giveText: "1.0b", want: "1.0b0"},
			{name: "returns a preview as a release candidate", giveText: "1.0preview2", want: "1.0rc2"},
			{name: "returns the normal form of a post-release", giveText: "1.0-1", want: "1.0.post1"},
			{name: "returns the normal form of a revision", giveText: "1.0rev3", want: "1.0.post3"},
			{name: "returns a development release", giveText: "1.0.dev4", want: "1.0.dev4"},
			{name: "returns an epoch", giveText: "2!1.0", want: "2!1.0"},
			{name: "returns a local label in lowercase", giveText: "1.0+Ubuntu-1", want: "1.0+ubuntu.1"},
			{name: "returns a version without its v and its spaces", giveText: " v1.0 ", want: "1.0"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := workspace.ParseVersion(tt.giveText)
				assert.NoError(t, err, "ParseVersion")
				assert.Equal(t, got.String(), tt.want, "the version")
			})
		}

		invalid := []struct {
			name     string
			giveText string
		}{
			{name: "returns ErrVersion for a text that is no version", giveText: "one"},
			{name: "returns ErrVersion for an empty text", giveText: ""},
			{name: "returns ErrVersion for a number above the range of a version", giveText: "1.99999999999999999999"},
			{
				name:     "returns ErrVersion for a number of a pre-release above the range of a version",
				giveText: "1.0rc99999999999999999999",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.ParseVersion(tt.giveText)
				assert.ErrorIs(t, err, workspace.ErrVersion, "ParseVersion")
			})
		}
	})

	t.Run("Compare", func(t *testing.T) {
		t.Parallel()

		// order are versions of PEP 440 in ascending order, as the examples of the specification
		// order them.
		order := []string{
			"1.0.dev456", "1.0a1", "1.0a2.dev456", "1.0a12.dev456", "1.0a12", "1.0b1.dev456", "1.0b2",
			"1.0b2.post345.dev456", "1.0b2.post345", "1.0rc1.dev456", "1.0rc1", "1.0", "1.0+abc.5", "1.0+abc.7",
			"1.0+5", "1.0.post456.dev34", "1.0.post456", "1.1.dev1", "1!0.1",
		}
		versions := make([]workspace.Version, 0, len(order))
		for _, text := range order {
			v, err := workspace.ParseVersion(text)
			assert.NoError(t, err, "ParseVersion of "+text)
			versions = append(versions, v)
		}

		t.Run("orders the versions of the specification", func(t *testing.T) {
			t.Parallel()
			assert.Pairwise(t, versions, func(a, b workspace.Version) bool { return a.Compare(&b) < 0 },
				"each version precedes the next")
		})

		t.Run("orders each version of the specification after the one before it", func(t *testing.T) {
			t.Parallel()
			assert.Pairwise(t, versions, func(a, b workspace.Version) bool { return b.Compare(&a) > 0 },
				"each version follows the one before it")
		})

		t.Run("returns 0 for releases that differ in their trailing zeros", func(t *testing.T) {
			t.Parallel()
			a, err := workspace.ParseVersion("1.2")
			assert.NoError(t, err, "ParseVersion of 1.2")
			b, err := workspace.ParseVersion("1.2.0.0")
			assert.NoError(t, err, "ParseVersion of 1.2.0.0")
			assert.Equal(t, a.Compare(&b), 0, "the order")
		})

		t.Run("orders a local label of a number after one of letters", func(t *testing.T) {
			t.Parallel()
			a, err := workspace.ParseVersion("1.0+abc.5")
			assert.NoError(t, err, "ParseVersion of 1.0+abc.5")
			b, err := workspace.ParseVersion("1.0+abc.5.1")
			assert.NoError(t, err, "ParseVersion of 1.0+abc.5.1")
			assert.Equal(t, a.Compare(&b), -1, "the order of a label and the longer label")
			assert.Equal(t, b.Compare(&a), 1, "the order of a longer label and the label")
		})
	})

	t.Run("IsPre", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveText string
			want     bool
		}{
			{name: "reports true for a pre-release", giveText: "1.0rc1", want: true},
			{name: "reports true for a development release", giveText: "1.0.dev1", want: true},
			{name: "reports false for a post-release", giveText: "1.0.post1", want: false},
			{name: "reports false for a release", giveText: "1.0", want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				v, err := workspace.ParseVersion(tt.giveText)
				assert.NoError(t, err, "ParseVersion")
				assert.Equal(t, v.IsPre(), tt.want, "the pre-release")
			})
		}
	})

	t.Run("Base", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the epoch and the release alone", func(t *testing.T) {
			t.Parallel()
			v, err := workspace.ParseVersion("1!1.2rc1.post2.dev3+local")
			assert.NoError(t, err, "ParseVersion")
			assert.Equal(t, v.Base().String(), "1!1.2", "the base")
		})
	})

	t.Run("SemVer", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveText string
			want     version.Version
		}{
			{
				name:     "returns a release of three numbers",
				giveText: "1.2.3",
				want:     version.Version{Major: 1, Minor: 2, Patch: 3},
			},
			{
				name:     "returns a release of two numbers with patch 0",
				giveText: "1.2",
				want:     version.Version{Major: 1, Minor: 2},
			},
			{
				name:     "returns a pre-release as its label and number",
				giveText: "2.0.0rc1",
				want:     version.Version{Major: 2, Pre: "rc.1"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				v, err := workspace.ParseVersion(tt.giveText)
				assert.NoError(t, err, "ParseVersion")
				got, err := v.SemVer()
				assert.NoError(t, err, "SemVer")
				assert.Equal(t, got, tt.want, "the version")
			})
		}

		invalid := []struct {
			name     string
			giveText string
		}{
			{name: "returns ErrVersion for an epoch", giveText: "1!1.0"},
			{name: "returns ErrVersion for a post-release", giveText: "1.0.post1"},
			{name: "returns ErrVersion for a development release", giveText: "1.0.dev1"},
			{name: "returns ErrVersion for a local label", giveText: "1.0+local"},
			{name: "returns ErrVersion for a release of four numbers", giveText: "1.2.3.4"},
			{
				name:     "returns ErrVersion for a number above the range of Semantic Versioning",
				giveText: "1.0.9007199254740992",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				v, err := workspace.ParseVersion(tt.giveText)
				assert.NoError(t, err, "ParseVersion")
				_, err = v.SemVer()
				assert.ErrorIs(t, err, workspace.ErrVersion, "SemVer")
			})
		}
	})

	t.Run("FromSemVer", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give version.Version
			want string
		}{
			{name: "returns a release", give: version.Version{Major: 1, Minor: 2}, want: "1.2.0"},
			{name: "returns an alpha", give: version.Version{Major: 1, Pre: "a.1"}, want: "1.0.0a1"},
			{name: "returns a beta", give: version.Version{Major: 1, Pre: "b.2"}, want: "1.0.0b2"},
			{name: "returns a release candidate", give: version.Version{Major: 1, Pre: "rc.3"}, want: "1.0.0rc3"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := workspace.FromSemVer(tt.give)
				assert.NoError(t, err, "FromSemVer")
				assert.Equal(t, got.String(), tt.want, "the version")
			})
		}

		invalid := []struct {
			name string
			give version.Version
		}{
			{name: "returns ErrVersion for build metadata", give: version.Version{Major: 1, Build: "20261016"}},
			{
				name: "returns ErrVersion for a pre-release that PEP 440 does not have",
				give: version.Version{Major: 1, Pre: "next.0"},
			},
			{name: "returns ErrVersion for a pre-release without a number", give: version.Version{Major: 1, Pre: "rc"}},
			{
				name: "returns ErrVersion for a number of a pre-release above the range of a version",
				give: version.Version{Major: 1, Pre: "rc.99999999999999999999"},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.FromSemVer(tt.give)
				assert.ErrorIs(t, err, workspace.ErrVersion, "FromSemVer")
			})
		}
	})
}
