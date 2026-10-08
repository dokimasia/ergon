// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// Platform is a system and an architecture, as <os>/<arch> in the names of Go, such as
// linux/amd64.
type Platform string

// The platforms of a release binary.
const (
	// LinuxAMD64 is Linux on x86-64.
	LinuxAMD64 Platform = "linux/amd64"

	// LinuxARM64 is Linux on 64-bit ARM.
	LinuxARM64 Platform = "linux/arm64"

	// DarwinAMD64 is macOS on x86-64.
	DarwinAMD64 Platform = "darwin/amd64"

	// DarwinARM64 is macOS on Apple silicon.
	DarwinARM64 Platform = "darwin/arm64"

	// WindowsAMD64 is Windows on x86-64.
	WindowsAMD64 Platform = "windows/amd64"

	// WindowsARM64 is Windows on 64-bit ARM.
	WindowsARM64 Platform = "windows/arm64"
)

// platforms are the platforms of a release binary.
var platforms = []Platform{LinuxAMD64, LinuxARM64, DarwinAMD64, DarwinARM64, WindowsAMD64, WindowsARM64}

// sha256Digest matches a SHA-256 digest in lowercase hexadecimal.
var sha256Digest = regexp.MustCompile(`^[0-9a-f]{64}$`)

// OS returns the system of p, the text before its slash, such as linux.
func (p Platform) OS() string {
	os, _, _ := strings.Cut(string(p), "/")
	return os
}

// Arch returns the architecture of p, the text after its slash, such as amd64.
func (p Platform) Arch() string {
	_, arch, _ := strings.Cut(string(p), "/")
	return arch
}

// Validate returns an error that wraps [ErrInvalid] for a p that is none of the platforms of a
// release binary: linux, darwin or windows, on amd64 or arm64.
func (p Platform) Validate() error {
	if !slices.Contains(platforms, p) {
		return fmt.Errorf("%w: platform %q, which is not linux, darwin or windows on amd64 or arm64", ErrInvalid, p)
	}
	return nil
}

// Asset is the file that a release publishes for one platform.
type Asset struct {
	// URL is the address of the asset, over https. Its suffix states the format of an archive:
	// .tar.gz or .zip. An asset with neither suffix is the program itself.
	URL string

	// Program is the slash-separated path of the program in the archive, and empty for an asset
	// that is the program itself.
	Program string
}

// Binary is the pin of a release binary: its version and the SHA-256 digest of the asset of each
// platform. ergon tool run checks a download against the digest of its platform before it unpacks
// it, so a section names no binary whose bytes it has not pinned. A tool of the kind embeds Binary
// in a type that states where its release publishes each asset, as [Release] requires.
type Binary struct {
	// SHA256 are the digests of the assets, by platform, in lowercase hexadecimal.
	SHA256 map[Platform]string `yaml:"sha256"`

	// Version is the version of the release, such as 0.12.0.
	Version string `yaml:"version"`
}

// Pin returns b. A type that embeds Binary returns its version and its digests through it, as
// [Release] requires.
func (b Binary) Pin() Binary {
	return b
}

// Validate returns an error that wraps [ErrInvalid] for a version that is empty or has a
// character other than a letter, a digit, '.', '_', '+' and '-', for no digest, for a digest of a
// platform that is not valid, as [Platform.Validate] states, and for a digest that is not 64
// lowercase hexadecimal digits. It reads the digests in the order of their platforms, so it
// returns the same error for the same pin.
func (b Binary) Validate() error {
	if !version.MatchString(b.Version) {
		return fmt.Errorf("%w: version %q, which is not the version of a release", ErrInvalid, b.Version)
	}
	if len(b.SHA256) == 0 {
		return fmt.Errorf("%w: version %s has no digest", ErrInvalid, b.Version)
	}
	for _, p := range slices.Sorted(maps.Keys(b.SHA256)) {
		if err := p.Validate(); err != nil {
			return err
		}
		if d := b.SHA256[p]; !sha256Digest.MatchString(d) {
			return fmt.Errorf("%w: the digest %q of %s, which is not 64 lowercase hexadecimal digits", ErrInvalid, d, p)
		}
	}
	return nil
}

// Release is a release binary of a section: a type that embeds [Binary] and states where its
// release publishes the asset of each platform. ergon tool run downloads the asset of the
// platform that it runs on, and a baseline update reads the releases of the repository.
type Release interface {
	// Pin returns the version and the digests of the binary.
	Pin() Binary

	// Asset returns the asset of p for the version of the pin. It returns an error that wraps
	// [ErrNoAsset] for a platform whose asset the release does not publish.
	Asset(p Platform) (Asset, error)

	// Repository returns the repository on GitHub whose releases publish the binary, as
	// owner/name, such as astral-sh/uv.
	Repository() string
}

// UV is the release binary of uv, through which ergon tool run runs the tools of the kind [PyPI]
// of its section. The release of uv publishes an asset for every platform.
type UV struct {
	Binary `yaml:",inline"`
}

var _ Release = UV{}

// uvRepository is the repository on GitHub whose releases publish uv.
const uvRepository = "astral-sh/uv"

// uvTargets are the target triples of the assets of uv, by platform.
var uvTargets = map[Platform]string{
	LinuxAMD64:   "x86_64-unknown-linux-gnu",
	LinuxARM64:   "aarch64-unknown-linux-gnu",
	DarwinAMD64:  "x86_64-apple-darwin",
	DarwinARM64:  "aarch64-apple-darwin",
	WindowsAMD64: "x86_64-pc-windows-msvc",
	WindowsARM64: "aarch64-pc-windows-msvc",
}

// Asset returns the archive of uv for p from the release of the version of u on GitHub: a
// .tar.gz with the program in a directory named after the target triple, and a .zip with uv.exe
// at its root on Windows. It returns an error that wraps [ErrNoAsset] for a platform that is not
// valid.
func (u UV) Asset(p Platform) (Asset, error) {
	target, ok := uvTargets[p]
	if !ok {
		return Asset{}, fmt.Errorf("%w: uv has no asset for %q", ErrNoAsset, p)
	}
	release := "https://github.com/" + uvRepository + "/releases/download/" + u.Version + "/uv-" + target
	if p.OS() == "windows" {
		return Asset{URL: release + ".zip", Program: "uv.exe"}, nil
	}
	return Asset{URL: release + ".tar.gz", Program: "uv-" + target + "/uv"}, nil
}

// Repository returns astral-sh/uv, the repository on GitHub whose releases publish uv.
func (UV) Repository() string {
	return uvRepository
}
