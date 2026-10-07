// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/service/vcs"
)

// ReleaseBranch is the prefix of the branch of a version pull request, before the name of the base
// branch, as in ergon-release/main.
const ReleaseBranch = "ergon-release/"

// maxPayload is the most bytes of base64 of the files of one commit of a version pull request.
// GitHub ends a request after 10 seconds, and a commit of 5,000,000 bytes, which is 6,666,668 bytes
// of base64, took 3.4 s.
const maxPayload = 6_666_668

// maxBody is the most bytes of the body of a version pull request, as changesets/action bounds it.
const maxBody = 60_000

// The parts of the body of a version pull request.
const (
	// proposalIntro opens the body.
	proposalIntro = "This pull request was opened by `ergon release ci version`. Merging it releases the " +
		"packages below: the release workflow publishes each package to its registry and tags it. A changeset " +
		"that is merged into %s before this pull request updates it."

	// releasesHeading starts the list of the releases.
	releasesHeading = "# Releases"

	// omittedChangelogs replaces the changelogs of a body over maxBody.
	omittedChangelogs = "\n> The changelog of each package is left out, because the body would exceed the size " +
		"limit.\n"

	// omittedReleases replaces the releases of a body that is still over maxBody.
	omittedReleases = "\n> The releases are left out, because the body would exceed the size limit."

	// lockIntro opens the body of a version pull request that rewrites lockfiles alone.
	lockIntro = "This pull request was opened by `ergon release ci version`. A package of a release that " +
		"waits for its publish changed after its version commit, and the lockfiles below still record its earlier " +
		"content. Merging this pull request rewrites them, so that the release workflow publishes and tags the " +
		"content that they record. A changeset that is merged into %s before this pull request updates it."

	// lockfilesHeading starts the list of the lockfiles.
	lockfilesHeading = "# Lockfiles"
)

// Proposer is the host of a repository that a version pull request needs: its branch, signed
// commits on the branch, and the pull request.
type Proposer interface {
	// SetBranch points the branch name of repo at the commit sha, and creates a missing branch.
	SetBranch(ctx context.Context, repo, name, sha string) error

	// Commit commits files and the deletion of deleted on the branch of repo whose head is head,
	// with message, and returns the new commit.
	Commit(
		ctx context.Context, repo, branch, head, message string, files map[string][]byte, deleted []string,
	) (string, error)

	// PullRequest returns the number of the open pull request of repo from head into base, and
	// reports whether one is open.
	PullRequest(ctx context.Context, repo, head, base string) (int, bool, error)

	// CreatePullRequest opens a pull request of repo from head into base and returns its number.
	CreatePullRequest(ctx context.Context, repo, head, base, title, body string) (int, error)

	// UpdatePullRequest sets the title and the body of the pull request number of repo.
	UpdatePullRequest(ctx context.Context, repo string, number int, title, body string) error
}

// Proposal is a version pull request: the files that a version wrote, and where the pull request
// goes. [NewProposal] returns one.
type Proposal struct {
	// Files are the files that the version added or changed, by their paths relative to the root of
	// the repository and slash-separated, with their content.
	Files map[string][]byte

	// Repo is the repository on the host, as owner/name.
	Repo string

	// Base is the branch that the pull request merges into, such as main.
	Base string

	// Head is the commit of Base that the version ran on.
	Head string

	// Title is the title of the pull request and the message of its commits, such as chore: version
	// packages.
	Title string

	// Body is the description of the pull request in Markdown, as [NewProposal] writes it.
	Body string

	// Deleted are the files that the version removed, such as the changesets.
	Deleted []string
}

// NewProposal returns the version pull request of plan into the branch base, after [Version] wrote
// plan into the repository at root. Repo, Head and Title are empty, for the caller to set.
//
//   - Files are the files of the working tree that differ from HEAD, with their content, and Deleted
//     the files that the working tree removed, each relative to root and slash-separated, in the
//     order of their paths.
//   - Body is the body of changesets/action v2: an introduction, the heading # Releases, and for each
//     release above none a heading ## <name>@<version> with the section of the version in the
//     changelog of the package. The releases of public packages come first, each group from the
//     highest level to the lowest. A body over 60,000 bytes leaves out the changelogs, and then the
//     releases.
//   - A plan without releases is the work of [Lock], which rewrites lockfiles alone. Its Body states
//     why, under the heading # Lockfiles with a list of the changed files.
//
// It returns the error of git, which wraps [vcs.ErrGit], the error of reading a changed file, and
// the error of reading a changelog, other than the error of a changelog that does not exist.
func NewProposal(ctx context.Context, root, base string, g *Graph, plan *Plan) (Proposal, error) {
	files, deleted, err := changedFiles(ctx, root)
	if err != nil {
		return Proposal{}, err
	}
	body := lockBody(base, files)
	if len(plan.Releases) > 0 {
		if body, err = proposalBody(root, base, g, plan); err != nil {
			return Proposal{}, err
		}
	}
	return Proposal{Files: files, Base: base, Body: body, Deleted: deleted}, nil
}

// section is the part of the body of a version pull request about one release.
type section struct {
	// heading is the heading of the release, such as ## pkg-a@1.2.0.
	heading string

	// changelog is the section of the release in the changelog of the package.
	changelog string

	// highest is the highest level that the changelog names up to the end of the section.
	highest version.Bump
}

