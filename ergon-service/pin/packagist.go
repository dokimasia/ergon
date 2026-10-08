// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// packagistMetadata is the metadata of version 2 of a package of Packagist, whose versions follow
// each other with the fields that changed, a version and a time in each.
type packagistMetadata struct {
	// Packages are the versions of the package, by its name.
	Packages map[string][]packagistVersion `json:"packages"`
}

// packagistVersion is a version of a package of Packagist.
type packagistVersion struct {
	// Published is the time of the release of the version.
	Published time.Time `json:"time"`

	// Version is the version, such as 2.2.17 or v3.95.27.
	Version string `json:"version"`
}

// packagist returns the stable versions of the Composer package name, as vendor/package, each with
// the time of its release, from the metadata of Packagist. It returns an error that wraps
// [ErrRegistry] for a package that Packagist does not have, a request that fails, and a document
// that does not decode.
func (r *Resolver) packagist(ctx context.Context, name string) ([]candidate, error) {
	address := r.Registries.Packagist + "/p2/" + name + ".json"
	data, err := r.document(ctx, address)
	if err != nil {
		return nil, err
	}
	var metadata packagistMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("%w: decode %s: %w", ErrRegistry, address, err)
	}
	versions, ok := metadata.Packages[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s has no versions of %s", ErrRegistry, address, name)
	}
	var candidates []candidate
	for _, v := range versions {
		if stableVersion.MatchString(v.Version) {
			candidates = append(
				candidates,
				candidate{published: v.Published, rank: order(v.Version), version: v.Version},
			)
		}
	}
	return candidates, nil
}
