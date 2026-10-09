// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/vcs"
)

// ErrTag is the error for a tag of a release that the repository has at another commit.
var ErrTag = errors.New("release: tag at another commit")

// ErrAssets is the error of a releaser that cannot attach assets to a release, such as
// [GitReleaser].
var ErrAssets = errors.New("release: the releaser cannot attach assets")

// ErrPublishPlan is the error for a publish plan that does not fit the packages of the repository: an
// entry of a package that the repository does not have, and an entry of [KindPublish] of a package
// whose toolchain has no publisher.
var ErrPublishPlan = errors.New("release: invalid publish plan")

// Releaser records the releases of a publish: it reads the tags of the repository, and creates the
// tag of a release with its notes and its assets.
type Releaser interface {
	// Tag returns the commit of the tag name, and reports whether the repository has the tag.
	Tag(ctx context.Context, name string) (string, bool, error)

	// Release creates the tag name at commit, with notes, the section of the changelog of the
	// release, as its annotation or as the notes of a release of the host, marked as a pre-release
	// for prerelease, and with the files of assets as the assets of the release. It completes a
	// release that an earlier publish left: it creates what the host lacks of the tag, the release
	// and the assets, and publishes a draft.
	Release(ctx context.Context, name, commit, notes string, prerelease bool, assets []string) error

	// Finish completes the releases that Release created, such as by pushing their tags.
	Finish(ctx context.Context) error
}

// Published is a package that a publish released, as changesets/action lists it in the output
// publishedPackages.
type Published struct {
	// Name is the name of the package that its registry knows.
	Name string `json:"name"`

	// Version is the version of the release.
	Version string `json:"version"`
}

// Publish publishes plan from the repository at root at the commit head, chunk by chunk, and
// returns the packages that it released, in the order of the plan:
//
//   - It uploads the packages of the entries of [KindPublish] of a chunk with the publisher of their
//     toolchain, from the artifacts in dir, in the order of the chunk. It leaves out a package
//     whose registry already has its version, as the publisher reports, so a publish that runs again
//     continues where an earlier one stopped.
//   - It then creates the tag of each entry of the chunk whose tag r does not have, at head, with the
//     section of the version in the changelog of the package as its notes, and with the files of
//     dir/assets/<tag>/ as the assets of its release, in the order of their names.
//   - It calls r.Release again for an entry whose tag r has at head and whose release has assets in
//     dir, so that a publish that runs again completes a release that an earlier one left.
//
// A chunk starts after the tags of the chunk before it. Publish calls r.Finish once, after the last
// tag or after an error.
//
// Before it publishes anything, it returns an error that wraps [ErrPublishPlan] for a plan that does
// not fit g, and an error that wraps [ErrStale] for lockfiles that record other content of the
// packages of plan than the working tree, as [Stale] reports them, with their paths. It returns an
// error that wraps [ErrTag] for a tag that r has at another commit than head, with both commits. It
// returns the error of Stale, of a publisher, of r and of reading a changelog or the assets of a
// release. On an error it also returns the packages that it released before the error.
func Publish(ctx context.Context, root string, g *Graph, plan *PublishPlan, head, dir string, r Releaser) (
	[]Published, error,
) {
	stale, err := Stale(ctx, root, g, plan)
	if err != nil {
		return nil, err
	}
	if len(stale) > 0 {
		return nil, fmt.Errorf("%w: %s, which ergon release version rewrites", ErrStale, strings.Join(stale, ", "))
	}
	var released []Published
	err = publish(ctx, root, g, plan, head, dir, r, &released)
	return released, errors.Join(err, r.Finish(ctx))
}

// publish publishes plan as [Publish] states, and appends each package that it releases to
// released.
func publish(
	ctx context.Context, root string, g *Graph, plan *PublishPlan, head, dir string, r Releaser, released *[]Published,
) error {
	for _, chunk := range plan.Plan {
		if err := upload(ctx, g, chunk, dir); err != nil {
			return err
		}
		for k := range chunk {
			e := &chunk[k]
			commit, found, err := r.Tag(ctx, e.Tag)
			if err != nil {
				return err
			}
			if found && commit != head {
				return fmt.Errorf(
					"%w: %s is at %s, and the release of %s is at %s",
					ErrTag,
					e.Tag,
					commit,
					e.Name,
					head,
				)
			}
			p := &g.pkgs[g.index[e.Name]]
			assets, err := assetsOf(dir, e.Tag)
			if err != nil {
				return err
			}
			if !found || len(assets) > 0 {
				notes, err := notesOf(root, p, e)
				if err != nil {
					return err
				}
				if err := r.Release(ctx, e.Tag, head, notes, e.Version.Pre != "", assets); err != nil {
					return err
				}
			}
			*released = append(*released, Published{Name: p.Name, Version: e.Version.String()})
		}
	}
	return nil
}

