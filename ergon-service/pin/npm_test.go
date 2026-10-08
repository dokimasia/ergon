// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/pin"
)

// The npm package of the cases: its name, its key, and the request of its document, in which the
// slash of the scope is escaped.
const (
	npmName     = "@biomejs/biome"
	npmKey      = "typescript.tools.biome"
	npmDocument = "GET /npm/@biomejs%2Fbiome"
)

func TestNPM(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want pin.Resolution
		}{
			{
				name: "returns the newest version that is not deprecated",
				give: `{"versions":{"1.0.0":{},"1.1.0":{},"1.2.0":{"deprecated":"broken"}},` +
					`"time":{"created":"2026-01-01T12:00:00Z","1.0.0":"2026-09-01T12:00:00Z",` +
					`"1.1.0":"2026-09-01T12:00:00Z","1.2.0":"2026-09-01T12:00:00Z"}}`,
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "1.1.0"}},
			},
			{
				name: "leaves out a pre-release",
				give: `{"versions":{"1.1.0-beta.1":{}},"time":{"1.1.0-beta.1":"2026-09-01T12:00:00Z"}}`,
			},
			{
				name: "dates a version by its time of publication",
				give: `{"versions":{"1.1.0":{}},"time":{"1.1.0":"2026-10-07T12:00:00Z"}}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{npmDocument: {body: tt.give}})
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindNPM, Key: npmKey, Name: npmName,
					Version: "1.0.0",
				})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		failures := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns ErrRegistry for a version without its time",
				give: `{"versions":{"1.1.0":{}},"time":{}}`,
				want: "the time of " + npmName + " 1.1.0",
			},
			{
				name: "returns ErrRegistry for a document that does not decode",
				give: `{"versions":[]}`,
				want: "decode",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{npmDocument: {body: tt.give}})
				_, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindNPM, Key: npmKey, Name: npmName,
					Version: "1.0.0",
				})
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}

		t.Run("returns ErrRegistry for a package that the registry does not have", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			_, err := r.Resolve(t.Context(), &pin.Pin{Kind: pin.KindNPM, Key: npmKey, Name: npmName, Version: "1.0.0"})
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.Contains(t, err.Error(), "the registry does not have it", "the error")
		})
	})
}
