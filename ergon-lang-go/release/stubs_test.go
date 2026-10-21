// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/workspace"
)

// The stubs of the libraries of macOS that a pack writes for the standard library of Go, by their
// paths in the directory of ERGON_DARWIN_STUBS.
const (
	resolvStub         = "libresolv.tbd"
	coreFoundationStub = "CoreFoundation.framework/CoreFoundation.tbd"
	securityStub       = "Security.framework/Security.tbd"
)

// resolv is the stub of libresolv, from the three symbols that the standard library of Go imports
// from it on darwin.
const resolv = `--- !tapi-tbd
tbd-version: 4
targets: [ x86_64-macos, arm64-macos ]
install-name: '/usr/lib/libresolv.9.dylib'
current-version: 1
compatibility-version: 1
exports:
  - targets: [ x86_64-macos, arm64-macos ]
    symbols: [ _res_9_nclose, _res_9_ninit, _res_9_nsearch ]
...
`

func TestStubs(t *testing.T) {
	t.Parallel()

	t.Run("Pack", func(t *testing.T) {
		t.Parallel()

		t.Run("writes a stub of each library of macOS that the standard library imports but libSystem",
			func(t *testing.T) {
				t.Parallel()
				got := slices.Sorted(maps.Keys(stubs(t)))
				assert.Equal(t, got, []string{coreFoundationStub, securityStub, resolvStub}, "the stubs")
			})

		t.Run("writes the path and the imported symbols of a library", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, stubs(t)[resolvStub], resolv, "the stub of libresolv")
		})

		t.Run("writes the path of a framework in its versions directory", func(t *testing.T) {
			t.Parallel()
			got := stubs(t)
			expect.Contains(t, got[coreFoundationStub],
				"\ninstall-name: '/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation'\n",
				"the stub of CoreFoundation")
			expect.Contains(t, got[securityStub],
				"\ninstall-name: '/System/Library/Frameworks/Security.framework/Versions/A/Security'\n",
				"the stub of Security")
		})
	})
}

// TestStubsEnv runs the cases of the stubs that change the environment of the process, one at a
// time.
func TestStubsEnv(t *testing.T) {
	t.Run("Pack", func(t *testing.T) {
		t.Run("returns an error for a Go toolchain without its source", func(t *testing.T) {
			root := lintRepository(t)
			files.Write(t, root, files.Tree{"lint/.goreleaser.yaml": files.Text(cgoConfig)})
			t.Setenv("GOROOT", files.Workspace(t, files.Tree{}))
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := r.packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.HasError(t, err, "Pack")
			assert.Contains(t, err.Error(), "the stubs of macOS for the builds with cgo of "+pathA, "the error")
			assert.Empty(t, r.runs, "the runs")
		})

		t.Run("returns the error of a source file that does not read", func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("Windows reads a file without the read permission of its owner")
			}
			if os.Geteuid() == 0 {
				t.Skip("root reads a file without the read permission of its owner")
			}
			root := lintRepository(t)
			files.Write(t, root, files.Tree{"lint/.goreleaser.yaml": files.Text(cgoConfig)})
			t.Setenv("GOROOT", files.Workspace(t, files.Tree{
				"src/net/net.go": files.Text("package net\n").WithMode(0o000),
			}))
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := r.packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.ErrorIs(t, err, fs.ErrPermission, "Pack")
			assert.Contains(t, err.Error(), "read net/net.go", "the error")
			assert.Empty(t, r.runs, "the runs")
		})
	})
}

// stubs packs the module lint of the cases with a build with cgo, and returns the content of each
// file of the directory of ERGON_DARWIN_STUBS as the run of GoReleaser finds it, by its path in the
// directory, slash-separated.
func stubs(t *testing.T) map[string]string {
	t.Helper()
	root := lintRepository(t)
	files.Write(t, root, files.Tree{"lint/.goreleaser.yaml": files.Text(cgoConfig)})
	p := (&checkout{}).packer()
	fake := p.Run
	found := map[string]string{}
	p.Run = func(ctx context.Context, dir, tool string, args, env []string) error {
		stubs, _ := variable(env, "ERGON_DARWIN_STUBS")
		err := filepath.WalkDir(stubs, func(name string, d fs.DirEntry, err error) error {
			if err == nil && d.Type().IsRegular() {
				rel, _ := filepath.Rel(stubs, name)
				found[filepath.ToSlash(rel)] = files.Read(t, name)
			}
			return err
		})
		assert.NoError(t, err, "WalkDir of the stubs")
		return fake(ctx, dir, tool, args, env)
	}
	pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
	assert.NoError(t, p.Pack(t.Context(), root, pkgs, t.TempDir()), "Pack")
	return found
}
