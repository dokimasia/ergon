// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/vcs"
)

// ErrStale is the error of [Publish] for a publish plan whose packages have lockfiles that record
// other content than the working tree, as [Stale] reports them.
var ErrStale = errors.New("release: stale lockfiles")

// toolchainPackages are the packages of a publish plan of one toolchain, and the locker of the
// toolchain.
type toolchainPackages struct {
	// locker reports and rewrites the lockfiles of the toolchain.
	locker language.Locker

	// name is the name of the toolchain.
	name workspace.Toolchain

	// pkgs are the packages, each at the version of its entry, in the order of the plan.
	pkgs []workspace.Package
}

// Stale returns the lockfiles of the repository at root that record, for a package of plan at the
// version of its entry, other content than the package has in the working tree, as the
// [language.Locker] of the toolchain of each package reports them. The version commit of a release
// records that content, and a change to a package of plan after the commit makes the record stale,
// so a publish would tag content that the lockfiles do not record. Stale skips a package of a
// toolchain without a locker. The paths are relative to root, slash-separated and sorted.
//
// It returns an error that wraps [ErrPublishPlan] for a plan that does not fit g, as [Publish]
// states it, and the error of a locker, with the name of its toolchain.
func Stale(ctx context.Context, root string, g *Graph, plan *PublishPlan) ([]string, error) {
	_, stale, err := staleLockers(ctx, root, g, plan)
	return stale, err
}

// Lock rewrites the lockfiles of the repository at root that [Stale] reports for plan, with the
// [language.Locker] of each toolchain that reports one, so that they record the content of the
// working tree. It returns the paths that it changed, relative to root and slash-separated, in the
// order of the toolchains in plan. It changes nothing when no lockfile is stale. It takes a snapshot
// of the working tree before its first write, and restores every path that it changed when a
// locker fails, including the paths that the locker wrote before its error.
//
// It returns the error of Stale, the error of git, which wraps [vcs.ErrGit], and the error of a
// locker, with the name of its toolchain, joined with the error of the restore.
func Lock(ctx context.Context, root string, g *Graph, plan *PublishPlan) ([]string, error) {
	stale, _, err := staleLockers(ctx, root, g, plan)
	if err != nil || len(stale) == 0 {
		return nil, err
	}
	tree, err := vcs.Snapshot(ctx, root)
	if err != nil {
		return nil, err
	}
	var written []string
	for _, group := range stale {
		paths, err := group.locker.Lock(ctx, root, group.pkgs)
		written = append(written, paths...)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("release: lock the toolchain %s: %w", group.name, err),
				vcs.Restore(context.WithoutCancel(ctx), root, tree, written))
		}
	}
	return written, nil
}

// staleLockers returns the packages of plan of each toolchain whose locker reports a stale
// lockfile, as [lockers] groups them, and the stale lockfiles of all toolchains, sorted and each
// once. It returns the error of lockers, and the error of a locker with the name of its toolchain.
func staleLockers(ctx context.Context, root string, g *Graph, plan *PublishPlan) (
	[]toolchainPackages, []string, error,
) {
	groups, err := lockers(g, plan)
	if err != nil {
		return nil, nil, err
	}
	var stale []toolchainPackages
	var files []string
	for _, group := range groups {
		paths, err := group.locker.Stale(ctx, root, group.pkgs)
		if err != nil {
			return nil, nil, fmt.Errorf("release: the lockfiles of the toolchain %s: %w", group.name, err)
		}
		if len(paths) > 0 {
			stale = append(stale, group)
			files = append(files, paths...)
		}
	}
	slices.Sort(files)
	return stale, slices.Compact(files), nil
}

// lockers returns the packages of the entries of plan, each at the version of its entry, grouped by
// the toolchains that have a locker, in the order of the plan. It returns an error that wraps
// [ErrPublishPlan] for a plan that does not fit g.
func lockers(g *Graph, plan *PublishPlan) ([]toolchainPackages, error) {
	if err := plan.check(g); err != nil {
		return nil, err
	}
	var groups []toolchainPackages
	at := map[*roles]int{}
	for _, chunk := range plan.Plan {
		for k := range chunk {
			i := g.index[chunk[k].Name]
			r := g.roles[i]
			if r.locker == nil {
				continue
			}
			slot, ok := at[r]
			if !ok {
				slot = len(groups)
				at[r] = slot
				groups = append(groups, toolchainPackages{locker: r.locker, name: g.pkgs[i].Toolchain})
			}
			p := g.pkgs[i]
			p.Version = chunk[k].Version
			groups[slot].pkgs = append(groups[slot].pkgs, p)
		}
	}
	return groups, nil
}
