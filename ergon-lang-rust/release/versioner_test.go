// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/rust/release"
	rustworkspace "go.dokimi.dev/ergon/lang/rust/workspace"
)

// The files of the workspace of the cases: probe-a inherits the version of the workspace, and
// probe-b requires it through the dependencies of the workspace and requires probe-c by its path.
const (
	lockFile     = "Cargo.lock"
	rootManifest = `[workspace]
members = ["crates/*"]

[workspace.package]
version = "0.1.0"

[workspace.dependencies]
probe-a = { path = "crates/a", version = "0.1.0" }
`
	manifestA = "[package]\nname = \"probe-a\"\nversion.workspace = true\n"
	manifestB = `[package]
name = "probe-b"
version = '0.2.0' # the version

[dependencies]
probe-a.workspace = true
probe-c = { path = "../c", version = "0.1" }
`
	manifestC  = "[package]\nname = \"probe-c\"\nversion = \"0.1.0\"\n"
	lockBefore = "# the lock of 0.1.0\n"
	lockAfter  = "# the lock of 0.2.0\n"
)

// The packages of the workspace of the cases, as the discovery of the toolchain returns them.
var (
	packageA = workspace.Package{
		Name: "probe-a", Toolchain: pinnedToolchain, Dir: "crates/a", Source: rustworkspace.VersionSource,
		Version: version.Version{Minor: 1},
	}
	packageB = workspace.Package{
		Name: "probe-b", Toolchain: pinnedToolchain, Dir: "crates/b", Version: version.Version{Minor: 2},
		Deps: []workspace.Dependency{
			{Name: "probe-a", Kind: workspace.KindRuntime, Req: "0.1.0"},
			{Name: "probe-c", Kind: workspace.KindRuntime, Req: "0.1"},
		},
	}
	packageC = workspace.Package{
		Name: "probe-c", Toolchain: pinnedToolchain, Dir: "crates/c", Version: version.Version{Minor: 1},
	}
)

// tree returns the files of the workspace of the cases, with Cargo.lock.
func tree() files.Tree {
	return files.Tree{
		"Cargo.toml":          files.Text(rootManifest),
		"crates/a/Cargo.toml": files.Text(manifestA),
		"crates/b/Cargo.toml": files.Text(manifestB),
		"crates/c/Cargo.toml": files.Text(manifestC),
		lockFile:              files.Text(lockBefore),
	}
}

