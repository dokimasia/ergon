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
	"go.dokimi.dev/ergon/lang/python/release"
	pyworkspace "go.dokimi.dev/ergon/lang/python/workspace"
)

// The paths of the cases.
const (
	manifestB = "packages/probe_b/pyproject.toml"
	lockFile  = "uv.lock"
)

// projectB is the pyproject.toml of probe-b in the cases, with a version in single quotes, a
// comment, and requirements on probe-a in each section.
const projectB = `[project]
name = "probe-b"
version = '0.1.0'  # the version
dependencies = [
  "probe-a[cli] >=0.1.0 ; python_version >= '3.12'",
  "tqdm>=4",
]
optional-dependencies = { cli = ['probe-a'] }

[dependency-groups]
dev = ["probe-a"]
`

// packageB is the package of probe-b as its discovery finds it in projectB.
var packageB = workspace.Package{
	Name: "probe-b", Toolchain: pyworkspace.Toolchain, Dir: "packages/probe_b", Version: version.Version{Minor: 1},
	Deps: []workspace.Dependency{
		{Name: "probe-a", Kind: workspace.KindRuntime, Req: ">=0.1.0"},
		{Name: "probe-a", Kind: workspace.KindOptional},
		{Name: "probe-a", Kind: workspace.KindDev},
	},
}

