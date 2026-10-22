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
	"go.dokimi.dev/ergon/lang/python/release"
	pyworkspace "go.dokimi.dev/ergon/lang/python/workspace"
)

// The distributions of probe-b at 0.2.0 that uv build writes.
const (
	sdistB = "probe_b-0.2.0.tar.gz"
	wheelB = "probe_b-0.2.0-py3-none-any.whl"
)

// The flag of uv build for the directory of the distributions.
const outDirFlag = "--out-dir"

// releasedA and releasedB are the packages of probe-a and probe-b at the versions of their release.
var (
	releasedA = workspace.Package{
		Name: "probe-a", Toolchain: pyworkspace.Toolchain, Dir: "packages/probe_a", Version: version.Version{Minor: 3},
	}
	releasedB = workspace.Package{
		Name: "probe-b", Toolchain: pyworkspace.Toolchain, Dir: "packages/probe_b", Version: version.Version{Minor: 2},
	}
)

// builder is a fake of ergon tool run for uv build. It records each call, writes the files of the
// project that the flag --package names into the directory of the flag --out-dir, under the pack
// directory pack, and returns err. uv build writes a .gitignore beside the distributions.
type builder struct {
	tb    testing.TB
	err   error
	files map[string][]string
	pack  string
	calls []call
}

// run records the call and acts as builder states.
func (f *builder) run(_ context.Context, dir, tool string, args, _ []string) error {
	f.calls = append(f.calls, call{dir: dir, tool: tool, args: args})
	out, err := filepath.Rel(f.pack, args[slices.Index(args, outDirFlag)+1])
	if err != nil {
		return err
	}
	tree := files.Tree{filepath.ToSlash(filepath.Join(out, ".gitignore")): files.Text("*")}
	for _, name := range f.files[args[slices.Index(args, "--package")+1]] {
		tree[filepath.ToSlash(filepath.Join(out, name))] = files.Text("")
	}
	files.Write(f.tb, f.pack, tree)
	return f.err
}

func TestPacker(t *testing.T) {
	t.Parallel()

	t.Run("Pack", func(t *testing.T) {
		t.Parallel()

		t.Run("runs uv build for each package in the order of the packages", func(t *testing.T) {
			t.Parallel()
			root, pack := t.TempDir(), t.TempDir()
			f := &builder{tb: t, pack: pack, files: map[string][]string{
				"probe-a": {"probe_a-0.3.0.tar.gz", "probe_a-0.3.0-py3-none-any.whl"},
				"probe-b": {sdistB, wheelB},
			}}
			err := release.Packer{Tools: f.run}.Pack(t.Context(), root, []workspace.Package{releasedB, releasedA}, pack)
			assert.NoError(t, err, "Pack")
			assert.Equal(t, f.calls, []call{
				{dir: root, tool: "uv", args: []string{
					"build", "--package", "probe-b", outDirFlag, filepath.Join(pack, release.DistDir, "probe-b"),
				}},
				{dir: root, tool: "uv", args: []string{
					"build", "--package", "probe-a", outDirFlag, filepath.Join(pack, release.DistDir, "probe-a"),
				}},
			}, "the runs of uv")
		})

		tests := []struct {
			name      string
			giveFiles []string
		}{
			{name: "returns ErrDistribution for a build without a distribution", giveFiles: nil},
			{
				name:      "returns ErrDistribution for a distribution of another version",
				giveFiles: []string{"probe_b-0.3.0.tar.gz", wheelB},
			},
			{
				name:      "returns ErrDistribution for a distribution of another project",
				giveFiles: []string{sdistB, "probe_a-0.2.0-py3-none-any.whl"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				pack := t.TempDir()
				f := &builder{tb: t, pack: pack, files: map[string][]string{"probe-b": tt.giveFiles}}
				err := release.Packer{Tools: f.run}.Pack(t.Context(), t.TempDir(), []workspace.Package{releasedB}, pack)
				assert.ErrorIs(t, err, release.ErrDistribution, "Pack")
			})
		}

		t.Run("returns the error of uv build", func(t *testing.T) {
			t.Parallel()
			pack := t.TempDir()
			f := &builder{tb: t, pack: pack, err: errUV}
			err := release.Packer{Tools: f.run}.Pack(t.Context(), t.TempDir(), []workspace.Package{releasedB}, pack)
			assert.ErrorIs(t, err, errUV, "Pack")
		})

		t.Run("returns ErrVersion for a version that PEP 440 does not have", func(t *testing.T) {
			t.Parallel()
			pack := t.TempDir()
			f := &builder{tb: t, pack: pack}
			p := releasedB
			p.Version.Build = "1"
			err := release.Packer{Tools: f.run}.Pack(t.Context(), t.TempDir(), []workspace.Package{p}, pack)
			assert.ErrorIs(t, err, pyworkspace.ErrVersion, "Pack")
			assert.Empty(t, f.calls, "the runs of uv")
		})
	})
}
