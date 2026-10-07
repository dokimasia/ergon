// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option_test

import (
	"fmt"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
)

// The digests of the assets of uv 0.12.23 for the three platforms of the baseline, as its
// release's sha256.sum states them.
const (
	uvLinux   = "9167d72b3319674b6303c4cbe071854bba13ebdf3d76b1a7cbdc175471fb66d6"
	uvDarwin  = "50487ae565ccd96e499056b4674d438f4c53170202617b4c759defe0c6a1b544"
	uvWindows = "75d05de6762778c31ee183398de7dd15093fad0ed90b1f236d8205ea5ec00c90"
)

// uvRelease is the address of the assets of uv 0.12.23.
const uvRelease = "https://github.com/astral-sh/uv/releases/download/0.12.23/"

func TestBinary(t *testing.T) {
	t.Parallel()

	t.Run("Platform", func(t *testing.T) {
		t.Parallel()

		t.Run("OS", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the system before the slash", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, option.DarwinARM64.OS(), "darwin", "OS")
			})
		})

		t.Run("Arch", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the architecture after the slash", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, option.DarwinARM64.Arch(), "arm64", "Arch")
			})
		})

		t.Run("Validate", func(t *testing.T) {
			t.Parallel()

			t.Run("returns nil for every platform of a release binary", func(t *testing.T) {
				t.Parallel()
				assert.Total(t, option.Platform.Validate, []option.Platform{
					option.LinuxAMD64, option.LinuxARM64, option.DarwinAMD64, option.DarwinARM64, option.WindowsAMD64,
					option.WindowsARM64,
				}, "Validate of each platform")
			})

			invalid := []struct {
				name string
				give option.Platform
			}{
				{name: "returns ErrInvalid for the empty platform", give: ""},
				{name: "returns ErrInvalid for a system without an architecture", give: "linux"},
				{name: "returns ErrInvalid for an architecture of 32 bits", give: "linux/386"},
				{name: "returns ErrInvalid for another system", give: "freebsd/amd64"},
			}
			for _, tt := range invalid {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					assert.ErrorIs(t, tt.give.Validate(), option.ErrInvalid, "Validate")
				})
			}
		})
	})

	t.Run("Pin", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the version and the digests", func(t *testing.T) {
			t.Parallel()
			b := uv()
			assert.Equal(t, b.Pin(), b.Binary, "Pin")
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a version with a digest of each platform", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, uv().Validate(), "Validate")
		})

		invalid := []struct {
			name string
			give func(*option.Binary)
		}{
			{name: "returns ErrInvalid for an empty version", give: func(b *option.Binary) { b.Version = "" }},
			{
				name: "returns ErrInvalid for a version with a space",
				give: func(b *option.Binary) { b.Version = "0.12 rc" },
			},
			{name: "returns ErrInvalid for no digest", give: func(b *option.Binary) { b.SHA256 = nil }},
			{
				name: "returns ErrInvalid for a digest of an invalid platform",
				give: func(b *option.Binary) { b.SHA256["linux/386"] = uvLinux },
			},
			{
				name: "returns ErrInvalid for a digest in uppercase",
				give: func(b *option.Binary) { b.SHA256[option.LinuxAMD64] = strings.ToUpper(uvLinux) },
			},
			{
				name: "returns ErrInvalid for a digest of 63 digits",
				give: func(b *option.Binary) { b.SHA256[option.LinuxAMD64] = uvLinux[1:] },
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				b := uv()
				tt.give(&b.Binary)
				assert.ErrorIs(t, b.Validate(), option.ErrInvalid, "Validate")
			})
		}
	})

	t.Run("UV", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the archive of Linux on x86-64",
					give: option.LinuxAMD64,
					want: option.Asset{
						URL:     uvRelease + "uv-x86_64-unknown-linux-gnu.tar.gz",
						Program: "uv-x86_64-unknown-linux-gnu/uv",
					},
				},
				{
					name: "returns the archive of Linux on ARM",
					give: option.LinuxARM64,
					want: option.Asset{
						URL:     uvRelease + "uv-aarch64-unknown-linux-gnu.tar.gz",
						Program: "uv-aarch64-unknown-linux-gnu/uv",
					},
				},
				{
					name: "returns the archive of macOS on x86-64",
					give: option.DarwinAMD64,
					want: option.Asset{
						URL:     uvRelease + "uv-x86_64-apple-darwin.tar.gz",
						Program: "uv-x86_64-apple-darwin/uv",
					},
				},
				{
					name: "returns the archive of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{
						URL:     uvRelease + "uv-aarch64-apple-darwin.tar.gz",
						Program: "uv-aarch64-apple-darwin/uv",
					},
				},
				{
					name: "returns the zip of Windows on x86-64 with the program at its root",
					give: option.WindowsAMD64,
					want: option.Asset{URL: uvRelease + "uv-x86_64-pc-windows-msvc.zip", Program: "uv.exe"},
				},
				{
					name: "returns the zip of Windows on ARM with the program at its root",
					give: option.WindowsARM64,
					want: option.Asset{URL: uvRelease + "uv-aarch64-pc-windows-msvc.zip", Program: "uv.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := uv().Asset(tt.give)
					assert.NoError(t, err, fmt.Sprintf("Asset of %s", tt.give))
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for an invalid platform", func(t *testing.T) {
				t.Parallel()
				_, err := uv().Asset("linux/386")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset of linux/386")
			})
		})
	})
}

// uv returns a new pin of uv 0.12.23 with the digests of the three platforms of the baseline.
func uv() option.UV {
	return option.UV{
		SHA256: map[option.Platform]string{
			option.LinuxAMD64:   uvLinux,
			option.DarwinARM64:  uvDarwin,
			option.WindowsAMD64: uvWindows,
		},
		Version: "0.12.23",
	}
}
