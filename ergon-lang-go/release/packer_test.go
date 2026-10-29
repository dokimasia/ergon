// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/go/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
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

// cgoConfig is the configuration of GoReleaser of the module lint of the cases with a build without
// cgo and a build with cgo.
const cgoConfig = lintConfig + "builds:\n  - id: assertlint\n    env: [CGO_ENABLED=0]\n" +
	"  - id: assertlint-darwin\n    env:\n      - CGO_ENABLED=1\n"

// draftConfig is the configuration of GoReleaser of the module lint of the cases with the manifest
// of a provider of the Terraform Registry as an extra file of its checksums, of the name template
// %s, and a release that remains a draft.
const draftConfig = lintConfig + "checksum:\n  extra_files:\n    - glob: lint/terraform-registry-manifest.json\n" +
	"      name_template: %q\nrelease:\n  disable: true\n  draft: true\n"

// manifest is the manifest of the Terraform Registry of the cases.
const manifest = `{"version":1,"metadata":{"protocol_versions":["6.0"]}}`

// zigProgram is the program of zig that the packer of the cases installs.
const zigProgram = "/cache/zig/zig"

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

// errZig is the error of an install of zig that fails.
var errZig = errors.New("zig does not install")

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
// packer created, the tools that it installed, and the runs of GoReleaser, which write the
// artifacts of artifacts into the dist directory of the module lint, or fail with fail.
type checkout struct {
	// fail is the error of a run, or nil.
	fail error

	// zig is the error of the install of a tool, or nil.
	zig error

	// tags are the tags of the repository and their commits.
	tags map[string]string

	// created are the tags that the packer created, each as <name> <commit>.
	created []string

	// installed are the tools that the packer installed, in their order.
	installed []string

	// runs are the runs of the tools, in their order.
	runs []run
}

