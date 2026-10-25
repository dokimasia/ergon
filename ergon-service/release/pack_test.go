// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
)

// errPack is the error of a packer that fails.
var errPack = errors.New("pack failed")

// packer is a [language.Packer] that records the names of the packages of each call, and fails
// when fail is set.
type packer struct {
	// packs are the names of the packages of each call, in the order of the calls.
	packs *[][]string

	// dirs records for each call whether the pack directory is a directory, when it is not nil.
	dirs *[]bool

	// fail makes Pack return errPack.
	fail bool
}

var _ language.Packer = packer{}

// Pack records the names of pkgs, and whether dir is a directory.
func (p packer) Pack(_ context.Context, _ string, pkgs []workspace.Package, dir string) error {
	if p.fail {
		return errPack
	}
	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	*p.packs = append(*p.packs, names)
	if p.dirs != nil {
		info, err := os.Stat(dir)
		*p.dirs = append(*p.dirs, err == nil && info.IsDir())
	}
	return nil
}

func TestPack(t *testing.T) {
	t.Parallel()

	t.Run("Pack", func(t *testing.T) {
		t.Parallel()

		t.Run("builds the artifacts of each package of the plan once with the packer of its toolchain",
			func(t *testing.T) {
				t.Parallel()
				s := newState(t)
				s.add("pkg-b", "1.0.0")
				packs := new([][]string)
				s.roles = []any{newRegistry(), packer{packs: packs}}
				plan := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{
					{{Kind: release.KindPublish, Name: "pkg-a"}, {Kind: release.KindTagOnly, Name: "pkg-b"}},
					{{Kind: release.KindPublish, Name: "pkg-b"}},
				}}
				dir := filepath.Join(t.TempDir(), "dist")
				assert.NoError(t, release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, dir), "Pack")
				assert.Equal(t, *packs, [][]string{{"pkg-a", "pkg-b"}}, "the packs")
			})

		t.Run("creates the pack directory before the first packer runs", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			dirs := new([]bool)
			s.roles = []any{newRegistry(), packer{packs: new([][]string), dirs: dirs}}
			plan := &release.PublishPlan{
				Version: 1,
				Plan:    [][]release.PublishEntry{{{Kind: release.KindPublish, Name: "pkg-a"}}},
			}
			dir := filepath.Join(t.TempDir(), "dist")
			assert.NoError(t, release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, dir), "Pack")
			assert.Equal(t, *dirs, []bool{true}, "the pack directory of each call")
		})

		t.Run("creates no pack directory for a plan without a packer", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.roles = []any{newRegistry()}
			plan := &release.PublishPlan{
				Version: 1,
				Plan:    [][]release.PublishEntry{{{Kind: release.KindPublish, Name: "pkg-a"}}},
			}
			dir := filepath.Join(t.TempDir(), "dist")
			assert.NoError(t, release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, dir), "Pack")
			files.Absent(t, dir, "the pack directory")
		})

		t.Run("returns the error of a pack directory that cannot be created", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			packs := new([][]string)
			s.roles = []any{newRegistry(), packer{packs: packs}}
			plan := &release.PublishPlan{
				Version: 1,
				Plan:    [][]release.PublishEntry{{{Kind: release.KindPublish, Name: "pkg-a"}}},
			}
			dir := filepath.Join(files.Workspace(t, files.Tree{"file": files.Text("a file")}), "file", "dist")
			assert.HasError(t, release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, dir), "Pack")
			assert.Empty(t, *packs, "the packs")
		})

		t.Run("builds the assets of a package that a publish tags alone", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			packs := new([][]string)
			s.roles = []any{packer{packs: packs}}
			plan := &release.PublishPlan{
				Version: 1,
				Plan:    [][]release.PublishEntry{{{Kind: release.KindTagOnly, Name: "pkg-a"}}},
			}
			dir := filepath.Join(t.TempDir(), "dist")
			assert.NoError(t, release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, dir), "Pack")
			assert.Equal(t, *packs, [][]string{{"pkg-a"}}, "the packs")
		})

		t.Run("builds nothing for a toolchain without a packer", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.roles = []any{newRegistry()}
			plan := &release.PublishPlan{
				Version: 1,
				Plan:    [][]release.PublishEntry{{{Kind: release.KindPublish, Name: "pkg-a"}}},
			}
			assert.NoError(t, release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, "dist"), "Pack")
		})

		t.Run("returns ErrPublishPlan for an entry of a package that the repository does not have", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			plan := &release.PublishPlan{
				Version: 1,
				Plan:    [][]release.PublishEntry{{{Kind: release.KindPublish, Name: "pkg-z"}}},
			}
			err := release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, "dist")
			assert.ErrorIs(t, err, release.ErrPublishPlan, "Pack")
		})

		t.Run("returns the error of a packer", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.roles = []any{newRegistry(), packer{packs: new([][]string), fail: true}}
			plan := &release.PublishPlan{
				Version: 1,
				Plan:    [][]release.PublishEntry{{{Kind: release.KindPublish, Name: "pkg-a"}}},
			}
			err := release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, filepath.Join(t.TempDir(), "dist"))
			assert.ErrorIs(t, err, errPack, "Pack")
		})
	})
}
