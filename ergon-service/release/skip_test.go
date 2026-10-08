// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/release"
)

// The commits of the cases beside those of the gate: the parent of the version commit, and the
// merge of a pull request that a run of the pull request checks out.
const (
	parentSHA = "f81cabac94301dc450d1aa074fb4b0a65b0657b9"
	mergeSHA  = "9999999999999999999999999999999999999999"
)

// parentPage is the page of the passed run of the parent of the version commit of the cases.
const parentPage = "https://github.com/dokimasia/ergon/actions/runs/37836591841"

// The address of the page of the run that writes the version commit of the cases.
const versionRun = "https://github.com/dokimasia/ergon/actions/runs/37838509558"

// The states of the statuses of the cases.
const (
	succeeded = "success"
	failed    = "failure"
)

// markForge is a [release.Marker] over the head of one branch. It records each call as a line, and
// returns errForge from the call whose number is failAt.
type markForge struct {
	// head is the commit of the branch, and empty for a branch that does not exist.
	head string

	// calls are the calls, each as a line, in their order.
	calls []string

	// failAt is the number of the call that fails, from 1, or 0 for none.
	failAt int
}

var _ release.Marker = (*markForge)(nil)

// call records line and returns errForge when it is the call failAt.
func (f *markForge) call(line string) error {
	f.calls = append(f.calls, line)
	if len(f.calls) == f.failAt {
		return errForge
	}
	return nil
}

// Branch returns the head of the branch.
func (f *markForge) Branch(_ context.Context, repo, name string) (string, bool, error) {
	if err := f.call("branch " + repo + " " + name); err != nil {
		return "", false, err
	}
	return f.head, f.head != "", nil
}

// SetStatus records the status.
func (f *markForge) SetStatus(_ context.Context, repo, sha, state, name, description, target string) error {
	return f.call("status " + repo + " " + sha + " " + state + " " + name + " " + description + " " + target)
}

