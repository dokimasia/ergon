// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"bytes"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/go/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
	"golang.org/x/mod/module"
	"golang.org/x/mod/sumdb/dirhash"
	modzip "golang.org/x/mod/zip"
)

// The modules of the cases.
const (
	// pathA is a module that requires no other module.
	pathA = "example.com/a"

	// pathB is a module that requires pathA.
	pathB = "example.com/b"

	// pathC is a module that requires pathB.
	pathC = "example.com/c"
)

// The Go files of the modules of the cases. Each module imports the module before it.
const (
	sourceA = "package a\n\n// A returns one.\nfunc A() int { return 1 }\n"
	sourceB = "package b\n\nimport \"example.com/a\"\n\n// B returns A.\nfunc B() int { return a.A() }\n"
	sourceC = "package c\n\nimport \"example.com/b\"\n\n// C returns B.\nfunc C() int { return b.B() }\n"
)

// toolchain is the toolchain of the packages of the cases.
const toolchain workspace.Toolchain = "go"

// zipPerm is the mode of the zip of a module that a case hashes.
const zipPerm fs.FileMode = 0o600

// versioner is the versioner of the cases, which takes its snapshots with the git of vcs.
var versioner = release.Versioner{Snapshot: vcs.Snapshot}

// TestMain runs the tests without the variables and the configuration of git of the environment,
// so that git reads the repositories of the tests alone.
func TestMain(m *testing.M) {
	vcstest.Isolate()
	os.Exit(m.Run())
}

