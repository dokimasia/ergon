// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/pin"
)

// The Composer package of the cases: its name, its key, and the request of its metadata.
const (
	composerName     = "vendor/tool"
	composerKey      = "php.tools.tool"
	composerMetadata = "GET /packagist/p2/vendor/tool.json"
)

func TestPackagist(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the newest stable version", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, map[string]route{composerMetadata: {body: `{"minified":"composer/2.0","packages":{` +
				`"vendor/tool":[{"version":"2.0.0-RC1","time":"2026-09-01T12:00:00+00:00"},` +
				`{"version":"1.2.0","time":"2026-09-01T12:00:00+00:00"},` +
				`{"version":"1.1.0","time":"2026-08-01T12:00:00+00:00"}]}}`}})
			got, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindComposer, Key: composerKey, Name: composerName,
				Version: "1.0.0",
			})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, got, pin.Resolution{Next: &pin.Release{Published: old, Version: "1.2.0"}}, "the resolution")
		})

		failures := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns ErrRegistry for metadata without the package",
				give: `{"minified":"composer/2.0","packages":{}}`,
				want: "has no versions of " + composerName,
			},
			{
				name: "returns ErrRegistry for metadata that does not decode",
				give: `{"packages":[]}`,
				want: "decode",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{composerMetadata: {body: tt.give}})
				_, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindComposer, Key: composerKey, Name: composerName,
					Version: "1.0.0",
				})
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}

		t.Run("returns ErrRegistry for a package that Packagist does not have", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			_, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindComposer, Key: composerKey, Name: composerName,
				Version: "1.0.0",
			})
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.Contains(t, err.Error(), "the registry does not have it", "the error")
		})
	})
}
