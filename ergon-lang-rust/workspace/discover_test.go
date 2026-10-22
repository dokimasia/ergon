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
	"go.dokimi.dev/ergon/lang/rust/workspace"
)

// pinnedToolchain is the spelling of the toolchain of Rust in configuration, which the test pins.
const pinnedToolchain coreworkspace.Toolchain = "rust"

// pinnedSource is the Source of a crate that inherits the version of its workspace, which the test
// pins.
const pinnedSource = "Cargo.toml workspace.package.version"

// rootCrate, crateA, crateB and crateC are the Cargo.toml files of the workspace of the cases. The
// root crate requires probe-a through the dependencies of the workspace, probe-b has a version of
// its own, and probe-c is private.
const (
	rootCrate = `[workspace]
members = ["crates/*"]
exclude = ["crates/seeds"]

[workspace.package]
version = "0.1.0"

[workspace.dependencies]
probe-a = { path = "crates/a", version = "0.1.0" }

[package]
name = "probe"
version.workspace = true

[dependencies]
probe-a.workspace = true
`
	crateA = "[package]\nname = \"probe-a\"\nversion.workspace = true\n"
	crateB = `[package]
name = "probe-b"
version = "0.3.0"

[dependencies]
probe-a.workspace = true
serde = "1.0"

[dev-dependencies]
probe-a = { path = "../a" }
probe-b = { path = "." }
`
	crateC = `[package]
name = "probe-c"
version.workspace = true
publish = false

[dependencies]
probe-b = { path = "../b", version = "0.3" }
`
)

