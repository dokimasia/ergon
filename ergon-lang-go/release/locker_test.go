// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/go/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
	"golang.org/x/mod/module"
)

// changedA is the Go file of the module a after a change that merged after its version commit.
const changedA = "package a\n\n// A returns two.\nfunc A() int { return 2 }\n"

// otherHash differs from the hash of every module of the cases.
const otherHash = "h1:other="

// sourcePerm is the mode of a file that a case rewrites.
const sourcePerm fs.FileMode = 0o644

// locker is the locker of the cases, which takes its snapshots with the git of vcs.
var locker = release.Locker{Snapshot: vcs.Snapshot}

func TestLocker(t *testing.T) {
	t.Parallel()

	t.Run("Stale", func(t *testing.T) {
		t.Parallel()

		t.Run("returns no go.sum that records the modules of the working tree", func(t *testing.T) {
			t.Parallel()
			got, err := locker.Stale(t.Context(), released(t), pendingA())
			assert.NoError(t, err, "Stale")
			assert.Empty(t, got, "the stale go.sum files")
		})

		t.Run("returns the go.sum that records another zip of a module", func(t *testing.T) {
			t.Parallel()
			root := released(t)
			rewrite(t, root, "a/a.go", changedA)
			vcstest.Commit(t, root, "change a")
			got, err := locker.Stale(t.Context(), root, pendingA())
			assert.NoError(t, err, "Stale")
			assert.Equal(t, got, []string{"b/go.sum"}, "the stale go.sum files")
		})

		t.Run("returns the go.sum that records another go.mod of a module", func(t *testing.T) {
			t.Parallel()
			root := released(t)
			rewrite(t, root, "b/go.sum", pathA+" v0.2.0/go.mod "+otherHash+"\n")
			got, err := locker.Stale(t.Context(), root, pendingA())
			assert.NoError(t, err, "Stale")
			assert.Equal(t, got, []string{"b/go.sum"}, "the stale go.sum files")
		})

		t.Run("returns no go.sum that records the go.mod of a module alone", func(t *testing.T) {
			t.Parallel()
			root := released(t)
			lines := strings.SplitAfter(sumOf(t, root, "a", module.Version{Path: pathA, Version: "v0.2.0"}), "\n")
			rewrite(t, root, "b/go.sum", lines[1])
			got, err := locker.Stale(t.Context(), root, pendingA())
			assert.NoError(t, err, "Stale")
			assert.Empty(t, got, "the stale go.sum files")
		})

		t.Run("returns no go.sum for the hash of another version of a module", func(t *testing.T) {
			t.Parallel()
			root := released(t)
			rewrite(t, root, "b/go.sum", pathA+" v0.1.0 "+otherHash+"\n")
			got, err := locker.Stale(t.Context(), root, pendingA())
			assert.NoError(t, err, "Stale")
			assert.Empty(t, got, "the stale go.sum files")
		})

		t.Run("returns no go.sum of a module that replaces the module with a directory", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{
				"b/go.mod": files.Text(modOf(pathB, pathA+" v0.2.0") + "\nreplace " + pathA + " => ../a\n"),
				"b/go.sum": files.Text(pathA + " v0.2.0 " + otherHash + "\n"),
			})
			got, err := locker.Stale(t.Context(), root, pendingA())
			assert.NoError(t, err, "Stale")
			assert.Empty(t, got, "the stale go.sum files")
		})

		t.Run("returns each stale go.sum in the order of their paths", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{
				"go.work":  files.Text("go 1.24\n\nuse (\n\t./a\n\t./b\n\t./c\n)\n"),
				"b/go.sum": files.Text(pathA + " v0.2.0 " + otherHash + "\n"),
				"c/go.mod": files.Text(modOf(pathC, pathA+" v0.2.0")),
				"c/go.sum": files.Text(pathA + " v0.2.0 " + otherHash + "\n"),
			})
			got, err := locker.Stale(t.Context(), root, pendingA())
			assert.NoError(t, err, "Stale")
			assert.Equal(t, got, []string{"b/go.sum", "c/go.sum"}, "the stale go.sum files")
		})

		t.Run("returns an error for a module that the repository does not have", func(t *testing.T) {
			t.Parallel()
			absent := []workspace.Package{{Name: "example.com/absent", Version: version.Version{Minor: 1}}}
			_, err := locker.Stale(t.Context(), released(t), absent)
			assert.HasError(t, err, "Stale")
			assert.Contains(t, err.Error(), "the module example.com/absent, which the repository does not have",
				"the error")
		})

		t.Run("returns the error of the modules of the repository", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"go.mod": files.Text("go 1.24\n")})
			_, err := locker.Stale(t.Context(), root, nil)
			assert.HasError(t, err, "Stale")
			assert.Contains(t, err.Error(), "go.mod declares no module", "the error")
		})

		t.Run("returns the error of the snapshot", func(t *testing.T) {
			t.Parallel()
			failing := release.Locker{
				Snapshot: func(context.Context, string) (string, error) { return "", errSnapshot },
			}
			_, err := failing.Stale(t.Context(), released(t), pendingA())
			assert.ErrorIs(t, err, errSnapshot, "Stale")
		})
	})

	t.Run("Lock", func(t *testing.T) {
		t.Parallel()

		t.Run("rewrites the go.sum that records another zip of a module", func(t *testing.T) {
			t.Parallel()
			root := released(t)
			rewrite(t, root, "a/a.go", changedA)
			vcstest.Commit(t, root, "change a")
			got, err := locker.Lock(t.Context(), root, pendingA())
			assert.NoError(t, err, "Lock")
			assert.Equal(t, got, []string{"b/go.sum"}, "the changed files")
			vcstest.Commit(t, root, "lock")
			a := module.Version{Path: pathA, Version: "v0.2.0"}
			files.HasContent(t, filepath.Join(root, "b", "go.sum"), sumOf(t, root, "a", a), "the go.sum of b")
			stale, err := locker.Stale(t.Context(), root, pendingA())
			assert.NoError(t, err, "Stale after Lock")
			assert.Empty(t, stale, "the stale go.sum files after Lock")
		})

		t.Run("changes no file for a go.sum that records the modules of the working tree", func(t *testing.T) {
			t.Parallel()
			got, err := locker.Lock(t.Context(), released(t), pendingA())
			assert.NoError(t, err, "Lock")
			assert.Empty(t, got, "the changed files")
		})

		t.Run("rewrites a go.sum after the go.sum of a module that it requires", func(t *testing.T) {
			t.Parallel()
			root := releasedWithC(t)
			rewrite(t, root, "a/a.go", changedA)
			vcstest.Commit(t, root, "change a")
			pkgs := append(pendingA(), workspace.Package{
				Name: pathB, Toolchain: toolchain, Dir: "b", Version: version.Version{Minor: 1, Patch: 1},
			})
			got, err := locker.Lock(t.Context(), root, pkgs)
			assert.NoError(t, err, "Lock")
			assert.Equal(t, got, []string{"b/go.sum", "c/go.sum"}, "the changed files")
			vcstest.Commit(t, root, "lock")
			a := module.Version{Path: pathA, Version: "v0.2.0"}
			b := module.Version{Path: pathB, Version: "v0.1.1"}
			files.HasContent(t, filepath.Join(root, "c", "go.sum"), sumOf(t, root, "a", a)+sumOf(t, root, "b", b),
				"the go.sum of c")
		})

		t.Run("returns ErrCycle for released modules that require each other", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{
				"a/go.mod": files.Text(modOf(pathA, pathB+" v0.2.0")),
				"a/go.sum": files.Text(pathB + " v0.2.0 " + otherHash + "\n"),
				"b/go.mod": files.Text(modOf(pathB, pathA+" v0.2.0")),
				"b/go.sum": files.Text(pathA + " v0.2.0 " + otherHash + "\n"),
			})
			pkgs := append(pendingA(), workspace.Package{
				Name: pathB, Toolchain: toolchain, Dir: "b", Version: version.Version{Minor: 2},
			})
			got, err := locker.Lock(t.Context(), root, pkgs)
			assert.ErrorIs(t, err, release.ErrCycle, "Lock")
			assert.Equal(t, got, []string{"a/go.sum", "b/go.sum"}, "the files that it began to change")
		})

		t.Run("returns the error of a go.sum that it cannot write", func(t *testing.T) {
			t.Parallel()
			root := released(t)
			assert.NoError(t, os.Chmod(filepath.Join(root, "b", "go.sum"), readOnly), "Chmod of the go.sum of b")
			got, err := locker.Lock(t.Context(), root, pendingA())
			assert.HasError(t, err, "Lock")
			assert.Contains(t, err.Error(), "write b/go.sum", "the error")
			assert.Equal(t, got, []string{"b/go.sum"}, "the files that it began to change")
		})

		t.Run("returns an error for a module that the repository does not have", func(t *testing.T) {
			t.Parallel()
			absent := []workspace.Package{{Name: "example.com/absent", Version: version.Version{Minor: 1}}}
			_, err := locker.Lock(t.Context(), released(t), absent)
			assert.HasError(t, err, "Lock")
			assert.Contains(t, err.Error(), "the module example.com/absent, which the repository does not have",
				"the error")
		})
	})
}