func TestVersioner(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a pre-release with build metadata", func(t *testing.T) {
			t.Parallel()
			assert.NoError(
				t,
				release.Versioner{}.Validate(&packageA, version.Version{Major: 1, Pre: "rc.1", Build: "7"}),
				"Validate",
			)
		})
	})

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the version and the requirements of each edit in place", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, tree())
			f := &commands{do: func(c call) error {
				files.Write(t, c.dir, files.Tree{lockFile: files.Text(lockAfter)})
				return nil
			}}
			got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{
				{Package: packageA, Version: version.Version{Minor: 2}},
				{
					Package: packageB, Version: version.Version{Minor: 2, Patch: 1},
					Requirements: []workspace.Dependency{
						{Name: "probe-a", Kind: workspace.KindRuntime, Req: "0.2.0"},
						{Name: "probe-c", Kind: workspace.KindRuntime, Req: "0.2"},
					},
				},
				{Package: packageC, Version: version.Version{Minor: 2}},
			})
			assert.NoError(t, err, "Apply")
			expect.That(t, got).Equal([]string{"Cargo.toml", "crates/b/Cargo.toml", "crates/c/Cargo.toml", lockFile},
				"the changed files")
			files.Equal(t, os.DirFS(root), files.Tree{
				"Cargo.toml": files.Text(`[workspace]
members = ["crates/*"]

[workspace.package]
version = "0.2.0"

[workspace.dependencies]
probe-a = { path = "crates/a", version = "0.2.0" }
`),
				"crates/a/Cargo.toml": files.Text(manifestA),
				"crates/b/Cargo.toml": files.Text(`[package]
name = "probe-b"
version = '0.2.1' # the version

[dependencies]
probe-a.workspace = true
probe-c = { path = "../c", version = "0.2" }
`),
				"crates/c/Cargo.toml": files.Text("[package]\nname = \"probe-c\"\nversion = \"0.2.0\"\n"),
				lockFile:              files.Text(lockAfter),
			}, "the workspace")
			expect.Equal(t, f.calls, []call{{dir: root, name: "cargo", args: []string{"update", "--workspace"}}},
				"the runs of cargo")
		})

		t.Run("writes the version that crates share once", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"Cargo.toml": files.Text("[workspace]\nmembers = [\"crates/*\"]\n\n[workspace.package]\n" +
					"version = \"0.1.0\"\n\n[package]\nname = \"probe\"\nversion.workspace = true\n"),
				"crates/a/Cargo.toml": files.Text(manifestA),
			})
			probe := workspace.Package{
				Name: "probe", Toolchain: pinnedToolchain, Dir: ".", Source: rustworkspace.VersionSource,
				Version: version.Version{Minor: 1},
			}
			got, err := release.Versioner{Command: (&commands{}).run}.Apply(t.Context(), root, []language.Edit{
				{Package: probe, Version: version.Version{Minor: 2}},
				{Package: packageA, Version: version.Version{Minor: 2}},
			})
			assert.NoError(t, err, "Apply")
			expect.That(t, got).Equal([]string{"Cargo.toml"}, "the changed files")
			files.HasContent(t, filepath.Join(root, "Cargo.toml"), "[workspace]\nmembers = [\"crates/*\"]\n\n"+
				"[workspace.package]\nversion = \"0.2.0\"\n\n[package]\nname = \"probe\"\nversion.workspace = true\n",
				"Cargo.toml")
		})

		t.Run("returns no lockfile that the refresh leaves as it is", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, tree())
			got, err := release.Versioner{Command: (&commands{}).run}.Apply(t.Context(), root, []language.Edit{
				{Package: packageC, Version: version.Version{Minor: 2}},
			})
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{"crates/c/Cargo.toml"}, "the changed files")
		})

		t.Run("refreshes no lockfile for edits that change no file", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, tree())
			f := &commands{}
			got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{
				{
					Package: packageB, Version: packageB.Version,
					Requirements: []workspace.Dependency{{Name: "probe-c", Kind: workspace.KindRuntime, Req: "0.1"}},
				},
			})
			assert.NoError(t, err, "Apply")
			expect.That(t, got).Empty("the changed files")
			expect.That(t, f.calls).Empty("the runs of cargo")
		})

		t.Run("keeps the delimiters of a string of more than one line", func(t *testing.T) {
			t.Parallel()
			give := tree()
			give["crates/c/Cargo.toml"] = files.Text("[package]\nname = \"probe-c\"\nversion = \"\"\"0.1.0\"\"\"\n")
			root := files.Workspace(t, give)
			_, err := release.Versioner{Command: (&commands{}).run}.Apply(t.Context(), root, []language.Edit{
				{Package: packageC, Version: version.Version{Minor: 2}},
			})
			assert.NoError(t, err, "Apply")
			files.HasContent(t, filepath.Join(root, "crates", "c", "Cargo.toml"),
				"[package]\nname = \"probe-c\"\nversion = \"\"\"0.2.0\"\"\"\n", "the Cargo.toml of probe-c")
		})

		t.Run("writes no file whose content the edits leave as it is", func(t *testing.T) {
			t.Parallel()
			give := tree()
			give["crates/c/Cargo.toml"] = files.Text("[package]\nname = \"probe-c\"\nversion = \"0.2.0\"\n")
			root := files.Workspace(t, give)
			f := &commands{}
			got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{
				{Package: packageC, Version: version.Version{Minor: 2}},
			})
			assert.NoError(t, err, "Apply")
			expect.That(t, got).Empty("the changed files")
			expect.That(t, f.calls).Empty("the runs of cargo")
		})

		t.Run("refreshes no lockfile in a workspace without Cargo.lock", func(t *testing.T) {
			t.Parallel()
			give := tree()
			delete(give, lockFile)
			root := files.Workspace(t, give)
			f := &commands{}
			got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{
				{Package: packageC, Version: version.Version{Minor: 2}},
			})
			assert.NoError(t, err, "Apply")
			expect.That(t, got).Equal([]string{"crates/c/Cargo.toml"}, "the changed files")
			expect.That(t, f.calls).Empty("the runs of cargo")
		})

		edits := []struct {
			name     string
			giveTree files.Tree
			give     []language.Edit
		}{
			{
				name:     "returns ErrEdit for a new version of a crate without package.version",
				giveTree: files.Tree{"crates/c/Cargo.toml": files.Text("[package]\nname = \"probe-c\"\n")},
				give:     []language.Edit{{Package: packageC, Version: version.Version{Minor: 2}}},
			},
			{
				name:     "returns ErrEdit for an inherited version that the root does not have",
				giveTree: files.Tree{"Cargo.toml": files.Text("[workspace]\nmembers = [\"crates/*\"]\n")},
				give:     []language.Edit{{Package: packageA, Version: version.Version{Minor: 2}}},
			},
			{
				name: "returns ErrEdit for a crate whose requirements on the crates of the workspace changed",
				giveTree: files.Tree{
					"crates/b/Cargo.toml": files.Text("[package]\nname = \"probe-b\"\nversion = \"0.2.0\"\n"),
				},
				give: []language.Edit{{Package: packageB, Version: version.Version{Minor: 3}}},
			},
			{
				name: "returns ErrEdit for a requirement that the crate does not have",
				give: []language.Edit{{
					Package: packageB, Version: packageB.Version,
					Requirements: []workspace.Dependency{{Name: "probe-c", Kind: workspace.KindDev, Req: "0.2"}},
				}},
			},
			{
				name: "returns ErrEdit for a requirement of a dependency without a version",
				giveTree: files.Tree{
					"crates/b/Cargo.toml": files.Text("[package]\nname = \"probe-b\"\nversion = \"0.2.0\"\n\n" +
						"[dependencies]\nprobe-a.workspace = true\nprobe-c = { path = \"../c\" }\n"),
				},
				give: []language.Edit{{
					Package: packageB, Version: packageB.Version,
					Requirements: []workspace.Dependency{{Name: "probe-c", Kind: workspace.KindRuntime, Req: "0.2"}},
				}},
			},
			{
				name: "returns ErrEdit for two versions of one inherited version",
				giveTree: files.Tree{
					"crates/d/Cargo.toml": files.Text("[package]\nname = \"probe-d\"\nversion.workspace = true\n"),
				},
				give: []language.Edit{
					{Package: packageA, Version: version.Version{Minor: 2}},
					{
						Package: workspace.Package{
							Name:      "probe-d",
							Toolchain: pinnedToolchain,
							Dir:       "crates/d",
							Source:    rustworkspace.VersionSource,
							Version:   version.Version{Minor: 1},
						},
						Version: version.Version{Minor: 3},
					},
				},
			},
			{
				name: "returns ErrEdit for two rewrites of one inherited requirement",
				giveTree: files.Tree{
					"crates/d/Cargo.toml": files.Text("[package]\nname = \"probe-d\"\nversion = \"0.1.0\"\n\n" +
						"[dependencies]\nprobe-a.workspace = true\n"),
				},
				give: []language.Edit{
					{
						Package: packageB,
						Version: packageB.Version,
						Requirements: []workspace.Dependency{
							{Name: "probe-a", Kind: workspace.KindRuntime, Req: "0.2.0"},
						},
					},
					{
						Package: workspace.Package{
							Name:      "probe-d",
							Toolchain: pinnedToolchain,
							Dir:       "crates/d",
							Version:   version.Version{Minor: 1},
							Deps: []workspace.Dependency{
								{Name: "probe-a", Kind: workspace.KindRuntime, Req: "0.1.0"},
							},
						},
						Version: version.Version{Minor: 1},
						Requirements: []workspace.Dependency{
							{Name: "probe-a", Kind: workspace.KindRuntime, Req: "0.3.0"},
						},
					},
				},
			},
		}
		for _, tt := range edits {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				give := tree()
				maps.Copy(give, tt.giveTree)
				root := files.Workspace(t, give)
				got, err := release.Versioner{Command: (&commands{}).run}.Apply(t.Context(), root, tt.give)
				expect.That(t, err).ErrorIs(release.ErrEdit, "Apply")
				expect.That(t, got).Empty("the changed files")
				files.Equal(t, os.DirFS(root), give, "the workspace")
			})
		}

		manifests := []struct {
			name     string
			giveTree files.Tree
		}{
			{
				name:     "returns ErrManifest for a Cargo.toml that does not parse",
				giveTree: files.Tree{"crates/c/Cargo.toml": files.Text("[package\n")},
			},
			{
				name:     "returns ErrManifest for an inherited dependency that the root does not have",
				giveTree: files.Tree{"Cargo.toml": files.Text("[workspace]\nmembers = [\"crates/*\"]\n")},
			},
		}
		for _, tt := range manifests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				give := tree()
				maps.Copy(give, tt.giveTree)
				_, err := release.Versioner{Command: (&commands{}).run}.Apply(t.Context(), files.Workspace(t, give),
					[]language.Edit{{Package: packageB, Version: version.Version{Minor: 3}}, {Package: packageC}})
				assert.ErrorIs(t, err, rustworkspace.ErrManifest, "Apply")
			})
		}

		reads := []struct {
			name     string
			giveTree files.Tree
		}{
			{
				name:     "returns the error of a Cargo.toml of a crate that does not read",
				giveTree: files.Tree{"crates/c/Cargo.toml": files.Dir()},
			},
			{
				name:     "returns the error of a Cargo.toml at the root that does not read",
				giveTree: files.Tree{"Cargo.toml": files.Dir()},
			},
		}
		for _, tt := range reads {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				give := tree()
				maps.Copy(give, tt.giveTree)
				got, err := release.Versioner{Command: (&commands{}).run}.Apply(t.Context(), files.Workspace(t, give),
					[]language.Edit{{Package: packageC, Version: version.Version{Minor: 2}}})
				expect.That(t, err).HasError("Apply")
				expect.That(t, got).Empty("the changed files")
			})
		}

		t.Run("returns the files that it wrote before a write that fails", func(t *testing.T) {
			t.Parallel()
			give := tree()
			give["crates/c/Cargo.toml"] = files.Text(manifestC).WithMode(0o444)
			root := files.Workspace(t, give)
			got, err := release.Versioner{Command: (&commands{}).run}.Apply(t.Context(), root, []language.Edit{
				{Package: packageB, Version: version.Version{Minor: 3}, Requirements: []workspace.Dependency{
					{Name: "probe-c", Kind: workspace.KindRuntime, Req: "0.2"},
				}},
				{Package: packageC, Version: version.Version{Minor: 2}},
			})
			expect.That(t, err).HasError("Apply")
			expect.That(t, got).Equal([]string{"crates/b/Cargo.toml"}, "the changed files")
		})

		lockfiles := []struct {
			name     string
			giveLock files.Entry
			giveDo   func(t *testing.T) func(c call) error
		}{
			{
				name:     "returns the error of a Cargo.lock that does not read",
				giveLock: files.Dir(),
				giveDo:   func(*testing.T) func(call) error { return nil },
			},
			{
				name:     "returns the error of the refresh of Cargo.lock",
				giveLock: files.Text(lockBefore),
				giveDo:   func(*testing.T) func(call) error { return func(call) error { return errProgram } },
			},
			{
				name:     "returns the error of a Cargo.lock that the refresh leaves unreadable",
				giveLock: files.Text(lockBefore),
				giveDo: func(t *testing.T) func(call) error {
					t.Helper()
					return func(c call) error {
						assert.NoError(t, os.Remove(filepath.Join(c.dir, lockFile)), "the removal of Cargo.lock")
						files.Write(t, c.dir, files.Tree{lockFile: files.Dir()})
						return nil
					}
				},
			},
		}
		for _, tt := range lockfiles {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				give := tree()
				give[lockFile] = tt.giveLock
				root := files.Workspace(t, give)
				f := &commands{do: tt.giveDo(t)}
				got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{
					{Package: packageC, Version: version.Version{Minor: 2}},
				})
				expect.That(t, err).HasError("Apply")
				expect.That(t, got).Contains("crates/c/Cargo.toml", "the changed files")
			})
		}

		t.Run("returns the error of a repository that does not open", func(t *testing.T) {
			t.Parallel()
			_, err := release.Versioner{Command: (&commands{}).run}.Apply(t.Context(),
				filepath.Join(t.TempDir(), "absent"), []language.Edit{{Package: packageC}})
			assert.HasError(t, err, "Apply")
		})
	})
}
