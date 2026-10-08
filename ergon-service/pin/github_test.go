// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/pin"
)

func TestGitHub(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the releases whose tags are stable versions", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{releases: map[string][]forge.Release{hooksRepo: {
				{Tag: "nightly", Published: old}, {Tag: "v6.2.0-rc.1", Published: old}, {Tag: "v6.1.0", Published: old},
			}}}
			got, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindGitHub, Key: hooksKey, Name: hooksRepo,
				Version: "v6.0.0",
			})
			assert.NoError(t, err, "Resolve")
			assert.Equal(
				t,
				got,
				pin.Resolution{Next: &pin.Release{Published: old, Version: "v6.1.0"}},
				"the resolution",
			)
		})

		t.Run("returns ErrRegistry for releases that GitHub does not return", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{err: errGitHub}
			_, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindGitHub, Key: hooksKey, Name: hooksRepo,
				Version: "v6.0.0",
			})
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.ErrorIs(t, err, errGitHub, "Resolve")
		})
	})
}
