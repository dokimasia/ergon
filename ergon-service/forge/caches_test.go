// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// The key of the caches of the cases, and a cache of that key as the API states it.
const (
	cacheKey  = "ergon-tools-"
	cacheJSON = `{"id":8766047619,"ref":"refs/heads/main","key":"ergon-tools-Linux-X64-check-go-ced0",` +
		`"version":"6c14","last_accessed_at":"2026-10-10T10:35:52.427293Z",` +
		`"created_at":"2026-10-10T10:35:52.427293Z","size_in_bytes":59096757}`
)

// The routes of the caches of the cases: the first two pages of the list, and the deletion of the
// cache of cacheJSON.
const (
	cachesRoute      = "GET /repos/" + repo + "/actions/caches?key=" + cacheKey + "&page=1&per_page=100"
	cachesPage2Route = "GET /repos/" + repo + "/actions/caches?key=" + cacheKey + "&page=2&per_page=100"
	deleteCacheRoute = "DELETE /repos/" + repo + "/actions/caches/8766047619"
)

func TestCaches(t *testing.T) {
	t.Parallel()

	t.Run("Caches", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the caches whose keys start with the key", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				cachesRoute: {{body: `{"total_count":1,"actions_caches":[` + cacheJSON + `]}`}},
			})
			got, err := c.Caches(t.Context(), repo, cacheKey)
			assert.NoError(t, err, "Caches")
			assert.Equal(t, got, []forge.Cache{{
				Created: time.Date(2026, 10, 10, 10, 35, 52, 427293000, time.UTC),
				Ref:     "refs/heads/main",
				Key:     "ergon-tools-Linux-X64-check-go-ced0",
				ID:      8766047619,
			}}, "the caches")
		})

		t.Run("reads the next page after a full page", func(t *testing.T) {
			t.Parallel()
			full := strings.Repeat(cacheJSON+",", 99) + cacheJSON
			c, fake := serve(t, map[string][]response{
				cachesRoute:      {{body: `{"total_count":101,"actions_caches":[` + full + `]}`}},
				cachesPage2Route: {{body: `{"total_count":101,"actions_caches":[` + cacheJSON + `]}`}},
			})
			got, err := c.Caches(t.Context(), repo, cacheKey)
			assert.NoError(t, err, "Caches")
			assert.Length(t, got, 101, "the caches")
			assert.Length(t, fake.recorded(), 2, "the requests")
		})

		t.Run("returns ErrGitHub for a token without the permission", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{cachesRoute: {{
				status: http.StatusForbidden, body: `{"message":"Resource not accessible by integration"}`,
			}}})
			_, err := c.Caches(t.Context(), repo, cacheKey)
			assert.ErrorIs(t, err, forge.ErrGitHub, "Caches")
		})
	})

	t.Run("DeleteCache", func(t *testing.T) {
		t.Parallel()

		t.Run("deletes the cache", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{deleteCacheRoute: {{status: http.StatusNoContent}}})
			assert.NoError(t, c.DeleteCache(t.Context(), repo, 8766047619), "DeleteCache")
			assert.Length(t, fake.recorded(), 1, "the requests")
		})

		t.Run("returns nil for a cache that the repository no longer has", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{})
			assert.NoError(t, c.DeleteCache(t.Context(), repo, 8766047619), "DeleteCache")
		})

		t.Run("returns ErrGitHub for a token without the permission", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{deleteCacheRoute: {{
				status: http.StatusForbidden, body: `{"message":"Resource not accessible by integration"}`,
			}}})
			err := c.DeleteCache(t.Context(), repo, 8766047619)
			assert.ErrorIs(t, err, forge.ErrGitHub, "DeleteCache")
		})
	})
}
