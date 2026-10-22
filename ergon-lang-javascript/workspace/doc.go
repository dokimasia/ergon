// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package workspace discovers the packages of an npm or a pnpm workspace as the packages of a
// release.
//
// [Discover] reads the globs of the packages from pnpm-workspace.yaml, or else from the workspaces
// of the package.json at the root, as pnpm and npm read them, and reads the package.json of each
// directory that the globs match. Each package is a package of [Toolchain]. Its requirements on the
// other packages are the requirements of the package: dependencies as runtime requirements,
// optionalDependencies as optional requirements, peerDependencies as peer requirements, and
// devDependencies as dev requirements. A repository without globs has one package, at its root.
//
// # The workspace protocol
//
// pnpm replaces a requirement of the workspace protocol when it packs a package, such as
// workspace:^ with ^1.2.0 for a package at 1.2.0. [Shorthand] reports the requirements that pnpm
// replaces with the version of the required package. Discover returns such a requirement as pnpm
// packs it, after the protocol, so a release resolves the requirement that the registry receives.
//
// # Manifests
//
// [ParseManifest] reads the members of a package.json that a release reads and writes, with the
// range of bytes of each string in the file, so a release rewrites a version or a requirement in
// place and keeps every other byte.
//
// # Errors
//
// The errors wrap [ErrManifest] for a package.json or a pnpm-workspace.yaml that a release cannot
// read.
//
// # Dependency position
//
// Imports the standard library, github.com/bmatcuk/doublestar/v4 for the globs of a workspace,
// go.yaml.in/yaml/v3 for pnpm-workspace.yaml, [go.dokimi.dev/ergon/core/version] and
// [go.dokimi.dev/ergon/core/workspace]. The package go.dokimi.dev/ergon/lang/javascript and its
// package release import it.
package workspace
