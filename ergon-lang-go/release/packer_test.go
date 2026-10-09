// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/go/release"
)

// The commits of the repository of the packer of the cases.
const (
	headCommit  = "1111111111111111111111111111111111111111"
	otherCommit = "2222222222222222222222222222222222222222"
)

// The configuration of GoReleaser of the module lint of the cases, and its dist directory.
const (
	lintConfig = "dist: dist/goreleaser/demo-lint\nversion: 2\n"
	lintDist   = "dist/goreleaser/demo-lint"
)

// artifacts is the artifacts.json that the fake GoReleaser of the cases writes: an archive, its
// SBOM, the checksums, their signature, a package, the source archive, a cask, and a binary and the
// metadata, which a release does not attach.
const artifacts = `[
	{"name":"assertlint","path":"dist/goreleaser/demo-lint/assertlint_linux_amd64_v1/assertlint","type":"Binary"},
	{"name":"assertlint_0.2.0_linux_amd64.tar.gz","path":"dist/goreleaser/demo-lint/assertlint_0.2.0_linux_amd64.tar.gz","type":"Archive"},
	{"name":"assertlint_0.2.0_linux_amd64.tar.gz.sbom.json","path":"dist/goreleaser/demo-lint/assertlint_0.2.0_linux_amd64.tar.gz.sbom.json","type":"SBOM"},
	{"name":"assertlint_0.2.0_amd64.deb","path":"dist/goreleaser/demo-lint/assertlint_0.2.0_amd64.deb","type":"Linux Package"},
	{"name":"demo-lint_0.2.0_source.tar.gz","path":"dist/goreleaser/demo-lint/demo-lint_0.2.0_source.tar.gz","type":"Source"},
	{"name":"checksums.txt","path":"dist/goreleaser/demo-lint/checksums.txt","type":"Checksum"},
	{"name":"checksums.txt.sigstore.json","path":"dist/goreleaser/demo-lint/checksums.txt.sigstore.json","type":"Signature"},
	{"name":"assertlint.rb","path":"dist/goreleaser/demo-lint/homebrew/Casks/assertlint.rb","type":"Homebrew Cask"},
	{"name":"metadata.json","path":"dist/goreleaser/demo-lint/metadata.json","type":"Metadata"}
]`

// errGoReleaser is the error of a run of GoReleaser that fails.
var errGoReleaser = errors.New("goreleaser failed")

// run is a run of a tool that the packer of the cases started.
type run struct {
	// dir is the directory of the run.
	dir string

	// tool is the tool of the section go.
	tool string

	// args are the arguments of the tool.
	args []string

	// env are the variables that the run adds to the environment.
	env []string
}

// checkout is the checkout of the repository of a packer of the cases: its tags, the tags that the
// packer created, and the runs of GoReleaser, which write the artifacts of artifacts into the dist
// directory of the module lint, or fail with fail.
type checkout struct {
	// fail is the error of a run, or nil.
	fail error

	// tags are the tags of the repository and their commits.
	tags map[string]string

	// created are the tags that the packer created, each as <name> <commit>.
	created []string

	// runs are the runs of the tools, in their order.
	runs []run
}

// packer returns a packer of r.
func (r *checkout) packer() release.Packer {
	return release.Packer{
		Tags: func(context.Context, string) (map[string]string, error) { return r.tags, nil },
		Head: func(context.Context, string) (string, error) { return headCommit, nil },
		Tag: func(_ context.Context, _, name, commit string) error {
			r.created = append(r.created, name+" "+commit)
			return nil
		},
		Run: func(_ context.Context, dir, tool string, args, env []string) error {
			r.runs = append(r.runs, run{dir: dir, tool: tool, args: args, env: env})
			if r.fail != nil {
				return r.fail
			}
			for name, content := range map[string]string{
				"artifacts.json":                                artifacts,
				"assertlint_linux_amd64_v1/assertlint":          "binary",
				"assertlint_0.2.0_linux_amd64.tar.gz":           "archive",
				"assertlint_0.2.0_linux_amd64.tar.gz.sbom.json": "sbom",
				"assertlint_0.2.0_amd64.deb":                    "deb",
				"demo-lint_0.2.0_source.tar.gz":                 "source",
				"checksums.txt":                                 "sums",
				"checksums.txt.sigstore.json":                   "bundle",
				"homebrew/Casks/assertlint.rb":                  "cask",
				"metadata.json":                                 "{}",
			} {
				file := filepath.Join(dir, filepath.FromSlash(lintDist), filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
					return fmt.Errorf("write %s: %w", name, err)
				}
				if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
					return fmt.Errorf("write %s: %w", name, err)
				}
			}
			return nil
		},
	}
}

