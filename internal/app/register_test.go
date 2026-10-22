// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package app_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/lang/bash"
	"go.dokimi.dev/ergon/lang/csharp"
	golang "go.dokimi.dev/ergon/lang/go"
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/kotlin"
	"go.dokimi.dev/ergon/lang/php"
	"go.dokimi.dev/ergon/lang/python"
	"go.dokimi.dev/ergon/lang/rust"
	"go.dokimi.dev/ergon/lang/terraform"
	"go.dokimi.dev/ergon/lang/typescript"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// recordEnv is the environment variable that contains the path of the record file. When the tools
// of the toolchain of Go run the test binary as ergon, the binary appends each argument and then
// ERGON_VERSION to that file, each on a line of its own. It then writes an empty artifacts.json
// into the dist directory of config, and writes zigProgram to its standard output for ergon tool
// path.
const recordEnv = "ERGON_APP_TEST_RECORD"

// The module that the cases pack, and its go.mod.
const (
	modulePath = "go.dokimi.dev/demo"
	goMod      = "module " + modulePath + "\n\ngo 1.24\n"
)

// The configurations of GoReleaser of the cases, without cgo and with a build with cgo, and the
// artifacts.json of their dist directory.
const (
	config    = "dist: dist/goreleaser/demo\n"
	cgoConfig = config + "builds:\n  - env: [CGO_ENABLED=1]\n"
	artifacts = "dist/goreleaser/demo/artifacts.json"
)

// zigProgram is the program of zig that the test binary writes as ergon tool path.
const zigProgram = "/cache/zig/zig"

// registry is a transport that records the address of each request and answers it with 404, as
// PyPI and the npm registry answer for a release that they do not have.
type registry struct {
	urls []string
}

// RoundTrip records the address of r and answers it with 404.
func (f *registry) RoundTrip(r *http.Request) (*http.Response, error) {
	f.urls = append(f.urls, r.URL.String())
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Header:     http.Header{},
		Body:       http.NoBody,
		Request:    r,
	}, nil
}

