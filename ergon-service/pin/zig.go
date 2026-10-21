// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"go.dokimi.dev/ergon/service/forge"
)

// zigDate is the key of the date of a version in the download index of zig.
const zigDate = "date"

// zigArchive is an archive of a version in the download index of zig.
type zigArchive struct {
	// Tarball is the address of the archive, such as
	// https://ziglang.org/download/0.16.0/zig-x86_64-linux-0.16.0.tar.xz.
	Tarball string `json:"tarball"`

	// Shasum is the SHA-256 of the archive in lowercase hexadecimal.
	Shasum string `json:"shasum"`
}

// zig returns the versions of the download index of zig at Registries.Zig that are stable, each
// with the time of its date and a release whose assets are its archives. The digest of each asset is
// the shasum that the index states for the archive. The index lists each version with its date, its
// archives and its documents, and lists master, which is no release. It returns an error that wraps
// [ErrRegistry] for an index that does not decode, and for a version without a date that parses.
func (r *Resolver) zig(ctx context.Context) ([]candidate, error) {
	data, err := r.document(ctx, r.Registries.Zig)
	if err != nil {
		return nil, err
	}
	var index map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("%w: decode %s: %w", ErrRegistry, r.Registries.Zig, err)
	}
	var candidates []candidate
	for version, entries := range index {
		if !stableVersion.MatchString(version) {
			continue
		}
		var date string
		err := json.Unmarshal(entries[zigDate], &date)
		var published time.Time
		if err == nil {
			published, err = time.Parse(time.DateOnly, date)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: the date of zig %s: %w", ErrRegistry, version, err)
		}
		release := &forge.Release{Published: published, Tag: version}
		for _, entry := range entries {
			var archive zigArchive
			if json.Unmarshal(entry, &archive) == nil && archive.Tarball != "" {
				release.Assets = append(release.Assets, forge.Asset{
					Name: path.Base(archive.Tarball), URL: archive.Tarball, Digest: archive.Shasum,
				})
			}
		}
		candidates = append(candidates, candidate{
			published: published, rank: order(version), release: release, version: version,
		})
	}
	return candidates, nil
}
