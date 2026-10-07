// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/go/release"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
	"golang.org/x/mod/module"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("tidies a module after the released modules that it requires", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{
				"go.work":  files.Text("go 1.24\n\nuse (\n\t./a\n\t./b\n\t./c\n)\n"),
				"c/go.mod": files.Text(modOf(pathC, pathB+" v0.1.0")),
				"c/c.go":   files.Text(sourceC),
			})
			c := workspace.Package{
				Name: pathC, Toolchain: toolchain, Dir: "c", Version: version.Version{Minor: 1},
				Deps: []workspace.Dependency{{Name: pathB, Kind: workspace.KindRuntime, Req: "v0.1.0"}},
			}
			ofA := releaseOfA()
			edits := []language.Edit{{
				Version:      version.Version{Minor: 1, Patch: 1},
				Requirements: []workspace.Dependency{{Name: pathB, Kind: workspace.KindRuntime, Req: "v0.1.1"}},
				Package:      c,
			}, ofA[1], ofA[0]}
			got, err := versioner.Apply(t.Context(), root, edits)
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{"c/go.mod", "b/go.mod", "go.work", "b/go.sum", "c/go.sum"},
				"the changed files")
			vcstest.Commit(t, root, "release")
			a := module.Version{Path: pathA, Version: "v0.2.0"}
			b := module.Version{Path: pathB, Version: "v0.1.1"}
			files.HasContent(t, filepath.Join(root, "c", "go.sum"), sumOf(t, root, "a", a)+sumOf(t, root, "b", b),
				"the go.sum of c")
		})

		t.Run("returns ErrCycle for released modules that require each other", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{
				"a/go.mod": files.Text(modOf(pathA, pathB+" v0.1.0")),
				"b/go.mod": files.Text(modOf(pathB, pathA+" v0.1.0")),
			})
			edits := []language.Edit{
				edit(pathA, "a", workspace.Dependency{Name: pathB, Kind: workspace.KindRuntime, Req: "v0.2.0"}),
				edit(pathB, "b", workspace.Dependency{Name: pathA, Kind: workspace.KindRuntime, Req: "v0.2.0"}),
			}
			got, err := versioner.Apply(t.Context(), root, edits)
			assert.ErrorIs(t, err, release.ErrCycle, "Apply")
			assert.Contains(t, err.Error(), pathA+", "+pathB, "the error")
			assert.Equal(t, got, []string{"a/go.mod", "b/go.mod", "go.work"}, "the files that it began to change")
		})
	})
}

// TestRunEnv runs the cases of Apply that change the environment of the process, one at a time.
func TestRunEnv(t *testing.T) {
	t.Run("Apply", func(t *testing.T) {
		t.Run("returns the error of a temporary directory that it cannot create", func(t *testing.T) {
			root := repository(t, nil)
			absent := filepath.Join(root, "absent")
			for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
				t.Setenv(name, absent)
			}
			got, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "create the module proxy", "the error")
			assert.Equal(t, got, []string{"b/go.mod", "go.work"}, "the files that it began to change")
		})

		t.Run("returns the error of go env", func(t *testing.T) {
			root := repository(t, nil)
			t.Setenv("PATH", "")
			_, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "go env", "the error")
		})

		t.Run("returns the error of go mod tidy with its output", func(t *testing.T) {
			root := repository(t, files.Tree{
				"b/go.mod": files.Text(modOf(pathB, pathA+" v0.1.0", "example.com/x v1.0.0")),
				"b/x.go":   files.Text("package b\n\nimport _ \"example.com/x\"\n"),
			})
			t.Setenv("GOPROXY", "off")
			got, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "go mod tidy in b", "the error")
			assert.Contains(t, err.Error(), "GOPROXY=off", "the output of go mod tidy")
			assert.Equal(t, got, []string{"b/go.mod", "go.work", "b/go.sum"}, "the files that it began to change")
		})
	})
}

// edit returns the edit of a release of the module path in the directory dir from 0.1.0 to 0.2.0,
// with the requirements reqs.
func edit(path, dir string, reqs ...workspace.Dependency) language.Edit {
	return language.Edit{
		Version:      version.Version{Minor: 2},
		Requirements: reqs,
		Package:      workspace.Package{Name: path, Toolchain: toolchain, Dir: dir, Version: version.Version{Minor: 1}},
	}
}
