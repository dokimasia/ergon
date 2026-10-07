// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"

	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/vcs"
)

// filePerm is the mode of a new changelog, before the umask.
const filePerm fs.FileMode = 0o644

// ErrNoChangesets is the error of [Version] for a plan without changesets, for which changeset
// version reports "No unreleased changesets found.".
var ErrNoChangesets = errors.New("release: no unreleased changesets")

// toolchainEdits are the edits of the packages of one toolchain, which its versioner applies.
type toolchainEdits struct {
	// versioner applies the edits.
	versioner language.Versioner

	// edits are the edits, in the order of the plan.
	edits []language.Edit
}

// Version writes plan into the repository at root, as changeset version writes a release plan, and
// returns the paths that it changed, relative to root and slash-separated, in the order of the
// writes:
//
//   - The CHANGELOG.md of each package with an entry of [Entries], with the entry before the first
//     heading of a version, under a new heading # and the Name of the package, or after the first
//     line of a changelog without such a heading.
//   - The removal of each changeset of the plan that does not name a skipped package.
//   - The manifests and lockfiles that the versioner of each toolchain writes for the edits of its
//     packages: the new version of each package of the plan, and the requirements that the release
//     rewrites.
//
// A release rewrites a requirement on a package that the plan releases above none when the
// requirement excludes or pins the new version, or when it selects the new version and the level
// of the release is at least updateInternalDependencies, except a peer requirement under
// onlyUpdatePeerDependentsWhenOutOfRange. It rewrites a requirement on any other package when the
// requirement pins the package's current version, so that a released Go module requires the
// versions that it was tested with. A requirement that its versioner does not read keeps its text.
//
// The changelog of each changeset links the commit that added its file. Version writes the
// changelogs and removes the changesets through an [os.Root] of root, so a path that leaves the
// repository fails. It takes a snapshot of the working tree before its first write, and restores
// every path that it changed when a later step fails, including the paths that a versioner wrote
// before its error.
//
// It returns [ErrNoChangesets] for a plan without changesets. It returns the error of git, which
// wraps [vcs.ErrGit], the error of [Entries], the error of a versioner that cannot rewrite a
// requirement, with the package and the requirement, the error of opening root, and the error of a
// write or of a versioner, joined with the error of the restore.
func Version(ctx context.Context, root string, g *Graph, c *Config, plan *Plan, h Host) ([]string, error) {
	if len(plan.Changesets) == 0 {
		return nil, ErrNoChangesets
	}
	commits := map[string]string{}
	if c.Changelog.Format != "" {
		for _, s := range plan.Changesets {
			commit, err := vcs.AddedBy(ctx, root, changesetPath(s.ID))
			if err != nil {
				return nil, err
			}
			commits[s.ID] = commit
		}
	}
	entries, err := Entries(ctx, g, c, plan, commits, h)
	if err != nil {
		return nil, err
	}
	groups, err := edits(g, c, plan)
	if err != nil {
		return nil, err
	}
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("release: open the repository: %w", err)
	}
	defer func() { _ = dir.Close() }()
	tree, err := vcs.Snapshot(ctx, root)
	if err != nil {
		return nil, err
	}
	written, err := write(ctx, dir, g, c, plan, entries, groups)
	if err != nil {
		return nil, errors.Join(err, vcs.Restore(context.WithoutCancel(ctx), root, tree, written))
	}
	return written, nil
}

