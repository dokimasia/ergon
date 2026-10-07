// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package vcs_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/vcs"
)

func TestGit(t *testing.T) {
	t.Parallel()

	t.Run("ErrGit", func(t *testing.T) {
		t.Parallel()

		t.Run("states the arguments and the standard error of git", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			_, err := vcs.Files(t.Context(), dir)
			assert.ErrorIs(t, err, vcs.ErrGit, "Files")
			assert.Contains(t, err.Error(), "vcs: git failed: git -C "+dir+" ls-files", "the arguments")
			assert.Contains(t, err.Error(), "not a git repository", "the standard error")
		})
	})
}
