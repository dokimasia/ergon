// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package workspace discovers the Python projects of a uv workspace as the packages of a release.
//
// [Discover] reads the pyproject.toml at the root of a repository and the pyproject.toml of each
// member of its tool.uv.workspace, as uv reads them: the globs of members, less the globs of
// exclude, and the directories among their matches that have a pyproject.toml. Each project is a
// package of [Toolchain]. Its requirements on the other projects are the requirements of the
// package: project.dependencies as runtime requirements, project.optional-dependencies as optional
// requirements, and the dependency groups and tool.uv.dev-dependencies as dev requirements.
//
// # Names and versions
//
// A package has the name of its project in the normal form of PEP 503, which [Normalize] returns,
// as uv.lock and PyPI name it. Its version is the version of its project in Semantic Versioning
// 2.0.0, which [Version.SemVer] converts from PEP 440. Semantic Versioning states a release of up
// to three numbers and a pre-release. A version with an epoch, a post-release, a development
// release or a local label is an error. [FromSemVer] converts a version of a release back.
//
// # Manifests
//
// [ParseManifest] reads the keys of a pyproject.toml that a release reads and writes, with the
// range of bytes of each string in the file, so a release rewrites a version or a requirement in
// place and keeps every other byte. [ParseRequirement] reads a requirement of PEP 508.
//
// # Errors
//
// The errors wrap [ErrVersion] for a version, [ErrRequirement] for a requirement, and [ErrManifest]
// for a pyproject.toml that a release cannot read.
//
// # Dependency position
//
// Imports the standard library, github.com/bmatcuk/doublestar/v4 for the globs of a workspace,
// github.com/pelletier/go-toml/v2 for the ranges of the values of a pyproject.toml,
// [go.dokimi.dev/ergon/core/changeset], [go.dokimi.dev/ergon/core/version] and
// [go.dokimi.dev/ergon/core/workspace]. The package go.dokimi.dev/ergon/lang/python and its
// package release import it.
package workspace
