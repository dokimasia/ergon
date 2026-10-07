// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/version"
	coreworkspace "go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/go/workspace"
)

// pinnedToolchain is the spelling of the toolchain of Go in configuration, which the test pins.
const pinnedToolchain coreworkspace.Toolchain = "go"

// errTags is the error of a repository whose tags git cannot read.
var errTags = errors.New("tags failed")

// tagged returns a function that returns tags as the tags of every repository.
func tagged(tags ...string) func(context.Context, string) (map[string]string, error) {
	return func(context.Context, string) (map[string]string, error) {
		out := make(map[string]string, len(tags))
		for _, name := range tags {
			out[name] = "0123456789abcdef0123456789abcdef01234567"
		}
		return out, nil
	}
}

func TestDiscoverer(t *testing.T) {
	t.Parallel()

	t.Run("Discover", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a package of the toolchain go with its requirements for each module", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"go.work":  files.Text("go 1.24\n\nuse (\n\t./a\n\t./b\n)\n"),
				"a/go.mod": files.Text(modA),
				"b/go.mod": files.Text("module example.com/b\n\ngo 1.24\n\nrequire (\n\texample.com/a v0.1.0\n" +
					"\texample.com/external v1.0.0 // indirect\n)\n"),
			})
			pkgs, err := workspace.Discoverer{Tags: tagged()}.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			onA := coreworkspace.Dependency{Name: "example.com/a", Kind: coreworkspace.KindRuntime, Req: "v0.1.0"}
			assert.Equal(t, pkgs, []coreworkspace.Package{
				{Name: "example.com/a", Toolchain: pinnedToolchain, Dir: "a"},
				{Name: "example.com/b", Toolchain: pinnedToolchain, Dir: "b", Deps: []coreworkspace.Dependency{onA}},
			}, "the packages")
		})

		versions := []struct {
			tree files.Tree
			name string
			want string
			tags []string
		}{
			{
				name: "reads the version of a module from the highest heading of its changelog",
				tree: files.Tree{
					"a/CHANGELOG.md": files.Text("# a\n\n## 1.2.0\n\n- b\n\n## 1.10.0\n\n- c\n\n## not a version\n"),
				},
				want: "1.10.0",
			},
			{
				name: "reads a heading with a v before its version",
				tree: files.Tree{"a/CHANGELOG.md": files.Text("# a\n\n## v0.3.0\n")},
				want: "0.3.0",
			},
			{
				name: "reads the version of a module in a directory from its highest tag",
				tags: []string{"a/v1.2.0", "a/v1.10.0", "a/vnext", "b/v9.0.0", "v8.0.0"},
				want: "1.10.0",
			},
			{
				name: "reads no version from a tag with build metadata",
				tags: []string{"a/v1.0.0", "a/v2.0.0+build"},
				want: "1.0.0",
			},
			{
				name: "takes the higher of the changelog and the tags",
				tree: files.Tree{"a/CHANGELOG.md": files.Text("# a\n\n## 0.4.0\n")},
				tags: []string{"a/v0.3.0"},
				want: "0.4.0",
			},
			{
				name: "returns the zero version for a module without a changelog and without tags",
				want: "0.0.0",
			},
		}
		for _, tt := range versions {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				tree := files.Tree{"go.work": files.Text("go 1.24\n\nuse ./a\n"), "a/go.mod": files.Text(modA)}
				maps.Copy(tree, tt.tree)
				root := files.Workspace(t, tree)
				pkgs, err := workspace.Discoverer{Tags: tagged(tt.tags...)}.Discover(t.Context(), root)
				assert.NoError(t, err, "Discover")
				assert.Length(t, pkgs, 1, "the packages")
				assert.Equal(t, pkgs[0].Version, parse(t, tt.want), "the version")
			})
		}

		t.Run("reads the tags v<version> for the module at the root", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"go.mod": files.Text(modRoot)})
			pkgs, err := workspace.Discoverer{Tags: tagged("v1.1.0", "a/v2.0.0")}.Discover(t.Context(), root)
			assert.NoError(t, err, "Discover")
			assert.Length(t, pkgs, 1, "the packages")
			assert.Equal(t, pkgs[0].Version, parse(t, "1.1.0"), "the version")
		})

		t.Run("returns no package for a repository without Go", func(t *testing.T) {
			t.Parallel()
			failing := func(context.Context, string) (map[string]string, error) { return nil, errTags }
			pkgs, err := workspace.Discoverer{Tags: failing}.Discover(t.Context(), t.TempDir())
			assert.NoError(t, err, "Discover")
			assert.Empty(t, pkgs, "the packages")
		})

		t.Run("returns the error of Modules", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"go.mod": files.Text("go 1.24\n")})
			_, err := workspace.Discoverer{Tags: tagged()}.Discover(t.Context(), root)
			assert.ErrorIs(t, err, workspace.ErrModules, "Discover")
		})

		t.Run("returns the error of the tags", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"go.mod": files.Text(modRoot)})
			failing := func(context.Context, string) (map[string]string, error) { return nil, errTags }
			_, err := workspace.Discoverer{Tags: failing}.Discover(t.Context(), root)
			assert.ErrorIs(t, err, errTags, "Discover")
		})

		t.Run("returns the error of a changelog that it cannot read", func(t *testing.T) {
			t.Parallel()
			tree := files.Tree{"go.mod": files.Text(modRoot), "CHANGELOG.md/inner": files.Text("x\n")}
			_, err := workspace.Discoverer{Tags: tagged()}.Discover(t.Context(), files.Workspace(t, tree))
			assert.HasError(t, err, "Discover")
			assert.Contains(t, err.Error(), "read CHANGELOG.md", "the error")
		})

		t.Run("returns the error of a repository that does not exist", func(t *testing.T) {
			t.Parallel()
			absent := filepath.Join(t.TempDir(), "absent")
			_, err := workspace.Discoverer{Tags: tagged()}.Discover(t.Context(), absent)
			assert.ErrorIs(t, err, fs.ErrNotExist, "Discover")
		})
	})
}

// parse returns the version that s states, and stops the test tb for an s that version.Parse
// refuses.
func parse(tb testing.TB, s string) version.Version {
	tb.Helper()
	v, err := version.Parse(s)
	assert.NoError(tb, err, "Parse of "+s)
	return v
}
