// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// releasesPage is the number of releases of one page of the list of the releases of a repository,
// the most that the API returns.
const releasesPage = 100

// digestPrefix starts the digest of an asset that GitHub states as a SHA-256, as sha256:<hex>.
const digestPrefix = "sha256:"

// newRelease is the body of a request that creates a GitHub Release.
type newRelease struct {
	// Tag is the tag of the release.
	Tag string `json:"tag_name"`

	// Name is the title of the release.
	Name string `json:"name"`

	// Body is the description of the release in Markdown.
	Body string `json:"body"`

	// Prerelease marks the release as a pre-release.
	Prerelease bool `json:"prerelease"`
}

// Asset is a file of a release.
type Asset struct {
	// Name is the name of the file, such as uv-x86_64-unknown-linux-gnu.tar.gz.
	Name string `json:"name"`

	// URL is the address from which the file downloads.
	URL string `json:"browser_download_url"`

	// Digest is the SHA-256 of the file in lowercase hexadecimal, and empty for a file without a
	// digest of that algorithm on GitHub.
	Digest string `json:"digest"`
}

// Release is a published release of a repository on GitHub.
type Release struct {
	// Published is the time at which the release was published.
	Published time.Time `json:"published_at"`

	// Tag is the tag of the release, such as v7.0.1.
	Tag string `json:"tag_name"`

	// Assets are the files of the release.
	Assets []Asset `json:"assets"`
}

// CreateRelease creates the GitHub Release of the tag of repo, titled after the tag, with the
// Markdown body, and marked as a pre-release for prerelease, as changesets/action creates the
// release of a published package. It returns an error that wraps [ErrGitHub] for a request that
// fails, such as for a tag that has a release.
func (c *Client) CreateRelease(ctx context.Context, repo, tag, body string, prerelease bool) error {
	in := newRelease{Tag: tag, Name: tag, Body: body, Prerelease: prerelease}
	_, err := c.rest(ctx, http.MethodPost, "/repos/"+repo+"/releases", in, nil)
	return err
}

// Releases returns the published releases of repo, as owner/name, in the order of the API: the
// newest first. It reads the list page by page, 100 releases at a time, and leaves out a draft. It
// returns an error that wraps [ErrGitHub] for a request that fails, such as for a repository that
// does not exist.
func (c *Client) Releases(ctx context.Context, repo string) ([]Release, error) {
	var releases []Release
	for page := 1; ; page++ {
		var batch []struct {
			Release

			// Draft reports a release that is not published.
			Draft bool `json:"draft"`
		}
		path := "/repos/" + repo + "/releases?per_page=" + strconv.Itoa(releasesPage) + "&page=" + strconv.Itoa(page)
		if _, err := c.rest(ctx, http.MethodGet, path, nil, &batch); err != nil {
			return nil, err
		}
		for _, r := range batch {
			if r.Draft {
				continue
			}
			for k := range r.Assets {
				digest, ok := strings.CutPrefix(r.Assets[k].Digest, digestPrefix)
				if !ok {
					digest = ""
				}
				r.Assets[k].Digest = digest
			}
			releases = append(releases, r.Release)
		}
		if len(batch) < releasesPage {
			return releases, nil
		}
	}
}
