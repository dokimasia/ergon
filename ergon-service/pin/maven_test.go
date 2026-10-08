// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"net/http"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/pin"
)

// The Maven artifact of the cases: its coordinates, its key, the request of its metadata, and the
// request of the POM of its version 1.1.0.
const (
	mavenName     = "com.example:tool"
	mavenKey      = "java.tools.tool"
	mavenMetadata = "GET /maven/com/example/tool/maven-metadata.xml"
	mavenPOM      = "HEAD /maven/com/example/tool/1.1.0/tool-1.1.0.pom"
)

// metadata is maven-metadata.xml of the artifact of the cases, with the versions 1.0.0, 1.1.0 and
// 1.2.0-RC1.
const metadata = `<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>com.example</groupId>
  <artifactId>tool</artifactId>
  <versioning>
    <latest>1.2.0-RC1</latest>
    <release>1.1.0</release>
    <versions>
      <version>1.0.0</version>
      <version>1.1.0</version>
      <version>1.2.0-RC1</version>
    </versions>
  </versioning>
</metadata>
`

func TestMaven(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the newest version with the time of its POM", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, map[string]route{
				mavenMetadata: {body: metadata},
				mavenPOM:      {header: http.Header{"Last-Modified": {old.Format(http.TimeFormat)}}},
			})
			got, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindMaven, Key: mavenKey, Name: mavenName,
				Version: "1.0.0",
			})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, got, pin.Resolution{Next: &pin.Release{Published: old, Version: "1.1.0"}}, "the resolution")
		})

		failures := []struct {
			name   string
			routes map[string]route
			want   string
		}{
			{
				name:   "returns ErrRegistry for a version without a POM",
				routes: map[string]route{mavenMetadata: {body: metadata}},
				want:   mavenName + " 1.1.0 has no POM",
			},
			{
				name:   "returns ErrRegistry for a request of a POM that fails",
				routes: map[string]route{mavenMetadata: {body: metadata}, mavenPOM: {status: http.StatusBadGateway}},
				want:   "502 Bad Gateway",
			},
			{
				name:   "returns ErrRegistry for a POM without its time",
				routes: map[string]route{mavenMetadata: {body: metadata}, mavenPOM: {}},
				want:   "the time of " + mavenName + " 1.1.0",
			},
			{
				name:   "returns ErrRegistry for metadata that does not decode",
				routes: map[string]route{mavenMetadata: {body: "<metadata>"}},
				want:   "decode the metadata of " + mavenName,
			},
			{
				name: "returns ErrRegistry for an artifact that the repository does not have",
				want: "the registry does not have it",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, tt.routes)
				_, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindMaven, Key: mavenKey, Name: mavenName,
					Version: "1.0.0",
				})
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}
	})
}