func TestVersioner(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrVersion for a version with build metadata", func(t *testing.T) {
			t.Parallel()
			_, err := release.Versioner{}.Resolve(">=1.0", version.Version{Major: 1, Build: "1"})
			assert.ErrorIs(t, err, pyworkspace.ErrVersion, "Resolve")
		})
	})

	t.Run("Rewrite", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrVersion for a version with build metadata", func(t *testing.T) {
			t.Parallel()
			_, err := release.Versioner{}.Rewrite(">=1.0", version.Version{Major: 1, Build: "1"})
			assert.ErrorIs(t, err, pyworkspace.ErrVersion, "Rewrite")
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a release", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, release.Versioner{}.Validate(&packageB, version.Version{Minor: 2}), "Validate")
		})

		t.Run("returns ErrVersion for a pre-release that PEP 440 does not have", func(t *testing.T) {
			t.Parallel()
			err := release.Versioner{}.Validate(&packageB, version.Version{Minor: 2, Pre: "next.0"})
			assert.ErrorIs(t, err, pyworkspace.ErrVersion, "Validate")
		})
	})

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the version and the requirements in place", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{manifestB: files.Text(projectB)})
			f := &tools{}
			got, err := release.Versioner{Tools: f.run}.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2},
				Requirements: []workspace.Dependency{
					{Name: "probe-a", Kind: workspace.KindRuntime, Req: ">=0.2.0"},
					{Name: "probe-a", Kind: workspace.KindDev, Req: ">=0.2.0"},
				},
				Package: packageB,
			}})
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{manifestB}, "the changed files")
			files.HasContent(t, filepath.Join(root, filepath.FromSlash(manifestB)), `[project]
name = "probe-b"
version = '0.2.0'  # the version
dependencies = [
  "probe-a[cli] >=0.2.0 ; python_version >= '3.12'",
  "tqdm>=4",
]
optional-dependencies = { cli = ['probe-a'] }

[dependency-groups]
dev = ["probe-a>=0.2.0"]
`, "the pyproject.toml")
			assert.Empty(t, f.calls, "the runs of uv without uv.lock")
		})

		t.Run("refreshes uv.lock with uv lock", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{manifestB: files.Text(projectB), lockFile: files.Text("old\n")})
			f := &tools{lock: "new\n"}
			got, err := release.Versioner{Tools: f.run}.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2}, Package: packageB,
			}})
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{manifestB, lockFile}, "the changed files")
			assert.Equal(t, f.calls, []call{{dir: root, tool: "uv", args: []string{"lock"}}}, "the runs of uv")
		})

		t.Run("leaves out an unchanged uv.lock", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{manifestB: files.Text(projectB), lockFile: files.Text("old\n")})
			got, err := release.Versioner{Tools: (&tools{}).run}.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2}, Package: packageB,
			}})
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{manifestB}, "the changed files")
		})

		t.Run("writes nothing for edits that change no byte", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{manifestB: files.Text(projectB), lockFile: files.Text("old\n")})
			f := &tools{}
			got, err := release.Versioner{Tools: f.run}.Apply(t.Context(), root, []language.Edit{{
				Version:      packageB.Version,
				Requirements: []workspace.Dependency{{Name: "probe-a", Kind: workspace.KindRuntime, Req: ">=0.1.0"}},
				Package:      packageB,
			}})
			assert.NoError(t, err, "Apply")
			assert.Empty(t, got, "the changed files")
			assert.Empty(t, f.calls, "the runs of uv")
		})

		t.Run("writes the requirements of a project with a dynamic version", func(t *testing.T) {
			t.Parallel()
			text := "[project]\nname = \"a\"\ndynamic = [\"version\"]\ndependencies = [\"probe-a\"]\n"
			root := files.Workspace(t, files.Tree{"a/pyproject.toml": files.Text(text)})
			pkg := workspace.Package{
				Name: "a",
				Dir:  "a",
				Deps: []workspace.Dependency{{Name: "probe-a", Kind: workspace.KindRuntime}},
			}
			_, err := release.Versioner{Tools: (&tools{}).run}.Apply(t.Context(), root, []language.Edit{{
				Version:      version.Version{Minor: 1},
				Requirements: []workspace.Dependency{{Name: "probe-a", Kind: workspace.KindRuntime, Req: ">=0.2.0"}},
				Package:      pkg,
			}})
			assert.NoError(t, err, "Apply")
			files.HasContent(
				t,
				filepath.Join(root, "a", "pyproject.toml"),
				"[project]\nname = \"a\"\ndynamic = [\"version\"]\ndependencies = [\"probe-a>=0.2.0\"]\n",
				"the pyproject.toml",
			)
		})

		t.Run("writes a basic string with escapes in place of a string of more than one line", func(t *testing.T) {
			t.Parallel()
			text := "[project]\nname = \"a\"\nversion = '''0.1.0'''\n" +
				"dependencies = [\"probe-a;\\tpython_version >= \\\"3\\u0001\\\"\"]\n"
			root := files.Workspace(t, files.Tree{"pyproject.toml": files.Text(text)})
			pkg := workspace.Package{
				Name: "a", Dir: ".", Version: version.Version{Minor: 1},
				Deps: []workspace.Dependency{{Name: "probe-a", Kind: workspace.KindRuntime}},
			}
			_, err := release.Versioner{Tools: (&tools{}).run}.Apply(t.Context(), root, []language.Edit{{
				Version:      version.Version{Minor: 2},
				Requirements: []workspace.Dependency{{Name: "probe-a", Kind: workspace.KindRuntime, Req: ">=0.2.0"}},
				Package:      pkg,
			}})
			assert.NoError(t, err, "Apply")
			files.HasContent(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = \"a\"\nversion = \"0.2.0\"\n"+
				"dependencies = [\"probe-a>=0.2.0;\\tpython_version >= \\\"3\\u0001\\\"\"]\n", "the pyproject.toml")
		})

		edits := []struct {
			name     string
			giveTree files.Tree
			giveEdit language.Edit
		}{
			{
				name:     "returns ErrEdit for a project without the version of its package",
				giveTree: files.Tree{"pyproject.toml": files.Text("[project]\nname = \"a\"\n")},
				giveEdit: language.Edit{
					Version: version.Version{Minor: 2},
					Package: workspace.Package{Name: "a", Dir: "."},
				},
			},
			{
				name:     "returns ErrEdit for a project without the requirements of its package",
				giveTree: files.Tree{manifestB: files.Text("[project]\nname = \"probe-b\"\nversion = \"0.1.0\"\n")},
				giveEdit: language.Edit{Version: version.Version{Minor: 2}, Package: packageB},
			},
			{
				name:     "returns ErrEdit for a requirement that its package does not have",
				giveTree: files.Tree{manifestB: files.Text(projectB)},
				giveEdit: language.Edit{
					Version:      packageB.Version,
					Requirements: []workspace.Dependency{{Name: "probe-z", Kind: workspace.KindRuntime, Req: ">=1"}},
					Package:      packageB,
				},
			},
		}
		for _, tt := range edits {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				root := files.Workspace(t, tt.giveTree)
				_, err := release.Versioner{
					Tools: (&tools{}).run,
				}.Apply(
					t.Context(),
					root,
					[]language.Edit{tt.giveEdit},
				)
				assert.ErrorIs(t, err, release.ErrEdit, "Apply")
			})
		}

		t.Run("returns ErrManifest for a pyproject.toml that does not parse", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{manifestB: files.Text("[project\n")})
			_, err := release.Versioner{
				Tools: (&tools{}).run,
			}.Apply(
				t.Context(),
				root,
				[]language.Edit{{Package: packageB}},
			)
			assert.ErrorIs(t, err, pyworkspace.ErrManifest, "Apply")
		})

		t.Run("returns the error of uv lock with uv.lock", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{manifestB: files.Text(projectB), lockFile: files.Text("old\n")})
			got, err := release.Versioner{Tools: (&tools{err: errUV}).run}.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2}, Package: packageB,
			}})
			assert.ErrorIs(t, err, errUV, "Apply")
			assert.Equal(t, got, []string{manifestB, lockFile}, "the changed files")
		})

		failures := []struct {
			name      string
			giveTree  files.Tree
			giveTools *tools
		}{
			{
				name:      "returns the error of a pyproject.toml that does not read",
				giveTree:  files.Tree{},
				giveTools: &tools{},
			},
			{
				name:      "returns the error of a uv.lock that does not read",
				giveTree:  files.Tree{manifestB: files.Text(projectB), lockFile: files.Dir()},
				giveTools: &tools{},
			},
			{
				name:      "returns the error of a uv.lock that uv lock removes",
				giveTree:  files.Tree{manifestB: files.Text(projectB), lockFile: files.Text("old\n")},
				giveTools: &tools{remove: true},
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				root := files.Workspace(t, tt.giveTree)
				_, err := release.Versioner{Tools: tt.giveTools.run}.Apply(t.Context(), root, []language.Edit{{
					Version: version.Version{Minor: 2}, Package: packageB,
				}})
				assert.HasError(t, err, "Apply")
			})
		}

		t.Run("returns the error of a repository that does not open", func(t *testing.T) {
			t.Parallel()
			_, err := release.Versioner{
				Tools: (&tools{}).run,
			}.Apply(
				t.Context(),
				filepath.Join(t.TempDir(), "absent"),
				nil,
			)
			assert.HasError(t, err, "Apply")
		})

		t.Run("returns the file and the error of a pyproject.toml that does not write", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{manifestB: files.Text(projectB).WithMode(0o444)})
			got, err := release.Versioner{Tools: (&tools{}).run}.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2}, Package: packageB,
			}})
			assert.HasError(t, err, "Apply")
			assert.Equal(t, got, []string{manifestB}, "the changed files")
		})
	})
}
