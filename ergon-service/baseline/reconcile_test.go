// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/baseline"
)

// newFile is a managed file that a later baseline adds.
const newFile = "new.txt"

func TestReconcile(t *testing.T) {
	t.Parallel()

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("Check", func(t *testing.T) {
			t.Parallel()

			t.Run("returns Missing for a managed file that the repository does not have", func(t *testing.T) {
				t.Parallel()
				r, root := initialized(t)
				assert.NoError(t, root.Remove(license), "Remove of the LICENSE")
				findings, err := r.Check()
				assert.NoError(t, err, "Check")
				want := []baseline.Finding{{Path: license, Problem: baseline.Missing}}
				assert.Equal(t, findings, want, "the findings")
			})

			t.Run("returns Edited for a managed file that differs from the lock and the rendering", func(t *testing.T) {
				t.Parallel()
				r, root := initialized(t)
				put(t, root, license, "Copyright someone else\n")
				findings, err := r.Check()
				assert.NoError(t, err, "Check")
				want := []baseline.Finding{{Path: license, Problem: baseline.Edited}}
				assert.Equal(t, findings, want, "the findings")
			})

			t.Run("returns Outdated for an unedited file whose local file changed", func(t *testing.T) {
				t.Parallel()
				r, root := initialized(t)
				put(t, root, localDir+ignore, "local/\n")
				findings, err := r.Check()
				assert.NoError(t, err, "Check")
				want := []baseline.Finding{{Path: ignore, Problem: baseline.Outdated}}
				assert.Equal(t, findings, want, "the findings")
			})

			t.Run("returns Outdated for a file at the rendering that the lock records otherwise", func(t *testing.T) {
				t.Parallel()
				r, root := initialized(t)
				put(t, root, localDir+ignore, "local/\n")
				put(t, root, ignore, "# common\nalpha/\nlocal/\n")
				findings, err := r.Check()
				assert.NoError(t, err, "Check")
				want := []baseline.Finding{{Path: ignore, Problem: baseline.Outdated}}
				assert.Equal(t, findings, want, "the findings")
			})

			t.Run("returns Missing for a file of a later baseline that the repository lacks", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				findings, err := later(t, root).Check()
				assert.NoError(t, err, "Check")
				want := []baseline.Finding{{Path: newFile, Problem: baseline.Missing}}
				assert.Equal(t, findings, want, "the findings")
			})

			t.Run("returns Outdated for a file of a later baseline that the lock does not record", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				put(t, root, newFile, "new\n")
				findings, err := later(t, root).Check()
				assert.NoError(t, err, "Check")
				want := []baseline.Finding{{Path: newFile, Problem: baseline.Outdated}}
				assert.Equal(t, findings, want, "the findings")
			})

			t.Run("returns Edited for a file of a later baseline that exists with other content", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				put(t, root, newFile, "ours\n")
				findings, err := later(t, root).Check()
				assert.NoError(t, err, "Check")
				want := []baseline.Finding{{Path: newFile, Problem: baseline.Edited}}
				assert.Equal(t, findings, want, "the findings")
			})

			t.Run("returns the recorded files that the rendering no longer has", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				assert.NoError(t, root.Remove(workflow), "Remove of the workflow")
				put(t, root, license, "Copyright someone else\n")
				findings, err := withoutCommon(t, root).Check()
				assert.NoError(t, err, "Check")
				assert.Equal(t, findings, []baseline.Finding{
					{Path: workflow, Problem: baseline.Outdated},
					{Path: ignore, Problem: baseline.Outdated},
					{Path: license, Problem: baseline.Edited},
				}, "the findings")
			})
		})

		t.Run("Sync", func(t *testing.T) {
			t.Parallel()

			t.Run("records a file of a later baseline that is at the rendering", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				put(t, root, newFile, "new\n")
				r := later(t, root)
				changes, err := r.Sync(nil, baseline.Options{})
				assert.NoError(t, err, "Sync")
				assert.Equal(t, paths(changes), []string{lockPath}, "the files that Sync wrote")
				findings, err := r.Check()
				assert.NoError(t, err, "Check")
				assert.Empty(t, findings, "the findings after Sync")
			})

			t.Run("leaves an edited file that the rendering no longer has", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				put(t, root, license, "Copyright someone else\n")
				_, err := withoutCommon(t, root).Sync(nil, baseline.Options{})
				assert.ErrorIs(t, err, baseline.ErrConflict, "Sync")
				assert.True(t, exists(t, root, license), "the edited LICENSE")
				assert.Contains(t, content(t, root, lockPath), license, "the lock")
			})

			t.Run("removes an edited file that the rendering no longer has with Force", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				put(t, root, license, "Copyright someone else\n")
				_, err := withoutCommon(t, root).Sync(nil, baseline.Options{Force: true})
				assert.NoError(t, err, "Sync with Force")
				assert.False(t, exists(t, root, license), "the LICENSE")
			})

			t.Run("drops the entry of a missing file that the rendering no longer has", func(t *testing.T) {
				t.Parallel()
				_, root := initialized(t)
				assert.NoError(t, root.Remove(license), "Remove of the LICENSE")
				r := withoutCommon(t, root)
				_, err := r.Sync(nil, baseline.Options{})
				assert.NoError(t, err, "Sync")
				assert.NotContains(t, content(t, root, lockPath), license, "the lock")
			})
		})
	})
}

// later returns a repository on root whose baseline adds newFile to the producers of the cases.
func later(t *testing.T, root baseline.FS) *baseline.Repository {
	t.Helper()
	r, err := baseline.Open(root, catalog(t), version, common, renders(
		language.File{Path: newFile, Class: language.Managed, Content: []byte("new\n")},
	))
	assert.NoError(t, err, "Open of the later baseline")
	return r
}

// withoutCommon returns a repository on root whose baseline no longer has the producer common.
func withoutCommon(t *testing.T, root baseline.FS) *baseline.Repository {
	t.Helper()
	r, err := baseline.Open(root, catalog(t), version)
	assert.NoError(t, err, "Open without common")
	return r
}
