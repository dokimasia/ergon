// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/ergon/service/forge"
)

// releaseRoute is the route of the releases of the repository of the cases.
const releaseRoute = "POST /repos/" + repo + "/releases"

// uploadURL is the upload URL of the release of the cases, with the template of its parameters.
const uploadURL = "https://uploads.github.com/repos/" + repo + "/releases/7/assets{?name,label}"

// The routes of the first two pages of the list of the releases of the repository of the cases.
const (
	firstPage  = "GET /repos/" + repo + "/releases?per_page=100&page=1"
	secondPage = "GET /repos/" + repo + "/releases?per_page=100&page=2"
)

// The asset of the release of the cases, with the digest that GitHub states for it.
const (
	assetName   = "uv-x86_64-unknown-linux-gnu.tar.gz"
	assetURL    = "https://github.com/astral-sh/uv/releases/download/0.12.23/" + assetName
	assetDigest = "42dca04bdc12ce3778b84256a791a26b1d17c07e19badffec4c40349c26db8c7"
)

// published is the time of the publication of the release of the cases.
const published = "2026-09-30T15:23:18Z"

func TestReleases(t *testing.T) {
	t.Parallel()

	t.Run("CreateDraft", func(t *testing.T) {
		t.Parallel()

		t.Run("creates the draft of a tag with its body", func(t *testing.T) {
			t.Parallel()
			created := response{status: http.StatusCreated, body: `{"id":7,"tag_name":"v1.0.0-rc.1","draft":true,` +
				`"upload_url":"` + uploadURL + `","assets":[]}`}
			c, fake := serve(t, map[string][]response{releaseRoute: {created}})
			got, err := c.CreateDraft(t.Context(), repo, "v1.0.0-rc.1", "### Major Changes", true)
			assert.NoError(t, err, "CreateDraft")
			assert.Equal(t, fake.recorded()[0].body, `{"tag_name":"v1.0.0-rc.1","name":"v1.0.0-rc.1",`+
				`"body":"### Major Changes","prerelease":true,"draft":true}`, "the release")
			assert.Equal(t, got, forge.Release{
				ID: 7, Tag: "v1.0.0-rc.1", Draft: true, UploadURL: uploadURL,
				Assets: []forge.Asset{},
			}, "the draft")
		})

		t.Run("returns ErrGitHub for a tag that has a release", func(t *testing.T) {
			t.Parallel()
			exists := response{status: http.StatusUnprocessableEntity, body: `{"message":"Validation Failed"}`}
			c, _ := serve(t, map[string][]response{releaseRoute: {exists}})
			_, err := c.CreateDraft(t.Context(), repo, "v1.0.0", "", false)
			assert.ErrorIs(t, err, forge.ErrGitHub, "CreateDraft")
		})
	})

	t.Run("ReleaseOf", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the draft of a tag", func(t *testing.T) {
			t.Parallel()
			body := `[{"id":8,"tag_name":"v2.0.0","draft":true,"published_at":null,"assets":[]},` +
				`{"id":7,"tag_name":"v1.0.0","draft":false,"published_at":"` + published + `","assets":[]}]`
			c, _ := serve(t, map[string][]response{firstPage: {{body: body}}})
			got, found, err := c.ReleaseOf(t.Context(), repo, "v2.0.0")
			assert.NoError(t, err, "ReleaseOf")
			assert.True(t, found, "whether the tag has a release")
			assert.Equal(t, got, forge.Release{ID: 8, Tag: "v2.0.0", Draft: true, Assets: []forge.Asset{}}, "the draft")
		})

		t.Run("reports false for a tag without a release", func(t *testing.T) {
			t.Parallel()
			body := `[{"id":7,"tag_name":"v1.0.0","published_at":"` + published + `","assets":[]}]`
			c, _ := serve(t, map[string][]response{firstPage: {{body: body}}})
			_, found, err := c.ReleaseOf(t.Context(), repo, "v2.0.0")
			assert.NoError(t, err, "ReleaseOf")
			assert.False(t, found, "whether the tag has a release")
		})

		t.Run("returns ErrGitHub for a repository that does not exist", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			_, _, err := c.ReleaseOf(t.Context(), repo, "v1.0.0")
			assert.ErrorIs(t, err, forge.ErrGitHub, "ReleaseOf")
		})
	})

	t.Run("UploadAsset", func(t *testing.T) {
		t.Parallel()

		t.Run("uploads the file to the upload URL of the release under its base name", func(t *testing.T) {
			t.Parallel()
			fake := &gitHub{routes: map[string][]response{
				"POST /uploads/releases/7/assets?name=ergon_0.6.0_linux_amd64.tar.gz": {{status: http.StatusCreated}},
			}}
			srv := httptest.NewServer(fake)
			t.Cleanup(srv.Close)
			c, err := forge.New(srv.Client(), forge.Config{Token: token, API: srv.URL})
			assert.NoError(t, err, "New")
			path := filepath.Join(t.TempDir(), "ergon_0.6.0_linux_amd64.tar.gz")
			assert.NoError(t, os.WriteFile(path, []byte("archive"), 0o600), "the asset")
			r := forge.Release{ID: 7, UploadURL: srv.URL + "/uploads/releases/7/assets{?name,label}"}
			assert.NoError(t, c.UploadAsset(t.Context(), &r, path), "UploadAsset")
			upload := fake.recorded()[0]
			expect.Equal(t, upload.body, "archive", "the body of the upload")
			expect.Equal(t, upload.header.Get("Content-Type"), "application/octet-stream", "the media type")
		})

		t.Run("returns the error of a file that does not exist", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			absent := filepath.Join(t.TempDir(), "absent")
			err := c.UploadAsset(t.Context(), &forge.Release{UploadURL: uploadURL}, absent)
			assert.ErrorIs(t, err, fs.ErrNotExist, "UploadAsset")
		})

		t.Run("returns ErrGitHub for an asset that the release already has", func(t *testing.T) {
			t.Parallel()
			fake := &gitHub{routes: map[string][]response{}}
			srv := httptest.NewServer(fake)
			t.Cleanup(srv.Close)
			c, err := forge.New(srv.Client(), forge.Config{Token: token, API: srv.URL})
			assert.NoError(t, err, "New")
			path := filepath.Join(t.TempDir(), "checksums.txt")
			assert.NoError(t, os.WriteFile(path, []byte("sums"), 0o600), "the asset")
			r := forge.Release{ID: 7, UploadURL: srv.URL + "/uploads/releases/7/assets{?name,label}"}
			assert.ErrorIs(t, c.UploadAsset(t.Context(), &r, path), forge.ErrGitHub, "UploadAsset")
		})
	})

	t.Run("Publish", func(t *testing.T) {
		t.Parallel()

		t.Run("publishes the draft", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{"PATCH /repos/" + repo + "/releases/7": {{}}})
			assert.NoError(t, c.Publish(t.Context(), repo, 7), "Publish")
			assert.Equal(t, fake.recorded()[0].body, `{"draft":false}`, "the change of the release")
		})

		t.Run("returns ErrGitHub for a release that does not exist", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			assert.ErrorIs(t, c.Publish(t.Context(), repo, 7), forge.ErrGitHub, "Publish")
		})
	})

	t.Run("Releases", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the tag, the time and the assets of each release", func(t *testing.T) {
			t.Parallel()
			body := `[{"tag_name":"0.12.23","published_at":"` + published + `","draft":false,"assets":[` +
				`{"name":"` + assetName + `","browser_download_url":"` + assetURL + `","digest":"sha256:` + assetDigest +
				`"}]}]`
			c, _ := serve(t, map[string][]response{firstPage: {{body: body}}})
			got, err := c.Releases(t.Context(), repo)
			assert.NoError(t, err, "Releases")
			when, _ := time.Parse(time.RFC3339, published)
			assert.Equal(t, got, []forge.Release{{
				Published: when,
				Tag:       "0.12.23",
				Assets:    []forge.Asset{{Name: assetName, URL: assetURL, Digest: assetDigest}},
			}}, "the releases")
		})

		t.Run("leaves the digest empty for an asset without a SHA-256", func(t *testing.T) {
			t.Parallel()
			body := `[{"tag_name":"v1.0.0","published_at":"` + published + `","assets":[` +
				`{"name":"a.zip","browser_download_url":"https://example.com/a.zip","digest":null},` +
				`{"name":"b.zip","browser_download_url":"https://example.com/b.zip","digest":"sha512:00ff"}]}]`
			c, _ := serve(t, map[string][]response{firstPage: {{body: body}}})
			got, err := c.Releases(t.Context(), repo)
			assert.NoError(t, err, "Releases")
			assert.Length(t, got, 1, "the releases")
			assert.Length(t, got[0].Assets, 2, "the assets")
			expect.Empty(t, got[0].Assets[0].Digest, "the digest of the asset without one")
			expect.Empty(t, got[0].Assets[1].Digest, "the digest of the asset of SHA-512")
		})

		t.Run("leaves out a draft", func(t *testing.T) {
			t.Parallel()
			body := `[{"tag_name":"v2.0.0","draft":true,"published_at":null,"assets":[]},` +
				`{"tag_name":"v1.0.0","draft":false,"published_at":"` + published + `","assets":[]}]`
			c, _ := serve(t, map[string][]response{firstPage: {{body: body}}})
			got, err := c.Releases(t.Context(), repo)
			assert.NoError(t, err, "Releases")
			assert.Length(t, got, 1, "the releases")
			assert.Equal(t, got[0].Tag, "v1.0.0", "the tag of the published release")
		})

		t.Run("reads the next page after a full page", func(t *testing.T) {
			t.Parallel()
			tags := make([]string, 0, 100)
			for k := range 100 {
				tags = append(
					tags,
					`{"tag_name":"v1.0.`+strconv.Itoa(k)+`","published_at":"`+published+`","assets":[]}`,
				)
			}
			c, fake := serve(t, map[string][]response{
				firstPage:  {{body: "[" + strings.Join(tags, ",") + "]"}},
				secondPage: {{body: `[{"tag_name":"v0.9.0","published_at":"` + published + `","assets":[]}]`}},
			})
			got, err := c.Releases(t.Context(), repo)
			assert.NoError(t, err, "Releases")
			assert.Length(t, got, 101, "the releases of both pages")
			expect.Equal(t, got[100].Tag, "v0.9.0", "the release of the second page")
			expect.Length(t, fake.recorded(), 2, "the requests")
		})

		t.Run("returns ErrGitHub for a repository that does not exist", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			_, err := c.Releases(t.Context(), repo)
			assert.ErrorIs(t, err, forge.ErrGitHub, "Releases")
		})
	})
}
