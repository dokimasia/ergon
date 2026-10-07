// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

// The lines of go.sum of a module outside the repository, which a Lock keeps.
const (
	sumX    = "example.com/x v1.0.0 h1:zip="
	modSumX = "example.com/x v1.0.0/go.mod h1:mod="
)

func TestSum(t *testing.T) {
	t.Parallel()

	t.Run("Stale", func(t *testing.T) {
		t.Parallel()

		t.Run("skips an empty line of a go.sum", func(t *testing.T) {
			t.Parallel()
			root := released(t)
			rewrite(t, root, "b/go.sum", "\n"+files.Read(t, filepath.Join(root, "b", "go.sum")))
			got, err := locker.Stale(t.Context(), root, pendingA())
			assert.NoError(t, err, "Stale")
			assert.Empty(t, got, "the stale go.sum files")
		})

		t.Run("returns an error for a line that is not a module path, a version and a hash", func(t *testing.T) {
			t.Parallel()
			root := released(t)
			rewrite(t, root, "b/go.sum", "\n"+pathA+" v0.2.0\n")
			_, err := locker.Stale(t.Context(), root, pendingA())
			assert.HasError(t, err, "Stale")
			assert.Contains(t, err.Error(), "read b/go.sum: line 2 is not a module path, a version and a hash",
				"the error")
		})

		t.Run("returns the error of a go.sum that it cannot read", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{"b/go.sum/inner": files.Text("inner\n")})
			_, err := locker.Stale(t.Context(), root, pendingA())
			assert.HasError(t, err, "Stale")
			assert.Contains(t, err.Error(), "read b/go.sum", "the error")
		})
	})

	t.Run("Lock", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the error of a go.sum that it cannot read", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{"b/go.sum/inner": files.Text("inner\n")})
			_, err := locker.Lock(t.Context(), root, pendingA())
			assert.HasError(t, err, "Lock")
			assert.Contains(t, err.Error(), "read b/go.sum", "the error")
		})
	})
}

// TestSumEnv runs the cases of Lock that change the environment of the process, one at a time.
func TestSumEnv(t *testing.T) {
	t.Run("Lock", func(t *testing.T) {
		t.Run("keeps the lines of the modules outside the release in their order", func(t *testing.T) {
			root := released(t)
			rewrite(t, root, "b/go.mod", modOf(pathB, pathA+" v0.2.0", "example.com/x v1.0.0"))
			rewrite(t, root, "b/x.go", "package b\n\nimport _ \"example.com/x\"\n")
			sums := files.Read(t, filepath.Join(root, "b", "go.sum"))
			rewrite(t, root, "b/go.sum", sumX+"\n"+sums+modSumX+"\n")
			t.Setenv("GOPROXY", "off")
			_, err := locker.Lock(t.Context(), root, pendingA())
			assert.HasError(t, err, "Lock")
			assert.Contains(t, err.Error(), "go mod tidy in b", "the error")
			files.HasContent(t, filepath.Join(root, "b", "go.sum"), sumX+"\n"+modSumX+"\n", "the go.sum of b")
		})
	})
}
