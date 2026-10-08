// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// The status and the conclusion of a run of GitHub Actions that a [Gate] reads.
const (
	// completed is the status of a run that ended.
	completed = "completed"

	// success is the conclusion of a run whose jobs passed.
	success = "success"
)

// pushEvent is the event of the run of the gate that a release waits for: the push of its commit.
const pushEvent = "push"

// ErrGate is the error for a run of the gate that completes without success, and for a commit that
// no run of the gate checks.
var ErrGate = errors.New("release: the gate did not pass")

// RunForge is the host of a repository whose workflow runs a release waits for.
type RunForge interface {
	// Run returns the newest run of the workflow file of repo for the commit sha and the event: its
	// status, its conclusion, which is empty until the run completes, and the address of its page.
	// It returns empty results for a commit without such a run.
	Run(ctx context.Context, repo, workflow, sha, event string) (status, conclusion, page string, err error)
}

// Gate is the gate of a repository in CI: the workflow whose run for the push of a commit the
// release steps of the commit wait for, such as ci.yml.
type Gate struct {
	// Forge reads the runs of the workflow.
	Forge RunForge

	// Repo is the repository on the host, as owner/name.
	Repo string

	// Workflow is the file of the workflow, such as ci.yml.
	Workflow string

	// Interval is the time between two reads of the run.
	Interval time.Duration

	// Appear is the time that the host may take to create the run of a commit, counted in the
	// intervals that Wait sleeps.
	Appear time.Duration
}

// Wait reads the newest run of the workflow of g for the push of the commit sha every Interval until
// the run completes, and returns the address of its page. It returns an error that wraps [ErrGate]
// for a run that completes with a conclusion other than success, and for a commit that has no run
// after Wait has slept Appear. It returns the error of the forge, and an error that wraps the
// error of ctx when ctx ends first.
func (g *Gate) Wait(ctx context.Context, sha string) (string, error) {
	var waited time.Duration
	for {
		status, conclusion, page, err := g.Forge.Run(ctx, g.Repo, g.Workflow, sha, pushEvent)
		if err != nil {
			return "", err
		}
		if conclusion == success {
			return page, nil
		}
		if status == completed {
			return page, fmt.Errorf("%w: the run of %s for %s completed with the conclusion %s: %s", ErrGate,
				g.Workflow, sha, conclusion, page)
		}
		if status == "" && waited >= g.Appear {
			return "", fmt.Errorf("%w: no run of %s checks %s after %s", ErrGate, g.Workflow, sha, g.Appear)
		}
		select {
		case <-ctx.Done():
			return page, fmt.Errorf("release: wait for the run of %s for %s: %w", g.Workflow, sha, ctx.Err())
		case <-time.After(g.Interval):
		}
		waited += g.Interval
	}
}
