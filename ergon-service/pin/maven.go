// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

// mavenMetadata is the part of maven-metadata.xml of an artifact that a Resolver reads.
type mavenMetadata struct {
	// Versions are the versions of the artifact.
	Versions []string `xml:"versioning>versions>version"`
}

// maven returns the stable versions of the Maven artifact of p newer than the version of p, each
// with the time of the last modification of its POM, from maven-metadata.xml of the artifact in the
// Maven repository. It returns an error that wraps [ErrRegistry] for an artifact that the
// repository does not have, a version without a POM or without its time, a request that fails, and
// a document that does not decode.
func (r *Resolver) maven(ctx context.Context, p *Pin) ([]candidate, error) {
	group, artifact, _ := strings.Cut(p.Name, ":")
	base := r.Registries.Maven + "/" + strings.ReplaceAll(group, ".", "/") + "/" + artifact
	data, err := r.document(ctx, base+"/maven-metadata.xml")
	if err != nil {
		return nil, err
	}
	var metadata mavenMetadata
	if err := xml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("%w: decode the metadata of %s: %w", ErrRegistry, p.Name, err)
	}
	current := order(p.Version)
	var candidates []candidate
	for _, v := range metadata.Versions {
		n := order(v)
		if !stableVersion.MatchString(v) || !n.above(current) {
			continue
		}
		pom := base + "/" + v + "/" + artifact + "-" + v + ".pom"
		_, header, found, err := r.get(ctx, http.MethodHead, pom)
		if err == nil && !found {
			err = fmt.Errorf("%w: %s %s has no POM", ErrRegistry, p.Name, v)
		}
		if err != nil {
			return nil, err
		}
		published, err := http.ParseTime(header.Get("Last-Modified"))
		if err != nil {
			return nil, fmt.Errorf("%w: the time of %s %s: %w", ErrRegistry, p.Name, v, err)
		}
		candidates = append(candidates, candidate{published: published, rank: n, version: v})
	}
	return candidates, nil
}
