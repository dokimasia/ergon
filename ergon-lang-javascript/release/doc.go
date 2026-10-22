// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package release releases the packages of an npm or a pnpm workspace, which
// [go.dokimi.dev/ergon/lang/javascript/workspace] discovers, to the npm registry.
//
// [Versioner] is the release role of the js toolchain. A requirement of a package on another
// package is a range of node-semver, such as ^1.2.0, and npm and pnpm install the newest version
// that it admits. So Resolve returns Selected for a version that the range admits, and Excluded
// otherwise, as for PyPI and Cargo. Resolve reads a range as node-semver 7.8.5 reads it in its
// strict mode: hyphen ranges, x-ranges, tildes, carets and the rule that a range admits a
// pre-release only for a comparator of the same version with a pre-release. Rewrite moves a range
// to the new version as changesets moves it.
//
// # Writes
//
// Apply writes the new version and each rewritten requirement of the package.json of its package
// in place, and keeps every other byte of the file. It leaves a shorthand of the workspace
// protocol, such as workspace:^, which pnpm replaces when it packs the package. It then refreshes
// pnpm-lock.yaml with pnpm install --lockfile-only, or package-lock.json with npm install
// --package-lock-only.
//
// # Pack and publish
//
// [Packer] packs the tarball of each package of a release with pnpm pack or npm pack into
// js/<name>/ of the pack directory, and checks that the tarball is the package at the version of
// the release. [Publisher] reads the npm registry for a release, and uploads the tarball of a
// package with npm publish of the section js. In a job of GitHub Actions with the permission
// id-token, npm publish exchanges the OIDC token of the job through the trusted publishing of npm.
//
// # Errors
//
// The errors wrap [ErrEdit] for an edit of a manifest that changed since its discovery,
// [ErrTarball] for a pack or a pack directory without the tarball of a release, and [ErrRegistry]
// for an unexpected status of the npm registry.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/version], [go.dokimi.dev/ergon/core/workspace] and
// [go.dokimi.dev/ergon/lang/javascript/workspace]. The package go.dokimi.dev/ergon/lang/javascript
// imports it.
package release
