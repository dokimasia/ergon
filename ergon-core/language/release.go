// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package language

import (
	"context"

	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Resolution is how the resolver of a consumer treats a version of a dependency under one
// requirement of a dependent: whether a consumer that installs the dependent receives the version
// without a new release of the dependent. The zero value is not a valid resolution.
type Resolution uint8

// The resolutions of a version under a requirement.
const (
	// ResolutionExcluded is a requirement that does not admit the version, such as ^1.2.0 of npm
	// for 2.0.0, or a require line of Go that names a higher version.
	ResolutionExcluded Resolution = 1

	// ResolutionPinned is a requirement that admits the version while a consumer that resolves the
	// requirement alone gets the version that the requirement names, as minimal version selection
	// does with a require line of Go for every higher version.
	ResolutionPinned Resolution = 2

	// ResolutionSelected is a requirement through which a consumer gets the version, as npm,
	// Cargo and pip pick the newest version that a range admits, and as Go picks the version that a
	// require line names.
	ResolutionSelected Resolution = 3
)

// Valid reports whether r is one of the three resolutions.
func (r Resolution) Valid() bool {
	return r >= ResolutionExcluded && r <= ResolutionSelected
}

// Edit is the change that a release makes to the manifest of one package.
type Edit struct {
	// Version is the new version of the package, or Package.Version when the release changes only
	// its requirements.
	Version version.Version

	// Requirements are the requirements of the package that the release rewrites, each with its new
	// Req, in the order of Package.Deps. It is nil when no requirement changes.
	Requirements []workspace.Dependency

	// Package is the package as its toolchain discovered it.
	Package workspace.Package
}

// Versioner is the release role of a toolchain: how its consumers resolve a requirement, the
// syntax of its requirements, the versions that it refuses, and the writes of a release into its
// manifests and lockfiles. A toolchain that releases packages implements it.
type Versioner interface {
	// Resolve returns how a consumer that resolves req alone treats v, where req is a requirement
	// in the syntax of the toolchain. It returns an error for a req that the toolchain does not
	// read.
	Resolve(req string, v version.Version) (Resolution, error)

	// Rewrite returns req moved to admit v with the same operator, such as ^1.2.0 moved to ^1.3.0,
	// or v1.2.0 moved to v1.3.0 in go.mod. It returns an error for a req that the toolchain does
	// not read.
	Rewrite(req string, v version.Version) (string, error)

	// Validate returns an error for a release of p at v that the toolchain refuses, such as a major
	// release of a Go module whose path does not end in the new major version, and nil for any
	// other release. It does not modify p.
	Validate(p *workspace.Package, v version.Version) error

	// Apply writes the new version and the rewritten requirements of each of edits into the
	// manifests under root, and refreshes the lockfiles. It returns the paths that it changed,
	// relative to root and slash-separated. On an error it also returns the paths that it changed
	// before the error, and the caller restores them.
	Apply(ctx context.Context, root string, edits []Edit) ([]string, error)
}

// Tagger is the release role of a toolchain whose tools read the version of a package from the
// name of its tag, as the go command reads <dir>/vX.Y.Z. A toolchain without it takes the tags of
// changesets: v<version> in a repository with one package, and <name>@<version> otherwise.
type Tagger interface {
	// Tag returns the name of the tag of p at v. It does not modify p.
	Tag(p *workspace.Package, v version.Version) string
}

// The directories of a pack directory that a [Packer] writes beside the artifacts of a registry.
const (
	// AssetsDir has the assets of the release of each tag, as assets/<tag>/<file>, which a publish
	// attaches to the release of the tag.
	AssetsDir = "assets"

	// CasksDir has the Homebrew casks of the release of each tag, as casks/<tag>/<name>.rb, which a
	// release commits to its tap after the publish.
	CasksDir = "casks"
)

// Packer is the release role of a toolchain whose packages a registry receives as artifacts, such
// as the tarballs of npm, or whose releases have assets, such as the archives of the commands of a
// Go module.
type Packer interface {
	// Pack builds the artifacts of pkgs from root into dir: the files that the registry of a
	// package receives, and the assets of the release of its tag under [AssetsDir]/<tag>/ of dir.
	// Each package of pkgs has the Version of its release. Pack returns the error of the build.
	Pack(ctx context.Context, root string, pkgs []workspace.Package, dir string) error
}

// Publisher is the release role of a toolchain with a registry. A toolchain without it publishes a
// release by its tag alone, as the go command reads a module from its tag.
type Publisher interface {
	// Published reports whether the registry has p at its Version. It returns the error of the
	// registry, and does not modify p.
	Published(ctx context.Context, p *workspace.Package) (bool, error)

	// Publish uploads the artifacts of pkgs from dir to the registry, in the order of pkgs. It
	// returns the error of the first upload that fails.
	Publish(ctx context.Context, dir string, pkgs []workspace.Package) error
}

// Locker is the release role of a toolchain whose lockfiles record the content of the packages of
// the repository, as go.sum records the hash of each version of a module that a module requires.
// The version commit of a release records the content of each released package. A change to such
// a package before its publish makes the record stale, and the tag of the publish would then name
// content that the lockfiles do not record. A toolchain whose lockfiles do not record such content,
// such as npm with its workspace links, does not implement Locker.
type Locker interface {
	// Stale returns the lockfiles under root that record, for a package of pkgs at its Version,
	// content other than the content of the package in the working tree. The paths are relative to
	// root, slash-separated and sorted. Stale modifies neither the files under root nor pkgs. It
	// returns the error of reading the lockfiles and of building the content of a package.
	Stale(ctx context.Context, root string, pkgs []workspace.Package) ([]string, error)

	// Lock rewrites the lockfiles under root that record content of a package of pkgs at its
	// Version, so that they record the content of the package in the working tree. It returns the
	// paths that it changed, relative to root and slash-separated. On an error it also returns the
	// paths that it changed before the error, and the caller restores them.
	Lock(ctx context.Context, root string, pkgs []workspace.Package) ([]string, error)
}
