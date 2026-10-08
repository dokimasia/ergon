// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/release"
)

// The repository and the workflow of the gates of the cases.
const (
	gateRepo     = "dokimasia/ergon"
	gateWorkflow = "ci.yml"
)

// The commits of the cases: the commit that a publish verifies, the head of the pull request that
// merged it, and the head of another pull request of the commit.
const (
	gateSHA   = "0749c74e25cc9171f23944f676f42168839ddc0e"
	headSHA   = "a8f03f8a8f03f8a8f03f8a8f03f8a8f03f8a8f03"
	otherHead = "ca02a08ca02a08ca02a08ca02a08ca02a08ca02a"
)

// The trees of the commits of the cases: the tree of the commit, and another tree.
const (
	gateTree  = "fc9f3dacda8a546eeed499c6cdc2722f653e4993"
	otherTree = "1111111111111111111111111111111111111111"
)

// The pages of the runs of the cases.
const (
	gatePage  = "https://github.com/dokimasia/ergon/actions/runs/37769836521"
	headPage  = "https://github.com/dokimasia/ergon/actions/runs/37821856662"
	otherPage = "https://github.com/dokimasia/ergon/actions/runs/37809885040"
)

// errForge is the error of the forge of the cases that fails.
var errForge = errors.New("forge failed")

// gateForge is a [release.GateForge] over maps: the page of the passed run of each commit, the tree
// of each commit, and the heads of the pull requests of each commit. It records each read as a line,
// and returns errForge from the read whose number is failAt.
type gateForge struct {
	// passed are the pages of the passed runs, by commit.
	passed map[string]string

	// trees are the trees, by commit.
	trees map[string]string

	// heads are the heads of the pull requests, by commit.
	heads map[string][]string

	// reads are the reads, each as a line, in their order.
	reads []string

	// failAt is the number of the read that fails, from 1, or 0 for none.
	failAt int
}

var _ release.GateForge = (*gateForge)(nil)

// read records line and returns errForge when it is the read failAt.
func (f *gateForge) read(line string) error {
	f.reads = append(f.reads, line)
	if len(f.reads) == f.failAt {
		return errForge
	}
	return nil
}

// Passed returns the page of the passed run of sha.
func (f *gateForge) Passed(_ context.Context, repo, workflow, sha string) (string, bool, error) {
	if err := f.read("passed " + repo + " " + workflow + " " + sha); err != nil {
		return "", false, err
	}
	page, ok := f.passed[sha]
	return page, ok, nil
}

// Tree returns the tree of sha.
func (f *gateForge) Tree(_ context.Context, repo, sha string) (string, error) {
	if err := f.read("tree " + repo + " " + sha); err != nil {
		return "", err
	}
	return f.trees[sha], nil
}

// PullHeads returns the heads of the pull requests of sha.
func (f *gateForge) PullHeads(_ context.Context, repo, sha string) ([]string, error) {
	if err := f.read("heads " + repo + " " + sha); err != nil {
		return nil, err
	}
	return f.heads[sha], nil
}

func TestGate(t *testing.T) {
	t.Parallel()

	t.Run("Verify", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the page of a run of the commit that passed", func(t *testing.T) {
			t.Parallel()
			f := &gateForge{passed: map[string]string{gateSHA: gatePage}}
			page, err := gate(f).Verify(t.Context(), gateSHA)
			assert.NoError(t, err, "Verify")
			assert.Equal(t, page, gatePage, "the page")
			assert.Equal(t, f.reads, []string{"passed " + gateRepo + " " + gateWorkflow + " " + gateSHA}, "the reads")
		})

		t.Run("returns the page of a passed run of a pull request head with the same tree", func(t *testing.T) {
			t.Parallel()
			f := merged()
			page, err := gate(f).Verify(t.Context(), gateSHA)
			assert.NoError(t, err, "Verify")
			assert.Equal(t, page, headPage, "the page")
			assert.Equal(t, f.reads, []string{
				"passed " + gateRepo + " " + gateWorkflow + " " + gateSHA,
				"tree " + gateRepo + " " + gateSHA,
				"heads " + gateRepo + " " + gateSHA,
				"tree " + gateRepo + " " + headSHA,
				"passed " + gateRepo + " " + gateWorkflow + " " + headSHA,
			}, "the reads")
		})

		t.Run("skips the head of a pull request with another tree", func(t *testing.T) {
			t.Parallel()
			f := merged()
			f.heads[gateSHA] = []string{otherHead, headSHA}
			f.trees[otherHead] = otherTree
			f.passed[otherHead] = otherPage
			page, err := gate(f).Verify(t.Context(), gateSHA)
			assert.NoError(t, err, "Verify")
			assert.Equal(t, page, headPage, "the page")
			assert.NotContains(t, f.reads, "passed "+gateRepo+" "+gateWorkflow+" "+otherHead,
				"the reads, which skip the runs of a head with another tree")
		})

		t.Run("returns ErrGate for a pull request whose head has no run that passed", func(t *testing.T) {
			t.Parallel()
			f := merged()
			delete(f.passed, headSHA)
			_, err := gate(f).Verify(t.Context(), gateSHA)
			assert.ErrorIs(t, err, release.ErrGate, "Verify")
			assert.Equal(t, err.Error(), "release: the gate did not pass: no run of "+gateWorkflow+
				" passed on the content of "+gateSHA+", so rerun this job after the run of "+gateWorkflow+
				" for its push passes", "the error")
		})

		t.Run("returns ErrGate for a commit without a pull request", func(t *testing.T) {
			t.Parallel()
			f := &gateForge{trees: map[string]string{gateSHA: gateTree}}
			_, err := gate(f).Verify(t.Context(), gateSHA)
			assert.ErrorIs(t, err, release.ErrGate, "Verify")
		})

		// The reads of a commit that a pull request merged, in their order.
		failures := []string{
			"returns the error of the runs of the commit",
			"returns the error of the tree of the commit",
			"returns the error of the pull requests of the commit",
			"returns the error of the tree of a head",
			"returns the error of the runs of a head",
		}
		for n, name := range failures {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				f := merged()
				f.failAt = n + 1
				_, err := gate(f).Verify(t.Context(), gateSHA)
				assert.ErrorIs(t, err, errForge, "Verify")
				assert.Length(t, f.reads, n+1, "the reads up to the one that fails")
			})
		}
	})
}

// gate returns the gate of the cases on f.
func gate(f *gateForge) *release.Gate {
	return &release.Gate{Forge: f, Repo: gateRepo, Workflow: gateWorkflow}
}

// merged returns the forge of a commit that a pull request merged: the commit has no run that
// passed, and the head of the pull request has the tree of the commit and a run that passed.
func merged() *gateForge {
	return &gateForge{
		passed: map[string]string{headSHA: headPage},
		trees:  map[string]string{gateSHA: gateTree, headSHA: gateTree},
		heads:  map[string][]string{gateSHA: {headSHA}},
	}
}