// write writes the entries into the changelogs, removes the changesets of plan that do not name a
// skipped package, and applies the edits of each toolchain, in that order, in the repository at
// dir. It returns the paths that it changed, each listed before its change, and on an error the
// paths that it changed or began to change before the error.
func write(
	ctx context.Context, dir *os.Root, g *Graph, c *Config, plan *Plan, entries []Entry, groups []toolchainEdits,
) ([]string, error) {
	var written []string
	for _, e := range entries {
		p := &g.pkgs[g.index[e.Name]]
		file := path.Join(p.Dir, ChangelogFile)
		data, err := dir.ReadFile(filepath.FromSlash(file))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return written, fmt.Errorf("release: read %s: %w", file, err)
		}
		written = append(written, file)
		if err := dir.WriteFile(filepath.FromSlash(file), updateChangelog(data, p.Name, e.Text), filePerm); err != nil {
			return written, fmt.Errorf("release: write %s: %w", file, err)
		}
	}
	for _, s := range plan.Changesets {
		skipped := slices.ContainsFunc(s.Releases, func(r changeset.Release) bool {
			return c.skips(r.Name, &g.pkgs[g.index[r.Name]])
		})
		if skipped {
			continue
		}
		file := changesetPath(s.ID)
		if err := dir.Remove(filepath.FromSlash(file)); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return written, fmt.Errorf("release: remove %s: %w", file, err)
		}
		written = append(written, file)
	}
	for _, group := range groups {
		paths, err := group.versioner.Apply(ctx, dir.Name(), group.edits)
		written = append(written, paths...)
		if err != nil {
			return written, fmt.Errorf("release: apply the release: %w", err)
		}
	}
	return written, nil
}

// edits returns the edits of the packages of plan whose version or requirements change, grouped by
// the toolchain of the package in the order of the plan.
func edits(g *Graph, c *Config, plan *Plan) ([]toolchainEdits, error) {
	releases := map[int]*Release{}
	for k := range plan.Releases {
		releases[g.index[plan.Releases[k].Name]] = &plan.Releases[k]
	}
	var groups []toolchainEdits
	at := map[*roles]int{}
	for k := range plan.Releases {
		r := &plan.Releases[k]
		i := g.index[r.Name]
		reqs, err := rewrittenRequirements(g, c, releases, i)
		if err != nil {
			return nil, err
		}
		if r.New == g.pkgs[i].Version && len(reqs) == 0 {
			continue
		}
		slot, ok := at[g.roles[i]]
		if !ok {
			slot = len(groups)
			at[g.roles[i]] = slot
			groups = append(groups, toolchainEdits{versioner: g.roles[i].versioner})
		}
		groups[slot].edits = append(groups[slot].edits, language.Edit{
			Version: r.New, Requirements: reqs, Package: g.pkgs[i],
		})
	}
	return groups, nil
}

// rewrittenRequirements returns the requirements of the package at index i that a release with the
// releases of each package by index rewrites, each with its new Req, in the order of its Deps. It
// returns the error of a versioner that cannot rewrite a requirement.
func rewrittenRequirements(g *Graph, c *Config, releases map[int]*Release, i int) ([]workspace.Dependency, error) {
	v := g.roles[i].versioner
	var out []workspace.Dependency
	for _, d := range g.pkgs[i].Deps {
		j, ok := g.sibling(i, d.Name)
		if !ok {
			continue
		}
		var target version.Version
		if rel := releases[j]; rel != nil && rel.Bump != version.BumpNone {
			resolution, err := v.Resolve(d.Req, rel.New)
			if err != nil || !c.rewrites(resolution, d.Kind, rel.Bump) {
				continue
			}
			target = rel.New
		} else {
			resolution, err := v.Resolve(d.Req, g.pkgs[j].Version)
			if err != nil || resolution != language.ResolutionPinned {
				continue
			}
			target = g.pkgs[j].Version
		}
		req, err := v.Rewrite(d.Req, target)
		if err != nil {
			return nil, fmt.Errorf("release: rewrite the requirement %s of %s on %s: %w", d.Req, g.names[i], d.Name,
				err)
		}
		if req != d.Req {
			out = append(out, workspace.Dependency{Name: d.Name, Kind: d.Kind, Req: req})
		}
	}
	return out, nil
}

// changesetPath returns the path of the file of the changeset id, relative to the root of the
// repository and slash-separated.
func changesetPath(id string) string {
	return path.Join(changeset.Dir, id+changeset.Ext)
}