// TestMain acts as ergon run by the tools of the toolchain of Go when recordEnv is set. Otherwise it
// isolates git from the configuration of the user and runs the tests.
func TestMain(m *testing.M) {
	if record := os.Getenv(recordEnv); record != "" {
		lines := slices.Concat(os.Args[1:], []string{os.Getenv("ERGON_VERSION")})
		f, err := os.OpenFile(record, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			_, err = f.WriteString(strings.Join(lines, "\n") + "\n")
			err = errors.Join(err, f.Close())
		}
		if err == nil {
			err = os.MkdirAll(filepath.Dir(artifacts), 0o755)
		}
		if err == nil {
			err = os.WriteFile(artifacts, []byte("[]"), 0o600)
		}
		if err == nil && len(os.Args) > 2 && os.Args[2] == "path" {
			_, err = fmt.Fprintln(os.Stdout, zigProgram)
		}
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	vcstest.Isolate()
	os.Exit(m.Run())
}

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("adds every language with the toolchain that builds it", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			assert.Equal(t, slices.Collect(c.Languages()), []language.Declaration{
				{Name: csharp.Language, Toolchain: csharp.Toolchain},
				{Name: java.Language, Toolchain: java.Toolchain},
				{Name: kotlin.Language, Toolchain: java.Toolchain},
				{Name: php.Language, Toolchain: php.Toolchain},
				{Name: javascript.Language, Toolchain: javascript.Toolchain},
				{Name: typescript.Language, Toolchain: javascript.Toolchain},
				{Name: golang.Language, Toolchain: golang.Toolchain},
				{Name: python.Language, Toolchain: python.Toolchain},
				{Name: rust.Language, Toolchain: rust.Toolchain},
				{Name: terraform.Language, Toolchain: terraform.Toolchain},
				{Name: bash.Language, Toolchain: bash.Toolchain},
			}, "the languages of the catalog")
		})

		t.Run("adds producers whose sections go-cli decodes with every key", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			var names []string
			for d := range c.Languages() {
				p, _ := language.Role[language.Producer](&c, d.Name)
				if conf, ok := p.(language.Configurable); ok {
					baselinetest.Keys(t, string(d.Name), conf)
					names = append(names, string(d.Name))
				}
			}
			for tc := range c.Toolchains() {
				p, _ := language.ToolchainRole[language.Producer](&c, tc.Name)
				if conf, ok := p.(language.Configurable); ok {
					baselinetest.Keys(t, string(tc.Name), conf)
					names = append(names, string(tc.Name))
				}
			}
			assert.Equal(t, names, []string{
				"csharp", "java", "kotlin", "php", "javascript", "typescript", "go", "python", "rust", "terraform",
				"bash", "jvm", "js",
			}, "the sections of the catalog")
		})

		publishers := []struct {
			name          string
			giveToolchain workspace.Toolchain
			wantURL       string
		}{
			{
				name:          "adds the publisher of Python, which reads PyPI through the transport",
				giveToolchain: python.Toolchain,
				wantURL:       "https://pypi.org/pypi/demo/0.2.0/json",
			},
			{
				name:          "adds the publisher of JavaScript, which reads the npm registry through the transport",
				giveToolchain: javascript.Toolchain,
				wantURL:       "https://registry.npmjs.org/demo/0.2.0",
			},
			{
				name:          "adds the publisher of Rust, which reads the index of crates.io through the transport",
				giveToolchain: rust.Toolchain,
				wantURL:       "https://index.crates.io/de/mo/demo",
			},
		}
		for _, tt := range publishers {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var c language.Catalog
				f := &registry{}
				assert.NoError(t, app.Register(&c, f), "Register of every language module")
				publisher, ok := language.ToolchainRole[language.Publisher](&c, tt.giveToolchain)
				assert.True(t, ok, "the publisher of the toolchain")
				pkg := &workspace.Package{Name: "demo", Version: version.Version{Minor: 2}}
				got, err := publisher.Published(t.Context(), pkg)
				assert.NoError(t, err, "Published")
				assert.False(t, got, "Published")
				assert.Equal(t, f.urls, []string{tt.wantURL}, "the requests")
			})
		}

		t.Run("returns ErrRegistered for a catalog that has a language of ergon", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, csharp.Register(&c), "Register of C#")
			assert.ErrorIs(
				t,
				app.Register(&c, &registry{}),
				language.ErrRegistered,
				"Register of every language module",
			)
		})
	})
}

