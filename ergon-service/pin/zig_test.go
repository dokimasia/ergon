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

// The zig of the cases: its key, the request of the download index, and the digests of the archives
// of its next version.
const (
	zigKey      = "go.tools.zig"
	zigIndex    = "GET /zig/index.json"
	nextLinux   = "3333333333333333333333333333333333333333333333333333333333333333"
	nextWindows = "4444444444444444444444444444444444444444444444444444444444444444"
)

// The versions of the download index of the cases: master, which is no release, the next version
// with the archives of Linux and Windows on x86-64, and the version of the pin.
const (
	masterEntry = `"master": {"version": "0.18.0-dev.367+187d0bf9e", "date": "2026-10-01", "x86_64-linux": ` +
		`{"tarball": "https://ziglang.org/builds/zig-x86_64-linux-0.18.0-dev.367+187d0bf9e.tar.xz", ` +
		`"shasum": "` + darwinDigest + `", "size": "1"}}`
	nextEntry = `"0.17.0": {"version": "0.17.0", "date": "2026-09-01", ` +
		`"docs": "https://ziglang.org/documentation/0.17.0/", ` +
		`"src": {"tarball": "https://ziglang.org/download/0.17.0/zig-0.17.0.tar.xz", "shasum": "` + darwinDigest +
		`", "size": "1"}, ` +
		`"x86_64-linux": {"tarball": "https://ziglang.org/download/0.17.0/zig-x86_64-linux-0.17.0.tar.xz", ` +
		`"shasum": "` + nextLinux + `", "size": "1"}, ` +
		`"x86_64-windows": {"tarball": "https://ziglang.org/download/0.17.0/zig-x86_64-windows-0.17.0.zip", ` +
		`"shasum": "` + nextWindows + `", "size": "1"}}`
	currentEntry = `"0.16.0": {"version": "0.16.0", "date": "2026-04-13", ` +
		`"x86_64-linux": {"tarball": "https://ziglang.org/download/0.16.0/zig-x86_64-linux-0.16.0.tar.xz", ` +
		`"shasum": "` + linuxDigest + `", "size": "1"}}`
)

// dated is the time of the date of the next version of zig in the cases.
var dated = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// zigPin is the pin of zig of the cases, at the version of zig of the options of the cases.
var zigPin = pin.Pin{Kind: pin.KindZig, Key: zigKey, Name: "zig", Version: zig.Version, Value: zig}

func TestZig(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want pin.Resolution
		}{
			{
				name: "returns the newest version with the digests of the archives of the platforms of the pin",
				give: "{" + masterEntry + ", " + nextEntry + ", " + currentEntry + "}",
				want: pin.Resolution{Next: &pin.Release{
					Published: dated, Version: "0.17.0",
					Digests: map[option.Platform]string{option.LinuxAMD64: nextLinux, option.WindowsAMD64: nextWindows},
				}},
			},
			{
				name: "returns no release for an index whose newest release is the version of the pin",
				give: "{" + masterEntry + ", " + currentEntry + "}",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{zigIndex: {body: tt.give}})
				p := zigPin
				got, err := r.Resolve(t.Context(), &p)
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		t.Run("returns ErrAsset for a version without the archive of a platform of the pin", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, map[string]route{zigIndex: {body: "{" + nextEntry + "}"}})
			p := zigPin
			p.Value = option.Zig{
				SHA256:  map[option.Platform]string{option.DarwinARM64: darwinDigest},
				Version: zig.Version,
			}
			_, err := r.Resolve(t.Context(), &p)
			assert.ErrorIs(t, err, pin.ErrAsset, "Resolve")
		})

		failures := []struct {
			name string
			give route
			want string
		}{
			{
				name: "returns ErrRegistry for an index that does not decode",
				give: route{body: "["},
				want: "decode",
			},
			{
				name: "returns ErrRegistry for a version whose date does not parse",
				give: route{body: `{"0.17.0": {"date": "yesterday"}}`},
				want: "the date of zig 0.17.0",
			},
			{
				name: "returns ErrRegistry for a version without a date",
				give: route{body: `{"0.17.0": {"version": "0.17.0"}}`},
				want: "the date of zig 0.17.0",
			},
			{
				name: "returns ErrRegistry with the status of an index that fails",
				give: route{status: http.StatusServiceUnavailable},
				want: "503 Service Unavailable",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{zigIndex: tt.give})
				p := zigPin
				_, err := r.Resolve(t.Context(), &p)
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}
	})
}
