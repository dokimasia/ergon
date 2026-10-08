// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"context"
	"fmt"
)

// github returns the releases of the repository repo, as owner/name, whose tags are stable versions,
// each with the time of its publication and its assets, from the API of GitHub. It returns an error
// that wraps [ErrRegistry] for a request that fails, such as for a repository that does not exist.
func (r *Resolver) github(ctx context.Context, repo string) ([]candidate, error) {
	releases, err := r.GitHub.Releases(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRegistry, err)
	}
	var candidates []candidate
	for k := range releases {
		if tag := releases[k].Tag; stableVersion.MatchString(tag) {
			candidates = append(candidates, candidate{
				published: releases[k].Published, rank: order(tag), release: &releases[k], version: tag,
			})
		}
	}
	return candidates, nil
}