// TestRegisterProcess sets the environment of the process and runs the case whose packer starts the
// test binary as ergon. The case does not call t.Parallel. A coverage run writes one coverage file
// for two processes that exit in the same nanosecond.
func TestRegisterProcess(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		t.Run("packs a module of Go with GoReleaser through ergon tool run at the tag of the module",
			func(t *testing.T) {
				root := vcstest.Repository(t, files.Tree{
					"go.mod":           files.Text(goMod),
					".goreleaser.yaml": files.Text(config),
				})
				head := vcstest.Commit(t, root, "first")
				record := filepath.Join(t.TempDir(), "record")
				t.Setenv(recordEnv, record)
				t.Chdir(root)
				var c language.Catalog
				assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
				packer, ok := language.ToolchainRole[language.Packer](&c, golang.Toolchain)
				assert.True(t, ok, "the packer of Go")
				v, err := version.Parse("1.0.0")
				assert.NoError(t, err, "Parse")
				pkgs := []workspace.Package{{Name: modulePath, Dir: ".", Version: v}}
				assert.NoError(t, packer.Pack(t.Context(), root, pkgs, t.TempDir()), "Pack")
				files.HasContent(t, record, "tool\nrun\ngo.goreleaser\n--\nrelease\n--snapshot\n--clean\n--config\n"+
					".goreleaser.yaml\n1.0.0\n", "the arguments and the version of the run")
				assert.Equal(t, strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "v1.0.0^{commit}")), head,
					"the commit of the tag of the module")
			})

		t.Run("returns the error of ergon tool run for a GoReleaser that fails", func(t *testing.T) {
			root := vcstest.Repository(t, files.Tree{
				"go.mod":           files.Text(goMod),
				".goreleaser.yaml": files.Text(config),
			})
			vcstest.Commit(t, root, "first")
			t.Setenv(recordEnv, t.TempDir())
			t.Chdir(root)
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			packer, ok := language.ToolchainRole[language.Packer](&c, golang.Toolchain)
			assert.True(t, ok, "the packer of Go")
			v, err := version.Parse("1.0.0")
			assert.NoError(t, err, "Parse")
			pkgs := []workspace.Package{{Name: modulePath, Dir: ".", Version: v}}
			err = packer.Pack(t.Context(), root, pkgs, t.TempDir())
			assert.HasError(t, err, "Pack")
			assert.Contains(t, err.Error(), "ergon tool run go.goreleaser", "the error")
		})

		t.Run("installs zig through ergon tool path for a build with cgo", func(t *testing.T) {
			root := vcstest.Repository(t, files.Tree{
				"go.mod":           files.Text(goMod),
				".goreleaser.yaml": files.Text(cgoConfig),
			})
			vcstest.Commit(t, root, "first")
			record := filepath.Join(t.TempDir(), "record")
			t.Setenv(recordEnv, record)
			t.Chdir(root)
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			packer, ok := language.ToolchainRole[language.Packer](&c, golang.Toolchain)
			assert.True(t, ok, "the packer of Go")
			v, err := version.Parse("1.0.0")
			assert.NoError(t, err, "Parse")
			pkgs := []workspace.Package{{Name: modulePath, Dir: ".", Version: v}}
			assert.NoError(t, packer.Pack(t.Context(), root, pkgs, t.TempDir()), "Pack")
			assert.HasPrefix(t, files.Read(t, record), "tool\npath\ngo.zig\n\ntool\nrun\ngo.goreleaser\n",
				"the run of ergon tool path before the run of GoReleaser")
		})

		t.Run("returns the error of ergon tool path for a build with cgo", func(t *testing.T) {
			root := vcstest.Repository(t, files.Tree{
				"go.mod":           files.Text(goMod),
				".goreleaser.yaml": files.Text(cgoConfig),
			})
			vcstest.Commit(t, root, "first")
			t.Setenv(recordEnv, t.TempDir())
			t.Chdir(root)
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			packer, ok := language.ToolchainRole[language.Packer](&c, golang.Toolchain)
			assert.True(t, ok, "the packer of Go")
			v, err := version.Parse("1.0.0")
			assert.NoError(t, err, "Parse")
			pkgs := []workspace.Package{{Name: modulePath, Dir: ".", Version: v}}
			err = packer.Pack(t.Context(), root, pkgs, t.TempDir())
			assert.HasError(t, err, "Pack")
			assert.Contains(t, err.Error(), "ergon tool path go.zig", "the error")
		})

		t.Run("refreshes uv.lock of a release of Python with uv through ergon tool run", func(t *testing.T) {
			root := files.Workspace(t, files.Tree{
				"pyproject.toml": files.Text("[project]\nname = \"demo\"\nversion = \"0.1.0\"\n"),
				"uv.lock":        files.Text("version = 1\n"),
			})
			record := filepath.Join(t.TempDir(), "record")
			t.Setenv(recordEnv, record)
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			versioner, ok := language.ToolchainRole[language.Versioner](&c, python.Toolchain)
			assert.True(t, ok, "the versioner of Python")
			_, err := versioner.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2},
				Package: workspace.Package{Name: "demo", Dir: ".", Version: version.Version{Minor: 1}},
			}})
			assert.NoError(t, err, "Apply")
			files.HasContent(t, record, "tool\nrun\npython.uv\n--\nlock\n\n", "the arguments of the run")
		})

		t.Run("refreshes package-lock.json of a release of JavaScript with npm of the PATH", func(t *testing.T) {
			self, err := os.Executable()
			assert.NoError(t, err, "Executable")
			npm := "npm"
			if runtime.GOOS == "windows" {
				npm += ".exe"
			}
			t.Setenv("PATH", files.Workspace(t, files.Tree{npm: files.Executable(files.Read(t, self))}))
			record := filepath.Join(t.TempDir(), "record")
			t.Setenv(recordEnv, record)
			root := files.Workspace(t, files.Tree{
				"package.json":      files.Text(`{"name": "demo", "version": "0.1.0"}`),
				"package-lock.json": files.Text("{}\n"),
			})
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			versioner, ok := language.ToolchainRole[language.Versioner](&c, javascript.Toolchain)
			assert.True(t, ok, "the versioner of JavaScript")
			_, err = versioner.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2},
				Package: workspace.Package{Name: "demo", Dir: ".", Version: version.Version{Minor: 1}},
			}})
			assert.NoError(t, err, "Apply")
			files.HasContent(t, record, "install\n--package-lock-only\n\n", "the arguments of the run of npm")
		})

		t.Run("returns the error of a program that the PATH does not have", func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			root := files.Workspace(t, files.Tree{
				"package.json":      files.Text(`{"name": "demo", "version": "0.1.0"}`),
				"package-lock.json": files.Text("{}\n"),
			})
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			versioner, ok := language.ToolchainRole[language.Versioner](&c, javascript.Toolchain)
			assert.True(t, ok, "the versioner of JavaScript")
			_, err := versioner.Apply(t.Context(), root, []language.Edit{{
				Version: version.Version{Minor: 2},
				Package: workspace.Package{Name: "demo", Dir: ".", Version: version.Version{Minor: 1}},
			}})
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "app: npm: ", "the error")
		})

		t.Run("uploads a tarball of JavaScript with npm through ergon tool run", func(t *testing.T) {
			pack := files.Workspace(t, files.Tree{"js/demo/demo-0.2.0.tgz": files.Text("")})
			record := filepath.Join(t.TempDir(), "record")
			t.Setenv(recordEnv, record)
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			publisher, ok := language.ToolchainRole[language.Publisher](&c, javascript.Toolchain)
			assert.True(t, ok, "the publisher of JavaScript")
			pkgs := []workspace.Package{{Name: "demo", Dir: ".", Version: version.Version{Minor: 2}}}
			assert.NoError(t, publisher.Publish(t.Context(), t.TempDir(), pkgs, pack), "Publish")
			files.HasContent(t, record, "tool\nrun\njs.npm\n--\npublish\n"+
				filepath.Join(pack, "js", "demo", "demo-0.2.0.tgz")+"\n\n", "the arguments of the run")
		})

		t.Run("uploads a crate of Rust with cargo of the PATH", func(t *testing.T) {
			self, err := os.Executable()
			assert.NoError(t, err, "Executable")
			cargo := "cargo"
			if runtime.GOOS == "windows" {
				cargo += ".exe"
			}
			t.Setenv("PATH", files.Workspace(t, files.Tree{cargo: files.Executable(files.Read(t, self))}))
			// The case runs outside a job of GitHub Actions with the permission id-token.
			t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
			record := filepath.Join(t.TempDir(), "record")
			t.Setenv(recordEnv, record)
			pack := files.Workspace(t, files.Tree{"rust/demo/demo-0.2.0.crate": files.Text("")})
			var c language.Catalog
			assert.NoError(t, app.Register(&c, &registry{}), "Register of every language module")
			publisher, ok := language.ToolchainRole[language.Publisher](&c, rust.Toolchain)
			assert.True(t, ok, "the publisher of Rust")
			pkgs := []workspace.Package{{Name: "demo", Dir: ".", Version: version.Version{Minor: 2}}}
			assert.NoError(t, publisher.Publish(t.Context(), t.TempDir(), pkgs, pack), "Publish")
			files.HasContent(
				t,
				record,
				"publish\n--no-verify\n--package\ndemo\n\n",
				"the arguments of the run of cargo",
			)
		})
	})
}
