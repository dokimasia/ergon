// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"

	"go.dokimi.dev/ergon/core/workspace"
)

// Pack builds the artifacts of the packages of the entries of [KindPublish] of plan, from the
// repository at root into dir, with the packer of each toolchain: one call of a packer for the
// packages of its toolchain, in the order of the plan. A package of a toolchain without a packer
// has no artifact, as the registry of such a toolchain reads the source of a package.
//
// It returns an error that wraps [ErrPublishPlan] for an entry of a package that the repository
// does not have, and the error of a packer.
func Pack(ctx context.Context, root string, g *Graph, plan *PublishPlan, dir string) error {
	var order []*roles
	pending := map[*roles][]workspace.Package{}
	for _, chunk := range plan.Plan {
		for k := range chunk {
			e := &chunk[k]
			if e.Kind != KindPublish {
				continue
			}
			i, ok := g.index[e.Name]
			if !ok {
				return fmt.Errorf("%w: the package %s, which the repository does not have", ErrPublishPlan, e.Name)
			}
			r := g.roles[i]
			if r.packer == nil {
				continue
			}
			if _, seen := pending[r]; !seen {
				order = append(order, r)
			}
			pending[r] = append(pending[r], g.pkgs[i])
		}
	}
	for _, r := range order {
		if err := r.packer.Pack(ctx, root, pending[r], dir); err != nil {
			return fmt.Errorf("release: pack: %w", err)
		}
	}
	return nil
}
