// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// crateVersion is a line of the file of a crate in the sparse index of crates.io.
type crateVersion struct {
	// Published is the time at which the version was published, and the zero time for a version
	// published before the index recorded it.
	Published time.Time `json:"pubtime"`

	// Version is the version, such as 0.22.2.
	Version string `json:"vers"`

	// Yanked reports a version that its crate withdrew.
	Yanked bool `json:"yanked"`
}

// crate returns the stable versions of the crate name, each with the time of its publication, from
// the sparse index of crates.io, which has one line of JSON for each version. It leaves out a
// yanked version. It returns an error that wraps [ErrRegistry] for a crate that the index does not
// have, a request that fails, and a line that does not decode.
func (r *Resolver) crate(ctx context.Context, name string) ([]candidate, error) {
	address := r.Registries.Crates + "/" + indexPath(name)
	data, err := r.document(ctx, address)
	if err != nil {
		return nil, err
	}
	var candidates []candidate
	for line := range bytes.SplitSeq(bytes.TrimSpace(data), []byte("\n")) {
		var v crateVersion
		if err := json.Unmarshal(line, &v); err != nil {
			return nil, fmt.Errorf("%w: decode %s: %w", ErrRegistry, address, err)
		}
		if !stableVersion.MatchString(v.Version) || v.Yanked {
			continue
		}
		candidates = append(candidates, candidate{published: v.Published, rank: order(v.Version), version: v.Version})
	}
	return candidates, nil
}

// indexPath returns the path of the file of the crate name in the index, as Cargo lays it out: the
// lower-case name under 1/ or 2/ for a name of one or two characters, under 3/ and its first
// character for a name of three, and under its first two and its next two characters otherwise.
func indexPath(name string) string {
	n := strings.ToLower(name)
	switch {
	case len(n) <= 2:
		return strconv.Itoa(len(n)) + "/" + n
	case len(n) == 3:
		return "3/" + n[:1] + "/" + n
	}
	return n[:2] + "/" + n[2:4] + "/" + n
}
