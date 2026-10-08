// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"net/http"
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

	t.Run("CreateRelease", func(t *testing.T) {
		t.Parallel()

		t.Run("creates the release of a tag with its body", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{releaseRoute: {{status: http.StatusCreated}}})
			assert.NoError(
				t,
				c.CreateRelease(t.Context(), repo, "v1.0.0-rc.1", "### Major Changes", true),
				"CreateRelease",
			)
			assert.Equal(
				t,
				fake.recorded()[0].body,
				`{"tag_name":"v1.0.0-rc.1","name":"v1.0.0-rc.1","body":"### Major Changes","prerelease":true}`,
				"the release",
			)
		})

		t.Run("returns ErrGitHub for a tag that has a release", func(t *testing.T) {
			t.Parallel()
			exists := response{status: http.StatusUnprocessableEntity, body: `{"message":"Validation Failed"}`}
			c, _ := serve(t, map[string][]response{releaseRoute: {exists}})
			err := c.CreateRelease(t.Context(), repo, "v1.0.0", "", false)
			assert.ErrorIs(t, err, forge.ErrGitHub, "CreateRelease")
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
