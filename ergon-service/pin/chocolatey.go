// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"time"
)

// chocolateyTime is the layout of a time of the OData API of Chocolatey, in UTC without a zone.
const chocolateyTime = "2006-01-02T15:04:05.999999999"

// chocolateyFeed is the part of the Atom feed of the OData API of Chocolatey that a Resolver reads.
type chocolateyFeed struct {
	// Entries are the versions of the feed.
	Entries []struct {
		// Properties are the properties of the version.
		Properties struct {
			// Version is the version, such as 4.4.1.
			Version string `xml:"Version"`

			// Published is the time of the publication of the version, as chocolateyTime.
			Published string `xml:"Published"`

			// Prerelease reports a version of a pre-release.
			Prerelease bool `xml:"IsPrerelease"`
		} `xml:"properties"`
	} `xml:"entry"`
}

// chocolatey returns the latest version of the package id of the Chocolatey community repository,
// with the time of its publication, when it is stable: the version that the repository marks as
// latest. It returns an error that wraps [ErrRegistry] for a package that the repository does not
// have, whose feed has no entry, a request that fails, and a feed or a time that does not decode.
func (r *Resolver) chocolatey(ctx context.Context, id string) ([]candidate, error) {
	address := r.Registries.Chocolatey + "/Packages()?$filter=" +
		url.PathEscape("Id eq '"+id+"' and IsLatestVersion")
	data, err := r.document(ctx, address)
	if err != nil {
		return nil, err
	}
	var feed chocolateyFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("%w: decode %s: %w", ErrRegistry, address, err)
	}
	if len(feed.Entries) == 0 {
		return nil, fmt.Errorf("%w: %s has no package %s", ErrRegistry, r.Registries.Chocolatey, id)
	}
	var candidates []candidate
	for _, entry := range feed.Entries {
		version := entry.Properties.Version
		if !stableVersion.MatchString(version) || entry.Properties.Prerelease {
			continue
		}
		published, err := time.Parse(chocolateyTime, entry.Properties.Published)
		if err != nil {
			return nil, fmt.Errorf("%w: the time of %s %s: %w", ErrRegistry, id, version, err)
		}
		candidates = append(candidates, candidate{published: published, rank: order(version), version: version})
	}
	return candidates, nil
}
