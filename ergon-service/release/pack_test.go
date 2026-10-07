// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"testing"

	"go.dokimi.dev/assert"
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

	// fail makes Pack return errPack.
	fail bool
}

var _ language.Packer = packer{}

// Pack records the names of pkgs.
func (p packer) Pack(_ context.Context, _ string, pkgs []workspace.Package, _ string) error {
	if p.fail {
		return errPack
	}
	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	*p.packs = append(*p.packs, names)
	return nil
}

func TestPack(t *testing.T) {
	t.Parallel()

	t.Run("Pack", func(t *testing.T) {
		t.Parallel()

		t.Run("builds the artifacts of the uploads of the plan with the packer of their toolchain", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.add("pkg-b", "1.0.0")
			packs := new([][]string)
			s.roles = []any{newRegistry(), packer{packs: packs}}
			plan := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{
				{{Kind: release.KindPublish, Name: "pkg-a"}, {Kind: release.KindTagOnly, Name: "pkg-b"}},
				{{Kind: release.KindPublish, Name: "pkg-b"}},
			}}
			assert.NoError(t, release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, "dist"), "Pack")
			assert.Equal(t, *packs, [][]string{{"pkg-a", "pkg-b"}}, "the packs")
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
			err := release.Pack(t.Context(), t.TempDir(), s.graph(t), plan, "dist")
			assert.ErrorIs(t, err, errPack, "Pack")
		})
	})
}