// released returns the repository of the cases after the version commit of a release of a at
// 0.2.0, which moved the require line of b and wrote the go.sum of b.
func released(tb testing.TB) string {
	tb.Helper()
	root := repository(tb, nil)
	_, err := versioner.Apply(tb.Context(), root, releaseOfA())
	assert.NoError(tb, err, "Apply")
	vcstest.Commit(tb, root, "release")
	return root
}

// releasedWithC returns the repository of the cases with the module c, which requires b, after the
// version commit of a release of a at 0.2.0, of b at 0.1.1 and of c at 0.2.0.
func releasedWithC(tb testing.TB) string {
	tb.Helper()
	root := repository(tb, files.Tree{
		"go.work":  files.Text("go 1.24\n\nuse (\n\t./a\n\t./b\n\t./c\n)\n"),
		"c/go.mod": files.Text(modOf(pathC, pathB+" v0.1.0")),
		"c/c.go":   files.Text(sourceC),
	})
	c := edit(pathC, "c", workspace.Dependency{Name: pathB, Kind: workspace.KindRuntime, Req: "v0.1.1"})
	c.Package.Deps = []workspace.Dependency{{Name: pathB, Kind: workspace.KindRuntime, Req: "v0.1.0"}}
	_, err := versioner.Apply(tb.Context(), root, append(releaseOfA(), c))
	assert.NoError(tb, err, "Apply")
	vcstest.Commit(tb, root, "release")
	return root
}

// pendingA returns the package of a release of a at 0.2.0 whose tag is missing.
func pendingA() []workspace.Package {
	return []workspace.Package{{Name: pathA, Toolchain: toolchain, Dir: "a", Version: version.Version{Minor: 2}}}
}

// rewrite writes content into the file name, a slash-separated path relative to the repository at
// root, for the test tb.
func rewrite(tb testing.TB, root, name, content string) {
	tb.Helper()
	err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), sourcePerm)
	assert.NoError(tb, err, "the write of "+name)
}
