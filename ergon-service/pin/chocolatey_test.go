// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"net/http"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/pin"
)

// The package of Chocolatey of the cases: its key, and the query of its latest version, whose
// filter is percent-encoded.
const (
	chocolateyKey   = "github.make"
	chocolateyQuery = "GET /chocolatey/Packages()?$filter=Id%20eq%20%27make%27%20and%20IsLatestVersion"
)

// published is the time of the publication of the version 4.4.1 of make in the cases.
var published = time.Date(2026, 9, 1, 12, 0, 0, 470000000, time.UTC)

// feed returns the Atom feed of the OData API of Chocolatey with one entry: the version, the time
// of its publication, and whether it is a pre-release.
func feed(version, published, prerelease string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<feed xml:base="https://community.chocolatey.org/api/v2/" xmlns="http://www.w3.org/2005/Atom"` +
		` xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"` +
		` xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <entry>
    <title type="text">make</title>
    <m:properties>
      <d:Version>` + version + `</d:Version>
      <d:Published m:type="Edm.DateTime">` + published + `</d:Published>
      <d:IsPrerelease m:type="Edm.Boolean">` + prerelease + `</d:IsPrerelease>
    </m:properties>
  </entry>
</feed>
`
}

func TestChocolatey(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want pin.Resolution
		}{
			{
				name: "returns the latest version with the time of its publication",
				give: feed("4.4.1", "2026-09-01T12:00:00.47", "false"),
				want: pin.Resolution{Next: &pin.Release{Published: published, Version: "4.4.1"}},
			},
			{
				name: "leaves out a pre-release",
				give: feed("4.4.1", "2026-09-01T12:00:00.47", "true"),
			},
			{
				name: "leaves out a version that is not stable",
				give: feed("4.4.1-beta", "2026-09-01T12:00:00.47", "false"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{chocolateyQuery: {body: tt.give}})
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindChocolatey, Key: chocolateyKey, Name: "make",
					Version: "4.3",
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
				name: "returns ErrRegistry for a time that does not parse",
				give: feed("4.4.1", "yesterday", "false"),
				want: "the time of make 4.4.1",
			},
			{
				name: "returns ErrRegistry for a feed that does not decode",
				give: "<feed>",
				want: "decode",
			},
			{
				name: "returns ErrRegistry for a package that the repository does not have",
				give: `<feed xmlns="http://www.w3.org/2005/Atom"><title type="text">Packages</title></feed>`,
				want: "has no package make",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{chocolateyQuery: {body: tt.give}})
				_, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindChocolatey, Key: chocolateyKey, Name: "make",
					Version: "4.3",
				})
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}

		t.Run("returns ErrRegistry with the status of a query that fails", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, map[string]route{chocolateyQuery: {status: http.StatusServiceUnavailable}})
			_, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindChocolatey, Key: chocolateyKey, Name: "make",
				Version: "4.3",
			})
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.Contains(t, err.Error(), "503 Service Unavailable", "the error")
		})
	})
}
