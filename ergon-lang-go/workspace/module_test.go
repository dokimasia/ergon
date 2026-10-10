// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"io/fs"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/lang/go/workspace"
)

// The files of the modules of the cases.
const (
	// modA is the go.mod of the module example.com/a.
	modA = "module example.com/a\n\ngo 1.24\n"

	// modRoot is the go.mod of the module example.com/root.
	modRoot = "module example.com/root\n\ngo 1.24\n"
)

func TestModule(t *testing.T) {
	t.Parallel()

	t.Run("Modules", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the module of each directory of go.work in the order of go.work", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{
				"go.work":  files.Text("go 1.24\n\nuse (\n\t./b\n\t.\n\t./a\n)\n"),
				"go.mod":   files.Text(modRoot),
				"a/go.mod": files.Text(modA),
				"b/go.mod": files.Text("module example.com/b\n\ngo 1.24\n"),
			})
			mods, err := workspace.Modules(root)
			assert.NoError(t, err, "Modules")
			assert.Equal(t, pathsOf(mods), []string{"example.com/b", "example.com/root", "example.com/a"}, "the paths")
			assert.Equal(t, dirsOf(mods), []string{"b", ".", "a"}, "the directories")
		})

		t.Run("returns the module at the root of a repository without go.work", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"go.mod": files.Text(modRoot), "a/go.mod": files.Text(modA)})
			mods, err := workspace.Modules(root)
			assert.NoError(t, err, "Modules")
			assert.Equal(t, pathsOf(mods), []string{"example.com/root"}, "the paths")
			assert.Equal(t, mods[0].File.Module.Mod.Path, "example.com/root", "the module of its go.mod")
		})

		t.Run("returns no module for a repository without Go", func(t *testing.T) {
			t.Parallel()
			mods, err := workspace.Modules(files.Workspace(t, files.Tree{"README.md": files.Text("# x\n")}))
			assert.NoError(t, err, "Modules")
			assert.Empty(t, mods, "the modules")
		})

		t.Run("skips the directories of go.work outside the repository", func(t *testing.T) {
			t.Parallel()
			outside := filepath.ToSlash(t.TempDir())
			root := files.Workspace(t, files.Tree{
				"go.work":  files.Text("go 1.24\n\nuse (\n\t./a\n\t../other\n\t..\n\t\"" + outside + "\"\n)\n"),
				"a/go.mod": files.Text(modA),
			})
			mods, err := workspace.Modules(root)
			assert.NoError(t, err, "Modules")
			assert.Equal(t, dirsOf(mods), []string{"a"}, "the directories")
		})

		refused := []struct {
			tree files.Tree
			name string
			text string
		}{
			{
				name: "returns ErrModules for a go.work that the go command refuses",
				tree: files.Tree{"go.work": files.Text("go 1.24\n\nunknown ./a\n")},
				text: "go.work:3",
			},
			{
				name: "returns ErrModules for a directory of go.work without a go.mod",
				tree: files.Tree{"go.work": files.Text("go 1.24\n\nuse ./a\n"), "a/README.md": files.Text("# a\n")},
				text: "the directory a, which go.work uses, has no go.mod",
			},
			{
				name: "returns ErrModules for a go.mod that the go command refuses",
				tree: files.Tree{"go.mod": files.Text("module example.com/root\n\nunknown x\n")},
				text: "go.mod:3",
			},
			{
				name: "returns ErrModules for a go.mod without a module directive",
				tree: files.Tree{"go.mod": files.Text("go 1.24\n")},
				text: "go.mod declares no module",
			},
		}
		for _, tt := range refused {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.Modules(files.Workspace(t, tt.tree))
				assert.ErrorIs(t, err, workspace.ErrModules, "Modules")
				assert.Contains(t, err.Error(), tt.text, "the error")
			})
		}

		unreadable := []struct {
			tree files.Tree
			name string
			text string
		}{
			{
				name: "returns the error of a go.work that it cannot read",
				tree: files.Tree{"go.work/inner": files.Text("inner\n")},
				text: "read go.work",
			},
			{
				name: "returns the error of a go.mod that it cannot read",
				tree: files.Tree{
					"go.work":        files.Text("go 1.24\n\nuse ./a\n"),
					"a/go.mod/inner": files.Text("inner\n"),
				},
				text: "read a/go.mod",
			},
		}
		for _, tt := range unreadable {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.Modules(files.Workspace(t, tt.tree))
				assert.That(t, err).
					HasError("Modules").
					ErrorIsNot(workspace.ErrModules, "the class of the error")
				assert.Contains(t, err.Error(), tt.text, "the error")
			})
		}

		t.Run("returns the error of a repository that does not exist", func(t *testing.T) {
			t.Parallel()
			_, err := workspace.Modules(filepath.Join(t.TempDir(), "absent"))
			assert.ErrorIs(t, err, fs.ErrNotExist, "Modules")
		})
	})
}

// pathsOf returns the module path of each of mods, in their order.
func pathsOf(mods []workspace.Module) []string {
	paths := make([]string, 0, len(mods))
	for _, m := range mods {
		paths = append(paths, m.Path)
	}
	return paths
}

// dirsOf returns the directory of each of mods, in their order.
func dirsOf(mods []workspace.Module) []string {
	dirs := make([]string, 0, len(mods))
	for _, m := range mods {
		dirs = append(dirs, m.Dir)
	}
	return dirs
}
