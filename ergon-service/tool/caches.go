// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/service/forge"
)

// CacheForge is the host of the caches of GitHub Actions of a repository, which [PruneCaches] reads
// and deletes. Both methods take the repository as owner/name and return the error of the host.
// The package go.dokimi.dev/ergon/service/forge implements it.
type CacheForge interface {
	// Caches returns the caches of repo whose keys start with key.
	Caches(ctx context.Context, repo, key string) ([]forge.Cache, error)

	// DeleteCache deletes the cache id of repo. It returns nil for a cache that repo no longer has.
	DeleteCache(ctx context.Context, repo string, id int64) error
}

// group is the job of a cache: the ref whose runs restore it, and its key up to the last dash,
// which a job restores when its own key misses.
type group struct {
	// ref is the ref of the cache, such as refs/heads/main.
	ref string

	// prefix is the key of the cache up to its last dash.
	prefix string
}

// PruneCaches deletes each cache of repo whose key starts with prefix and that a newer cache of the
// same job replaced, and returns the deleted caches in the order of the deletions. Caches with the
// same ref and the same key up to its last dash belong to one job. The part after the last dash is
// the digest of the files of the key, and a job whose key misses restores the newest cache of the
// part before it. PruneCaches keeps the newest cache of each job. Of two caches with the same
// creation time, the cache of the higher ID is the newer.
//
// It returns the error of the host. After an error of DeleteCache it returns the caches that it
// deleted before the error.
func PruneCaches(ctx context.Context, f CacheForge, repo, prefix string) ([]forge.Cache, error) {
	caches, err := f.Caches(ctx, repo, prefix)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(caches, func(a, b forge.Cache) int {
		return cmp.Or(b.Created.Compare(a.Created), cmp.Compare(b.ID, a.ID))
	})
	kept := make(map[group]bool, len(caches))
	var deleted []forge.Cache
	for _, c := range caches {
		g := group{ref: c.Ref, prefix: c.Key[:strings.LastIndex(c.Key, "-")+1]}
		if !kept[g] {
			kept[g] = true
			continue
		}
		if err := f.DeleteCache(ctx, repo, c.ID); err != nil {
			return deleted, err
		}
		deleted = append(deleted, c)
	}
	return deleted, nil
}
