// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/lang/python/release"
)

// v returns the version that text states, for the cases.
func v(t *testing.T, text string) version.Version {
	t.Helper()
	parsed, err := version.Parse(text)
	assert.NoError(t, err, "Parse of "+text)
	return parsed
}

func TestSpecifier(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			giveReq     string
			giveVersion string
			want        language.Resolution
		}{
			{
				name:        "selects every version for no specifier",
				giveReq:     "",
				giveVersion: "1.0.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "selects a version above a lower bound",
				giveReq:     ">=0.1.0",
				giveVersion: "0.2.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "excludes a version below a lower bound",
				giveReq:     ">=0.3",
				giveVersion: "0.2.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "selects a version that equals with trailing zeros",
				giveReq:     "==0.2",
				giveVersion: "0.2.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "selects a version that starts with the numbers of a wildcard",
				giveReq:     "==0.2.*",
				giveVersion: "0.2.5",
				want:        language.ResolutionSelected,
			},
			{
				name:        "selects a version with fewer numbers than a wildcard of zeros",
				giveReq:     "==1.0.0.*",
				giveVersion: "1.0.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "excludes a version with other numbers than a wildcard",
				giveReq:     "==0.2.*",
				giveVersion: "0.3.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "excludes a version of another epoch than a wildcard",
				giveReq:     "==1!0.2.*",
				giveVersion: "0.2.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "excludes the version of a clause of !=",
				giveReq:     "!=0.2.0",
				giveVersion: "0.2.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "selects a version outside a wildcard of !=",
				giveReq:     "!=0.2.*",
				giveVersion: "0.3.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "selects a compatible release",
				giveReq:     "~=1.2",
				giveVersion: "1.9.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "excludes a release of the next major version under ~=",
				giveReq:     "~=1.2",
				giveVersion: "2.0.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "excludes a release of the next minor version under ~= of three numbers",
				giveReq:     "~=1.2.0",
				giveVersion: "1.3.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "selects a version below an upper bound",
				giveReq:     "<2",
				giveVersion: "1.9.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "excludes the version of an upper bound",
				giveReq:     "<2",
				giveVersion: "2.0.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "excludes a pre-release of the version of an upper bound",
				giveReq:     "<2",
				giveVersion: "2.0.0-rc.1",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "selects a pre-release below an upper bound of a pre-release",
				giveReq:     "<2.0rc2",
				giveVersion: "2.0.0-rc.1",
				want:        language.ResolutionSelected,
			},
			{
				name:        "selects the version of an inclusive upper bound",
				giveReq:     "<=1.2",
				giveVersion: "1.2.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "excludes the version of an exclusive lower bound",
				giveReq:     ">1.2",
				giveVersion: "1.2.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "selects a version above an exclusive lower bound",
				giveReq:     "> 1.2",
				giveVersion: "1.3.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "excludes a version below the post-release of an exclusive lower bound",
				giveReq:     ">1.0.post1",
				giveVersion: "1.0.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "selects a version above a development release",
				giveReq:     ">=1.0.dev1",
				giveVersion: "1.0.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "selects the text of a clause of ===",
				giveReq:     "===1.2.0",
				giveVersion: "1.2.0",
				want:        language.ResolutionSelected,
			},
			{
				name:        "excludes another text than a clause of ===",
				giveReq:     "===1.2",
				giveVersion: "1.2.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "excludes a version without the local label of ==",
				giveReq:     "==1.0+local",
				giveVersion: "1.0.0",
				want:        language.ResolutionExcluded,
			},
			{
				name:        "selects a version inside every clause",
				giveReq:     ">=1.0, <2",
				giveVersion: "1.5.0",
				want:        language.ResolutionSelected,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := release.Versioner{}.Resolve(tt.giveReq, v(t, tt.giveVersion))
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		invalid := []struct {
			name    string
			giveReq string
		}{
			{name: "returns ErrSpecifier for a clause without an operator", giveReq: "1.0"},
			{name: "returns ErrSpecifier for a version that does not parse", giveReq: ">=one"},
			{name: "returns ErrSpecifier for ~= with one number", giveReq: "~=1"},
			{name: "returns ErrSpecifier for a wildcard of >=", giveReq: ">=1.*"},
			{name: "returns ErrSpecifier for a wildcard after a pre-release", giveReq: "==1.0rc1.*"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := release.Versioner{}.Resolve(tt.giveReq, v(t, "1.0.0"))
				assert.ErrorIs(t, err, release.ErrSpecifier, "Resolve")
			})
		}
	})

	t.Run("Rewrite", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			giveReq     string
			giveVersion string
			want        string
		}{
			{name: "returns a lower bound for no specifier", giveReq: "", giveVersion: "0.2.0", want: ">=0.2.0"},
			{name: "moves a lower bound", giveReq: ">=0.1.0", giveVersion: "0.2.0", want: ">=0.2.0"},
			{
				name:        "keeps an upper bound that admits the version",
				giveReq:     ">= 0.1.0, <1",
				giveVersion: "0.2.0",
				want:        ">=0.2.0, <1",
			},
			{name: "moves an equality", giveReq: "==0.1.0", giveVersion: "0.2.0", want: "==0.2.0"},
			{name: "moves a wildcard", giveReq: "==0.1.*", giveVersion: "0.2.0", want: "==0.2.*"},
			{name: "moves a compatible release of two numbers", giveReq: "~=0.1", giveVersion: "0.2.0", want: "~=0.2"},
			{
				name:        "moves a compatible release of three numbers",
				giveReq:     "~=0.1.0",
				giveVersion: "0.2.0",
				want:        "~=0.2.0",
			},
			{
				name:        "moves a compatible release to a pre-release",
				giveReq:     "~=0.1",
				giveVersion: "0.2.0-rc.1",
				want:        "~=0.2.0rc1",
			},
			{
				name:        "moves an upper bound past the major version",
				giveReq:     ">=1.2,<2",
				giveVersion: "2.0.0",
				want:        ">=2.0.0,<3",
			},
			{name: "moves an upper bound past the minor version", giveReq: "<1.3", giveVersion: "1.3.0", want: "<1.4"},
			{name: "moves an inclusive upper bound", giveReq: "<=1.2.0", giveVersion: "1.3.0", want: "<=1.3.0"},
			{name: "keeps an exclusion of another version", giveReq: "!=1.2.0", giveVersion: "1.3.0", want: "!=1.2.0"},
			{name: "moves a clause of ===", giveReq: "===0.1.0", giveVersion: "0.2.0", want: "===0.2.0"},
			{
				name:        "moves a wildcard of more numbers than a release",
				giveReq:     "==1.0.0.0.*",
				giveVersion: "1.1.0",
				want:        "==1.1.0.0.*",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := release.Versioner{}.Rewrite(tt.giveReq, v(t, tt.giveVersion))
				assert.NoError(t, err, "Rewrite")
				assert.Equal(t, got, tt.want, "the specifier")
			})
		}

		invalid := []struct {
			name        string
			giveReq     string
			giveVersion string
		}{
			{
				name:        "returns ErrSpecifier for an exclusive lower bound that excludes the version",
				giveReq:     ">1.3.0",
				giveVersion: "1.3.0",
			},
			{name: "returns ErrSpecifier for an exclusion of the version", giveReq: "!=1.3.0", giveVersion: "1.3.0"},
			{name: "returns ErrSpecifier for a specifier that does not parse", giveReq: "one", giveVersion: "1.3.0"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := release.Versioner{}.Rewrite(tt.giveReq, v(t, tt.giveVersion))
				assert.ErrorIs(t, err, release.ErrSpecifier, "Rewrite")
			})
		}
	})
}
