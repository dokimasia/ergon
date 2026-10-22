// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package release releases the projects of a uv workspace, which
// [go.dokimi.dev/ergon/lang/python/workspace] discovers, to PyPI.
//
// [Versioner] is the release role of the toolchain python. A requirement of a project on another
// project is a version specifier of PEP 440, such as >=1.2.0, and pip and uv select the newest
// version that it admits. So Resolve returns Selected for a version that the specifier admits, and
// Excluded otherwise, as for npm and Cargo. Rewrite moves each clause of a specifier to
// the new version with its operator, and writes >=X.Y.Z for a requirement without a specifier,
// because uv build publishes a requirement on a member of the workspace as the requirement writes
// it.
//
// # Writes
//
// Apply writes the new version into project.version and each rewritten requirement into its string
// of the pyproject.toml of its project, in place, and keeps every other byte of the file. It then
// runs uv lock through ergon tool run, which records the new versions in uv.lock. A repository
// without uv.lock gets none.
//
// # Pack and publish
//
// [Packer] builds the sdist and the wheel of each project of a release with uv build into
// python/<name>/ of the pack directory, and checks that each one is a distribution of the project at
// the version of the release. [Publisher] reads the JSON API of PyPI for a release, and uploads the
// distributions of a project with uv publish. In a job of GitHub Actions with the permission
// id-token, uv publish exchanges the OIDC token of the job through the trusted publishing of PyPI.
//
// # Errors
//
// The errors wrap [ErrSpecifier] for a version specifier, [ErrEdit] for an edit of a manifest that
// changed since its discovery, [ErrDistribution] for a build or a pack directory without the
// distributions of a release, and [ErrIndex] for an unexpected status of PyPI.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/version], [go.dokimi.dev/ergon/core/workspace] and
// [go.dokimi.dev/ergon/lang/python/workspace]. The package go.dokimi.dev/ergon/lang/python imports
// it.
package release