// packer returns a packer of r, which takes its snapshots with the git of vcs.
func (r *checkout) packer() release.Packer {
	return release.Packer{
		Snapshot: vcs.Snapshot,
		Tags:     func(context.Context, string) (map[string]string, error) { return r.tags, nil },
		Head:     func(context.Context, string) (string, error) { return headCommit, nil },
		Tag: func(_ context.Context, _, name, commit string) error {
			r.created = append(r.created, name+" "+commit)
			return nil
		},
		Program: func(_ context.Context, _, tool string) (string, error) {
			r.installed = append(r.installed, tool)
			return zigProgram, r.zig
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
			root := lintRepository(t)
			dir := filepath.Join(t.TempDir(), "pack")
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, r.packer().Pack(t.Context(), root, pkgs, dir), "Pack")
			expect.Equal(t, r.created, []string{"lint/v0.2.0 " + headCommit}, "the tags")
			assert.Length(t, r.runs, 1, "the runs")
			expect.Equal(t, r.runs[0].dir, root, "the directory of the run")
			expect.Equal(t, r.runs[0].tool, "goreleaser", "the tool of the run")
			args := []string{"release", "--snapshot", "--clean", "--config", "lint/.goreleaser.yaml"}
			expect.Equal(t, r.runs[0].args, args, "the arguments of the run")
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

		t.Run("attaches the extra files of the checksums under their names and writes the draft", func(t *testing.T) {
			t.Parallel()
			root := lintRepository(t)
			config := fmt.Sprintf(draftConfig, "demo_{{ .Version }}_manifest.json")
			files.Write(t, root, files.Tree{
				"lint/.goreleaser.yaml":                 files.Text(config),
				"lint/terraform-registry-manifest.json": files.Text(manifest),
			})
			dir := filepath.Join(t.TempDir(), "pack")
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, (&checkout{}).packer().Pack(t.Context(), root, pkgs, dir), "Pack")
			files.Contains(t, os.DirFS(dir), files.Tree{
				"assets/lint/v0.2.0/demo_0.2.0_manifest.json":    files.Text(manifest),
				"assets/lint/v0.2.0/checksums.txt.sigstore.json": files.Text("bundle"),
				"drafts/lint/v0.2.0":                             files.Text(""),
			}, "the pack directory")
		})

		t.Run("names an extra file without a name template after its base name", func(t *testing.T) {
			t.Parallel()
			root := lintRepository(t)
			files.Write(t, root, files.Tree{
				"lint/.goreleaser.yaml":                 files.Text(fmt.Sprintf(draftConfig, "")),
				"lint/terraform-registry-manifest.json": files.Text(manifest),
			})
			dir := filepath.Join(t.TempDir(), "pack")
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, (&checkout{}).packer().Pack(t.Context(), root, pkgs, dir), "Pack")
			files.HasContent(t, filepath.Join(dir, "assets", "lint", "v0.2.0", "terraform-registry-manifest.json"),
				manifest, "the extra file")
		})

		templates := []struct {
			name string
			give string
		}{
			{
				name: "returns an error for a name template of a field other than the version",
				give: "{{ .ProjectName }}.json",
			},
			{name: "returns an error for a name template that does not parse", give: "{{ .Version"},
		}
		for _, tt := range templates {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				root := lintRepository(t)
				files.Write(t, root, files.Tree{"lint/.goreleaser.yaml": files.Text(fmt.Sprintf(draftConfig, tt.give))})
				r := &checkout{}
				pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
				err := r.packer().Pack(t.Context(), root, pkgs, t.TempDir())
				assert.HasError(t, err, "Pack")
				assert.Contains(t, err.Error(), "the name of the extra file lint/terraform-registry-manifest.json",
					"the error")
				assert.Empty(t, r.runs, "the runs")
			})
		}

		t.Run("returns the error of an extra file that the repository lacks", func(t *testing.T) {
			t.Parallel()
			root := lintRepository(t)
			config := fmt.Sprintf(draftConfig, "demo_{{ .Version }}_manifest.json")
			files.Write(t, root, files.Tree{"lint/.goreleaser.yaml": files.Text(config)})
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := (&checkout{}).packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.ErrorIs(t, err, fs.ErrNotExist, "Pack")
			assert.Contains(t, err.Error(), "the assets of "+pathA, "the error")
		})

		t.Run("returns the error of a draft that it cannot write", func(t *testing.T) {
			t.Parallel()
			root := lintRepository(t)
			files.Write(t, root, files.Tree{
				"lint/.goreleaser.yaml":                 files.Text(fmt.Sprintf(draftConfig, "")),
				"lint/terraform-registry-manifest.json": files.Text(manifest),
			})
			dir := files.Workspace(t, files.Tree{"drafts": files.Text("a file\n")})
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := (&checkout{}).packer().Pack(t.Context(), root, pkgs, dir)
			assert.HasError(t, err, "Pack")
			assert.Contains(t, err.Error(), "write the draft of lint/v0.2.0", "the error")
		})

		t.Run("runs GoReleaser outside the workspace with the version and the modules of the release",
			func(t *testing.T) {
				t.Parallel()
				r := &checkout{}
				pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
				assert.NoError(t, r.packer().Pack(t.Context(), lintRepository(t), pkgs, t.TempDir()), "Pack")
				assert.Length(t, r.runs, 1, "the runs")
				env := r.runs[0].env
				expect.Equal(t, env[0], "ERGON_VERSION=0.2.0", "the first variable")
				expect.That(t, env).
					Contains("GOWORK=off", "the variables").
					Contains("GOFLAGS=-modcacherw", "the variables").
					Contains("ERGON_MODULES="+pathA, "the variables")
				goproxy, _ := variable(env, "GOPROXY")
				expect.HasPrefix(t, goproxy, "file://", "the first proxy")
				nosumdb, _ := variable(env, "GONOSUMDB")
				expect.Contains(t, nosumdb, pathA, "the modules without the checksum database")
			})

		t.Run("resolves each module of the release at its version in the runs of GoReleaser", func(t *testing.T) {
			t.Parallel()
			root := repository(t, nil)
			_, err := versioner.Apply(t.Context(), root, releaseOfA())
			assert.NoError(t, err, "Apply")
			files.Write(t, root, files.Tree{"b/.goreleaser.yaml": files.Text(lintConfig)})
			vcstest.Commit(t, root, "release")
			r := &checkout{}
			p := r.packer()
			fake := p.Run
			var resolved string
			p.Run = func(ctx context.Context, dir, tool string, args, env []string) error {
				list := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Version}}", pathA)
				list.Dir, list.Env = filepath.Join(dir, "b"), append(os.Environ(), env...)
				out, err := list.CombinedOutput()
				if err != nil {
					return fmt.Errorf("go list: %w\n%s", err, out)
				}
				resolved = strings.TrimSpace(string(out))
				build := exec.CommandContext(ctx, "go", "build", "./...")
				build.Dir, build.Env = filepath.Join(dir, "b"), append(os.Environ(), env...)
				if out, err := build.CombinedOutput(); err != nil {
					return fmt.Errorf("go build: %w\n%s", err, out)
				}
				return fake(ctx, dir, tool, args, env)
			}
			pkgs := []workspace.Package{
				{Name: pathA, Dir: "a", Version: parse(t, "0.2.0")},
				{Name: pathB, Dir: "b", Version: parse(t, "0.1.1")},
			}
			assert.NoError(t, p.Pack(t.Context(), root, pkgs, t.TempDir()), "Pack")
			expect.Equal(t, resolved, "v0.2.0", "the version of a in the run")
			assert.Length(t, r.runs, 1, "the runs")
			expect.Contains(t, r.runs[0].env, "ERGON_MODULES="+pathA+","+pathB, "the variables")
		})

		t.Run("runs a build with cgo in the environment of zig", func(t *testing.T) {
			t.Parallel()
			root := lintRepository(t)
			files.Write(t, root, files.Tree{"lint/.goreleaser.yaml": files.Text(cgoConfig)})
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, r.packer().Pack(t.Context(), root, pkgs, t.TempDir()), "Pack")
			expect.Equal(t, r.installed, []string{"zig"}, "the installed tools")
			assert.Length(t, r.runs, 1, "the runs")
			env := r.runs[0].env
			expect.Contains(t, env, "ERGON_ZIG="+zigProgram, "the variables")
			stubs, _ := variable(env, "ERGON_DARWIN_STUBS")
			global, _ := variable(env, "ZIG_GLOBAL_CACHE_DIR")
			local, _ := variable(env, "ZIG_LOCAL_CACHE_DIR")
			expect.NotEmpty(t, stubs, "the directory of the stubs of macOS")
			expect.NotEmpty(t, global, "the global cache of zig")
			expect.Equal(t, local, global, "the local cache of zig")
		})

		t.Run("installs no zig for builds without cgo", func(t *testing.T) {
			t.Parallel()
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, r.packer().Pack(t.Context(), lintRepository(t), pkgs, t.TempDir()), "Pack")
			expect.Empty(t, r.installed, "the installed tools")
			assert.Length(t, r.runs, 1, "the runs")
			_, found := variable(r.runs[0].env, "ERGON_ZIG")
			expect.False(t, found, "the run has ERGON_ZIG")
		})

		t.Run("returns the error of the install of zig with the module", func(t *testing.T) {
			t.Parallel()
			root := lintRepository(t)
			files.Write(t, root, files.Tree{"lint/.goreleaser.yaml": files.Text(cgoConfig)})
			r := &checkout{zig: errZig}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := r.packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.ErrorIs(t, err, errZig, "Pack")
			assert.Contains(t, err.Error(), "install zig for the builds with cgo of "+pathA, "the error")
			assert.Empty(t, r.runs, "the runs")
		})

		t.Run("removes the module proxy of the release after the runs", func(t *testing.T) {
			t.Parallel()
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, r.packer().Pack(t.Context(), lintRepository(t), pkgs, t.TempDir()), "Pack")
			assert.Length(t, r.runs, 1, "the runs")
			cache, ok := variable(r.runs[0].env, "GOMODCACHE")
			assert.True(t, ok, "the run has a module cache")
			files.Absent(t, filepath.Dir(cache), "the temporary directory of the module proxy")
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
			r := &checkout{tags: map[string]string{"lint/v0.2.0": headCommit}}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			assert.NoError(t, r.packer().Pack(t.Context(), lintRepository(t), pkgs, t.TempDir()), "Pack")
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

		t.Run("returns an error for a module of the release that the repository does not have", func(t *testing.T) {
			t.Parallel()
			r := &checkout{}
			pkgs := []workspace.Package{{Name: "example.com/absent", Dir: "lint", Version: parse(t, "0.2.0")}}
			err := r.packer().Pack(t.Context(), lintRepository(t), pkgs, t.TempDir())
			assert.HasError(t, err, "Pack")
			assert.Contains(t, err.Error(), "the module example.com/absent, which the repository does not have",
				"the error")
			assert.Empty(t, r.runs, "the runs")
		})

		t.Run("returns the error of the snapshot", func(t *testing.T) {
			t.Parallel()
			r := &checkout{}
			p := r.packer()
			p.Snapshot = func(context.Context, string) (string, error) { return "", errSnapshot }
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := p.Pack(t.Context(), lintRepository(t), pkgs, t.TempDir())
			assert.ErrorIs(t, err, errSnapshot, "Pack")
			assert.Empty(t, r.runs, "the runs")
		})

		t.Run("returns the error of GoReleaser with the module", func(t *testing.T) {
			t.Parallel()
			r := &checkout{fail: errGoReleaser}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := r.packer().Pack(t.Context(), lintRepository(t), pkgs, t.TempDir())
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
			root := vcstest.Repository(t, files.Tree{
				"go.mod":           files.Text(modOf(pathA)),
				"a.go":             files.Text(sourceA),
				".goreleaser.yaml": files.Text("dist: dist/goreleaser/demo\n"),
			})
			vcstest.Commit(t, root, "add the module")
			p := (&checkout{}).packer()
			p.Run = func(context.Context, string, string, []string, []string) error { return nil }
			pkgs := []workspace.Package{{Name: pathA, Dir: ".", Version: parse(t, "1.0.0")}}
			err := p.Pack(t.Context(), root, pkgs, t.TempDir())
			assert.ErrorIs(t, err, os.ErrNotExist, "Pack")
			assert.Contains(t, err.Error(), "read the artifacts of GoReleaser", "the error")
		})
	})
}

