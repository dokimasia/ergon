// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package buildinfo returns the version of a build of ergon.
//
// A build of a release sets three variables with -X flags of the linker. [Full] writes them with
// [Format], and [Version] returns the version of the release alone:
//
//	go.dokimi.dev/ergon/internal/buildinfo.version  the version of the release
//	go.dokimi.dev/ergon/internal/buildinfo.commit   the commit of the release
//	go.dokimi.dev/ergon/internal/buildinfo.date     the date of the commit
//
// A build without the flags has the version of its main module when go install built a release of
// it, as [Release] resolves it, and the version dev otherwise.
//
// # Dependency position
//
// Imports the standard library, golang.org/x/mod/module and golang.org/x/mod/semver. cmd/ergon
// imports it.
package buildinfo
