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

// pypiProject is the part of the JSON of a project of PyPI that a Resolver reads: the files of each
// release, by version.
type pypiProject struct {
	// Releases are the files of each release, by version.
	Releases map[string][]pypiFile `json:"releases"`
}

// pypiFile is a file of a release of PyPI.
type pypiFile struct {
	// Uploaded is the time at which the file was uploaded.
	Uploaded time.Time `json:"upload_time_iso_8601"`

	// Yanked reports a file that its project withdrew.
	Yanked bool `json:"yanked"`
}

// pypi returns the stable releases of the PyPI package name, each with the time of its first file,
// from the JSON API of PyPI. It leaves out a release without files, and a release whose every file
// was yanked. It returns an error that wraps [ErrRegistry] for a package that PyPI does not have, a
// request that fails, and a document that does not decode.
func (r *Resolver) pypi(ctx context.Context, name string) ([]candidate, error) {
	address := r.Registries.PyPI + "/pypi/" + url.PathEscape(name) + "/json"
	data, err := r.document(ctx, address)
	if err != nil {
		return nil, err
	}
	var project pypiProject
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("%w: decode %s: %w", ErrRegistry, address, err)
	}
	var candidates []candidate
	for _, version := range slices.Sorted(maps.Keys(project.Releases)) {
		files := project.Releases[version]
		available := slices.ContainsFunc(files, func(f pypiFile) bool { return !f.Yanked })
		if !stableVersion.MatchString(version) || !available {
			continue
		}
		candidates = append(candidates, candidate{published: files[0].Uploaded, rank: order(version), version: version})
	}
	return candidates, nil
}
