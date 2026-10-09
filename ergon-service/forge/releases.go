// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// releasesPage is the number of releases of one page of the list of the releases of a repository,
// the most that the API returns.
const releasesPage = 100

// digestPrefix starts the digest of an asset that GitHub states as a SHA-256, as sha256:<hex>.
const digestPrefix = "sha256:"

// The upload of an asset: the media type of its body, and the parameters that end the upload URL
// of a release, as {?name,label}.
const (
	assetType     = "application/octet-stream"
	uploadOptions = "{"
)

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

	// Draft creates the release as a draft, which no one but the collaborators of the repository
	// sees, and whose assets change until its publish.
	Draft bool `json:"draft"`
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

// Release is a release of a repository on GitHub.
type Release struct {
	// Published is the time at which the release was published, and the zero time for a draft.
	Published time.Time `json:"published_at"`

	// Tag is the tag of the release, such as v7.0.1.
	Tag string `json:"tag_name"`

	// UploadURL is the address to which an asset of the release uploads, followed by the template
	// of its parameters, such as https://uploads.github.com/repos/a/b/releases/1/assets{?name,label}.
	UploadURL string `json:"upload_url"`

	// Assets are the files of the release.
	Assets []Asset `json:"assets"`

	// ID is the identifier of the release in the API.
	ID int64 `json:"id"`

	// Draft reports a release that is not published.
	Draft bool `json:"draft"`
}

// CreateDraft creates the release of the tag of repo as a draft, titled after the tag, with the
// Markdown body, and marked as a pre-release for prerelease, and returns it. It returns an error
// that wraps [ErrGitHub] for a request that fails, such as for a tag that has a release.
func (c *Client) CreateDraft(ctx context.Context, repo, tag, body string, prerelease bool) (Release, error) {
	in := newRelease{Tag: tag, Name: tag, Body: body, Prerelease: prerelease, Draft: true}
	var r Release
	if _, err := c.rest(ctx, http.MethodPost, "/repos/"+repo+"/releases", in, &r); err != nil {
		return Release{}, err
	}
	return r, nil
}

// Releases returns the published releases of repo, as owner/name, in the order of the API: the
// newest first. It reads the list page by page, 100 releases at a time, and leaves out a draft. It
// returns an error that wraps [ErrGitHub] for a request that fails, such as for a repository that
// does not exist.
func (c *Client) Releases(ctx context.Context, repo string) ([]Release, error) {
	all, err := c.releases(ctx, repo)
	if err != nil {
		return nil, err
	}
	published := make([]Release, 0, len(all))
	for _, r := range all {
		if !r.Draft {
			published = append(published, r)
		}
	}
	return published, nil
}

// ReleaseOf returns the release of the tag of repo, a draft included, and reports whether repo has
// one. A token sees the drafts of a repository only with write access to its contents. It returns
// the errors of [Client.Releases].
func (c *Client) ReleaseOf(ctx context.Context, repo, tag string) (Release, bool, error) {
	all, err := c.releases(ctx, repo)
	if err != nil {
		return Release{}, false, err
	}
	for _, r := range all {
		if r.Tag == tag {
			return r, true, nil
		}
	}
	return Release{}, false, nil
}

// UploadAsset uploads the file at path as an asset of the release r, named after the base name of
// the file, to the upload URL of r. It returns the error of reading the file, and an error that
// wraps [ErrGitHub] for a request that fails, such as for a release that has an asset of the name.
func (c *Client) UploadAsset(ctx context.Context, r *Release, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("forge: read the asset %s: %w", path, err)
	}
	address, _, _ := strings.Cut(r.UploadURL, uploadOptions)
	address += "?name=" + url.QueryEscape(filepath.Base(path))
	_, err = c.send(ctx, http.MethodPost, address, bytes.NewReader(data), assetType, nil, nil)
	return err
}

// Publish publishes the draft release id of repo. It returns an error that wraps [ErrGitHub] for a
// request that fails.
func (c *Client) Publish(ctx context.Context, repo string, id int64) error {
	in := struct {
		Draft bool `json:"draft"`
	}{Draft: false}
	_, err := c.rest(ctx, http.MethodPatch, "/repos/"+repo+"/releases/"+strconv.FormatInt(id, 10), in, nil)
	return err
}

// releases returns every release of repo, the drafts included, with the digest of each asset in
// lowercase hexadecimal, as [Client.Releases] reads them.
func (c *Client) releases(ctx context.Context, repo string) ([]Release, error) {
	var releases []Release
	for page := 1; ; page++ {
		var batch []Release
		path := "/repos/" + repo + "/releases?per_page=" + strconv.Itoa(releasesPage) + "&page=" + strconv.Itoa(page)
		if _, err := c.rest(ctx, http.MethodGet, path, nil, &batch); err != nil {
			return nil, err
		}
		for _, r := range batch {
			for k := range r.Assets {
				digest, ok := strings.CutPrefix(r.Assets[k].Digest, digestPrefix)
				if !ok {
					digest = ""
				}
				r.Assets[k].Digest = digest
			}
			releases = append(releases, r)
		}
		if len(batch) < releasesPage {
			return releases, nil
		}
	}
}
