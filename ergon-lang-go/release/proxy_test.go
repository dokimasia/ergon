// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/lang/go/release"
	"golang.org/x/mod/sumdb/dirhash"
)

// errSnapshot is the error of a repository whose working tree git cannot snapshot.
var errSnapshot = errors.New("snapshot failed")

func TestProxy(t *testing.T) {
	t.Parallel()

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("serves the go.mod that the go command synthesizes for a module whose tree has none", func(t *testing.T) {
			t.Parallel()
			root := repository(t, files.Tree{"a/.gitignore": files.Text("go.mod\n")})
			_, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.NoError(t, err, "Apply")
			synthesized := "module " + pathA + "\n"
			hash, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(synthesized)), nil
			})
			assert.NoError(t, err, "Hash1")
			assert.Contains(t, files.Read(t, filepath.Join(root, "b", "go.sum")), pathA+" v0.2.0/go.mod "+hash+"\n",
				"the go.sum of b")
		})

		t.Run("returns the error of a module whose path the file system cannot name", func(t *testing.T) {
			t.Parallel()
			long := "example.com/" + strings.Repeat("a", 300)
			root := repository(t, files.Tree{
				"a/go.mod": files.Text(modOf(long)),
				"b/go.mod": files.Text(modOf(pathB, long+" v0.1.0")),
				"b/b.go":   files.Text("package b\n"),
			})
			edits := releaseOfA()
			edits[0].Package.Name = long
			edits[1].Requirements[0].Name = long
			_, err := versioner.Apply(t.Context(), root, edits)
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "serve "+long+"@v0.2.0", "the error")
		})

		t.Run("returns the error of a version whose files the file system cannot name", func(t *testing.T) {
			t.Parallel()
			root := repository(t, nil)
			edits := releaseOfA()
			edits[0].Version = version.Version{Minor: 2, Pre: strings.Repeat("a", 300)}
			edits[1].Requirements[0].Req = "v" + edits[0].Version.String()
			_, err := versioner.Apply(t.Context(), root, edits)
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "serve "+pathA+"@v0.2.0-", "the error")
		})

		t.Run("returns the error of the snapshot", func(t *testing.T) {
			t.Parallel()
			failing := release.Versioner{
				Snapshot: func(context.Context, string) (string, error) { return "", errSnapshot },
			}
			_, err := failing.Apply(t.Context(), repository(t, nil), releaseOfA())
			assert.ErrorIs(t, err, errSnapshot, "Apply")
		})

		t.Run("returns the error of the zip of a tree that the repository does not have", func(t *testing.T) {
			t.Parallel()
			absent := release.Versioner{Snapshot: func(context.Context, string) (string, error) {
				return strings.Repeat("0", 40), nil
			}}
			_, err := absent.Apply(t.Context(), repository(t, nil), releaseOfA())
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "write the zip of "+pathA+"@v0.2.0", "the error")
		})
	})
}