// TestPackerEnv runs the cases of Pack that change the environment of the process, one at a time.
func TestPackerEnv(t *testing.T) {
	t.Run("Pack", func(t *testing.T) {
		t.Run("returns the error of go env GOROOT for a build with cgo", func(t *testing.T) {
			root := lintRepository(t)
			files.Write(t, root, files.Tree{"lint/.goreleaser.yaml": files.Text(cgoConfig)})
			t.Setenv("GOROOT", filepath.Join(root, "absent"))
			r := &checkout{}
			pkgs := []workspace.Package{{Name: pathA, Dir: "lint", Version: parse(t, "0.2.0")}}
			err := r.packer().Pack(t.Context(), root, pkgs, t.TempDir())
			assert.HasError(t, err, "Pack")
			assert.Contains(t, err.Error(), "go env GOROOT for the builds with cgo of "+pathA, "the error")
			assert.Empty(t, r.runs, "the runs")
		})
	})
}

// lintRepository returns a working tree of git with the module a in the directory lint of a
// workspace, with the configuration of GoReleaser of the cases, all files committed.
func lintRepository(tb testing.TB) string {
	tb.Helper()
	root := vcstest.Repository(tb, files.Tree{
		"go.work":               files.Text("go 1.24\n\nuse ./lint\n"),
		"lint/go.mod":           files.Text(modOf(pathA)),
		"lint/a.go":             files.Text(sourceA),
		"lint/.goreleaser.yaml": files.Text(lintConfig),
	})
	vcstest.Commit(tb, root, "add the module")
	return root
}

// variable returns the value of the last variable name of env, and reports whether env has one.
func variable(env []string, name string) (string, bool) {
	value, found := "", false
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, name+"="); ok {
			value, found = v, true
		}
	}
	return value, found
}
