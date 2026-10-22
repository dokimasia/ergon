// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"testing"

	"go.dokimi.dev/assert"
	coreworkspace "go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/rust/workspace"
)

// manifestFile is the name of the file in the errors of the cases.
const manifestFile = "Cargo.toml"

// crateManifest is the Cargo.toml of a crate with every key of a package that a release reads.
const crateManifest = `[package]
name = "probe-b"
version = "0.2.0"
publish = ["crates-io"]

[dependencies]
probe-a = { path = "../a", version = "0.2.0" }
serde = '1.0'
probe-c.path = "../c"
probe-c.version = "=0.2.0"
alias = { package = "probe-d", path = "../d", version = "~0.2", optional = true }
probe-e.workspace = true

[dependencies.probe-f]
path = "../f"
version = "0.2"

[dev-dependencies]
probe-a = { path = "../a" }

[build-dependencies]
probe-g = { workspace = true, optional = true }

[target.'cfg(unix)'.dependencies]
probe-h = { path = "../h", version = "0.1" }

[[bin]]
name = "tool"
path = "src/main.rs"
`

// workspaceManifest is the Cargo.toml at the root of a workspace with every key of a workspace that
// a release reads.
const workspaceManifest = `[workspace]
members = ["crates/*", "tools/gen"]
exclude = ["crates/seeds"]
resolver = "3"

[workspace.package]
version = "1.2.0"
edition = "2024"

[workspace.dependencies]
probe-a = { path = "crates/a", version = "1.2.0" }
serde = { version = "1.0", features = ["derive"] }
`

