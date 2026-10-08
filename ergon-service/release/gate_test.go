// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/release"
)

// The repository, the workflow, the event, the commit and the page of the run of the cases.
const (
	gateRepo     = "dokimasia/ergon"
	gateWorkflow = "ci.yml"
	gateEvent    = "push"
	gateSHA      = "0749c74e25cc9171f23944f676f42168839ddc0e"
	gatePage     = "https://github.com/dokimasia/ergon/actions/runs/37769836521"
)

// The statuses and the conclusions of a run of the cases, as GitHub states them.
const (
	completedStatus   = "completed"
	inProgressStatus  = "in_progress"
	successConclusion = "success"
	failureConclusion = "failure"
)

// The intervals of the gates of the cases: one between two reads, and the time that GitHub may take
// to create a run, which is three intervals.
const (
	readInterval = time.Millisecond
	appearTime   = 3 * time.Millisecond
)

// errRuns is the error of the forge of the cases that fails.
var errRuns = errors.New("runs failed")

// run is the result of one read of a run.
type run struct {
	// status, conclusion and page are the status, the conclusion and the page of the run.
	status, conclusion, page string

	// err is the error of the read, or nil.
	err error
}

// runForge is a [release.RunForge] that returns the next result of runs on each read, and the last
// one again after the list ends. It records each read, and calls cancel on the first read when
// cancel is not nil.
type runForge struct {
	// runs are the results of the reads, in their order.
	runs []run

	// reads are the arguments of each read, each as one line.
	reads []string

	// cancel ends the context of the case on the first read, or is nil.
	cancel context.CancelFunc
}

var _ release.RunForge = (*runForge)(nil)

// Run records its arguments and returns the next result of f.
func (f *runForge) Run(_ context.Context, repo, workflow, sha, event string) (string, string, string, error) {
	f.reads = append(f.reads, repo+" "+workflow+" "+sha+" "+event)
	if f.cancel != nil {
		f.cancel()
	}
	r := f.runs[min(len(f.reads), len(f.runs))-1]
	return r.status, r.conclusion, r.page, r.err
}

func TestGate(t *testing.T) {
	t.Parallel()

	t.Run("Wait", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the page of a run that completes with success", func(t *testing.T) {
			t.Parallel()
			f := &runForge{runs: []run{
				{status: inProgressStatus, page: gatePage},
				{status: completedStatus, conclusion: successConclusion, page: gatePage},
			}}
			page, err := gate(f).Wait(t.Context(), gateSHA)
			assert.NoError(t, err, "Wait")
			assert.Equal(t, page, gatePage, "the page")
			read := gateRepo + " " + gateWorkflow + " " + gateSHA + " " + gateEvent
			assert.Equal(t, f.reads, []string{read, read}, "the reads of the run of the push")
		})

		t.Run("waits for a run that GitHub creates within the time to appear", func(t *testing.T) {
			t.Parallel()
			f := &runForge{
				runs: []run{{}, {}, {status: completedStatus, conclusion: successConclusion, page: gatePage}},
			}
			page, err := gate(f).Wait(t.Context(), gateSHA)
			assert.NoError(t, err, "Wait")
			assert.Equal(t, page, gatePage, "the page")
		})

		t.Run("waits for a run that runs longer than the time to appear", func(t *testing.T) {
			t.Parallel()
			running := run{status: inProgressStatus, page: gatePage}
			f := &runForge{runs: []run{
				running, running, running, running, running,
				{status: completedStatus, conclusion: successConclusion, page: gatePage},
			}}
			page, err := gate(f).Wait(t.Context(), gateSHA)
			assert.NoError(t, err, "Wait")
			assert.Equal(t, page, gatePage, "the page")
		})

		t.Run("returns ErrGate for a run that completes with another conclusion", func(t *testing.T) {
			t.Parallel()
			f := &runForge{runs: []run{{status: completedStatus, conclusion: failureConclusion, page: gatePage}}}
			page, err := gate(f).Wait(t.Context(), gateSHA)
			assert.ErrorIs(t, err, release.ErrGate, "Wait")
			assert.Contains(t, err.Error(), "the conclusion "+failureConclusion+": "+gatePage, "the error")
			assert.Equal(t, page, gatePage, "the page")
		})

		t.Run("returns ErrGate for a commit without a run after the time to appear", func(t *testing.T) {
			t.Parallel()
			f := &runForge{runs: []run{{}}}
			_, err := gate(f).Wait(t.Context(), gateSHA)
			assert.ErrorIs(t, err, release.ErrGate, "Wait")
			assert.Length(t, f.reads, 4, "the reads at 0, 1, 2 and 3 intervals")
		})

		t.Run("returns the error of the forge", func(t *testing.T) {
			t.Parallel()
			f := &runForge{runs: []run{{err: errRuns}}}
			_, err := gate(f).Wait(t.Context(), gateSHA)
			assert.ErrorIs(t, err, errRuns, "Wait")
		})

		t.Run("returns the error of a context that ends during the wait", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			f := &runForge{runs: []run{{status: inProgressStatus, page: gatePage}}, cancel: cancel}
			g := gate(f)
			g.Interval = time.Hour
			_, err := g.Wait(ctx, gateSHA)
			assert.ErrorIs(t, err, context.Canceled, "Wait")
		})
	})
}

// gate returns the gate of the cases on f.
func gate(f *runForge) *release.Gate {
	return &release.Gate{Forge: f, Repo: gateRepo, Workflow: gateWorkflow, Interval: readInterval, Appear: appearTime}
}
