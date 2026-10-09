// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/lang/go/baseline"
)

// The downloads of the releases of the tools of the cases.
const (
	goreleaserDownload = "https://github.com/goreleaser/goreleaser/releases/download/v2.18.2/"
	cosignDownload     = "https://github.com/sigstore/cosign/releases/download/v3.1.3/"
	syftDownload       = "https://github.com/anchore/syft/releases/download/v1.54.1/"
	upxDownload        = "https://github.com/upx/upx/releases/download/v5.2.1/"
)

func TestBinaries(t *testing.T) {
	t.Parallel()

	t.Run("GoReleaser", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the archive of Linux on x86-64 with the system capitalized and amd64 as x86_64",
					give: option.LinuxAMD64,
					want: option.Asset{
						URL:     goreleaserDownload + "goreleaser_Linux_x86_64.tar.gz",
						Program: "goreleaser",
					},
				},
				{
					name: "returns the archive of macOS on Apple silicon",
					give: option.DarwinARM64,
					want: option.Asset{
						URL:     goreleaserDownload + "goreleaser_Darwin_arm64.tar.gz",
						Program: "goreleaser",
					},
				},
				{
					name: "returns a .zip with goreleaser.exe on Windows",
					give: option.WindowsAMD64,
					want: option.Asset{
						URL:     goreleaserDownload + "goreleaser_Windows_x86_64.zip",
						Program: "goreleaser.exe",
					},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := goOptions().Tools.GoReleaser.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for a platform that is not valid", func(t *testing.T) {
				t.Parallel()
				_, err := goOptions().Tools.GoReleaser.Asset("linux/386")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
			})
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns goreleaser/goreleaser", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.GoReleaser{}.Repository(), "goreleaser/goreleaser", "the repository")
			})
		})
	})

	t.Run("Cosign", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the program of Linux on 64-bit ARM",
					give: option.LinuxARM64,
					want: option.Asset{URL: cosignDownload + "cosign-linux-arm64"},
				},
				{
					name: "returns the program with .exe on Windows",
					give: option.WindowsAMD64,
					want: option.Asset{URL: cosignDownload + "cosign-windows-amd64.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := goOptions().Tools.Cosign.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			for _, p := range []option.Platform{option.WindowsARM64, "linux/386"} {
				t.Run("returns ErrNoAsset for "+string(p), func(t *testing.T) {
					t.Parallel()
					_, err := goOptions().Tools.Cosign.Asset(p)
					assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
				})
			}
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns sigstore/cosign", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.Cosign{}.Repository(), "sigstore/cosign", "the repository")
			})
		})
	})

	t.Run("Syft", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the archive of macOS on x86-64 with the version in its name",
					give: option.DarwinAMD64,
					want: option.Asset{URL: syftDownload + "syft_1.54.1_darwin_amd64.tar.gz", Program: "syft"},
				},
				{
					name: "returns a .zip with syft.exe on Windows",
					give: option.WindowsARM64,
					want: option.Asset{URL: syftDownload + "syft_1.54.1_windows_arm64.zip", Program: "syft.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := goOptions().Tools.Syft.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			t.Run("returns ErrNoAsset for a platform that is not valid", func(t *testing.T) {
				t.Parallel()
				_, err := goOptions().Tools.Syft.Asset("darwin/386")
				assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
			})
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns anchore/syft", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.Syft{}.Repository(), "anchore/syft", "the repository")
			})
		})
	})

	t.Run("UPX", func(t *testing.T) {
		t.Parallel()

		t.Run("Asset", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name string
				give option.Platform
				want option.Asset
			}{
				{
					name: "returns the .tar.xz of Linux on x86-64 with the program in the directory of its name",
					give: option.LinuxAMD64,
					want: option.Asset{
						URL:     upxDownload + "upx-5.2.1-amd64_linux.tar.xz",
						Program: "upx-5.2.1-amd64_linux/upx",
					},
				},
				{
					name: "returns the .tar.xz of Linux on 64-bit ARM",
					give: option.LinuxARM64,
					want: option.Asset{
						URL:     upxDownload + "upx-5.2.1-arm64_linux.tar.xz",
						Program: "upx-5.2.1-arm64_linux/upx",
					},
				},
				{
					name: "returns the .zip of Windows on x86-64 with upx.exe",
					give: option.WindowsAMD64,
					want: option.Asset{URL: upxDownload + "upx-5.2.1-win64.zip", Program: "upx-5.2.1-win64/upx.exe"},
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := goOptions().Tools.UPX.Asset(tt.give)
					assert.NoError(t, err, "Asset")
					assert.Equal(t, got, tt.want, "the asset")
				})
			}

			for _, p := range []option.Platform{option.DarwinARM64, option.WindowsARM64, "linux/386"} {
				t.Run("returns ErrNoAsset for "+string(p), func(t *testing.T) {
					t.Parallel()
					_, err := goOptions().Tools.UPX.Asset(p)
					assert.ErrorIs(t, err, option.ErrNoAsset, "Asset")
				})
			}
		})

		t.Run("Repository", func(t *testing.T) {
			t.Parallel()

			t.Run("returns upx/upx", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, baseline.UPX{}.Repository(), "upx/upx", "the repository")
			})
		})
	})
}
