// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/lang/rust/release"
)

// requirementVector is a requirement of testdata/requirements.json: whether the semver crate 1.0.28
// parses it, the versions of the grid that it matches, and the requirement that cargo-release
// writes for each version of the grid where that requirement matches the version.
type requirementVector struct {
	Rewrites    map[string]string `json:"rewrites"`
	Requirement string            `json:"requirement"`
	Matches     []string          `json:"matches"`
	Valid       bool              `json:"valid"`
}

// requirementVectors is the content of testdata/requirements.json.
type requirementVectors struct {
	Versions     []string            `json:"versions"`
	Requirements []requirementVector `json:"requirements"`
}

// vectors returns the content of testdata/requirements.json, and stops the test when the file does
// not decode or has no requirement.
func vectors(t *testing.T) requirementVectors {
	t.Helper()
	var got requirementVectors
	data := files.Read(t, filepath.Join("testdata", "requirements.json"))
	assert.NoError(t, json.Unmarshal([]byte(data), &got), "the decoding of the expectations")
	assert.NotEmpty(t, got.Requirements, "the requirements of the expectations")
	return got
}

func TestRequirement(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()
		grid := vectors(t)
		for _, tt := range grid.Requirements {
			if !tt.Valid {
				t.Run(fmt.Sprintf("returns ErrRequirement for %q", tt.Requirement), func(t *testing.T) {
					t.Parallel()
					_, err := release.Versioner{}.Resolve(tt.Requirement, version.Version{})
					assert.ErrorIs(t, err, release.ErrRequirement, "Resolve")
				})
				continue
			}
			for _, give := range grid.Versions {
				want, word := language.ResolutionExcluded, "Excluded"
				if slices.Contains(tt.Matches, give) {
					want, word = language.ResolutionSelected, "Selected"
				}
				t.Run(fmt.Sprintf("returns %s for %s under %q", word, give, tt.Requirement), func(t *testing.T) {
					t.Parallel()
					v, err := version.Parse(give)
					assert.NoError(t, err, "Parse")
					got, err := release.Versioner{}.Resolve(tt.Requirement, v)
					assert.NoError(t, err, "Resolve")
					assert.Equal(t, got, want, "the resolution")
				})
			}
		}

		t.Run("returns Selected for each version under a dependency without a version", func(t *testing.T) {
			t.Parallel()
			got, err := release.Versioner{}.Resolve("", version.Version{Major: 2, Pre: "rc.1"})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, got, language.ResolutionSelected, "the resolution")
		})
	})

	t.Run("Rewrite", func(t *testing.T) {
		t.Parallel()
		grid := vectors(t)
		for _, tt := range grid.Requirements {
			for give, want := range tt.Rewrites {
				t.Run(fmt.Sprintf("returns %q for %q at %s", want, tt.Requirement, give), func(t *testing.T) {
					t.Parallel()
					v, err := version.Parse(give)
					assert.NoError(t, err, "Parse")
					got, err := release.Versioner{}.Rewrite(tt.Requirement, v)
					assert.NoError(t, err, "Rewrite")
					assert.Equal(t, got, want, "the requirement")
				})
			}
		}

		for _, tt := range grid.Requirements {
			if !tt.Valid {
				continue
			}
			t.Run(
				fmt.Sprintf("returns a requirement that admits each version for %q", tt.Requirement),
				func(t *testing.T) {
					t.Parallel()
					for _, give := range grid.Versions {
						v, err := version.Parse(give)
						assert.NoError(t, err, "Parse")
						got, err := release.Versioner{}.Rewrite(tt.Requirement, v)
						if err != nil {
							expect.ErrorIs(t, err, release.ErrRequirement, "Rewrite at "+give)
							continue
						}
						resolution, err := release.Versioner{}.Resolve(got, v)
						expect.NoError(t, err, "Resolve of "+got)
						expect.Equal(
							t,
							resolution,
							language.ResolutionSelected,
							"the resolution of "+give+" under "+got,
						)
					}
				},
			)
		}

		tests := []struct {
			name        string
			giveReq     string
			giveVersion version.Version
			want        string
		}{
			{
				name:    "moves a lower bound to the numbers of the version",
				giveReq: ">=1.2", giveVersion: version.Version{Major: 1, Minor: 3}, want: ">=1.3",
			},
			{
				name:    "moves an upper bound that excludes the version past it",
				giveReq: ">=1.2, <2", giveVersion: version.Version{Major: 2, Minor: 1}, want: ">=2.1, <3",
			},
			{
				name:    "keeps an upper bound that admits the version",
				giveReq: ">=1.2, <2", giveVersion: version.Version{Major: 1, Minor: 5}, want: ">=1.5, <2",
			},
			{
				name:    "keeps the spaces of the requirement",
				giveReq: ">= 1.2.0 , < 1.3", giveVersion: version.Version{Major: 1, Minor: 3}, want: ">= 1.3.0 , < 1.4",
			},
			{
				name:    "moves an inclusive upper bound to the numbers of the version",
				giveReq: "<=1.2", giveVersion: version.Version{Major: 1, Minor: 3, Patch: 5}, want: "<=1.3",
			},
			{
				name:        "keeps an inclusive upper bound that admits the version",
				giveReq:     ">=1.0.0, <=1.2.3",
				giveVersion: version.Version{Major: 1, Minor: 2, Patch: 3},
				want:        ">=1.2.3, <=1.2.3",
			},
			{
				name:    "moves an upper bound with a wildcard past the version",
				giveReq: ">=1.0, <1.*", giveVersion: version.Version{Major: 1, Minor: 5}, want: ">=1.5, <2.*",
			},
			{
				name:    "keeps a lower exclusive bound that admits the version",
				giveReq: ">1.0", giveVersion: version.Version{Major: 1, Minor: 5}, want: ">1.0",
			},
			{
				name:        "writes the whole pre-release of the version for a requirement of fewer numbers",
				giveReq:     "^1.2",
				giveVersion: version.Version{Major: 1, Minor: 3, Pre: "alpha.1"},
				want:        "^1.3.0-alpha.1",
			},
			{
				name:    "writes the whole pre-release of the version for an inclusive upper bound",
				giveReq: ">=1.2.0, <=1.2.3", giveVersion: version.Version{Major: 1, Minor: 2, Patch: 4, Pre: "rc.1"},
				want: ">=1.2.4-rc.1, <=1.2.4-rc.1",
			},
			{
				name:    "drops the build metadata of the requirement",
				giveReq: "^1.2.3+build", giveVersion: version.Version{Major: 1, Minor: 3}, want: "^1.3.0",
			},
			{
				name:    "keeps the wildcard of a tilde requirement",
				giveReq: "~1.2.*", giveVersion: version.Version{Major: 1, Minor: 3}, want: "~1.3.*",
			},
			{
				name:    "keeps the spaces around the requirement",
				giveReq: " ^1.2 ", giveVersion: version.Version{Major: 1, Minor: 3}, want: " ^1.3 ",
			},
			{
				name:    "keeps a dependency without a version",
				giveReq: "", giveVersion: version.Version{Major: 2, Pre: "rc.1"}, want: "",
			},
			{
				name:        "keeps the wildcard of every version",
				giveReq:     "*",
				giveVersion: version.Version{Major: 2},
				want:        "*",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := release.Versioner{}.Rewrite(tt.giveReq, tt.giveVersion)
				assert.NoError(t, err, "Rewrite")
				assert.Equal(t, got, tt.want, "the requirement")
			})
		}

		invalid := []struct {
			name        string
			giveReq     string
			giveVersion version.Version
		}{
			{name: "returns ErrRequirement for a requirement that does not parse", giveReq: "~>1.2"},
			{
				name:    "returns ErrRequirement for a lower exclusive bound that excludes the version",
				giveReq: ">1.5", giveVersion: version.Version{Major: 1, Minor: 5},
			},
			{
				name:    "returns ErrRequirement for a wildcard that cannot admit a pre-release",
				giveReq: "1.*", giveVersion: version.Version{Major: 2, Pre: "alpha"},
			},
			{
				name:    "returns ErrRequirement for the wildcard of every version at a pre-release",
				giveReq: "*", giveVersion: version.Version{Major: 2, Pre: "alpha"},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := release.Versioner{}.Rewrite(tt.giveReq, tt.giveVersion)
				assert.ErrorIs(t, err, release.ErrRequirement, "Rewrite")
			})
		}
	})
}
