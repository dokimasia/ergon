// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/pin"
)

// The crate of the cases: its key, and the request of its file in the index.
const (
	crateKey  = "rust.tools.tool"
	crateFile = "GET /crates/to/ol/tool"
)

// crateLine is a line of the index of a version of 0.2.0 published more than a week before the run.
const crateLine = `{"name":"tool","vers":"0.2.0","yanked":false,"pubtime":"2026-09-01T12:00:00Z"}`

func TestCrates(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want pin.Resolution
		}{
			{
				name: "returns the newest version that is not yanked",
				give: `{"vers":"0.1.0","yanked":false,"pubtime":"2026-08-01T12:00:00Z"}` + "\n" + crateLine + "\n" +
					`{"vers":"0.3.0","yanked":true,"pubtime":"2026-09-01T12:00:00Z"}` + "\n",
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "0.2.0"}},
			},
			{
				name: "leaves out a pre-release",
				give: `{"vers":"0.2.0-alpha.1","yanked":false,"pubtime":"2026-09-01T12:00:00Z"}` + "\n",
			},
			{
				name: "takes a version without a time of publication",
				give: `{"vers":"0.2.0","yanked":false}` + "\n",
				want: pin.Resolution{Next: &pin.Release{Published: time.Time{}, Version: "0.2.0"}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{crateFile: {body: tt.give}})
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindCrate, Key: crateKey, Name: "tool",
					Version: "0.1.0",
				})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		paths := []struct {
			name string
			give string
			file string
		}{
			{name: "reads the file of a crate of one character under 1", give: "a", file: "GET /crates/1/a"},
			{name: "reads the file of a crate of two characters under 2", give: "ab", file: "GET /crates/2/ab"},
			{
				name: "reads the file of a crate of three characters under 3 and its first character",
				give: "abc",
				file: "GET /crates/3/a/abc",
			},
			{
				name: "reads the file of a longer crate under its first two and its next two characters",
				give: "serde",
				file: "GET /crates/se/rd/serde",
			},
			{
				name: "reads the file of a crate under its name in lower case",
				give: "Serde",
				file: "GET /crates/se/rd/serde",
			},
		}
		for _, tt := range paths {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{tt.file: {body: crateLine + "\n"}})
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindCrate, Key: crateKey, Name: tt.give,
					Version: "0.1.0",
				})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, pin.Resolution{Next: &pin.Release{Published: old, Version: "0.2.0"}},
					"the resolution")
			})
		}

		t.Run("returns ErrRegistry for a line that does not decode", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, map[string]route{crateFile: {body: crateLine + "\n{\n"}})
			_, err := r.Resolve(
				t.Context(),
				&pin.Pin{Kind: pin.KindCrate, Key: crateKey, Name: "tool", Version: "0.1.0"},
			)
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.Contains(t, err.Error(), "decode", "the error")
		})

		t.Run("returns ErrRegistry for a crate that the index does not have", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			_, err := r.Resolve(
				t.Context(),
				&pin.Pin{Kind: pin.KindCrate, Key: crateKey, Name: "tool", Version: "0.1.0"},
			)
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.Contains(t, err.Error(), "the registry does not have it", "the error")
		})
	})
}
