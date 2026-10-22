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
	jsworkspace "go.dokimi.dev/ergon/lang/javascript/workspace"
)

// pinnedToolchain is the spelling of the js toolchain in configuration, which the test pins.
const pinnedToolchain coreworkspace.Toolchain = "js"

// The package.json files of the packages of the cases: probe-a, and probe-b, which requires probe-a
// in each section, itself and a package of the registry.
const (
	packageA = `{"name": "@probe/a", "version": "0.1.0"}`
	packageB = `{
  "name": "@probe/b",
  "version": "0.2.0",
  "dependencies": {"@probe/a": "^0.1.0", "@probe/b": "*", "left-pad": "^1.3.0"},
  "peerDependencies": {"@probe/a": ">=0.1.0"}
}`
)

// The packages of probe-a and probe-b that [packageA] and [packageB] state, in packages/a and
// packages/b.
var (
	discoveredA = coreworkspace.Package{
		Name: "@probe/a", Toolchain: pinnedToolchain, Dir: "packages/a", Version: version.Version{Minor: 1},
	}
	discoveredB = coreworkspace.Package{
		Name:      "@probe/b",
		Toolchain: pinnedToolchain,
		Dir:       "packages/b",
		Version:   version.Version{Minor: 2},
		Deps: []coreworkspace.Dependency{
			{Name: "@probe/a", Kind: coreworkspace.KindRuntime, Req: "^0.1.0"},
			{Name: "@probe/a", Kind: coreworkspace.KindPeer, Req: ">=0.1.0"},
		},
	}
)

