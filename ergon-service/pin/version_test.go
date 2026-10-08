// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/pin"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			version string
			give    []string
			want    pin.Resolution
		}{
			{
				name:    "takes a version of four numbers",
				version: "1.2.3",
				give:    []string{"1.2.3.1"},
				want:    pin.Resolution{Next: &pin.Release{Published: old, Version: "1.2.3.1"}},
			},
			{
				name:    "leaves out a version of five numbers",
				version: "1.2.3",
				give:    []string{"1.2.3.4.5"},
			},
			{
				name:    "leaves out a release candidate",
				version: "v6.0.0",
				give:    []string{"v6.1.0-rc.1"},
			},
			{
				name:    "compares versions by their numbers and not by their text",
				version: "v6.9.0",
				give:    []string{"v6.10.0", "v6.9.1"},
				want:    pin.Resolution{Next: &pin.Release{Published: old, Version: "v6.10.0"}},
			},
			{
				name:    "ranks a version equal to the same version with a 0 at its end",
				version: "v6.1",
				give:    []string{"v6.1.0"},
			},
			{
				name:    "ranks a version equal to the same version without the 0 at its end",
				version: "v6.1.0",
				give:    []string{"v6.1"},
			},
			{
				name:    "takes a version with a number after the numbers of the pin",
				version: "v6.1",
				give:    []string{"v6.1.1"},
				want:    pin.Resolution{Next: &pin.Release{Published: old, Version: "v6.1.1"}},
			},
			{
				name:    "takes a version with fewer numbers that ranks higher",
				version: "v6.1.0",
				give:    []string{"v6.2"},
				want:    pin.Resolution{Next: &pin.Release{Published: old, Version: "v6.2"}},
			},
			{
				name:    "leaves out a version with a number of 20 digits",
				version: "v6.0.0",
				give:    []string{"v6.18446744073709551616.0"},
			},
			{
				name:    "takes the release of the numbers of a pin at a pre-release",
				version: "v6.1.0-rc.1",
				give:    []string{"v6.1.0"},
				want:    pin.Resolution{Next: &pin.Release{Published: old, Version: "v6.1.0"}},
			},
			{
				name:    "returns the version without a v for a pin without one",
				version: "6.0.0",
				give:    []string{"v6.1.0"},
				want:    pin.Resolution{Next: &pin.Release{Published: old, Version: "6.1.0"}},
			},
			{
				name:    "returns the version with a v for a pin with one",
				version: "v6.0.0",
				give:    []string{"6.1.0"},
				want:    pin.Resolution{Next: &pin.Release{Published: old, Version: "v6.1.0"}},
			},
			{
				name:    "takes a release of the major version 0 for a pin without numbers",
				version: "main",
				give:    []string{"0.1.0"},
				want:    pin.Resolution{Next: &pin.Release{Published: old, Version: "0.1.0"}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				releases := make([]forge.Release, 0, len(tt.give))
				for _, tag := range tt.give {
					releases = append(releases, forge.Release{Tag: tag, Published: old})
				}
				r, _ := serve(t, nil)
				r.GitHub = &gitHub{releases: map[string][]forge.Release{hooksRepo: releases}}
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindGitHub, Key: hooksKey, Name: hooksRepo,
					Version: tt.version,
				})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}
	})
}
