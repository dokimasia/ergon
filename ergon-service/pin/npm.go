// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"time"
)

// npmPackage is the part of the document of a package of the npm registry that a Resolver reads.
type npmPackage struct {
	// Versions are the published versions, by version.
	Versions map[string]npmVersion `json:"versions"`

	// Time are the times of publication, by version, beside the keys created and modified. An
	// unpublished package has an object under the key unpublished, so each value is read on its
	// own.
	Time map[string]json.RawMessage `json:"time"`
}

// npmVersion is a published version of a package of the npm registry.
type npmVersion struct {
	// Deprecated is the message with which the package deprecated the version, and empty for a
	// version that it did not deprecate.
	Deprecated string `json:"deprecated"`
}

// npm returns the stable versions of the npm package name, each with the time of its publication,
// from the npm registry. It leaves out a deprecated version. It returns an error that wraps
// [ErrRegistry] for a package that the registry does not have, a request that fails, and a document
// or a time that does not decode.
func (r *Resolver) npm(ctx context.Context, name string) ([]candidate, error) {
	address := r.Registries.NPM + "/" + url.PathEscape(name)
	data, err := r.document(ctx, address)
	if err != nil {
		return nil, err
	}
	var pkg npmPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("%w: decode %s: %w", ErrRegistry, address, err)
	}
	var candidates []candidate
	for _, version := range slices.Sorted(maps.Keys(pkg.Versions)) {
		if !stableVersion.MatchString(version) || pkg.Versions[version].Deprecated != "" {
			continue
		}
		var published time.Time
		if err := json.Unmarshal(pkg.Time[version], &published); err != nil {
			return nil, fmt.Errorf("%w: the time of %s %s: %w", ErrRegistry, name, version, err)
		}
		candidates = append(candidates, candidate{published: published, rank: order(version), version: version})
	}
	return candidates, nil
}
