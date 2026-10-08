// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"net/http"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/pin"
)

// The module of the cases: its key, the package of its command, and the requests of the proxy for
// its @latest, its list of versions and the documents of its tags.
const (
	moduleKey      = "go.tools.tool"
	modulePackage  = "example.com/tool/cmd/tool"
	moduleLatest   = "GET /goproxy/example.com/tool/@latest"
	moduleList     = "GET /goproxy/example.com/tool/@v/list"
	moduleInfo     = "GET /goproxy/example.com/tool/@v/v1.1.0.info"
	moduleNextInfo = "GET /goproxy/example.com/tool/@v/v1.2.0.info"
)

// The pseudo-versions of the cases: a commit of 2026-09-01 and a commit of 2026-09-20 of a module
// without tags, and a commit of 2026-09-20 after the tag v1.2.3.
const (
	firstCommit  = "v0.0.0-20260901120000-aaaaaaaaaaaa"
	secondCommit = "v0.0.0-20260920120000-bbbbbbbbbbbb"
	afterTag     = "v1.2.4-0.20260920120000-bbbbbbbbbbbb"
)

// committed is the time of the commits of secondCommit and afterTag.
var committed = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

func TestGoProxy(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			version string
			routes  map[string]route
			want    pin.Resolution
		}{
			{
				name:    "returns the newest tag of the module of a package",
				version: "v1.0.0",
				routes: map[string]route{
					moduleLatest:   {body: `{"Version":"v1.2.0","Time":"2026-09-01T12:00:00Z"}`},
					moduleList:     {body: "v1.0.0\nv1.1.0\nv1.2.0\nv1.3.0-rc.1\n"},
					moduleInfo:     {body: `{"Version":"v1.1.0","Time":"2026-09-01T12:00:00Z"}`},
					moduleNextInfo: {body: `{"Version":"v1.2.0","Time":"2026-09-01T12:00:00Z"}`},
				},
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "v1.2.0"}},
			},
			{
				name:    "takes the pseudo-version of the newest commit of a module without tags",
				version: firstCommit,
				routes: map[string]route{
					moduleLatest: {body: `{"Version":"` + secondCommit + `","Time":"2026-09-20T12:00:00Z"}`},
					moduleList:   {},
				},
				want: pin.Resolution{Next: &pin.Release{Published: committed, Version: secondCommit}},
			},
			{
				name:    "leaves out the pseudo-version of the commit of the pin",
				version: secondCommit,
				routes: map[string]route{
					moduleLatest: {body: `{"Version":"` + secondCommit + `","Time":"2026-09-20T12:00:00Z"}`},
					moduleList:   {},
				},
			},
			{
				name:    "takes the tag of the numbers of the pseudo-version of the pin",
				version: afterTag,
				routes: map[string]route{
					moduleLatest: {body: `{"Version":"v1.2.4","Time":"2026-09-01T12:00:00Z"}`},
					moduleList:   {body: "v1.2.3\nv1.2.4\n"},
					"GET /goproxy/example.com/tool/@v/v1.2.4.info": {
						body: `{"Version":"v1.2.4","Time":"2026-09-01T12:00:00Z"}`,
					},
				},
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "v1.2.4"}},
			},
			{
				name:    "ranks a tag above a pseudo-version of the same numbers",
				version: "v1.2.3",
				routes: map[string]route{
					moduleLatest: {body: `{"Version":"` + afterTag + `","Time":"2026-09-20T12:00:00Z"}`},
					moduleList:   {body: "v1.2.4\n"},
					"GET /goproxy/example.com/tool/@v/v1.2.4.info": {
						body: `{"Version":"v1.2.4","Time":"2026-09-01T12:00:00Z"}`,
					},
				},
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "v1.2.4"}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, tt.routes)
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindModule, Key: moduleKey, Name: modulePackage,
					Version: tt.version, Value: option.Module(modulePackage + "@" + tt.version),
				})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		t.Run("escapes the upper-case letters of the path of a module", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, map[string]route{
				"GET /goproxy/github.com/!burnt!sushi/toml/@latest": {
					body: `{"Version":"v1.6.0","Time":"2026-09-01T12:00:00Z"}`,
				},
				"GET /goproxy/github.com/!burnt!sushi/toml/@v/list": {body: "v1.5.0\nv1.6.0\n"},
				"GET /goproxy/github.com/!burnt!sushi/toml/@v/v1.6.0.info": {
					body: `{"Version":"v1.6.0","Time":"2026-09-01T12:00:00Z"}`,
				},
			})
			got, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindModule, Key: moduleKey,
				Name: "github.com/BurntSushi/toml/cmd/tomlv", Version: "v1.5.0",
			})
			assert.NoError(t, err, "Resolve")
			assert.Equal(
				t,
				got,
				pin.Resolution{Next: &pin.Release{Published: old, Version: "v1.6.0"}},
				"the resolution",
			)
		})

		failures := []struct {
			name   string
			pkg    string
			routes map[string]route
			want   string
		}{
			{
				name: "returns ErrRegistry for a package of no module",
				pkg:  modulePackage,
				want: "no module of the proxy has the package " + modulePackage,
			},
			{
				name: "returns ErrRegistry for a package path that is no module path",
				pkg:  "example.com/tool.",
				want: `the module path "example.com/tool."`,
			},
			{
				name: "returns ErrRegistry for a request of @latest that fails",
				pkg:  modulePackage,
				routes: map[string]route{
					"GET /goproxy/example.com/tool/cmd/tool/@latest": {status: http.StatusBadGateway},
				},
				want: "502 Bad Gateway",
			},
			{
				name:   "returns ErrRegistry for a document of @latest that does not decode",
				pkg:    modulePackage,
				routes: map[string]route{moduleLatest: {body: "{"}},
				want:   "decode",
			},
			{
				name: "returns ErrRegistry for a module without a list of versions",
				pkg:  modulePackage,
				routes: map[string]route{
					moduleLatest: {body: `{"Version":"v1.1.0","Time":"2026-09-01T12:00:00Z"}`},
				},
				want: "the registry does not have it",
			},
			{
				name: "returns ErrRegistry for a tag that the proxy lists without its document",
				pkg:  modulePackage,
				routes: map[string]route{
					moduleLatest: {body: `{"Version":"v1.1.0","Time":"2026-09-01T12:00:00Z"}`},
					moduleList:   {body: "v1.1.0\n"},
				},
				want: "lists v1.1.0 without its document",
			},
			{
				name: "returns ErrRegistry for a request of the document of a tag that fails",
				pkg:  modulePackage,
				routes: map[string]route{
					moduleLatest: {body: `{"Version":"v1.1.0","Time":"2026-09-01T12:00:00Z"}`},
					moduleList:   {body: "v1.1.0\n"},
					moduleInfo:   {status: http.StatusBadGateway},
				},
				want: "502 Bad Gateway",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, tt.routes)
				_, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindModule, Key: moduleKey, Name: tt.pkg,
					Version: "v1.0.0",
				})
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}
	})
}
