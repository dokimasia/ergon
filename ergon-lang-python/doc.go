// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package python declares Python and its toolchain, uv.
//
// [Register] adds the toolchain and the language to a catalog. The toolchain discovers the projects
// of a uv workspace as the packages of a release, and writes their versions, their requirements on
// each other and uv.lock in a release. A release builds the distributions of each project and
// uploads them to PyPI. It runs uv through the [Tools] that the composition root provides, and reads
// PyPI through the client that the composition root provides.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/workspace] and its packages baseline, release and workspace. Only the
// root module of ergon imports it.
package python