func TestSkip(t *testing.T) {
	t.Parallel()

	t.Run("Tested", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the page of a passed run on the content of a push", func(t *testing.T) {
			t.Parallel()
			f := &gateForge{passed: map[string]string{gateSHA: gatePage}}
			page, ok, err := skip(f).Tested(t.Context(), gateSHA, gateTree, gateSHA)
			assert.NoError(t, err, "Tested")
			assert.True(t, ok, "whether a passed run covers the push")
			assert.Equal(t, page, gatePage, "the page")
		})

		t.Run("returns the page of the run of the parent of a merged version commit", func(t *testing.T) {
			t.Parallel()
			f := mergedVersion()
			page, ok, err := skip(f).Tested(t.Context(), gateSHA, gateTree, gateSHA)
			assert.NoError(t, err, "Tested")
			assert.True(t, ok, "whether a passed run covers the push")
			assert.Equal(t, page, parentPage, "the page")
			assert.Equal(t, f.reads, []string{
				"passed " + gateRepo + " " + gateWorkflow + " " + gateSHA,
				"tree " + gateRepo + " " + gateSHA,
				"heads " + gateRepo + " " + gateSHA,
				"tree " + gateRepo + " " + headSHA,
				"passed " + gateRepo + " " + gateWorkflow + " " + headSHA,
				"heads " + gateRepo + " " + gateSHA,
				"status " + gateRepo + " " + gateSHA + " " + release.VersionStatus,
				"status " + gateRepo + " " + headSHA + " " + release.VersionStatus,
				"tree " + gateRepo + " " + headSHA,
				"parent " + gateRepo + " " + headSHA,
				"passed " + gateRepo + " " + gateWorkflow + " " + parentSHA,
			}, "the reads")
		})

		t.Run("returns the page of the run of the parent of the version commit of a pull request", func(t *testing.T) {
			t.Parallel()
			f := mergedVersion()
			page, ok, err := skip(f).Tested(t.Context(), mergeSHA, gateTree, headSHA)
			assert.NoError(t, err, "Tested")
			assert.True(t, ok, "whether a passed run covers the pull request")
			assert.Equal(t, page, parentPage, "the page")
			assert.Equal(t, f.reads[0], "status "+gateRepo+" "+headSHA+" "+release.VersionStatus,
				"the first read, which is no run of the merge that the pull request checks out")
		})

		// The version commits of pull requests that a passed run does not cover, each from
		// mergedVersion with one change.
		uncovered := []struct {
			name   string
			change func(f *gateForge)
		}{
			{
				name:   "reports false for a head without the status of a version commit",
				change: func(f *gateForge) { delete(f.statuses, headSHA) },
			},
			{
				name:   "reports false for a version commit whose status did not succeed",
				change: func(f *gateForge) { f.statuses[headSHA] = failed },
			},
			{
				name:   "reports false for a version commit with another tree",
				change: func(f *gateForge) { f.trees[headSHA] = otherTree },
			},
			{
				name:   "reports false for a version commit without a parent",
				change: func(f *gateForge) { delete(f.parents, headSHA) },
			},
			{
				name:   "reports false for a version commit whose parent has no passed run",
				change: func(f *gateForge) { delete(f.passed, parentSHA) },
			},
		}
		for _, tt := range uncovered {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f := mergedVersion()
				tt.change(f)
				page, ok, err := skip(f).Tested(t.Context(), mergeSHA, gateTree, headSHA)
				assert.NoError(t, err, "Tested")
				assert.False(t, ok, "whether a passed run covers the pull request")
				assert.Empty(t, page, "the page")
			})
		}

		// The reads of a push of a merged version commit, in their order.
		failures := []string{
			"returns the error of the runs of the commit",
			"returns the error of the tree of the commit",
			"returns the error of the pull requests of the gate",
			"returns the error of the tree of a head in the gate",
			"returns the error of the runs of a head",
			"returns the error of the pull requests of the commit",
			"returns the error of the status of the commit",
			"returns the error of the status of a head",
			"returns the error of the tree of a version commit",
			"returns the error of the parent of a version commit",
			"returns the error of the runs of the parent",
		}
		for n, name := range failures {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				f := mergedVersion()
				f.failAt = n + 1
				_, _, err := skip(f).Tested(t.Context(), gateSHA, gateTree, gateSHA)
				assert.ErrorIs(t, err, errForge, "Tested")
				assert.Length(t, f.reads, n+1, "the reads up to the one that fails")
			})
		}
	})

	t.Run("MarkVersion", func(t *testing.T) {
		t.Parallel()

		t.Run("sets the status of a version commit on the head of the branch", func(t *testing.T) {
			t.Parallel()
			f := &markForge{head: headSHA}
			marked, err := release.MarkVersion(t.Context(), f, proposal(nil), versionRun)
			assert.NoError(t, err, "MarkVersion")
			assert.Equal(t, marked, headSHA, "the commit")
			assert.Equal(t, f.calls, []string{
				"branch " + proposalRepo + " " + proposalBranch,
				"status " + proposalRepo + " " + headSHA + " " + succeeded + " " + release.VersionStatus + " The version of " +
					commitA + " that ergon release ci version wrote. " + versionRun,
			}, "the calls")
		})

		t.Run("returns an error for a branch that does not exist", func(t *testing.T) {
			t.Parallel()
			_, err := release.MarkVersion(t.Context(), &markForge{}, proposal(nil), versionRun)
			assert.HasError(t, err, "MarkVersion")
			assert.Equal(t, err.Error(), "release: mark the version: "+proposalRepo+" has no branch "+proposalBranch,
				"the error")
		})

		for n, name := range []string{"returns the error of the branch", "returns the error of the status"} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				f := &markForge{head: headSHA, failAt: n + 1}
				_, err := release.MarkVersion(t.Context(), f, proposal(nil), versionRun)
				assert.ErrorIs(t, err, errForge, "MarkVersion")
			})
		}
	})
}

// skip returns the skip of the cases on f.
func skip(f *gateForge) *release.Skip {
	return &release.Skip{Forge: f, Repo: gateRepo, Workflow: gateWorkflow}
}

// mergedVersion returns the forge of a version commit that a pull request merged: the merged commit
// has the tree of the head of the pull request and no passed run, the head has the status of a
// version commit and no passed run, and the parent of the head has a passed run.
func mergedVersion() *gateForge {
	return &gateForge{
		passed:   map[string]string{parentSHA: parentPage},
		trees:    map[string]string{gateSHA: gateTree, headSHA: gateTree},
		heads:    map[string][]string{gateSHA: {headSHA}},
		statuses: map[string]string{headSHA: succeeded},
		parents:  map[string]string{headSHA: parentSHA},
	}
}
