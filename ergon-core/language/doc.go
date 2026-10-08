// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package language declares the toolchains and the languages of ergon, the catalog that registers
// them, and the roles that they implement.
//
// A [Toolchain] builds packages, such as the go command or Gradle. A [Declaration] states a
// language and the toolchain that builds it: Java and Kotlin name the toolchain jvm, and Go names
// the toolchain go. A composition root registers each toolchain with [RegisterToolchain] and each
// language with [Register] into one [Catalog], before any command reads the catalog.
//
// # Roles
//
// A toolchain and a language register their roles, and a command selects one with [Role] or
// [ToolchainRole]. The roles of ergon init divide the files of a repository by concern:
//
//   - A [Producer] returns the templates of its concern.
//   - A [Calculator] computes values that its templates read, from the answers, its options and the
//     contributions.
//   - A [Configurable] producer returns its [Options]: a struct of its own, which composes the
//     types of go.dokimi.dev/ergon/core/option and states its section of .ergon.yaml, [Config].
//   - A [Contributor] returns its part of the workflows as a
//     [go.dokimi.dev/ergon/core/workflow.Contribution], which the producer of the GitHub files
//     renders.
//   - A [Placer] returns each [File] whose path its options state, such as the LICENSE of a
//     directory, which no template tree can mirror.
//   - A [LocalChecker] refuses the local file of one of its managed files, such as a local file of
//     the configuration of Dependabot that would let Dependabot edit another managed file.
//
// A producer configures its own concern alone: a section names the tools, the actions and the
// runtime versions of its own producer, and never those of another ecosystem.
//
// The roles of ergon release belong to a toolchain, which discovers the packages through
// [Toolchain.Discover]:
//
//   - A [Versioner] reads and rewrites the requirements of the toolchain, refuses the versions that
//     the toolchain cannot release, and writes each [Edit] of a release into the manifests and the
//     lockfiles.
//   - A [Tagger] names the tags of a toolchain whose tools read a version from a tag.
//   - A [Packer] builds the artifacts that a registry receives.
//   - A [Publisher] uploads them. A toolchain without one publishes a release by its tag.
//   - A [Locker] reports and rewrites the lockfiles that record the content of a package whose
//     release waits for its publish, such as a go.sum after a change to a released module.
//
// # Answers
//
// [Answers] are the answers of ergon init, which every producer reads. [Answers.Validate] checks
// them once, before a producer renders, so no producer checks an answer again.
//
// # Errors
//
// RegisterToolchain and Register leave the catalog unchanged when they return an error. Each
// error wraps one of [ErrInvalidName], [ErrRegistered], [ErrUnknownToolchain] and [ErrUnknownRole],
// and quotes the name that caused it. Answers.Validate returns an error that wraps
// [ErrInvalidAnswer], and a LocalChecker an error that wraps [ErrInvalidLocal].
//
// # Concurrency
//
// Registration writes the catalog and is not safe for concurrent use. The methods of a Catalog
// only read it, so they are safe for concurrent use once the last registration has returned.
//
// # Dependency position
//
// Position 1 of ergon-core. Imports the standard library, [go.dokimi.dev/ergon/core/spdx],
// [go.dokimi.dev/ergon/core/version], [go.dokimi.dev/ergon/core/workflow] and
// [go.dokimi.dev/ergon/core/workspace].
package language
