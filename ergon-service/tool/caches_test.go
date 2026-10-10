// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/tool"
)

// The repository of the cache cases, and the start of the keys of its caches.
const (
	cachesRepo = "dokimasia/ergon"
	cachesKey  = "ergon-tools-"
)

// The keys of the caches of the cache cases: three digests of the job check-go, a digest of the job
// check-go on the runtime version 1.27, and a digest of the job commits.
const (
	checkA       = "ergon-tools-Linux-X64-check-go-aaaa"
	checkB       = "ergon-tools-Linux-X64-check-go-bbbb"
	checkC       = "ergon-tools-Linux-X64-check-go-cccc"
	checkVersion = "ergon-tools-Linux-X64-check-go-1.27-aaaa"
	commitsA     = "ergon-tools-Linux-X64-commits-aaaa"
)

// The refs of the caches of the cache cases.
const (
	mainRef = "refs/heads/main"
	pullRef = "refs/pull/14/merge"
)

// epoch is the time of the oldest cache of the cache cases.
var epoch = time.Date(2026, 10, 10, 3, 0, 0, 0, time.UTC)

// errHost is the error of the host of the cache cases.
var errHost = errors.New("forge: GitHub failed")

// caches generates a cache of a ref of the cases, of one of the keys of the cases, and of a time in
// the four hours after epoch.
var caches = prop.Composite(func(c *prop.Case) forge.Cache {
	return forge.Cache{
		Created: epoch.Add(time.Duration(c.Draw(prop.Integer(0, 3), "hours")) * time.Hour),
		Ref:     c.Draw(prop.SampledFrom(mainRef, pullRef), "ref"),
		Key:     c.Draw(prop.SampledFrom(checkA, checkB, checkVersion, commitsA), "key"),
	}
})

// cacheHost is the [tool.CacheForge] of the cache cases. It lists caches, or fails the list with
// err, fails the deletion of the cache failID, and records each call.
type cacheHost struct {
	// err is the error of Caches, or nil.
	err error

	// caches are the caches that Caches returns.
	caches []forge.Cache

	// calls are the method and the arguments of each call, in their order.
	calls []string

	// deleted are the caches that DeleteCache deleted, in their order.
	deleted []int64

	// failID is the cache whose deletion fails with errHost, or 0 for none.
	failID int64
}

// Caches records the call, and returns a copy of h.caches, or h.err.
func (h *cacheHost) Caches(_ context.Context, repo, key string) ([]forge.Cache, error) {
	h.calls = append(h.calls, "Caches "+repo+" "+key)
	if h.err != nil {
		return nil, h.err
	}
	return slices.Clone(h.caches), nil
}

// DeleteCache records the call, and deletes the cache id, or returns errHost for h.failID.
func (h *cacheHost) DeleteCache(_ context.Context, repo string, id int64) error {
	h.calls = append(h.calls, "DeleteCache "+repo+" "+strconv.FormatInt(id, 10))
	if id == h.failID {
		return errHost
	}
	h.deleted = append(h.deleted, id)
	return nil
}

