// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/javascript/release"
)

// The tarballs of @probe/a and @probe/b at the versions of their release.
const (
	tarballA = "probe-a-0.3.0.tgz"
	tarballB = "probe-b-0.2.0.tgz"
)

// destinationFlag is the flag of npm pack and pnpm pack for the directory of the tarball.
const destinationFlag = "--pack-destination"

// packs is a fake of the package manager in a pack. It records each call, writes the tarball that
// tarballs names for the directory of the call into the directory of the flag --pack-destination,
// and returns err.
type packs struct {
	tb       testing.TB
	err      error
	tarballs map[string]string
	calls    []call
}

// run records the call and acts as packs states.
func (f *packs) run(_ context.Context, dir, name string, args, _ []string) error {
	f.calls = append(f.calls, call{dir: dir, name: name, args: args})
	if file, ok := f.tarballs[dir]; ok {
		files.Write(f.tb, args[slices.Index(args, destinationFlag)+1], files.Tree{file: files.Text("")})
	}
	return f.err
}

func TestPacker(t *testing.T) {
	t.Parallel()

	t.Run("Pack", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			giveTree    files.Tree
			wantProgram string
		}{
			{name: "runs npm pack in the directory of each package", giveTree: files.Tree{}, wantProgram: "npm"},
			{
				name:        "runs pnpm pack in a repository with pnpm-lock.yaml",
				giveTree:    files.Tree{release.PNPMLockFile: files.Text("lockfileVersion: '9.0'\n")},
				wantProgram: "pnpm",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				root, pack := files.Workspace(t, tt.giveTree), t.TempDir()
				dirA, dirB := filepath.Join(root, "packages", "a"), filepath.Join(root, "packages", "b")
				f := &packs{tb: t, tarballs: map[string]string{dirA: tarballA, dirB: tarballB}}
				err := release.Packer{
					Command: f.run,
				}.Pack(
					t.Context(),
					root,
					[]workspace.Package{releasedB, releasedA},
					pack,
				)
				assert.NoError(t, err, "Pack")
				assert.Equal(t, f.calls, []call{
					{dir: dirB, name: tt.wantProgram, args: []string{
						"pack", destinationFlag, filepath.Join(pack, release.DistDir, "@probe", "b"),
					}},
					{dir: dirA, name: tt.wantProgram, args: []string{
						"pack", destinationFlag, filepath.Join(pack, release.DistDir, "@probe", "a"),
					}},
				}, "the runs of the package manager")
			})
		}

		t.Run(
			"checks the tarball of a package without a scope at its version without build metadata",
			func(t *testing.T) {
				t.Parallel()
				root, pack := t.TempDir(), t.TempDir()
				p := workspace.Package{Name: "left", Toolchain: pinnedToolchain, Dir: ".", Version: version.Version{
					Major: 1, Pre: "rc.1", Build: "5",
				}}
				f := &packs{tb: t, tarballs: map[string]string{root: "left-1.0.0-rc.1.tgz"}}
				err := release.Packer{Command: f.run}.Pack(t.Context(), root, []workspace.Package{p}, pack)
				assert.NoError(t, err, "Pack")
			},
		)

		t.Run("returns ErrTarball for a pack that writes the tarball of another version", func(t *testing.T) {
			t.Parallel()
			root, pack := t.TempDir(), t.TempDir()
			f := &packs{tb: t, tarballs: map[string]string{filepath.Join(root, "packages", "b"): "probe-b-0.3.0.tgz"}}
			err := release.Packer{Command: f.run}.Pack(t.Context(), root, []workspace.Package{releasedB}, pack)
			assert.ErrorIs(t, err, release.ErrTarball, "Pack")
		})

		t.Run("returns the error of the package manager", func(t *testing.T) {
			t.Parallel()
			f := &packs{tb: t, err: errProgram}
			err := release.Packer{
				Command: f.run,
			}.Pack(
				t.Context(),
				t.TempDir(),
				[]workspace.Package{releasedB},
				t.TempDir(),
			)
			assert.ErrorIs(t, err, errProgram, "Pack")
		})

		t.Run("returns the error of a directory of tarballs that cannot be created", func(t *testing.T) {
			t.Parallel()
			pack := files.Workspace(t, files.Tree{release.DistDir: files.Text("")})
			f := &packs{tb: t}
			err := release.Packer{Command: f.run}.Pack(t.Context(), t.TempDir(), []workspace.Package{releasedB}, pack)
			assert.HasError(t, err, "Pack")
			assert.Empty(t, f.calls, "the runs of the package manager")
		})
	})
}
