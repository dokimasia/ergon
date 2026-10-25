// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/rust/release"
)

// The flags of cargo package that a fake reads.
const (
	flagTargetDir = "--target-dir"
	flagPackage   = "--package"
)

// unreadableMode is the mode of a directory that its owner can enter and write but not read.
const unreadableMode = 0o300

// packages returns a fake of cargo package that writes the .crate file of each crate of the flags
// --package of its run into target/ergon/package of the directory of the run, with the version of
// the crate of pkgs, and leaves out the crates of skip.
func packages(t *testing.T, pkgs []workspace.Package, skip ...string) func(c call) error {
	t.Helper()
	return func(c call) error {
		out := files.Tree{}
		for k, arg := range c.args {
			if arg != flagPackage || slices.Contains(skip, c.args[k+1]) {
				continue
			}
			p := pkgs[slices.IndexFunc(pkgs, func(p workspace.Package) bool { return p.Name == c.args[k+1] })]
			out["target/ergon/package/"+p.Name+"-"+p.Version.String()+".crate"] = files.Text("the crate " + p.Name)
		}
		files.Write(t, c.dir, out)
		return nil
	}
}

func TestPacker(t *testing.T) {
	t.Parallel()

	t.Run("Pack", func(t *testing.T) {
		t.Parallel()

		t.Run("packs each crate with cargo package into the directory rust of dir", func(t *testing.T) {
			t.Parallel()
			root, dir := files.Workspace(t, files.Tree{"target": files.Dir()}), files.Workspace(t, files.Tree{})
			pkgs := []workspace.Package{releasedA, releasedB}
			f := &commands{do: packages(t, pkgs)}
			assert.NoError(t, release.Packer{Command: f.run}.Pack(t.Context(), root, pkgs, dir), "Pack")
			expect.Equal(t, f.calls, []call{{
				dir:  root,
				name: "cargo",
				args: []string{
					"package", flagTargetDir, filepath.Join(root, "target", "ergon"),
					flagPackage, "probe-a", flagPackage, "probe-b",
				},
			}}, "the runs of cargo")
			files.Equal(t, os.DirFS(dir), files.Tree{
				"rust/probe-a/probe-a-0.3.0.crate": files.Text("the crate probe-a"),
				"rust/probe-b/probe-b-0.2.1.crate": files.Text("the crate probe-b"),
			}, "the pack directory")
		})

		t.Run("creates the pack directory when it does not exist", func(t *testing.T) {
			t.Parallel()
			root, dir := files.Workspace(t, files.Tree{"target": files.Dir()}), filepath.Join(t.TempDir(), "dist")
			pkgs := []workspace.Package{releasedA}
			f := &commands{do: packages(t, pkgs)}
			assert.NoError(t, release.Packer{Command: f.run}.Pack(t.Context(), root, pkgs, dir), "Pack")
			files.Equal(t, os.DirFS(dir), files.Tree{
				"rust/probe-a/probe-a-0.3.0.crate": files.Text("the crate probe-a"),
			}, "the pack directory")
		})

		t.Run("runs no cargo package for no crate", func(t *testing.T) {
			t.Parallel()
			f := &commands{}
			assert.NoError(t, release.Packer{Command: f.run}.Pack(t.Context(), t.TempDir(), nil, t.TempDir()), "Pack")
			assert.Empty(t, f.calls, "the runs of cargo")
		})

		t.Run("returns ErrCrate for a crate that cargo package does not write", func(t *testing.T) {
			t.Parallel()
			pkgs := []workspace.Package{releasedA, releasedB}
			f := &commands{do: packages(t, pkgs, "probe-b")}
			err := release.Packer{Command: f.run}.Pack(t.Context(), files.Workspace(t, files.Tree{}), pkgs, t.TempDir())
			assert.ErrorIs(t, err, release.ErrCrate, "Pack")
		})

		t.Run("returns the error of cargo package", func(t *testing.T) {
			t.Parallel()
			f := &commands{do: func(call) error { return errProgram }}
			err := release.Packer{
				Command: f.run,
			}.Pack(
				t.Context(),
				t.TempDir(),
				[]workspace.Package{releasedA},
				t.TempDir(),
			)
			assert.ErrorIs(t, err, errProgram, "Pack")
		})

		t.Run("returns the error of a .crate file that does not write", func(t *testing.T) {
			t.Parallel()
			pkgs := []workspace.Package{releasedA}
			f := &commands{do: packages(t, pkgs)}
			dir := files.Workspace(t, files.Tree{"rust/probe-a/probe-a-0.3.0.crate": files.Dir()})
			err := release.Packer{Command: f.run}.Pack(t.Context(), files.Workspace(t, files.Tree{}), pkgs, dir)
			assert.HasError(t, err, "Pack")
		})

		t.Run("returns the error of a directory of crates that cannot be created", func(t *testing.T) {
			t.Parallel()
			pkgs := []workspace.Package{releasedA}
			f := &commands{do: packages(t, pkgs)}
			dir := files.Workspace(t, files.Tree{release.DistDir: files.Text("a file")})
			err := release.Packer{Command: f.run}.Pack(t.Context(), files.Workspace(t, files.Tree{}), pkgs, dir)
			assert.HasError(t, err, "Pack")
		})

		t.Run("returns the error of a pack directory that cannot be created", func(t *testing.T) {
			t.Parallel()
			pkgs := []workspace.Package{releasedA}
			f := &commands{do: packages(t, pkgs)}
			file := filepath.Join(files.Workspace(t, files.Tree{"dist": files.Text("a file")}), "dist")
			err := release.Packer{Command: f.run}.Pack(t.Context(), files.Workspace(t, files.Tree{}), pkgs, file)
			assert.HasError(t, err, "Pack")
		})

		t.Run("returns the error of a pack directory that does not open", func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == "windows" {
				t.Skip("Windows does not deny the read of a directory by its mode")
			}
			pkgs := []workspace.Package{releasedA}
			f := &commands{do: packages(t, pkgs)}
			dir := filepath.Join(files.Workspace(t, files.Tree{"dist": files.Dir().WithMode(unreadableMode)}), "dist")
			err := release.Packer{Command: f.run}.Pack(t.Context(), files.Workspace(t, files.Tree{}), pkgs, dir)
			assert.HasError(t, err, "Pack")
		})
	})
}
