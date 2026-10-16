// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/version"
	coreworkspace "go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/python/workspace"
)

// pinnedToolchain is the spelling of the toolchain of Python in configuration, which the test pins.
const pinnedToolchain coreworkspace.Toolchain = "python"

// The pyproject.toml files of the workspace of the cases: a virtual root that requires probe-a, and
// the members probe-a and probe-b, which requires probe-a.
const (
	rootProject = `[project]
name = "probe"
version = "0.1.0"
dependencies = ["probe-a", "tqdm>=4"]

[tool.uv]
package = false

[tool.uv.sources]
probe-a = { workspace = true }

[tool.uv.workspace]
members = ["packages/*"]
exclude = ["packages/seeds"]
`
	projectA = "[project]\nname = \"Probe_A\"\nversion = \"0.1\"\n"
	projectB = "[project]\nname = \"probe-b\"\nversion = \"0.1.0\"\n" +
		"dependencies = [\"probe-a>=0.1.0\", \"probe-b[cli]\", \"probe-a @ file:///probe_a\"]\n"
)

func TestDiscover(t *testing.T) {
	t.Parallel()

	t.Run("Discover", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a package of the toolchain python for each project of the workspace", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"pyproject.toml":                  files.Text(rootProject),
				"packages/probe_a/pyproject.toml": files.Text(projectA),
				"packages/probe_b/pyproject.toml": files.Text(projectB),
				"packages/seeds/pyproject.toml":   files.Text("[project]\nname = \"seeds\"\nversion = \"0.1.0\"\n"),
				"packages/empty":                  files.Dir(),
				"packages/README.md":              files.Text("# packages\n"),
			})
			got, err := workspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{
					Name:      "probe",
					Toolchain: pinnedToolchain,
					Dir:       ".",
					Version:   version.Version{Minor: 1},
					Private:   true,
					Deps:      []coreworkspace.Dependency{{Name: "probe-a", Kind: coreworkspace.KindRuntime}},
				},
				{
					Name:      "probe-a",
					Toolchain: pinnedToolchain,
					Dir:       "packages/probe_a",
					Version:   version.Version{Minor: 1},
				},
				{
					Name:      "probe-b",
					Toolchain: pinnedToolchain,
					Dir:       "packages/probe_b",
					Version:   version.Version{Minor: 1},
					Deps: []coreworkspace.Dependency{
						{Name: "probe-a", Kind: coreworkspace.KindRuntime, Req: ">=0.1.0"},
					},
				},
			}, "the packages")
		})

		none := []struct {
			name     string
			giveTree files.Tree
		}{
			{name: "returns no package for a repository without pyproject.toml", giveTree: files.Tree{}},
			{
				name:     "returns no package for a pyproject.toml without a project and a workspace",
				giveTree: files.Tree{"pyproject.toml": files.Text("[tool.ruff]\nline-length = 100\n")},
			},
		}
		for _, tt := range none {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := workspace.Discover(t.Context(), files.Workspace(t, tt.giveTree))
				assert.NoError(t, err, "Discover")
				assert.Empty(t, got, "the packages")
			})
		}

		t.Run("returns each project once for globs that match it twice", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"pyproject.toml": files.Text(
					"[project]\nname = \"probe\"\nversion = \"1.0.0\"\n\n[tool.uv.workspace]\nmembers = [\".\", \"a\", \"a*\"]\n",
				),
				"a/pyproject.toml": files.Text(projectA),
			})
			got, err := workspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{Name: "probe", Toolchain: pinnedToolchain, Dir: ".", Version: version.Version{Major: 1}},
				{Name: "probe-a", Toolchain: pinnedToolchain, Dir: "a", Version: version.Version{Minor: 1}},
			}, "the packages")
		})

		t.Run("returns the members of a workspace without a project at its root", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"pyproject.toml":   files.Text("[tool.uv.workspace]\nmembers = [\"./a/\"]\n"),
				"a/pyproject.toml": files.Text(projectA),
			})
			got, err := workspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{Name: "probe-a", Toolchain: pinnedToolchain, Dir: "a", Version: version.Version{Minor: 1}},
			}, "the packages")
		})

		dynamic := []struct {
			name     string
			giveTree files.Tree
			want     version.Version
		}{
			{
				name:     "reads a dynamic version from the highest heading of the changelog",
				giveTree: files.Tree{"CHANGELOG.md": files.Text("# a\n\n## 1.2.0\n\n## 1.10.0\n")},
				want:     version.Version{Major: 1, Minor: 10},
			},
			{name: "reads a dynamic version without a changelog as the zero version", giveTree: files.Tree{}},
		}
		for _, tt := range dynamic {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				tt.giveTree["pyproject.toml"] = files.Text("[project]\nname = \"a\"\ndynamic = [\"version\"]\n")
				got, err := workspace.Discover(t.Context(), files.Workspace(t, tt.giveTree))
				assert.NoError(t, err, "Discover")
				assert.Equal(t, got, []coreworkspace.Package{
					{Name: "a", Toolchain: pinnedToolchain, Dir: ".", Version: tt.want},
				}, "the packages")
			})
		}

		invalid := []struct {
			name     string
			giveTree files.Tree
		}{
			{
				name:     "returns ErrManifest for a pyproject.toml that does not parse",
				giveTree: files.Tree{"pyproject.toml": files.Text("[project\n")},
			},
			{
				name: "returns ErrManifest for a member without a project",
				giveTree: files.Tree{
					"pyproject.toml":   files.Text("[tool.uv.workspace]\nmembers = [\"a\"]\n"),
					"a/pyproject.toml": files.Text("[tool.ruff]\n"),
				},
			},
			{
				name:     "returns ErrManifest for a project without a name",
				giveTree: files.Tree{"pyproject.toml": files.Text("[project]\nversion = \"0.1.0\"\n")},
			},
			{
				name:     "returns ErrManifest for a project without a version",
				giveTree: files.Tree{"pyproject.toml": files.Text("[project]\nname = \"a\"\n")},
			},
			{
				name: "returns ErrManifest for a version that Semantic Versioning does not have",
				giveTree: files.Tree{
					"pyproject.toml": files.Text("[project]\nname = \"a\"\nversion = \"1.0.post1\"\n"),
				},
			},
			{
				name:     "returns ErrManifest for a version that does not parse",
				giveTree: files.Tree{"pyproject.toml": files.Text("[project]\nname = \"a\"\nversion = \"one\"\n")},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.Discover(t.Context(), files.Workspace(t, tt.giveTree))
				assert.ErrorIs(t, err, workspace.ErrManifest, "Discover")
			})
		}

		failures := []struct {
			name     string
			giveTree files.Tree
		}{
			{
				name:     "returns the error of a glob of the members that does not parse",
				giveTree: files.Tree{"pyproject.toml": files.Text("[tool.uv.workspace]\nmembers = [\"a/[\"]\n")},
			},
			{
				name: "returns the error of a glob of the exclude that does not parse",
				giveTree: files.Tree{
					"pyproject.toml":   files.Text("[tool.uv.workspace]\nmembers = [\"a\"]\nexclude = [\"[\"]\n"),
					"a/pyproject.toml": files.Text(projectA),
				},
			},
			{
				name: "returns the error of a changelog that does not read",
				giveTree: files.Tree{
					"pyproject.toml": files.Text("[project]\nname = \"a\"\ndynamic = [\"version\"]\n"),
					"CHANGELOG.md":   files.Dir(),
				},
			},
			{
				name: "returns the error of a pyproject.toml of a member that does not read",
				giveTree: files.Tree{
					"pyproject.toml":   files.Text("[tool.uv.workspace]\nmembers = [\"a\"]\n"),
					"a/pyproject.toml": files.Dir(),
				},
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.Discover(t.Context(), files.Workspace(t, tt.giveTree))
				assert.HasError(t, err, "Discover")
			})
		}

		t.Run("returns the error of a repository that does not open", func(t *testing.T) {
			t.Parallel()
			_, err := workspace.Discover(t.Context(), filepath.Join(t.TempDir(), "absent"))
			assert.HasError(t, err, "Discover")
		})

		t.Run("returns the error of a pyproject.toml at the root that does not read", func(t *testing.T) {
			t.Parallel()
			_, err := workspace.Discover(t.Context(), files.Workspace(t, files.Tree{"pyproject.toml": files.Dir()}))
			assert.HasError(t, err, "Discover")
		})
	})
}