func TestVersioner(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			req  string
			give string
			want language.Resolution
		}{
			{
				name: "returns ResolutionExcluded for a version below the require line",
				req:  "v1.2.0",
				give: "1.1.9",
				want: language.ResolutionExcluded,
			},
			{
				name: "returns ResolutionSelected for the version of the require line",
				req:  "v1.2.0",
				give: "1.2.0",
				want: language.ResolutionSelected,
			},
			{
				name: "returns ResolutionPinned for a version above the require line",
				req:  "v1.2.0",
				give: "1.3.0",
				want: language.ResolutionPinned,
			},
			{
				name: "returns ResolutionPinned for a release above a pseudo-version",
				req:  "v0.0.0-20261007120000-abcdef123456",
				give: "0.1.0",
				want: language.ResolutionPinned,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := versioner.Resolve(tt.req, parse(t, tt.give))
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		for _, req := range []string{"1.2.0", "vnext", ""} {
			t.Run("returns ErrRequirement for the require line "+req, func(t *testing.T) {
				t.Parallel()
				_, err := versioner.Resolve(req, parse(t, "1.0.0"))
				assert.ErrorIs(t, err, release.ErrRequirement, "Resolve")
			})
		}
	})

	t.Run("Rewrite", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the version with a v before it", func(t *testing.T) {
			t.Parallel()
			got, err := versioner.Rewrite("v0.1.0", parse(t, "1.2.0-rc.1"))
			assert.NoError(t, err, "Rewrite")
			assert.Equal(t, got, "v1.2.0-rc.1", "the require line")
		})

		t.Run("returns ErrRequirement for a requirement that is no require line", func(t *testing.T) {
			t.Parallel()
			_, err := versioner.Rewrite("^0.1.0", parse(t, "1.2.0"))
			assert.ErrorIs(t, err, release.ErrRequirement, "Rewrite")
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			path string
			give string
		}{
			{
				name: "returns nil for a release at major 0 of a path without a major element",
				path: pathA,
				give: "0.4.0",
			},
			{
				name: "returns nil for a release at major 1 of a path without a major element",
				path: pathA,
				give: "1.0.0",
			},
			{
				name: "returns nil for a release at major 2 of a path that ends in v2",
				path: pathA + "/v2",
				give: "2.1.0",
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := versioner.Validate(&workspace.Package{Name: tt.path}, parse(t, tt.give))
				assert.NoError(t, err, "Validate")
			})
		}

		refused := []struct {
			name string
			path string
			give string
		}{
			{
				name: "returns ErrMajor for a release at major 2 of a path without a major element",
				path: pathA,
				give: "2.0.0",
			},
			{
				name: "returns ErrMajor for a release at major 1 of a path that ends in v2",
				path: pathA + "/v2",
				give: "1.9.0",
			},
			{
				name: "returns ErrMajor for a release at major 3 of a path that ends in v2",
				path: pathA + "/v2",
				give: "3.0.0",
			},
		}
		for _, tt := range refused {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := versioner.Validate(&workspace.Package{Name: tt.path}, parse(t, tt.give))
				assert.ErrorIs(t, err, release.ErrMajor, "Validate")
				assert.Contains(t, err.Error(), tt.path+" at "+tt.give, "the error")
			})
		}
	})

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the hash of a released module at its tag into the go.sum of a dependent", func(t *testing.T) {
			t.Parallel()
			root := repository(t, nil)
			got, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{"b/go.mod", "go.work", "b/go.sum"}, "the changed files")
			want := modOf(pathB, pathA+" v0.2.0")
			files.HasContent(t, filepath.Join(root, "b", "go.mod"), want, "the go.mod of b")
			vcstest.Commit(t, root, "release")
			a := module.Version{Path: pathA, Version: "v0.2.0"}
			files.HasContent(t, filepath.Join(root, "b", "go.sum"), sumOf(t, root, "a", a), "the go.sum of b")
		})

		t.Run("writes a require line on a module of a directory replace without go mod tidy", func(t *testing.T) {
			t.Parallel()
			replacedB := modOf(pathB, pathA+" v0.1.0") + "\nreplace " + pathA + " => ../a\n"
			root := repository(t, files.Tree{"b/go.mod": files.Text(replacedB)})
			got, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{"b/go.mod", "go.work"}, "the changed files")
			files.Absent(t, filepath.Join(root, "b", "go.sum"), "the go.sum of b")
		})

		t.Run("replaces each version of a module that a go.mod requires in go.work", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{"go.work": files.Text(workOf("replace (\n\t" + pathA +
				" v0.1.0 => ./a\n\texample.com/x v1.0.0 => ../x\n\texample.com/y => ../y\n)\n"))})
			_, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.NoError(t, err, "Apply")
			files.HasContent(t, filepath.Join(root, "go.work"), workOf("replace (\n\t"+pathA+" v0.2.0 => ./a\n"+
				"\texample.com/x v1.0.0 => ../x\n\texample.com/y => ../y\n)\n"), "the go.work")
		})

		t.Run("replaces a version of the module at the root by the directory ./", func(t *testing.T) {
			t.Parallel()
			root := vcstest.Repository(t, files.Tree{
				"go.work":  files.Text("go 1.24\n\nuse (\n\t./\n\t./b\n)\n"),
				"go.mod":   files.Text(modOf(pathA)),
				"a.go":     files.Text(sourceA),
				"b/go.mod": files.Text(modOf(pathB, pathA+" v0.1.0")),
				"b/b.go":   files.Text(sourceB),
			})
			vcstest.Commit(t, root, "add the modules")
			edits := releaseOfA()
			edits[0].Package.Dir = "."
			_, err := versioner.Apply(t.Context(), root, edits)
			assert.NoError(t, err, "Apply")
			files.HasContent(t, filepath.Join(root, "go.work"), "go 1.24\n\nuse (\n\t./\n\t./b\n)\n\nreplace "+pathA+
				" v0.2.0 => ./\n", "the go.work")
		})

		t.Run("writes no go.work for a repository without one", func(t *testing.T) {
			t.Parallel()
			root := vcstest.Repository(t, files.Tree{
				"go.mod": files.Text(modOf(pathA, "example.com/x v1.0.0") + "\nreplace example.com/x => ./x\n"),
			})
			vcstest.Commit(t, root, "add the module")
			edits := []language.Edit{edit(pathA, ".", workspace.Dependency{
				Name: "example.com/x", Kind: workspace.KindRuntime, Req: "v1.1.0",
			})}
			got, err := versioner.Apply(t.Context(), root, edits)
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{"go.mod"}, "the changed files")
			files.Absent(t, filepath.Join(root, "go.work"), "the go.work")
		})

		t.Run("returns the error of a go.work that it cannot read", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(
				t,
				files.Tree{"go.work/inner": files.Text("x\n"), "go.mod": files.Text(modOf(pathA))},
			)
			_, err := versioner.Apply(t.Context(), root, nil)
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "read go.work", "the error")
		})

		t.Run("returns the error of a go.work that the go command refuses", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"go.work": files.Text("go 1.24\n\nunknown ./a\n")})
			_, err := versioner.Apply(t.Context(), root, nil)
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "go.work:3", "the error")
		})

		t.Run("changes no file for edits without requirements", func(t *testing.T) {
			t.Parallel()
			root := repository(t, nil)
			got, err := versioner.Apply(t.Context(), root, releaseOfA()[:1])
			assert.NoError(t, err, "Apply")
			assert.Empty(t, got, "the changed files")
		})

		t.Run("returns an error for an edit of a module that the repository does not have", func(t *testing.T) {
			t.Parallel()
			absent := workspace.Package{Name: "example.com/absent"}
			edits := []language.Edit{{Version: parse(t, "1.0.0"), Package: absent}}
			_, err := versioner.Apply(t.Context(), repository(t, nil), edits)
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "the module example.com/absent, which the repository does not have",
				"the error")
		})

		t.Run("returns the error of the modules of the repository", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"go.mod": files.Text("go 1.24\n")})
			_, err := versioner.Apply(t.Context(), root, nil)
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "go.mod declares no module", "the error")
		})
	})
}

