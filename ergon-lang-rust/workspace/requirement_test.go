// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"testing"

	"go.dokimi.dev/assert"
	coreworkspace "go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/rust/workspace"
)

// topManifest is the root of the workspace of the cases. memberManifest is a crate in crates/b that
// requires crates of a path directly and through the dependencies of the workspace.
const (
	topManifest = `[workspace]
members = ["crates/*"]

[workspace.dependencies]
probe-e = { path = "crates/e", version = "1.0" }
alias = { package = "probe-x", path = "./crates/x/", version = "2" }
serde = "1.0"
`
	memberManifest = `[package]
name = "probe-b"
version = "1.0.0"

[dependencies]
probe-a = { path = "../a", version = "1.0" }
serde.workspace = true
probe-e = { workspace = true, optional = true }
renamed = { package = "probe-d", path = "../d", version = "~1" }
alias.workspace = true
outside = { path = "/opt/outside", version = "3" }

[dev-dependencies]
probe-a = { path = "../a" }
probe-e.workspace = true
`
)

// The paths of the manifests of the cases, relative to the root of the repository.
const (
	topFile    = "Cargo.toml"
	memberFile = "crates/b/Cargo.toml"
)

func TestRequirement(t *testing.T) {
	t.Parallel()

	t.Run("Requirements", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the requirements on the crates of a path in the order of the manifest", func(t *testing.T) {
			t.Parallel()
			top, m := parse(t, topFile, topManifest), parse(t, memberFile, memberManifest)
			got, err := workspace.Requirements(top, m, "crates/b")
			assert.NoError(t, err, "Requirements")
			assert.Equal(t, got, []workspace.Requirement{
				{
					Name: "probe-a", Dir: "crates/a", Req: "1.0", File: memberFile, Kind: coreworkspace.KindRuntime,
					Span: spanAfter(t, memberManifest, `probe-a = { path = "../a", version = `, `"1.0"`),
				},
				{
					Name: "probe-e", Dir: "crates/e", Req: "1.0", File: topFile, Kind: coreworkspace.KindOptional,
					Span: spanAfter(t, topManifest, "probe-e = ", `"1.0"`),
				},
				{
					Name: "probe-d", Dir: "crates/d", Req: "~1", File: memberFile, Kind: coreworkspace.KindRuntime,
					Span: spanAfter(t, memberManifest, "renamed = ", `"~1"`),
				},
				{
					Name: "probe-x", Dir: "crates/x", Req: "2", File: topFile, Kind: coreworkspace.KindRuntime,
					Span: spanAfter(t, topManifest, "alias = ", `"2"`),
				},
				{
					Name: "outside", Dir: "/opt/outside", Req: "3", File: memberFile, Kind: coreworkspace.KindRuntime,
					Span: spanAfter(t, memberManifest, "outside = ", `"3"`),
				},
				{Name: "probe-a", Dir: "crates/a", File: memberFile, Kind: coreworkspace.KindDev},
				{
					Name: "probe-e", Dir: "crates/e", Req: "1.0", File: topFile, Kind: coreworkspace.KindDev,
					Span: spanAfter(t, topManifest, "probe-e = ", `"1.0"`),
				},
			}, "the requirements")
		})

		t.Run("reads the paths of the crate at the root from the root", func(t *testing.T) {
			t.Parallel()
			m := parse(t, topFile, "[package]\nname = \"probe\"\n\n[dependencies]\nprobe-a = { path = \"crates/a\" }\n")
			got, err := workspace.Requirements(m, m, ".")
			assert.NoError(t, err, "Requirements")
			assert.Equal(t, got, []workspace.Requirement{
				{Name: "probe-a", Dir: "crates/a", File: topFile, Kind: coreworkspace.KindRuntime},
			}, "the requirements")
		})

		t.Run("returns ErrManifest for an inherited dependency that the workspace does not have", func(t *testing.T) {
			t.Parallel()
			top := parse(t, topFile, "[workspace]\nmembers = [\"crates/*\"]\n")
			m := parse(t, memberFile, "[package]\nname = \"probe-b\"\n\n[dependencies]\nprobe-a.workspace = true\n")
			_, err := workspace.Requirements(top, m, "crates/b")
			assert.ErrorIs(t, err, workspace.ErrManifest, "Requirements")
		})
	})
}
