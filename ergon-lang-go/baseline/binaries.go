// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"strings"

	"go.dokimi.dev/ergon/core/option"
)

// The values of GOOS and GOARCH of a platform.
const (
	// linux is the GOOS of Linux.
	linux = "linux"

	// darwin is the GOOS of macOS.
	darwin = "darwin"

	// windows is the GOOS of Windows, whose programs end in .exe.
	windows = "windows"

	// amd64 is the GOARCH of x86-64 processors.
	amd64 = "amd64"

	// arm64 is the GOARCH of 64-bit ARM processors.
	arm64 = "arm64"
)

// The repositories on GitHub whose releases publish the release binaries of the section.
const (
	goreleaserRepository = "goreleaser/goreleaser"
	cosignRepository     = "sigstore/cosign"
	syftRepository       = "anchore/syft"
	upxRepository        = "upx/upx"
)

// goreleaserArchs are the architectures of the assets of GoReleaser, by the architecture of Go.
var goreleaserArchs = map[string]string{amd64: "x86_64", arm64: arm64}

// upxTargets are the platforms of the assets of UPX, as the names of the assets state them. UPX
// publishes no program for macOS and for Windows on ARM.
var upxTargets = map[option.Platform]string{
	option.LinuxAMD64:   "amd64_linux",
	option.LinuxARM64:   "arm64_linux",
	option.WindowsAMD64: "win64",
}

// GoReleaser is the release binary of GoReleaser, goreleaser/goreleaser on GitHub, which builds,
// packs and signs the commands of a module in the job pack of release.yml. Its release publishes
// an asset for every platform.
type GoReleaser struct {
	option.Binary `yaml:",inline"`
}

// Cosign is the release binary of cosign, sigstore/cosign on GitHub, which signs the checksums of a
// release without a key. Its release publishes the program itself for every platform but
// windows/arm64.
type Cosign struct {
	option.Binary `yaml:",inline"`
}

// Syft is the release binary of syft, anchore/syft on GitHub, which writes the SBOM of each archive
// and package of a release. Its release publishes an asset for every platform.
type Syft struct {
	option.Binary `yaml:",inline"`
}

// UPX is the release binary of UPX, upx/upx on GitHub, which packs each Linux binary of a command
// in the job pack of release.yml. Its release publishes the program for Linux on x86-64 and on ARM
// and for Windows on x86-64.
type UPX struct {
	option.Binary `yaml:",inline"`
}

var (
	_ option.Release = GoReleaser{}
	_ option.Release = Cosign{}
	_ option.Release = Syft{}
	_ option.Release = UPX{}
)

// Asset returns the archive of GoReleaser for p from the release of the version of g on GitHub:
// goreleaser_<OS>_<arch>.tar.gz with the system capitalized and amd64 as x86_64, a .zip on
// Windows, and the program at its root. It returns an error that wraps [option.ErrNoAsset] for a
// platform that is not valid.
func (g GoReleaser) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil {
		return option.Asset{}, fmt.Errorf("%w: GoReleaser has no asset for %q", option.ErrNoAsset, p)
	}
	program, ext := "goreleaser", ".tar.gz"
	if p.OS() == windows {
		program, ext = program+".exe", ".zip"
	}
	system := strings.ToUpper(p.OS()[:1]) + p.OS()[1:]
	download := "https://github.com/" + goreleaserRepository + "/releases/download/v" + g.Version
	return option.Asset{
		URL:     download + "/goreleaser_" + system + "_" + goreleaserArchs[p.Arch()] + ext,
		Program: program,
	}, nil
}

// Repository returns goreleaser/goreleaser, the repository on GitHub whose releases publish
// GoReleaser.
func (GoReleaser) Repository() string {
	return goreleaserRepository
}

// Asset returns the program of cosign for p from the release of the version of c on GitHub:
// cosign-<os>-<arch>, with .exe on Windows. It returns an error that wraps [option.ErrNoAsset] for
// windows/arm64, whose program the release does not publish, and for a platform that is not valid.
func (c Cosign) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil || p == option.WindowsARM64 {
		return option.Asset{}, fmt.Errorf("%w: cosign has no asset for %q", option.ErrNoAsset, p)
	}
	program := "cosign-" + p.OS() + "-" + p.Arch()
	if p.OS() == windows {
		program += ".exe"
	}
	return option.Asset{
		URL: "https://github.com/" + cosignRepository + "/releases/download/v" + c.Version + "/" + program,
	}, nil
}

// Repository returns sigstore/cosign, the repository on GitHub whose releases publish cosign.
func (Cosign) Repository() string {
	return cosignRepository
}

// Asset returns the archive of syft for p from the release of the version of s on GitHub:
// syft_<version>_<os>_<arch>.tar.gz, a .zip on Windows, with the program at its root. It returns an
// error that wraps [option.ErrNoAsset] for a platform that is not valid.
func (s Syft) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil {
		return option.Asset{}, fmt.Errorf("%w: syft has no asset for %q", option.ErrNoAsset, p)
	}
	program, ext := "syft", ".tar.gz"
	if p.OS() == windows {
		program, ext = program+".exe", ".zip"
	}
	download := "https://github.com/" + syftRepository + "/releases/download/v" + s.Version
	return option.Asset{
		URL:     download + "/syft_" + s.Version + "_" + p.OS() + "_" + p.Arch() + ext,
		Program: program,
	}, nil
}

// Repository returns anchore/syft, the repository on GitHub whose releases publish syft.
func (Syft) Repository() string {
	return syftRepository
}

// Asset returns the archive of UPX for p from the release of the version of u on GitHub:
// upx-<version>-<target>.tar.xz on Linux and upx-<version>-win64.zip on Windows, each with the
// program in the directory of the name of the archive. It returns an error that wraps
// [option.ErrNoAsset] for macOS, for Windows on ARM and for a platform that is not valid.
func (u UPX) Asset(p option.Platform) (option.Asset, error) {
	target, ok := upxTargets[p]
	if !ok {
		return option.Asset{}, fmt.Errorf("%w: UPX has no asset for %q", option.ErrNoAsset, p)
	}
	dir := "upx-" + u.Version + "-" + target
	download := "https://github.com/" + upxRepository + "/releases/download/v" + u.Version + "/" + dir
	if p.OS() == windows {
		return option.Asset{URL: download + ".zip", Program: dir + "/upx.exe"}, nil
	}
	return option.Asset{URL: download + ".tar.xz", Program: dir + "/upx"}, nil
}

// Repository returns upx/upx, the repository on GitHub whose releases publish UPX.
func (UPX) Repository() string {
	return upxRepository
}