// upload uploads the packages of the entries of [KindPublish] of chunk whose registry does not have
// their version, with the publisher of each toolchain, from the artifacts in dir. It returns the
// error of a publisher, with the name of the package.
func upload(ctx context.Context, g *Graph, chunk []PublishEntry, dir string) error {
	var order []*roles
	pending := map[*roles][]workspace.Package{}
	for k := range chunk {
		e := &chunk[k]
		if e.Kind != KindPublish {
			continue
		}
		i := g.index[e.Name]
		publisher := g.roles[i].publisher
		published, err := publisher.Published(ctx, &g.pkgs[i])
		if err != nil {
			return fmt.Errorf("release: the registry of %s: %w", e.Name, err)
		}
		if published {
			continue
		}
		if _, ok := pending[g.roles[i]]; !ok {
			order = append(order, g.roles[i])
		}
		pending[g.roles[i]] = append(pending[g.roles[i]], g.pkgs[i])
	}
	for _, r := range order {
		if err := r.publisher.Publish(ctx, dir, pending[r]); err != nil {
			return fmt.Errorf("release: publish: %w", err)
		}
	}
	return nil
}

// notesOf returns the section of the version of e in the changelog of p in the repository at root,
// and the empty string for a changelog without it. It returns the error of reading the changelog,
// other than the error of a changelog that does not exist.
func notesOf(root string, p *workspace.Package, e *PublishEntry) (string, error) {
	file := path.Join(p.Dir, ChangelogFile)
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("release: read %s: %w", file, err)
	}
	notes, _, _ := Section(data, e.Version)
	return notes, nil
}

// assetsOf returns the paths of the files of dir/assets/<tag>/, the assets of the release of tag
// that a packer built, in the order of their names. It returns no asset for an empty dir and for a
// tag without the directory, and the error of reading the directory.
func assetsOf(dir, tag string) ([]string, error) {
	if dir == "" {
		return nil, nil
	}
	assetsDir := filepath.Join(dir, language.AssetsDir, filepath.FromSlash(tag))
	entries, err := os.ReadDir(assetsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("release: read the assets of %s: %w", tag, err)
	}
	assets := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			assets = append(assets, filepath.Join(assetsDir, entry.Name()))
		}
	}
	return assets, nil
}

// GitReleaser releases through the git of a repository: annotated tags with the notes as their
// annotation, which git signs when the configuration sets tag.gpgSign, pushed in one atomic push.
// A tag of git has no assets, so a GitReleaser refuses a release with assets.
//
// # Concurrency
//
// A GitReleaser is not safe for concurrent use.
type GitReleaser struct {
	// tags are the tags of the repository, which Tag reads once.
	tags map[string]string

	// Terminal is the terminal of git tag and git push, on which the signing program of a tag and
	// the ssh of the remote prompt for the PIN and the touch of a key. The zero value is none.
	Terminal vcs.Terminal

	// Root is the root of the repository.
	Root string

	// Remote is the remote that Finish pushes the tags to, such as origin.
	Remote string

	// created are the refs of the tags that Release created, in their order.
	created []string
}

// Tag returns the commit of the tag name of the repository, and reports whether it has the tag. It
// reads the tags once, and returns the error of git, which wraps [vcs.ErrGit].
func (r *GitReleaser) Tag(ctx context.Context, name string) (string, bool, error) {
	if r.tags == nil {
		tags, err := vcs.Tags(ctx, r.Root)
		if err != nil {
			return "", false, err
		}
		r.tags = tags
	}
	commit, ok := r.tags[name]
	return commit, ok, nil
}

