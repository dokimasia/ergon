// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
)

// VersionStatus is the context of the commit status that marks the commit of a version pull
// request as the version of its parent. [MarkVersion] sets it, and [Skip.Tested] reads it.
const VersionStatus = "ergon/version"

// statusSuccess is the state of the status that [MarkVersion] sets and [Skip.Tested] requires.
const statusSuccess = "success"

// SkipForge is the host of a repository whose runs of the gate can skip their jobs: the runs, the
// trees and the pull requests of [GateForge], and the statuses and the parents of commits.
type SkipForge interface {
	GateForge

	// Status returns the state of the newest status of the commit sha of repo whose context is
	// name, and reports whether the commit has such a status.
	Status(ctx context.Context, repo, sha, name string) (string, bool, error)

	// Parent returns the first parent of the commit sha of repo, and reports whether the commit has
	// a parent.
	Parent(ctx context.Context, repo, sha string) (string, bool, error)
}

// Skip decides whether a run of the gate of a repository can skip its jobs, because a passed run
// already tested the content that the run would test.
type Skip struct {
	// Forge reads the runs, the trees, the pull requests, the statuses and the parents.
	Forge SkipForge

	// Repo is the repository on the host, as owner/name.
	Repo string

	// Workflow is the file of the workflow of the gate, such as ci.yml.
	Workflow string
}

// Tested returns the page of a passed run of the gate that covers the content of a run, and
// reports whether one does. head is the commit that the run checked out and tree is its tree.
// commit is the commit that the run tests: head for a push or a merge group, and the head of the
// pull request for a pull request, whose checkout merges that head into its base. A passed run
// covers the run in two cases:
//
//   - It passed on the content of head, as [Gate.Verify] finds it, and commit is head. A run of a
//     pull request does not take this case, because the content of its checkout depends on its base.
//   - It passed on the first parent of a version commit with the tree of the run, as Gate.Verify
//     finds it. A version commit is a commit with the status [VersionStatus] in the state success.
//     The candidates are commit, and for a push also the heads of the pull requests that merged
//     head. The content of a version commit is the output of ergon release ci version on its parent.
//
// It reads each run once and waits for none. It returns the error of the forge.
func (s *Skip) Tested(ctx context.Context, head, tree, commit string) (string, bool, error) {
	gate := Gate{Forge: s.Forge, Repo: s.Repo, Workflow: s.Workflow}
	candidates := []string{commit}
	if commit == head {
		page, err := gate.Verify(ctx, head)
		if err == nil {
			return page, true, nil
		}
		if !errors.Is(err, ErrGate) {
			return "", false, err
		}
		heads, err := s.Forge.PullHeads(ctx, s.Repo, head)
		if err != nil {
			return "", false, err
		}
		candidates = append(candidates, heads...)
	}
	for _, candidate := range candidates {
		page, ok, err := s.version(ctx, &gate, candidate, tree)
		if err != nil {
			return "", false, err
		}
		if ok {
			return page, true, nil
		}
	}
	return "", false, nil
}

// version returns the page of the passed run of the first parent of the commit candidate, and
// reports whether candidate is a version commit with tree whose first parent passed gate, as
// [Skip.Tested] states. It returns the error of the forge.
func (s *Skip) version(ctx context.Context, gate *Gate, candidate, tree string) (string, bool, error) {
	state, marked, err := s.Forge.Status(ctx, s.Repo, candidate, VersionStatus)
	if err != nil {
		return "", false, err
	}
	if !marked || state != statusSuccess {
		return "", false, nil
	}
	candidateTree, err := s.Forge.Tree(ctx, s.Repo, candidate)
	if err != nil {
		return "", false, err
	}
	if candidateTree != tree {
		return "", false, nil
	}
	parent, ok, err := s.Forge.Parent(ctx, s.Repo, candidate)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	page, err := gate.Verify(ctx, parent)
	if errors.Is(err, ErrGate) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return page, true, nil
}

// Marker is the host of a repository on which a version pull request marks its commit: the heads
// of branches and the statuses of commits.
type Marker interface {
	// Branch returns the commit of the branch name of repo, and reports whether repo has the
	// branch.
	Branch(ctx context.Context, repo, name string) (string, bool, error)

	// SetStatus sets the status of the commit sha of repo whose context is name, with state, the
	// short description and the address target of its page.
	SetStatus(ctx context.Context, repo, sha, state, name, description, target string) error
}

// MarkVersion sets the status [VersionStatus] in the state success on the head of the branch of the
// version pull request p, after [Propose] committed p, with the address target of the page of the
// run that wrote p, and returns that commit. [Skip.Tested] then lets a run of the gate on that
// content skip its jobs once the run of its first parent passed. A proposal of more than one commit
// marks its last commit, whose first parent is the commit before it, so its runs do not skip. It
// returns an error for a branch that does not exist, and the error of the forge.
func MarkVersion(ctx context.Context, f Marker, p *Proposal, target string) (string, error) {
	head, ok, err := f.Branch(ctx, p.Repo, p.Branch)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("release: mark the version: %s has no branch %s", p.Repo, p.Branch)
	}
	description := "The version of " + p.Head + " that ergon release ci version wrote."
	if err := f.SetStatus(ctx, p.Repo, head, statusSuccess, VersionStatus, description, target); err != nil {
		return "", err
	}
	return head, nil
}