// Propose opens or updates the version pull request p through f, and returns its number. It points
// the branch ergon-release/<base> at p.Head, and commits the files of p on it, in commits of at
// most 6,666,668 bytes of base64 each, so that GitHub accepts each commit within 10 seconds: the
// files in the order of their paths, a file over the bound in a commit of its own, and the
// deletions in the first commit, each with p.Title as its message. It then updates the open pull
// request from the branch into p.Base with p.Title and p.Body, or opens one. It returns the error of
// f.
func Propose(ctx context.Context, f Proposer, p *Proposal) (int, error) {
	branch := ReleaseBranch + p.Base
	if err := f.SetBranch(ctx, p.Repo, branch, p.Head); err != nil {
		return 0, err
	}
	head, deleted := p.Head, p.Deleted
	for _, files := range batches(p.Files) {
		next, err := f.Commit(ctx, p.Repo, branch, head, p.Title, files, deleted)
		if err != nil {
			return 0, err
		}
		head, deleted = next, nil
	}
	if len(deleted) > 0 {
		if _, err := f.Commit(ctx, p.Repo, branch, head, p.Title, nil, deleted); err != nil {
			return 0, err
		}
	}
	number, open, err := f.PullRequest(ctx, p.Repo, branch, p.Base)
	switch {
	case err != nil:
		return 0, err
	case open:
		return number, f.UpdatePullRequest(ctx, p.Repo, number, p.Title, p.Body)
	}
	return f.CreatePullRequest(ctx, p.Repo, branch, p.Base, p.Title, p.Body)
}

// changedFiles returns the files of the working tree at root that differ from HEAD with their
// content, and the files that the working tree removed, as [NewProposal] states. It returns the
// error of git and the error of reading a file.
func changedFiles(ctx context.Context, root string) (map[string][]byte, []string, error) {
	changed, err := vcs.Changed(ctx, root, "HEAD")
	if err != nil {
		return nil, nil, err
	}
	files := map[string][]byte{}
	var deleted []string
	for _, name := range changed {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			deleted = append(deleted, name)
		case err != nil:
			return nil, nil, fmt.Errorf("release: read %s: %w", name, err)
		default:
			files[name] = data
		}
	}
	return files, deleted, nil
}

// batches returns files in batches of at most maxPayload bytes of base64 each, in the order of the
// paths, and a file over maxPayload in a batch of its own.
func batches(files map[string][]byte) []map[string][]byte {
	var out []map[string][]byte
	size := 0
	for _, name := range slices.Sorted(maps.Keys(files)) {
		encoded := base64.StdEncoding.EncodedLen(len(files[name]))
		if len(out) == 0 || size+encoded > maxPayload {
			out = append(out, map[string][]byte{})
			size = 0
		}
		out[len(out)-1][name] = files[name]
		size += encoded
	}
	return out
}

// proposalBody returns the body of the version pull request of plan into the branch base of the
// repository at root, as [NewProposal] states. It returns the error of reading a changelog, other
// than the error of a changelog that does not exist.
func proposalBody(root, base string, g *Graph, plan *Plan) (string, error) {
	var public, private []section
	for k := range plan.Releases {
		r := &plan.Releases[k]
		if r.Bump == version.BumpNone {
			continue
		}
		p := &g.pkgs[g.index[r.Name]]
		file := path.Join(p.Dir, ChangelogFile)
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("release: read %s: %w", file, err)
		}
		text, highest, _ := Section(data, r.New)
		s := section{heading: "## " + p.Name + "@" + r.New.String(), changelog: text, highest: highest}
		if p.Private {
			private = append(private, s)
		} else {
			public = append(public, s)
		}
	}
	head := fmt.Sprintf(proposalIntro, base) + "\n" + releasesHeading
	var full, short strings.Builder
	full.WriteString(head)
	short.WriteString(head + "\n" + omittedChangelogs)
	for _, s := range slices.Concat(byLevel(public), byLevel(private)) {
		full.WriteString("\n" + s.heading + "\n\n" + s.changelog)
		short.WriteString("\n" + s.heading + "\n\n")
	}
	for _, body := range []string{full.String(), short.String()} {
		if len(body) <= maxBody {
			return body, nil
		}
	}
	return head + "\n" + omittedReleases, nil
}

// lockBody returns the body of a version pull request into the branch base that rewrites the
// lockfiles of files alone, as [NewProposal] states.
func lockBody(base string, files map[string][]byte) string {
	lines := make([]string, 0, len(files)+3)
	lines = append(lines, fmt.Sprintf(lockIntro, base), lockfilesHeading, "")
	for _, name := range slices.Sorted(maps.Keys(files)) {
		lines = append(lines, "- `"+name+"`")
	}
	return strings.Join(lines, "\n")
}

// byLevel returns sections sorted from the highest level to the lowest, in their order within a
// level.
func byLevel(sections []section) []section {
	slices.SortStableFunc(sections, func(a, b section) int {
		switch {
		case a.highest == b.highest:
			return 0
		case a.highest.AtLeast(b.highest):
			return -1
		}
		return 1
	})
	return sections
}
