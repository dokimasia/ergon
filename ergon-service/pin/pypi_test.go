// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/pin"
)

// The files of a release of PyPI in the cases: uploaded more than a week before the run, uploaded
// less than a week before it, and yanked.
const (
	oldFile    = `{"upload_time_iso_8601":"2026-09-01T12:00:00Z","yanked":false}`
	youngFile  = `{"upload_time_iso_8601":"2026-10-07T12:00:00Z","yanked":false}`
	yankedFile = `{"upload_time_iso_8601":"2026-09-01T12:00:00Z","yanked":true}`
)

func TestPyPI(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want pin.Resolution
		}{
			{
				name: "returns the newest release with a file",
				give: `{"releases":{"1.0.0":[` + oldFile + `],"1.1.0":[` + oldFile + `],"1.0.5":[` + oldFile + `]}}`,
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "1.1.0"}},
			},
			{
				name: "leaves out a release whose every file was yanked",
				give: `{"releases":{"1.1.0":[` + yankedFile + `],"1.0.5":[` + oldFile + `]}}`,
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "1.0.5"}},
			},
			{
				name: "takes a release with a file that was not yanked",
				give: `{"releases":{"1.1.0":[` + yankedFile + `,` + oldFile + `]}}`,
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "1.1.0"}},
			},
			{
				name: "leaves out a release without files",
				give: `{"releases":{"1.1.0":[],"1.0.5":[` + oldFile + `]}}`,
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "1.0.5"}},
			},
			{
				name: "leaves out a pre-release",
				give: `{"releases":{"1.1.0rc1":[` + oldFile + `]}}`,
			},
			{
				name: "dates a release by its first file",
				give: `{"releases":{"1.1.0":[` + youngFile + `,` + oldFile + `]}}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{pypiDocument: {body: tt.give}})
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindPyPI, Key: pypiKey, Name: pypiName,
					Version: "1.0.0",
				})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		t.Run("returns ErrRegistry for a document that does not decode", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, map[string]route{pypiDocument: {body: `{"releases":[]}`}})
			_, err := r.Resolve(
				t.Context(),
				&pin.Pin{Kind: pin.KindPyPI, Key: pypiKey, Name: pypiName, Version: "1.0.0"},
			)
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.Contains(t, err.Error(), "decode", "the error")
		})
	})
}
