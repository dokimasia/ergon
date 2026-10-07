// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import "context"

// Forge is the host of a repository on GitHub, which the changelog of GitHub reads the links of
// commits and pull requests from. Every method takes the repository as owner/name and returns the
// error of the host. The package go.dokimi.dev/ergon/service/forge implements it without importing
// this package, because its methods take and return only types of the standard library.
type Forge interface {
	// CommitLinks returns the Markdown links of the commit sha, of the pull request that merged it
	// first, and of the author of that pull request or of the commit, as @changesets/get-github-info
	// renders them: [`abc1234`](url), [#12](url) and [@login](url). It returns an empty link for a
	// commit, a pull request or an author that the host does not have.
	CommitLinks(ctx context.Context, repo, sha string) (commit, pull, author string, err error)

	// PullLinks returns the Markdown links of the pull request number, of its merge commit and of
	// its author, in the form of CommitLinks. It returns an empty link for a pull request, a merge
	// commit or an author that the host does not have.
	PullLinks(ctx context.Context, repo string, number int) (pull, commit, author string, err error)
}

// Host is the repository on GitHub that a release links to and writes to.
type Host struct {
	// Forge reads the links of the changelog of GitHub. It is nil for a release without the
	// changelog of GitHub.
	Forge Forge

	// Repo is the repository of the workflow as owner/name, from GITHUB_REPOSITORY, which the
	// changelog of GitHub links to when its options name no repository.
	Repo string

	// Server is the address of GitHub, from GITHUB_SERVER_URL, or empty for https://github.com.
	Server string
}