func TestPacker(t *testing.T) {
	t.Parallel()

	t.Run("Pack", func(t *testing.T) {
		t.Parallel()

		t.Run("builds the assets and the casks of a module with a configuration of GoReleaser", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"lint/.goreleaser.yaml": files.Text(lintConfig)})
			dir := filepath.Join(t.TempDir(), "pack")
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, r.packer().Pack(t.Context(), root, pkgs, dir), "Pack")
			expect.Equal(t, r.created, []string{"lint/v0.2.0 " + headCommit}, "the tags")
			expect.Equal(t, r.runs, []run{{
				dir:  root,
				tool: "goreleaser",
				args: []string{"release", "--snapshot", "--clean", "--config", "lint/.goreleaser.yaml"},
				env:  []string{"ERGON_VERSION=0.2.0"},
			}}, "the runs")
			files.Equal(t, os.DirFS(dir), files.Tree{
				"assets/lint/v0.2.0/assertlint_0.2.0_linux_amd64.tar.gz":           files.Text("archive"),
				"assets/lint/v0.2.0/assertlint_0.2.0_linux_amd64.tar.gz.sbom.json": files.Text("sbom"),
				"assets/lint/v0.2.0/assertlint_0.2.0_amd64.deb":                    files.Text("deb"),
				"assets/lint/v0.2.0/demo-lint_0.2.0_source.tar.gz":                 files.Text("source"),
				"assets/lint/v0.2.0/checksums.txt":                                 files.Text("sums"),
				"assets/lint/v0.2.0/checksums.txt.sigstore.json":                   files.Text("bundle"),
				"casks/lint/v0.2.0/assertlint.rb":                                  files.Text("cask"),
			}, "the pack directory")
			files.Absent(t, filepath.Join(root, filepath.FromSlash(lintDist)), "the dist directory of GoReleaser")
		})

		t.Run("builds nothing for a module without a configuration of GoReleaser", func(t *testing.T) {
			t.Parallel()
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: ".", Version: parse(t, "1.0.0")}}
			assert.NoError(t, r.packer().Pack(t.Context(), t.TempDir(), pkgs, t.TempDir()), "Pack")
			expect.Empty(t, r.created, "the tags")
			expect.Empty(t, r.runs, "the runs")
		})

		t.Run("leaves a tag of the module at HEAD", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"lint/.goreleaser.yaml": files.Text(lintConfig)})
			r := &checkout{tags: map[string]string{"lint/v0.2.0": headCommit}}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, r.packer().Pack(t.Context(), root, pkgs, t.TempDir()), "Pack")
			expect.Empty(t, r.created, "the tags")
			expect.Length(t, r.runs, 1, "the runs")
		})

		t.Run("returns ErrTag for a tag of the module at another commit", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"lint/.goreleaser.yaml": files.Text(lintConfig)})
			r := &checkout{tags: map[string]string{"lint/v0.2.0": otherCommit}}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := r.packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.ErrorIs(t, err, release.ErrTag, "Pack")
			assert.Equal(t, err.Error(), "release: tag at another commit: lint/v0.2.0 is at "+otherCommit+
				", and HEAD is at "+headCommit, "the error")
			assert.Empty(t, r.runs, "the runs")
		})

		t.Run("returns the error of GoReleaser with the module", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{"lint/.goreleaser.yaml": files.Text(lintConfig)})
			r := &checkout{fail: errGoReleaser}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := r.packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.ErrorIs(t, err, errGoReleaser, "Pack")
			assert.Contains(t, err.Error(), "GoReleaser of "+pathA, "the error")
		})

		t.Run("returns an error for a configuration without a dist directory", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{".goreleaser.yaml": files.Text("version: 2\n")})
			pkgs := []workspace.Package{{Name: pathA, Dir: ".", Version: parse(t, "1.0.0")}}
			err := (&checkout{}).packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.HasError(t, err, "Pack")
			assert.Equal(t, err.Error(), "release: .goreleaser.yaml states no dist directory", "the error")
		})

		t.Run("returns the error of a configuration that is no YAML", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{".goreleaser.yaml": files.Text("dist: [\n")})
			pkgs := []workspace.Package{{Name: pathA, Dir: ".", Version: parse(t, "1.0.0")}}
			err := (&checkout{}).packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.HasError(t, err, "Pack")
			assert.Contains(t, err.Error(), "release: read .goreleaser.yaml", "the error")
		})

		t.Run("returns an error for a run that writes no artifacts.json", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{".goreleaser.yaml": files.Text("dist: dist/goreleaser/demo\n")})
			p := (&checkout{}).packer()
			p.Run = func(context.Context, string, string, []string, []string) error { return nil }
			pkgs := []workspace.Package{{Name: pathA, Dir: ".", Version: parse(t, "1.0.0")}}
			err := p.Pack(t.Context(), root, pkgs, t.TempDir())
			assert.ErrorIs(t, err, os.ErrNotExist, "Pack")
			assert.Contains(t, err.Error(), "read the artifacts of GoReleaser", "the error")
		})
	})
}