func TestCaches(t *testing.T) {
	t.Parallel()

	t.Run("PruneCaches", func(t *testing.T) {
		t.Parallel()

		t.Run("deletes each cache that a newer cache of its job replaced", func(t *testing.T) {
			t.Parallel()
			a := forge.Cache{ID: 1, Ref: mainRef, Key: checkA, Created: epoch}
			b := forge.Cache{ID: 2, Ref: mainRef, Key: checkB, Created: epoch.Add(time.Hour)}
			c := forge.Cache{ID: 3, Ref: mainRef, Key: checkC, Created: epoch.Add(2 * time.Hour)}
			pull := forge.Cache{ID: 4, Ref: pullRef, Key: checkA, Created: epoch}
			version := forge.Cache{ID: 5, Ref: mainRef, Key: checkVersion, Created: epoch}
			commits := forge.Cache{ID: 6, Ref: mainRef, Key: commitsA, Created: epoch}
			h := &cacheHost{caches: []forge.Cache{a, pull, b, version, c, commits}}
			got, err := tool.PruneCaches(t.Context(), h, cachesRepo, cachesKey)
			assert.NoError(t, err, "PruneCaches")
			assert.Equal(t, got, []forge.Cache{b, a}, "the deleted caches")
			assert.Equal(t, h.calls, []string{
				"Caches " + cachesRepo + " " + cachesKey,
				"DeleteCache " + cachesRepo + " 2",
				"DeleteCache " + cachesRepo + " 1",
			}, "the calls of the host")
		})

		t.Run("deletes the cache of the lower ID of two caches of one time", func(t *testing.T) {
			t.Parallel()
			older := forge.Cache{ID: 7, Ref: mainRef, Key: checkA, Created: epoch}
			newer := forge.Cache{ID: 8, Ref: mainRef, Key: checkB, Created: epoch}
			h := &cacheHost{caches: []forge.Cache{newer, older}}
			got, err := tool.PruneCaches(t.Context(), h, cachesRepo, cachesKey)
			assert.NoError(t, err, "PruneCaches")
			assert.Equal(t, got, []forge.Cache{older}, "the deleted caches")
		})

		t.Run("deletes no cache of a job with one cache", func(t *testing.T) {
			t.Parallel()
			h := &cacheHost{caches: []forge.Cache{
				{ID: 1, Ref: mainRef, Key: checkA, Created: epoch},
				{ID: 4, Ref: pullRef, Key: checkB, Created: epoch.Add(time.Hour)},
				{ID: 5, Ref: mainRef, Key: checkVersion, Created: epoch.Add(time.Hour)},
			}}
			got, err := tool.PruneCaches(t.Context(), h, cachesRepo, cachesKey)
			assert.NoError(t, err, "PruneCaches")
			assert.Empty(t, got, "the deleted caches")
			assert.Empty(t, h.deleted, "the deletions of the host")
		})

		t.Run("returns the error of the list", func(t *testing.T) {
			t.Parallel()
			h := &cacheHost{err: errHost}
			got, err := tool.PruneCaches(t.Context(), h, cachesRepo, cachesKey)
			assert.ErrorIs(t, err, errHost, "PruneCaches")
			assert.Nil(t, got, "the deleted caches")
		})

		t.Run("returns the caches that it deleted before an error of a deletion", func(t *testing.T) {
			t.Parallel()
			a := forge.Cache{ID: 1, Ref: mainRef, Key: checkA, Created: epoch}
			b := forge.Cache{ID: 2, Ref: mainRef, Key: checkB, Created: epoch.Add(time.Hour)}
			c := forge.Cache{ID: 3, Ref: mainRef, Key: checkC, Created: epoch.Add(2 * time.Hour)}
			h := &cacheHost{caches: []forge.Cache{a, b, c}, failID: a.ID}
			got, err := tool.PruneCaches(t.Context(), h, cachesRepo, cachesKey)
			assert.ErrorIs(t, err, errHost, "PruneCaches")
			assert.Equal(t, got, []forge.Cache{b}, "the deleted caches")
		})

		t.Run("keeps the newest cache of each job", func(t *testing.T) {
			t.Parallel()
			contract := "PruneCaches deletes each cache that a newer cache of its ref and its job replaced"
			prop.ForAll(t, contract, func(c *prop.Case) {
				listed := c.Draw(prop.List(caches, prop.MaxSize(12)), "caches")
				for i := range listed {
					listed[i].ID = int64(i + 1)
				}
				h := &cacheHost{caches: listed}
				_, err := tool.PruneCaches(c.Context(), h, cachesRepo, cachesKey)
				assert.NoError(c, err, "PruneCaches")
				var replaced []int64
				for _, x := range listed {
					job := x.Key[:strings.LastIndex(x.Key, "-")]
					for _, y := range listed {
						sameJob := y.Ref == x.Ref && y.Key[:strings.LastIndex(y.Key, "-")] == job
						newer := y.Created.After(x.Created) || y.Created.Equal(x.Created) && y.ID > x.ID
						if sameJob && newer {
							replaced = append(replaced, x.ID)
							break
						}
					}
				}
				assert.Permutation(c, h.deleted, replaced, "the deleted caches", assert.EquateEmpty())
			})
		})
	})
}
