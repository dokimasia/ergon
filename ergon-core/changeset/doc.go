// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package changeset reads and writes the changeset files of .changeset, in the format of
// changesets.
//
// A changeset file states the packages that a change releases, each with the level of its
// release, in its front matter, and the summary of the change in its body:
//
//	---
//	"go.dokimi.dev/ergon/core": minor
//	"dokimi-assert": patch
//	---
//
//	Add the Unit type to the vocabulary.
//
// [Parse] reads the subset of YAML that changesets writes: one name and one level per line of the
// front matter, the name in double quotes, in single quotes or unquoted, and the level one of
// none, patch, minor and major. [Format] writes a [Changeset] as changesets writes one, with every
// name in double quotes. A name that two toolchains share is spelled <toolchain>:<name>, and the
// planner of a release resolves it.
//
// [Dir] is the directory of the changesets, and [Changelog] the name of the changelog of a package,
// into which a release writes the summaries of its changesets.
//
// # Errors
//
// Parse returns an error that wraps [ErrInvalid] and names the file and the line.
//
// # Dependency position
//
// Position 0 of ergon-core. Imports the standard library and [go.dokimi.dev/ergon/core/version].
package changeset