// repository returns a working tree of git with the modules a and b in a workspace, all files
// committed: b requires a at v0.1.0, imports it, and has no go.sum. The files of extra replace the
// files of the same name.
func repository(tb testing.TB, extra files.Tree) string {
	tb.Helper()
	tree := files.Tree{
		"go.work":  files.Text("go 1.24\n\nuse (\n\t./a\n\t./b\n)\n"),
		"LICENSE":  files.Text("MIT\n"),
		"a/go.mod": files.Text(modOf(pathA)),
		"a/a.go":   files.Text(sourceA),
		"b/go.mod": files.Text(modOf(pathB, pathA+" v0.1.0")),
		"b/b.go":   files.Text(sourceB),
	}
	maps.Copy(tree, extra)
	root := vcstest.Repository(tb, tree)
	vcstest.Commit(tb, root, "add the modules")
	return root
}

// workOf returns the go.work of the modules a and b of the cases with rest after its use block, in
// the format of golang.org/x/mod/modfile.
func workOf(rest string) string {
	return "go 1.24\n\nuse (\n\t./a\n\t./b\n)\n\n" + rest
}

// releaseOfA returns the edits of a release of a at 0.2.0, which moves the require line of b.
func releaseOfA() []language.Edit {
	a := workspace.Package{Name: pathA, Toolchain: toolchain, Dir: "a", Version: version.Version{Minor: 1}}
	b := workspace.Package{
		Name: pathB, Toolchain: toolchain, Dir: "b", Version: version.Version{Minor: 1},
		Deps: []workspace.Dependency{{Name: pathA, Kind: workspace.KindRuntime, Req: "v0.1.0"}},
	}
	return []language.Edit{
		{Version: version.Version{Minor: 2}, Package: a},
		{
			Version:      version.Version{Minor: 1, Patch: 1},
			Requirements: []workspace.Dependency{{Name: pathA, Kind: workspace.KindRuntime, Req: "v0.2.0"}},
			Package:      b,
		},
	}
}

// modOf returns the go.mod of the module path with the go line of the cases and a require line for
// each of reqs, a module path and a version, in the format of golang.org/x/mod/modfile.
func modOf(path string, reqs ...string) string {
	mod := "module " + path + "\n\ngo 1.24\n"
	if len(reqs) == 1 {
		return mod + "\nrequire " + reqs[0] + "\n"
	}
	if len(reqs) > 1 {
		return mod + "\nrequire (\n\t" + strings.Join(reqs, "\n\t") + "\n)\n"
	}
	return mod
}

// sumOf returns the lines of go.sum of the module version m in the directory dir of the repository
// at root, from its commit HEAD as the go command reads it from a tag, for the test tb.
func sumOf(tb testing.TB, root, dir string, m module.Version) string {
	tb.Helper()
	var data bytes.Buffer
	assert.NoError(tb, modzip.CreateFromVCS(&data, m, root, "HEAD", dir), "CreateFromVCS")
	file := filepath.Join(tb.TempDir(), "module.zip")
	assert.NoError(tb, os.WriteFile(file, data.Bytes(), zipPerm), "the write of the zip")
	zipHash, err := dirhash.HashZip(file, dirhash.Hash1)
	assert.NoError(tb, err, "HashZip")
	mod := files.Read(tb, filepath.Join(root, filepath.FromSlash(path.Join(dir, "go.mod"))))
	modHash, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(mod)), nil
	})
	assert.NoError(tb, err, "Hash1")
	return m.Path + " " + m.Version + " " + zipHash + "\n" + m.Path + " " + m.Version + "/go.mod " + modHash + "\n"
}

// parse returns the version that s states, and stops the test tb for an s that version.Parse
// refuses.
func parse(tb testing.TB, s string) version.Version {
	tb.Helper()
	v, err := version.Parse(s)
	assert.NoError(tb, err, "Parse of "+s)
	return v
}
