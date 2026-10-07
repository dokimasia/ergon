// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
)

// readOnly is the mode of a go.mod that Apply cannot write.
const readOnly = 0o444

func TestChanges(t *testing.T) {
	t.Parallel()

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("returns no path for a go.mod whose content it does not change", func(t *testing.T) {
			t.Parallel()
			replacedB := modOf(pathB, pathA+" v0.1.0") + "\nreplace " + pathA + " => ../a\n"
			root := repository(t, files.Tree{
				"b/go.mod": files.Text(replacedB),
				"go.work":  files.Text(workOf("replace " + pathA + " v0.1.0 => ./a\n")),
			})
			edits := []language.Edit{
				edit(pathB, "b", workspace.Dependency{Name: pathA, Kind: workspace.KindRuntime, Req: "v0.1.0"}),
			}
			got, err := versioner.Apply(t.Context(), root, edits)
			assert.NoError(t, err, "Apply")
			assert.Empty(t, got, "the changed files")
			files.HasContent(t, filepath.Join(root, "b", "go.mod"), replacedB, "the go.mod of b")
		})

		t.Run("returns the error of a go.sum that it cannot read", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{"b/go.sum/inner": files.Text("inner\n")})
			got, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "read b/go.sum", "the error")
			assert.Equal(t, got, []string{"b/go.mod", "go.work"}, "the files that it began to change")
		})

		t.Run("returns the error of a go.work that it cannot write", func(t *testing.T) {
			t.Parallel()
			root := repository(t, nil)
			assert.NoError(t, os.Chmod(filepath.Join(root, "go.work"), readOnly), "Chmod of go.work")
			got, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "write go.work", "the error")
			assert.Equal(t, got, []string{"b/go.mod", "go.work"}, "the files that it began to change")
		})

		t.Run("returns the error of a go.mod that it cannot write", func(t *testing.T) {
			t.Parallel()
			root := repository(t, nil)
			assert.NoError(t, os.Chmod(filepath.Join(root, "b", "go.mod"), readOnly), "Chmod of the go.mod of b")
			got, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "write b/go.mod", "the error")
			assert.Equal(t, got, []string{"b/go.mod"}, "the files that it began to change")
		})
	})
}