func TestManifest(t *testing.T) {
	t.Parallel()

	t.Run("ParseManifest", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the keys of a package that a release reads", func(t *testing.T) {
			t.Parallel()
			data := crateManifest
			assert.Equal(t, parse(t, manifestFile, data), &workspace.Manifest{
				Name:        "probe-b",
				Version:     "0.2.0",
				VersionSpan: spanAfter(t, data, `name = "probe-b"`, `"0.2.0"`),
				Package:     true,
				Dependencies: []workspace.Dependency{
					{
						Key: "probe-a", Path: "../a", Version: "0.2.0", Kind: coreworkspace.KindRuntime,
						VersionSpan: spanAfter(t, data, `probe-a = { path = "../a", version = `, `"0.2.0"`),
					},
					{
						Key: "serde", Version: "1.0", Kind: coreworkspace.KindRuntime,
						VersionSpan: spanAfter(t, data, "serde = ", `'1.0'`),
					},
					{
						Key: "probe-c", Path: "../c", Version: "=0.2.0", Kind: coreworkspace.KindRuntime,
						VersionSpan: spanAfter(t, data, "probe-c.version = ", `"=0.2.0"`),
					},
					{
						Key:         "alias",
						Package:     "probe-d",
						Path:        "../d",
						Version:     "~0.2",
						Kind:        coreworkspace.KindOptional,
						VersionSpan: spanAfter(t, data, "alias = ", `"~0.2"`),
					},
					{Key: "probe-e", Kind: coreworkspace.KindRuntime, Workspace: true},
					{
						Key: "probe-f", Path: "../f", Version: "0.2", Kind: coreworkspace.KindRuntime,
						VersionSpan: spanAfter(t, data, "[dependencies.probe-f]", `"0.2"`),
					},
					{Key: "probe-a", Path: "../a", Kind: coreworkspace.KindDev},
					{Key: "probe-g", Kind: coreworkspace.KindOptional, Workspace: true},
					{
						Key: "probe-h", Path: "../h", Version: "0.1", Kind: coreworkspace.KindRuntime,
						VersionSpan: spanAfter(t, data, "probe-h = ", `"0.1"`),
					},
				},
			}, "the manifest")
		})

		t.Run("returns the keys of a workspace that a release reads", func(t *testing.T) {
			t.Parallel()
			data := workspaceManifest
			assert.Equal(t, parse(t, manifestFile, data), &workspace.Manifest{
				Members:              []string{"crates/*", "tools/gen"},
				Exclude:              []string{"crates/seeds"},
				WorkspaceVersion:     "1.2.0",
				WorkspaceVersionSpan: spanAfter(t, data, "[workspace.package]", `"1.2.0"`),
				Workspace:            true,
				WorkspaceDependencies: []workspace.Dependency{
					{
						Key: "probe-a", Path: "crates/a", Version: "1.2.0",
						VersionSpan: spanAfter(t, data, "probe-a = ", `"1.2.0"`),
					},
					{Key: "serde", Version: "1.0", VersionSpan: spanAfter(t, data, "serde = ", `"1.0"`)},
				},
			}, "the manifest")
		})

		inherited := []struct {
			name string
			give string
		}{
			{
				name: "reads version.workspace as a dotted key",
				give: "[package]\nname = \"a\"\nversion.workspace = true\n",
			},
			{
				name: "reads version.workspace in a table",
				give: "[package]\nname = \"a\"\nversion = { workspace = true }\n",
			},
		}
		for _, tt := range inherited {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, parse(t, manifestFile, tt.give),
					&workspace.Manifest{Name: "a", Package: true, InheritsVersion: true}, "the manifest")
			})
		}

		publish := []struct {
			name string
			give string
			want bool
		}{
			{name: "reports a crate private for publish false", give: "false", want: true},
			{name: "reports a crate private for an empty list of registries", give: "[]", want: true},
			{name: "reports a crate private for registries without crates-io", give: `["internal"]`, want: true},
			{name: "reports a crate public for registries with crates-io", give: `["internal", "crates-io"]`},
			{name: "reports a crate public for publish true", give: "true"},
		}
		for _, tt := range publish {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := parse(t, manifestFile, "[package]\nname = \"a\"\npublish = "+tt.give+"\n")
				assert.Equal(t, got.Private, tt.want, "Private")
			})
		}

		invalid := []struct {
			name string
			give string
		}{
			{name: "returns ErrManifest for a file that does not parse", give: "[package\n"},
			{name: "returns ErrManifest for a name that is not a string", give: "[package]\nname = 1\n"},
			{name: "returns ErrManifest for a version that is not a string", give: "[package]\nversion = 1\n"},
			{
				name: "returns ErrManifest for a version.workspace that is not a boolean",
				give: "[package]\nversion.workspace = \"yes\"\n",
			},
			{name: "returns ErrManifest for a publish that is not a boolean", give: "[package]\npublish = \"no\"\n"},
			{name: "returns ErrManifest for a registry that is not a string", give: "[package]\npublish = [1]\n"},
			{name: "returns ErrManifest for members that are not a list", give: "[workspace]\nmembers = \"a\"\n"},
			{name: "returns ErrManifest for an exclude that is not a string", give: "[workspace]\nexclude = [1]\n"},
			{
				name: "returns ErrManifest for a workspace version that is not a string",
				give: "[workspace.package]\nversion = 1\n",
			},
			{name: "returns ErrManifest for a dependency that is not a table", give: "[dependencies]\na = 1\n"},
			{name: "returns ErrManifest for a path that is not a string", give: "[dependencies]\na = { path = 1 }\n"},
			{
				name: "returns ErrManifest for a version requirement that is not a string",
				give: "[dev-dependencies]\na = { version = 1 }\n",
			},
			{
				name: "returns ErrManifest for a renamed crate that is not a string",
				give: "[build-dependencies]\na = { package = 1 }\n",
			},
			{
				name: "returns ErrManifest for a key workspace that is not a boolean",
				give: "[workspace.dependencies]\na = { workspace = \"yes\" }\n",
			},
			{
				name: "returns ErrManifest for a key optional that is not a boolean",
				give: "[target.'cfg(unix)'.dependencies]\na = { optional = \"yes\" }\n",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.ParseManifest(manifestFile, []byte(tt.give))
				assert.ErrorIs(t, err, workspace.ErrManifest, "ParseManifest")
			})
		}
	})
}
