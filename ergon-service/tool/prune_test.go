// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/tool"
)

// The installs of the prune cases: the release binary tool and golangci-lint at the versions of
// demo, as the slash-separated start of their paths in the cache.
const (
	toolInstall = "release/tool/"
	lintInstall = "module/github.com/golangci/golangci-lint/v2/cmd/golangci-lint/v2.14.0/"
)

// stale is the content of each file of the prune cases that is not the install of a tool.
var stale = files.Text("stale")

func TestPrune(t *testing.T) {
	t.Parallel()

	t.Run("Prune", func(t *testing.T) {
		t.Parallel()

		t.Run("removes each file that is not the install of a tool of the sections", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.Tool = release(served, "tool.tar.gz", entry, toolTarGz)
			o.Tools.UV = uv(served, host())
			r, _, _ := runner(t, served)
			for _, name := range []string{"tool", "ruff", "golangci-lint", "cargo-audit", "phpstan"} {
				_, err := r.Run(t.Context(), section, o, name, nil)
				assert.NoError(t, err, "the Run of "+name)
			}
			installed := cached(t, r.Cache)
			program := slices.IndexFunc(installed, func(p string) bool {
				return strings.HasPrefix(filepath.ToSlash(p), toolInstall)
			})
			module := slices.IndexFunc(installed, func(p string) bool {
				return strings.HasPrefix(filepath.ToSlash(p), lintInstall)
			})
			assert.NotEqual(t, program, -1, "the index of the program of the release binary")
			assert.NotEqual(t, module, -1, "the index of the program of golangci-lint")
			download := path.Join(path.Dir(filepath.ToSlash(installed[program])), ".download-1")
			otherGo := path.Join(lintInstall, "0000000000000000")
			removed := []string{
				"release/tool/0.9", "release/gone", download, otherGo, "module/example.com", "crate/cargo-audit/0.21.0",
				"maven/gone.jar", "composer/demo/0000000000000000", "composer/other", "unknown",
			}
			want := make([]string, 0, len(removed))
			for _, p := range removed {
				want = append(want, filepath.FromSlash(p))
			}
			var got []string
			var err error
			// files.Write adds the files that are not the install of a tool to the cache, and Prune
			// removes them again, so the cache ends as it started.
			assert.Pure(t, func() []string { return cached(t, r.Cache) }, func() {
				files.Write(t, r.Cache, files.Tree{
					"release/tool/0.9/tool":                        stale,
					"release/gone/1.0/gone":                        stale,
					download:                                       stale,
					otherGo + "/golangci-lint":                     stale,
					"module/example.com/gone/v1.0.0/gone":          stale,
					"crate/cargo-audit/0.21.0/bin/cargo-audit":     stale,
					"maven/gone.jar":                               stale,
					"composer/demo/0000000000000000/composer.json": stale,
					"composer/other/composer.json":                 stale,
					"unknown/file":                                 stale,
				})
				got, err = r.Prune(t.Context(), map[string]language.Options{section: o})
			}, "the files of the cache")
			assert.NoError(t, err, "Prune")
			assert.Permutation(t, got, want, "the removed paths")
		})

		t.Run("keeps the program of golangci-lint of the plugins of its section", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			first := withPlugins(option.Plugins{"lint": "example.com/lint@v1.0.0"})
			_, err := r.Run(t.Context(), section, first, "golangci-lint", nil)
			assert.NoError(t, err, "the Run with the first plugins")
			before := cached(t, r.Cache)
			second := withPlugins(option.Plugins{"lint": "example.com/lint@v2.0.0"})
			_, err = r.Run(t.Context(), section, second, "golangci-lint", nil)
			assert.NoError(t, err, "the Run with the second plugins")
			after := cached(t, r.Cache)
			built := slices.IndexFunc(before, func(p string) bool {
				return strings.Contains(filepath.ToSlash(p), "/"+pluginsDir+"/")
			})
			assert.NotEqual(t, built, -1, "the index of a file of the first build")
			dir := filepath.Dir(before[built])
			got, err := r.Prune(t.Context(), map[string]language.Options{section: second})
			assert.NoError(t, err, "Prune")
			assert.Equal(t, got, []string{dir}, "the removed paths")
			want := slices.DeleteFunc(after, func(p string) bool {
				return strings.HasPrefix(p, dir+string(filepath.Separator))
			})
			assert.Equal(t, cached(t, r.Cache), want, "the files of the cache")
		})

		t.Run("keeps every Go module when go env GOVERSION does not report a version", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, demo(), "golangci-lint", nil)
			assert.NoError(t, err, "Run")
			files.Write(t, r.Cache, files.Tree{"module/example.com/gone/v1.0.0/gone": stale})
			r.Env = append(r.Env, versionFailsEnv+"=1")
			var got []string
			assert.Pure(t, func() []string { return cached(t, r.Cache) }, func() {
				got, err = r.Prune(t.Context(), map[string]language.Options{section: demo()})
			}, "the files of the cache")
			assert.NoError(t, err, "Prune")
			assert.Empty(t, got, "the removed paths")
		})

		t.Run("removes every install for sections without tools", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.Tool = release(served, "tool.tar.gz", entry, toolTarGz)
			r, _, _ := runner(t, served)
			for _, name := range []string{"tool", "golangci-lint"} {
				_, err := r.Run(t.Context(), section, o, name, nil)
				assert.NoError(t, err, "the Run of "+name)
			}
			got, err := r.Prune(t.Context(), map[string]language.Options{section: &toolless{}})
			assert.NoError(t, err, "Prune")
			assert.Permutation(t, got, []string{"module", "release"}, "the removed paths")
			assert.Empty(t, cached(t, r.Cache), "the files of the cache")
		})

		t.Run("returns nil for a cache that does not exist", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			r.Cache = filepath.Join(r.Cache, "missing")
			got, err := r.Prune(t.Context(), map[string]language.Options{section: demo()})
			assert.NoError(t, err, "Prune")
			assert.Nil(t, got, "the removed paths")
			files.Absent(t, r.Cache, "the cache")
		})

		t.Run("returns ErrInstall when the section has no plugins at the key of the tag plugins", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			_, err := r.Prune(t.Context(), map[string]language.Options{section: &unplugged{}})
			assert.ErrorIs(t, err, tool.ErrInstall, "Prune")
		})

		t.Run("returns the error of a file that it cannot remove", func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == windows {
				t.Skip("Windows removes the entries of a directory with the read-only attribute")
			}
			if os.Geteuid() == 0 {
				t.Skip("root removes the entries of a directory without write permission")
			}
			r, _, _ := runner(t, map[string][]byte{})
			files.Write(t, r.Cache, files.Tree{"release/gone/1.0/gone": stale})
			locked := filepath.Join(r.Cache, "release")
			assert.NoError(t, os.Chmod(locked, 0o555), "Chmod of the directory")
			t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
			_, err := r.Prune(t.Context(), nil)
			assert.HasError(t, err, "Prune")
			assert.HasPrefix(t, err.Error(), "tool: remove release from the cache: ", "the error")
		})

		t.Run("returns the error of a cache that is not a directory", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			r.Cache = filepath.Join(r.Cache, "file")
			assert.NoError(t, os.WriteFile(r.Cache, nil, 0o644), "WriteFile of the cache")
			_, err := r.Prune(t.Context(), nil)
			assert.HasError(t, err, "Prune")
			assert.HasPrefix(t, err.Error(), "tool: open the cache: ", "the error")
		})
	})
}

// cached returns the files below root, relative to it, in lexical order.
func cached(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(root, p)
			paths = append(paths, rel)
		}
		return err
	})
	assert.NoError(t, err, "WalkDir of the cache")
	return paths
}
