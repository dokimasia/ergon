// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"slices"

	"go.dokimi.dev/ergon/core/version"
)

// Kind is the section of a manifest that declares a requirement, which decides how a release of
// the required package moves the package that requires it. The zero value is not a valid kind.
type Kind string

// The sections of a manifest that declare a requirement.
const (
	// KindRuntime is a requirement of the code that the package ships, such as a require line of
	// go.mod or a dependency of package.json.
	KindRuntime Kind = "runtime"

	// KindOptional is a requirement that the package uses when it is installed, such as an
	// optional dependency of package.json.
	KindOptional Kind = "optional"

	// KindPeer is a requirement that the user of the package installs, such as a peer dependency
	// of package.json.
	KindPeer Kind = "peer"

	// KindDev is a requirement of the build and the tests alone, such as a dev dependency of
	// package.json.
	KindDev Kind = "dev"
)

// kinds are the sections of a manifest that declare a requirement.
var kinds = []Kind{KindRuntime, KindOptional, KindPeer, KindDev}

// Valid reports whether k is one of the four kinds.
func (k Kind) Valid() bool {
	return slices.Contains(kinds, k)
}

// Dependency is a requirement of a package on another package of the same repository. A
// requirement on a package of another repository is no Dependency.
type Dependency struct {
	// Name is the Name of the required package.
	Name string

	// Kind is the section of the manifest that declares the requirement.
	Kind Kind

	// Req is the requirement as the manifest writes it, in the syntax of the toolchain, such as
	// v1.2.0 in go.mod or ^1.2.0 in package.json.
	Req string
}

// Package is one unit that a toolchain releases: a Go module, a crate, an npm package, a Python
// distribution, or a Gradle or Maven project. The discovery of its toolchain returns it.
type Package struct {
	// Name is the name that the registry of the package knows: a module path, a crate name, an npm
	// name, a distribution name or group:artifact.
	Name string

	// Toolchain is the toolchain that discovered the package.
	Toolchain Toolchain

	// Dir is the directory of the package, relative to the root of the repository and
	// slash-separated. "." is the root.
	Dir string

	// Source names the file and the field that the version is written to when other packages read
	// the same field, such as [workspace.package] of Cargo.toml. Packages with the same non-empty
	// Source share one version.
	Source string

	// Deps are the requirements of the package on the other packages of the repository, in the
	// order of the manifest.
	Deps []Dependency

	// Version is the version that the manifest of the package states. For a toolchain whose
	// manifest has no version, such as Go, it is the version of the newest heading of the
	// package's CHANGELOG.md, and the zero version for a package that has never been released.
	Version version.Version

	// Private reports that the package is never published to a registry.
	Private bool
}