func TestDiscover(t *testing.T) {
	t.Parallel()

	t.Run("Discover", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a package of the toolchain rust for each crate of the workspace", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"Cargo.toml":              files.Text(rootCrate),
				"crates/a/Cargo.toml":     files.Text(crateA),
				"crates/b/Cargo.toml":     files.Text(crateB),
				"crates/c/Cargo.toml":     files.Text(crateC),
				"crates/seeds/Cargo.toml": files.Text("[package]\nname = \"seeds\"\nversion = \"0.1.0\"\n"),
				"crates/README.md":        files.Text("# crates\n"),
			})
			got, err := workspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{
					Name:      "probe",
					Toolchain: pinnedToolchain,
					Dir:       ".",
					Source:    pinnedSource,
					Version:   version.Version{Minor: 1},
					Deps: []coreworkspace.Dependency{
						{Name: "probe-a", Kind: coreworkspace.KindRuntime, Req: "0.1.0"},
					},
				},
				{
					Name:      "probe-a",
					Toolchain: pinnedToolchain,
					Dir:       "crates/a",
					Source:    pinnedSource,
					Version:   version.Version{Minor: 1},
				},
				{
					Name:      "probe-b",
					Toolchain: pinnedToolchain,
					Dir:       "crates/b",
					Version:   version.Version{Minor: 3},
					Deps: []coreworkspace.Dependency{
						{Name: "probe-a", Kind: coreworkspace.KindRuntime, Req: "0.1.0"},
						{Name: "probe-a", Kind: coreworkspace.KindDev},
					},
				},
				{
					Name:      "probe-c",
					Toolchain: pinnedToolchain,
					Dir:       "crates/c",
					Source:    pinnedSource,
					Version:   version.Version{Minor: 1},
					Private:   true,
					Deps: []coreworkspace.Dependency{
						{Name: "probe-b", Kind: coreworkspace.KindRuntime, Req: "0.3"},
					},
				},
			}, "the packages")
		})

		t.Run("adds the crates of the paths of the members inside the root of the workspace", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"Cargo.toml": files.Text("[workspace]\nmembers = [\"a\"]\nexclude = [\"vendor\"]\n"),
				"a/Cargo.toml": files.Text("[package]\nname = \"probe-a\"\nversion = \"1.0.0\"\n\n[dependencies]\n" +
					"probe-b = { path = \"../b\", version = \"1\" }\n" +
					"probe-v = { path = \"../vendor/v\", version = \"1\" }\n" +
					"probe-o = { path = \"../../outside\", version = \"1\" }\n"),
				"b/Cargo.toml": files.Text("[package]\nname = \"probe-b\"\nversion = \"1.1.0\"\n"),
			})
			got, err := workspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{
					Name:      "probe-a",
					Toolchain: pinnedToolchain,
					Dir:       "a",
					Version:   version.Version{Major: 1},
					Deps:      []coreworkspace.Dependency{{Name: "probe-b", Kind: coreworkspace.KindRuntime, Req: "1"}},
				},
				{Name: "probe-b", Toolchain: pinnedToolchain, Dir: "b", Version: version.Version{Major: 1, Minor: 1}},
			}, "the packages")
		})

		t.Run("returns the crate at the root alone without a workspace", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"Cargo.toml": files.Text("[package]\nname = \"probe\"\nversion = \"2.0.0\"\n\n[dependencies]\n" +
					"probe-a = { path = \"a\", version = \"1\" }\n"),
				"a/Cargo.toml": files.Text("[package]\nname = \"probe-a\"\nversion = \"1.0.0\"\n"),
			})
			got, err := workspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{Name: "probe", Toolchain: pinnedToolchain, Dir: ".", Version: version.Version{Major: 2}},
			}, "the packages")
		})

		t.Run("returns each crate once for members that match it twice", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"Cargo.toml":   files.Text("[workspace]\nmembers = [\"a\", \"a*\", \"./a/\"]\n"),
				"a/Cargo.toml": files.Text("[package]\nname = \"probe-a\"\nversion = \"1.0.0\"\n"),
			})
			got, err := workspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{Name: "probe-a", Toolchain: pinnedToolchain, Dir: "a", Version: version.Version{Major: 1}},
			}, "the packages")
		})

		t.Run("reads a crate without a version as a private crate at the zero version", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"Cargo.toml": files.Text("[package]\nname = \"probe\"\n")})
			got, err := workspace.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Equal(t, got, []coreworkspace.Package{
				{Name: "probe", Toolchain: pinnedToolchain, Dir: ".", Private: true},
			}, "the packages")
		})

		none := []struct {
			name     string
			giveTree files.Tree
		}{
			{name: "returns no package for a repository without Cargo.toml", giveTree: files.Tree{}},
			{
				name:     "returns no package for a Cargo.toml without a package and a workspace",
				giveTree: files.Tree{"Cargo.toml": files.Text("[patch.crates-io]\nserde = { path = \"serde\" }\n")},
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

		invalid := []struct {
			name     string
			giveTree files.Tree
		}{
			{
				name:     "returns ErrManifest for a Cargo.toml that does not parse",
				giveTree: files.Tree{"Cargo.toml": files.Text("[package\n")},
			},
			{
				name: "returns ErrManifest for a member that matches no directory",
				giveTree: files.Tree{
					"Cargo.toml": files.Text("[workspace]\nmembers = [\"crates/*\"]\n"),
					"crates":     files.Dir(),
				},
			},
			{
				name: "returns ErrManifest for a member without Cargo.toml",
				giveTree: files.Tree{
					"Cargo.toml":  files.Text("[workspace]\nmembers = [\"a\"]\n"),
					"a/README.md": files.Text("# a\n"),
				},
			},
			{
				name: "returns ErrManifest for a member without a package",
				giveTree: files.Tree{
					"Cargo.toml":   files.Text("[workspace]\nmembers = [\"a\"]\n"),
					"a/Cargo.toml": files.Text("[dependencies]\nserde = \"1.0\"\n"),
				},
			},
			{
				name: "returns ErrManifest for the crate of a path without Cargo.toml",
				giveTree: files.Tree{
					"Cargo.toml": files.Text("[workspace]\nmembers = [\".\"]\n\n[package]\nname = \"probe\"\n\n" +
						"[dependencies]\nprobe-a = { path = \"a\" }\n"),
					"a/README.md": files.Text("# a\n"),
				},
			},
			{
				name:     "returns ErrManifest for a crate without a name",
				giveTree: files.Tree{"Cargo.toml": files.Text("[package]\nversion = \"0.1.0\"\n")},
			},
			{
				name:     "returns ErrManifest for a crate whose version does not parse",
				giveTree: files.Tree{"Cargo.toml": files.Text("[package]\nname = \"probe\"\nversion = \"1.0\"\n")},
			},
			{
				name: "returns ErrManifest for an inherited version that the workspace does not have",
				giveTree: files.Tree{
					"Cargo.toml": files.Text("[workspace]\n\n[package]\nname = \"probe\"\nversion.workspace = true\n"),
				},
			},
			{
				name: "returns ErrManifest for a workspace version that does not parse",
				giveTree: files.Tree{
					"Cargo.toml": files.Text("[workspace.package]\nversion = \"one\"\n\n[package]\nname = \"probe\"\n" +
						"version.workspace = true\n"),
				},
			},
			{
				name: "returns ErrManifest for an inherited dependency that the workspace does not have",
				giveTree: files.Tree{
					"Cargo.toml": files.Text("[workspace]\n\n[package]\nname = \"probe\"\n\n" +
						"[dependencies]\nprobe-a.workspace = true\n"),
				},
			},
			{
				name: "returns ErrManifest for an inherited dependency of a crate without a workspace",
				giveTree: files.Tree{
					"Cargo.toml": files.Text(
						"[package]\nname = \"probe\"\n\n[dependencies]\nprobe-a.workspace = true\n",
					),
				},
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
				name:     "returns the error of a member that does not parse as a glob",
				giveTree: files.Tree{"Cargo.toml": files.Text("[workspace]\nmembers = [\"a/[\"]\n")},
			},
			{
				name: "returns the error of a Cargo.toml of a member that does not read",
				giveTree: files.Tree{
					"Cargo.toml":   files.Text("[workspace]\nmembers = [\"a\"]\n"),
					"a/Cargo.toml": files.Dir(),
				},
			},
			{
				name:     "returns the error of a Cargo.toml at the root that does not read",
				giveTree: files.Tree{"Cargo.toml": files.Dir()},
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
	})
}