func TestDiscover(t *testing.T) {
	t.Parallel()

	t.Run("Discover", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a package of the toolchain js for each package of an npm workspace", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"package.json": files.Text(`{"name": "probe", "private": true, ` +
					`"workspaces": ["packages/*", "!packages/legacy"], "devDependencies": {"@probe/a": "^0.1.0"}}`),
				"packages/a/package.json":      files.Text(packageA),
				"packages/b/package.json":      files.Text(packageB),
				"packages/legacy/package.json": files.Text(`{"name": "legacy", "version": "1.0.0"}`),
				"packages/empty":               files.Dir(),
				"packages/README.md":           files.Text("# packages\n"),
			})
			got, err := jsworkspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{discoveredA, discoveredB}, "the packages")
		})

		t.Run("returns the requirements of the workspace protocol as pnpm packs them", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"package.json": files.Text(`{"name": "probe", "private": true}`),
				"pnpm-workspace.yaml": files.Text(
					"packages:\n  - packages/**\n  - '!**/fixtures/**'\n",
				),
				"packages/a/package.json":                  files.Text(`{"name": "@probe/a", "version": "1.2.0"}`),
				"packages/a/node_modules/dep/package.json": files.Text(`{"name": "dep", "version": "1.0.0"}`),
				"packages/a/fixtures/x/package.json":       files.Text(`{"name": "x"}`),
				"packages/b/package.json": files.Text(`{"name": "@probe/b", "version": "0.2.0",
  "dependencies": {"@probe/a": "workspace:*"}, "optionalDependencies": {"@probe/a": "workspace:^"},
  "peerDependencies": {"@probe/a": "workspace:~"}, "devDependencies": {"@probe/a": "workspace:../a"}}`),
				"packages/c/package.json": files.Text(
					`{"name": "@probe/c", "version": "0.3.0", "dependencies": {"@probe/a": "workspace:^1.0.0"}}`,
				),
			})
			got, err := jsworkspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			a := coreworkspace.Package{
				Name:      "@probe/a",
				Toolchain: pinnedToolchain,
				Dir:       "packages/a",
				Version:   version.Version{Major: 1, Minor: 2},
			}
			assert.Equal(t, got, []coreworkspace.Package{
				a,
				{
					Name:      "@probe/b",
					Toolchain: pinnedToolchain,
					Dir:       "packages/b",
					Version:   version.Version{Minor: 2},
					Deps: []coreworkspace.Dependency{
						{Name: "@probe/a", Kind: coreworkspace.KindRuntime, Req: "workspace:1.2.0"},
						{Name: "@probe/a", Kind: coreworkspace.KindOptional, Req: "workspace:^1.2.0"},
						{Name: "@probe/a", Kind: coreworkspace.KindPeer, Req: "workspace:~1.2.0"},
						{Name: "@probe/a", Kind: coreworkspace.KindDev, Req: "workspace:1.2.0"},
					},
				},
				{
					Name:      "@probe/c",
					Toolchain: pinnedToolchain,
					Dir:       "packages/c",
					Version:   version.Version{Minor: 3},
					Deps: []coreworkspace.Dependency{
						{Name: "@probe/a", Kind: coreworkspace.KindRuntime, Req: "workspace:^1.0.0"},
					},
				},
			}, "the packages")
		})

		single := []struct {
			name     string
			giveTree files.Tree
		}{
			{
				name:     "returns the package at the root of a repository without a workspace",
				giveTree: files.Tree{"package.json": files.Text(`{"name": "probe", "version": "1.0.0"}`)},
			},
			{
				name: "returns the package at the root of a pnpm-workspace.yaml without packages",
				giveTree: files.Tree{
					"package.json":        files.Text(`{"name": "probe", "version": "1.0.0"}`),
					"pnpm-workspace.yaml": files.Text("onlyBuiltDependencies:\n  - esbuild\n"),
				},
			},
			{
				name:     "drops a v before the version",
				giveTree: files.Tree{"package.json": files.Text(`{"name": "probe", "version": "v1.0.0"}`)},
			},
		}
		for _, tt := range single {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := jsworkspace.Discover(t.Context(), files.Workspace(t, tt.giveTree))
				assert.NoError(t, err, "Discover")
				assert.Equal(t, got, []coreworkspace.Package{
					{Name: "probe", Toolchain: pinnedToolchain, Dir: ".", Version: version.Version{Major: 1}},
				}, "the packages")
			})
		}

		none := []struct {
			name     string
			giveTree files.Tree
		}{
			{name: "returns no package for a repository without package.json", giveTree: files.Tree{}},
			{
				name:     "returns no package for a package.json without a name and a workspace",
				giveTree: files.Tree{"package.json": files.Text(`{"devDependencies": {"@biomejs/biome": "2.5.15"}}`)},
			},
		}
		for _, tt := range none {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := jsworkspace.Discover(t.Context(), files.Workspace(t, tt.giveTree))
				assert.NoError(t, err, "Discover")
				assert.Empty(t, got, "the packages")
			})
		}

		t.Run("returns a private package without a version at the zero version", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"package.json":          files.Text(`{"workspaces": ["apps/*"]}`),
				"apps/web/package.json": files.Text(`{"name": "web", "private": true}`),
			})
			got, err := jsworkspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{Name: "web", Toolchain: pinnedToolchain, Dir: "apps/web", Private: true},
			}, "the packages")
		})

		t.Run("returns each package once for globs that match it twice", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"package.json":            files.Text(`{"workspaces": ["packages/*", "./packages/a", "/packages/a/"]}`),
				"packages/a/package.json": files.Text(packageA),
			})
			got, err := jsworkspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{discoveredA}, "the packages")
		})

		invalid := []struct {
			name     string
			giveTree files.Tree
		}{
			{
				name:     "returns ErrManifest for a package.json that does not parse",
				giveTree: files.Tree{"package.json": files.Text(`{"name": `)},
			},
			{
				name: "returns ErrManifest for a pnpm-workspace.yaml that does not parse",
				giveTree: files.Tree{
					"package.json":        files.Text(`{"name": "probe", "version": "1.0.0"}`),
					"pnpm-workspace.yaml": files.Text("packages: packages/*\n"),
				},
			},
			{
				name: "returns ErrManifest for a package of a workspace without a name",
				giveTree: files.Tree{
					"package.json":            files.Text(`{"workspaces": ["packages/*"]}`),
					"packages/a/package.json": files.Text(`{"version": "1.0.0"}`),
				},
			},
			{
				name: "returns ErrManifest for a package of a workspace that does not parse",
				giveTree: files.Tree{
					"package.json":            files.Text(`{"workspaces": ["packages/*"]}`),
					"packages/a/package.json": files.Text(`["a"]`),
				},
			},
			{
				name:     "returns ErrManifest for a package without a version",
				giveTree: files.Tree{"package.json": files.Text(`{"name": "probe"}`)},
			},
			{
				name:     "returns ErrManifest for a version that Semantic Versioning does not state",
				giveTree: files.Tree{"package.json": files.Text(`{"name": "probe", "version": "1.0"}`)},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := jsworkspace.Discover(t.Context(), files.Workspace(t, tt.giveTree))
				assert.ErrorIs(t, err, jsworkspace.ErrManifest, "Discover")
			})
		}

		failures := []struct {
			name     string
			giveTree files.Tree
			wantText string
		}{
			{
				name:     "returns the error of a glob that does not parse",
				giveTree: files.Tree{"package.json": files.Text(`{"workspaces": ["packages/["]}`)},
				wantText: `workspace: the glob "packages/[" of the workspace: `,
			},
			{
				name: "returns the error of an exclusion that does not parse",
				giveTree: files.Tree{
					"package.json":            files.Text(`{"workspaces": ["packages/*", "!["]}`),
					"packages/a/package.json": files.Text(packageA),
				},
				wantText: `workspace: the glob "packages/*" of the workspace: the exclusion "[": `,
			},
			{
				name:     "returns the error of a package.json at the root that does not read",
				giveTree: files.Tree{"package.json": files.Dir()},
				wantText: "workspace: read package.json: ",
			},
			{
				name: "returns the error of a pnpm-workspace.yaml that does not read",
				giveTree: files.Tree{
					"package.json":        files.Text(`{"name": "probe", "version": "1.0.0"}`),
					"pnpm-workspace.yaml": files.Dir(),
				},
				wantText: "workspace: read pnpm-workspace.yaml: ",
			},
			{
				name: "returns the error of a package.json of a package that does not read",
				giveTree: files.Tree{
					"package.json":            files.Text(`{"workspaces": ["packages/*"]}`),
					"packages/a/package.json": files.Dir(),
				},
				wantText: "workspace: read packages/a/package.json: ",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := jsworkspace.Discover(t.Context(), files.Workspace(t, tt.giveTree))
				assert.HasError(t, err, "Discover")
				assert.HasPrefix(t, err.Error(), tt.wantText, "the text of the error")
			})
		}

		t.Run("returns the error of a repository that does not open", func(t *testing.T) {
			t.Parallel()
			_, err := jsworkspace.Discover(t.Context(), filepath.Join(t.TempDir(), "absent"))
			assert.HasError(t, err, "Discover")
			assert.HasPrefix(t, err.Error(), "workspace: open the repository: ", "the text of the error")
		})
	})
}
