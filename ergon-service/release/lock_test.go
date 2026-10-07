// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
)

// The content of a lockfile of a case before and after a Lock.
const (
	earlierLock = "earlier\n"
	lockedLock  = "locked\n"
)

// lockA is the lockfile of pkg-a that a locker of the cases reports.
var lockA = path.Join(packagesDir, "pkg-a", "lock")

// errLockfiles is the error of a locker that fails.
var errLockfiles = errors.New("lockfiles failed")

// lockfiles is a [language.Locker] of the cases. Stale records the packages of each call and
// returns stale. Lock records the packages of each call, writes lockedLock into each file of stale,
// and returns their paths with failLock.
type lockfiles struct {
	// tb is the test of the case, which a write that fails stops. A case whose Lock writes sets it.
	tb testing.TB

	// failStale is the error of Stale, or nil.
	failStale error

	// failLock is the error of Lock after its writes, or nil.
	failLock error

	// stale are the lockfiles that Stale reports and Lock writes.
	stale []string

	// staleCalls are the packages of each call of Stale, in the order of the calls.
	staleCalls [][]workspace.Package

	// lockCalls are the packages of each call of Lock, in the order of the calls.
	lockCalls [][]workspace.Package
}

var _ language.Locker = (*lockfiles)(nil)

// Stale records pkgs and returns l.stale with l.failStale.
func (l *lockfiles) Stale(_ context.Context, _ string, pkgs []workspace.Package) ([]string, error) {
	l.staleCalls = append(l.staleCalls, pkgs)
	return l.stale, l.failStale
}

// Lock records pkgs, writes lockedLock into each file of l.stale under root, and returns their
// paths with l.failLock.
func (l *lockfiles) Lock(_ context.Context, root string, pkgs []workspace.Package) ([]string, error) {
	l.lockCalls = append(l.lockCalls, pkgs)
	for _, file := range l.stale {
		err := os.WriteFile(filepath.Join(root, filepath.FromSlash(file)), []byte(lockedLock), versionPerm)
		assert.NoError(l.tb, err, "the write of "+file)
	}
	return l.stale, l.failLock
}

func TestLock(t *testing.T) {
	t.Parallel()

	t.Run("Stale", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the lockfiles of the lockers sorted, each once", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			s.roles = []any{&lockfiles{stale: []string{"z/lock", lockA, lockA}}}
			got, err := release.Stale(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.NoError(t, err, "Stale")
			assert.Equal(t, got, []string{lockA, "z/lock"}, "the stale lockfiles")
		})

		t.Run("passes the packages of the plan at the versions of their entries", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			l := &lockfiles{}
			s.roles = []any{l}
			plan.Plan[0][0].Version = parse(t, "1.1.0")
			_, err := release.Stale(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.NoError(t, err, "Stale")
			a, b := s.pkgs[0], s.pkgs[1]
			a.Version = parse(t, "1.1.0")
			assert.Equal(t, l.staleCalls, [][]workspace.Package{{a, b}}, "the packages of Stale")
		})

		t.Run("returns no lockfile for a toolchain without a locker", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			got, err := release.Stale(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.NoError(t, err, "Stale")
			assert.Empty(t, got, "the stale lockfiles")
		})

		t.Run("returns the error of a locker with the name of its toolchain", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			s.roles = []any{&lockfiles{failStale: errLockfiles}}
			_, err := release.Stale(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.ErrorIs(t, err, errLockfiles, "Stale")
			assert.Contains(t, err.Error(), "the lockfiles of the toolchain npm", "the error")
		})

		t.Run("returns ErrPublishPlan for an entry of a package that the repository does not have", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			plan.Plan[0][0].Name = "pkg-z"
			_, err := release.Stale(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.ErrorIs(t, err, release.ErrPublishPlan, "Stale")
		})
	})

	t.Run("Lock", func(t *testing.T) {
		t.Parallel()

		t.Run("rewrites the stale lockfiles and returns their paths", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			l := &lockfiles{tb: t, stale: []string{lockA}}
			s.roles = []any{l}
			dir, _ := repository(t, s, files.Tree{lockA: files.Text(earlierLock)})
			got, err := release.Lock(t.Context(), dir, s.graph(t), plan)
			assert.NoError(t, err, "Lock")
			assert.Equal(t, got, []string{lockA}, "the changed paths")
			files.HasContent(t, filepath.Join(dir, lockA), lockedLock, "the lockfile")
			assert.Length(t, l.lockCalls, 1, "the calls of Lock")
		})

		t.Run("changes no file without a stale lockfile", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			l := &lockfiles{}
			s.roles = []any{l}
			got, err := release.Lock(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.NoError(t, err, "Lock")
			assert.Empty(t, got, "the changed paths")
			assert.Empty(t, l.lockCalls, "the calls of Lock")
		})

		t.Run("restores every path that it changed when a locker fails", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			s.roles = []any{&lockfiles{tb: t, stale: []string{lockA}, failLock: errLockfiles}}
			dir, _ := repository(t, s, files.Tree{lockA: files.Text(earlierLock)})
			got, err := release.Lock(t.Context(), dir, s.graph(t), plan)
			assert.ErrorIs(t, err, errLockfiles, "Lock")
			assert.Contains(t, err.Error(), "lock the toolchain npm", "the error")
			assert.Empty(t, got, "the changed paths")
			files.HasContent(t, filepath.Join(dir, lockA), earlierLock, "the lockfile")
		})

		t.Run("returns the error of a locker that reports no lockfile", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			s.roles = []any{&lockfiles{failStale: errLockfiles}}
			_, err := release.Lock(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.ErrorIs(t, err, errLockfiles, "Lock")
			assert.Contains(t, err.Error(), "the lockfiles of the toolchain npm", "the error")
		})

		t.Run("returns ErrGit outside a working tree", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			s.roles = []any{&lockfiles{stale: []string{lockA}}}
			_, err := release.Lock(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.ErrorIs(t, err, vcs.ErrGit, "Lock")
		})

		t.Run("returns ErrPublishPlan for an entry of a package that the repository does not have", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			plan.Plan[0][0].Name = "pkg-z"
			_, err := release.Lock(t.Context(), t.TempDir(), s.graph(t), plan)
			assert.ErrorIs(t, err, release.ErrPublishPlan, "Lock")
		})
	})
}