// Release creates the annotated tag name at commit, with notes as its annotation, or the name for
// empty notes, with the terminal of r. It returns an error that wraps [ErrAssets] for a release
// with assets, before it creates the tag, and the error of git, which wraps [vcs.ErrGit].
func (r *GitReleaser) Release(ctx context.Context, name, commit, notes string, _ bool, assets []string) error {
	if len(assets) > 0 {
		return fmt.Errorf("%w: the release %s has %d assets, which a publish through release.yml attaches",
			ErrAssets, name, len(assets))
	}
	if err := vcs.Tag(ctx, r.Root, r.Terminal, name, commit, cmp.Or(notes, name)); err != nil {
		return err
	}
	r.created = append(r.created, "refs/tags/"+name)
	return nil
}

// Finish pushes the tags that Release created to the remote in one atomic push, with the terminal
// of r, and pushes nothing when Release created none. It returns the error of git, which wraps
// [vcs.ErrGit].
func (r *GitReleaser) Finish(ctx context.Context) error {
	if len(r.created) == 0 {
		return nil
	}
	return vcs.Push(ctx, r.Root, r.Terminal, r.Remote, r.created...)
}

// TagForge is the host of a repository that a publish in CI tags and releases through.
// *forge.Client implements it.
type TagForge interface {
	// Tag returns the commit of the tag name of repo, and reports whether repo has the tag.
	Tag(ctx context.Context, repo, name string) (string, bool, error)

	// CreateTag creates the lightweight tag name of repo at the commit sha.
	CreateTag(ctx context.Context, repo, name, sha string) error

	// ReleaseOf returns the release of the tag of repo, a draft included, and reports whether repo
	// has one.
	ReleaseOf(ctx context.Context, repo, tag string) (forge.Release, bool, error)

	// CreateDraft creates the release of the tag of repo as a draft with the Markdown body, and
	// returns it.
	CreateDraft(ctx context.Context, repo, tag, body string, prerelease bool) (forge.Release, error)

	// UploadAsset uploads the file at path as an asset of the release r.
	UploadAsset(ctx context.Context, r *forge.Release, path string) error

	// Publish publishes the draft release id of repo.
	Publish(ctx context.Context, repo string, id int64) error
}

// ForgeReleaser releases through the host of a repository, as changesets/action releases a package:
// a lightweight tag, and a release of the host with the notes as its body. It creates a release as
// a draft, uploads its assets, and then publishes it, the order that a repository with immutable
// releases requires.
type ForgeReleaser struct {
	// Forge is the host.
	Forge TagForge

	// Repo is the repository on the host, as owner/name.
	Repo string
}

// Tag returns the commit of the tag name of the repository on the host, and reports whether it has
// the tag. It returns the error of the host.
func (r ForgeReleaser) Tag(ctx context.Context, name string) (string, bool, error) {
	return r.Forge.Tag(ctx, r.Repo, name)
}

// Release releases the tag name at commit on the host in four steps, each of which it leaves out
// when an earlier publish took it: it creates the tag, creates its release as a draft with notes as
// its body, uploads each file of assets that the release lacks by its base name, and publishes the
// draft. It returns the error of the host.
func (r ForgeReleaser) Release(
	ctx context.Context, name, commit, notes string, prerelease bool, assets []string,
) error {
	_, tagged, err := r.Forge.Tag(ctx, r.Repo, name)
	if err != nil {
		return err
	}
	if !tagged {
		err = r.Forge.CreateTag(ctx, r.Repo, name, commit)
		if err != nil {
			return err
		}
	}
	rel, found, err := r.Forge.ReleaseOf(ctx, r.Repo, name)
	if err != nil {
		return err
	}
	if !found {
		if rel, err = r.Forge.CreateDraft(ctx, r.Repo, name, notes, prerelease); err != nil {
			return err
		}
	}
	for _, path := range assets {
		base := filepath.Base(path)
		if slices.ContainsFunc(rel.Assets, func(a forge.Asset) bool { return a.Name == base }) {
			continue
		}
		if err := r.Forge.UploadAsset(ctx, &rel, path); err != nil {
			return err
		}
	}
	if !rel.Draft {
		return nil
	}
	return r.Forge.Publish(ctx, r.Repo, rel.ID)
}

// Finish returns nil: the host has each tag and release once Release returns.
func (ForgeReleaser) Finish(context.Context) error {
	return nil
}
