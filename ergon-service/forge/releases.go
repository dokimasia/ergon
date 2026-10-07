// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"net/http"
)

// release is the body of a request that creates a GitHub Release.
type release struct {
	// Tag is the tag of the release.
	Tag string `json:"tag_name"`

	// Name is the title of the release.
	Name string `json:"name"`

	// Body is the description of the release in Markdown.
	Body string `json:"body"`

	// Prerelease marks the release as a pre-release.
	Prerelease bool `json:"prerelease"`
}

// CreateRelease creates the GitHub Release of the tag of repo, titled after the tag, with the
// Markdown body, and marked as a pre-release for prerelease, as changesets/action creates the
// release of a published package. It returns an error that wraps [ErrGitHub] for a request that
// fails, such as for a tag that has a release.
func (c *Client) CreateRelease(ctx context.Context, repo, tag, body string, prerelease bool) error {
	in := release{Tag: tag, Name: tag, Body: body, Prerelease: prerelease}
	_, err := c.rest(ctx, http.MethodPost, "/repos/"+repo+"/releases", in, nil)
	return err
}
