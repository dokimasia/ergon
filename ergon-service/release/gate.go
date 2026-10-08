// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
)

// ErrGate is the error for a commit on whose content no run of the gate passed.
var ErrGate = errors.New("release: the gate did not pass")

// GateForge is the host of a repository whose runs of the gate a publish verifies.
type GateForge interface {
	// Passed returns the page of a run of the workflow file of repo for the commit sha that
	// completed with the conclusion success, and reports whether one did.
	Passed(ctx context.Context, repo, workflow, sha string) (string, bool, error)

	// Tree returns the tree of the commit sha of repo.
	Tree(ctx context.Context, repo, sha string) (string, error)

	// PullHeads returns the head commit of each pull request of repo that the host associates with
	// the commit sha, such as the pull request whose merge brought the commit to its branch.
	PullHeads(ctx context.Context, repo, sha string) ([]string, error)
}

// Gate is the gate of a repository in CI: the workflow, such as ci.yml, that a publish requires a
// passed run of.
type Gate struct {
	// Forge reads the runs of the workflow, the trees of the commits and the heads of their pull
	// requests.
	Forge GateForge

	// Repo is the repository on the host, as owner/name.
	Repo string

	// Workflow is the file of the workflow, such as ci.yml.
	Workflow string
}

// Verify returns the page of a run of the workflow of g that passed on the content of the commit
// sha. It accepts a run for sha itself, such as the run of its push or of its merge group, and a run
// for the head commit of a pull request that the host associates with sha when that head has the
// tree of sha. A squash, a rebase or a merge of a pull request that is up to date with its base
// leaves that tree. Verify reads each run once and waits for none. It returns an error that wraps
// [ErrGate] when no such run passed, and the error of the forge.
func (g *Gate) Verify(ctx context.Context, sha string) (string, error) {
	page, ok, err := g.Forge.Passed(ctx, g.Repo, g.Workflow, sha)
	if err != nil {
		return "", err
	}
	if ok {
		return page, nil
	}
	tree, err := g.Forge.Tree(ctx, g.Repo, sha)
	if err != nil {
		return "", err
	}
	heads, err := g.Forge.PullHeads(ctx, g.Repo, sha)
	if err != nil {
		return "", err
	}
	for _, head := range heads {
		headTree, err := g.Forge.Tree(ctx, g.Repo, head)
		if err != nil {
			return "", err
		}
		if headTree != tree {
			continue
		}
		page, ok, err := g.Forge.Passed(ctx, g.Repo, g.Workflow, head)
		if err != nil {
			return "", err
		}
		if ok {
			return page, nil
		}
	}
	return "", fmt.Errorf("%w: no run of %s passed on the content of %s, so rerun this job after the run of %s "+
		"for its push passes", ErrGate, g.Workflow, sha, g.Workflow)
}
