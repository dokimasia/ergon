// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package release plans and makes the releases of the packages of a repository, by the rules of
// changesets 3.0.3, for every toolchain that registers the release roles of
// [go.dokimi.dev/ergon/core/language].
//
// # Packages
//
// [Discover] runs the discovery of every toolchain of a catalog, and [NewGraph] builds the
// [Graph] of the packages: the name of each in a changeset, the release roles and the facts of its
// toolchain, and the requirements between the packages. A package requires another of its
// toolchain when the versioner of the toolchain resolves the requirement as pinning or selecting
// the current version of the other, as changesets counts a dependent through a range that admits
// the current version.
//
// # Configuration and changesets
//
// [ParseConfig] reads .changeset/config.json into a [Config], with the defaults and the rules of
// changesets 4.0.1. ergon reads two changelog formats and refuses every other module, because it
// does not run JavaScript. [ReadChangesets] reads the changesets of .changeset and skips the files
// that changesets skips. [ChangesetID] names a new changeset after its summary, and
// [AddChangeset] writes it.
//
// # Planning
//
// [NewPlan] is a pure function of the graph, the configuration and the changesets. It returns the
// [Plan] of the packages that the changesets name, of their dependents, and of their fixed and
// linked groups. A dependent is released when a consumer that installs it would not receive the
// new version of its dependency. The dependent's requirement then excludes the new version, as an
// npm range excludes a version outside it, or pins the version that it names, as a require line of
// Go does under minimal version selection. For npm packages the plan equals the plan of changesets
// on the same files. A Go module that requires a released module is released too, because its
// require line pins an older version.
//
// [NewStatus] reports the plan of the changesets that a branch added, the packages that the branch
// changed without a changeset, and the requirements that exclude the current version of a package.
//
// # Changelogs
//
// [Entries] renders the changelog entry of each release of a plan, as @changesets/cli/changelog
// or @changesets/changelog-github 1.0.1 renders it. The changelog of GitHub reads the links of
// commits and pull requests from the [Forge] of a [Host]. [Section] returns the section of one
// version of a changelog, the body of a release and of a version pull request.
//
// # Versioning
//
// [Version] writes a plan into the repository, as changeset version writes a release plan: the
// entries into the changelogs, the removal of the changesets, and the edits that the versioner of
// each toolchain applies to the manifests of its packages. It restores every path that it changed
// when a step fails. [NewProposal] collects the files that a version wrote into a version pull
// request, and [Propose] opens or updates the pull request through a [Proposer].
//
// # Publishing
//
// [NewPublishPlan] returns the [PublishPlan] of the packages that a publish releases: each package
// whose registry lacks its version, and each package without a registry whose tag is missing, in
// chunks of dependency order. [Pack] builds the artifacts of the plan, and [Publish] uploads the
// packages and tags them through a [Releaser]: a [GitReleaser] on a workstation, and a
// [ForgeReleaser] in CI. [SelectMode] chooses the job of the release workflow from the changesets
// and the publish plan.
//
// # Errors
//
// Each error of the package wraps one of [ErrPackages], [ErrConfig], [ErrChangeset],
// [ErrChangelog], [ErrPublishPlan] and [ErrTag], and names the package, the key, the file or the tag
// that caused it, or is [ErrNoChangesets]. The package also returns the error of git, which wraps
// [go.dokimi.dev/ergon/service/vcs.ErrGit], the error of the file system with its path, and the
// error of a toolchain role, of a [Forge], a [Proposer], a [Releaser] or of
// [go.dokimi.dev/ergon/core/version].
//
// # Concurrency
//
// A [Graph], a [Config], a [Plan] and a [PublishPlan] are values that no function of the package
// modifies after it returns them, so they are safe for concurrent use. [Version], [AddChangeset] and
// [Publish] write the working tree or the tags of a repository, so two calls must not write one
// repository at once.
//
// # Dependency position
//
// Imports the standard library, github.com/bmatcuk/doublestar/v4,
// [go.dokimi.dev/ergon/core/changeset], [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/version], [go.dokimi.dev/ergon/core/workspace] and
// [go.dokimi.dev/ergon/service/vcs]. internal/cli of the root module and
// [go.dokimi.dev/ergon/service/baseline/common] import it.
package release
