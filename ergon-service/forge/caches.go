// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// cachesPage is the number of caches of one page of the list of the caches of a repository, the
// most that the API returns.
const cachesPage = 100

// Cache is a cache of GitHub Actions of a repository.
type Cache struct {
	// Created is the time at which a run saved the cache.
	Created time.Time `json:"created_at"`

	// Ref is the ref whose runs restore the cache, such as refs/heads/main or refs/pull/12/merge.
	Ref string `json:"ref"`

	// Key is the key under which the run saved the cache.
	Key string `json:"key"`

	// ID is the identifier of the cache in the API.
	ID int64 `json:"id"`
}

// Caches returns the caches of repo whose keys start with key, in the order of the API. It reads
// the list page by page, 100 caches at a time. It returns an error that wraps [ErrGitHub] for a
// request that fails, such as for a token without the permission actions: read.
func (c *Client) Caches(ctx context.Context, repo, key string) ([]Cache, error) {
	var caches []Cache
	for page := 1; ; page++ {
		var batch struct {
			Caches []Cache `json:"actions_caches"`
		}
		query := url.Values{"key": {key}, perPage: {strconv.Itoa(cachesPage)}, "page": {strconv.Itoa(page)}}
		path := "/repos/" + repo + "/actions/caches?" + query.Encode()
		if _, err := c.rest(ctx, http.MethodGet, path, nil, &batch); err != nil {
			return nil, err
		}
		caches = append(caches, batch.Caches...)
		if len(batch.Caches) < cachesPage {
			return caches, nil
		}
	}
}

// DeleteCache deletes the cache id of repo. A cache that repo no longer has, such as one that
// GitHub evicted after the list, is no error. It returns an error that wraps [ErrGitHub] for a
// request that fails, such as for a token without the permission actions: write.
func (c *Client) DeleteCache(ctx context.Context, repo string, id int64) error {
	path := "/repos/" + repo + "/actions/caches/" + strconv.FormatInt(id, 10)
	_, err := c.rest(ctx, http.MethodDelete, path, nil, nil, http.StatusNotFound)
	return err
}
