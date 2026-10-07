// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/tool"
)

// fakeEnv is the variable of the environment that makes the test binary act as the program that the
// base name of its path states: a toolchain, which installs a copy of the test binary, or a tool,
// which writes its name and its arguments to the standard output.
const fakeEnv = "ERGON_TOOL_FAKE"

// section is the name of the section of the cases.
const section = "demo"

// exe ends the name of a program on Windows.
const exe = ".exe"

// The programs of the cases: the test binary, which acts as a tool, and the archives of the
// release binaries that contain it.
var (
	// self is the content of the test binary.
	self []byte

	// toolTarGz is a .tar.gz with the test binary as tool-1.0/tool.
	toolTarGz []byte

	// toolZip is a .zip with the test binary as tool.exe.
	toolZip []byte
)

// binary is a release binary of the cases, whose asset is asset on every platform, and none for an
// asset without an address.
type binary struct {
	option.Binary `yaml:",inline"`

	// asset is the asset of every platform.
	asset option.Asset
}

// Asset returns the asset of b, and an error that wraps option.ErrNoAsset for b without one.
func (b binary) Asset(option.Platform) (option.Asset, error) {
	if b.asset.URL == "" {
		return option.Asset{}, option.ErrNoAsset
	}
	return b.asset, nil
}

// tools are the tools of the section of the cases, one of each kind.
type tools struct {
	Tool    binary          `yaml:"tool"`
	UV      option.UV       `yaml:"uv"`
	Lint    option.Module   `yaml:"golangci-lint"`
	Ruff    option.PyPI     `yaml:"ruff"`
	Mypy    option.PyPI     `yaml:"mypy"                 run:"project"`
	Biome   option.NPM      `yaml:"biome"`
	TSC     option.NPM      `yaml:"typescript"                         program:"tsc"`
	Audit   option.Crate    `yaml:"cargo-audit"`
	PMD     option.Maven    `yaml:"pmd"`
	Ktlint  option.Maven    `yaml:"ktlint"                                                    classifier:"all"`
	PHPStan option.Composer `yaml:"phpstan"`
	Strict  option.Composer `yaml:"phpstan-strict-rules"`
	Fixer   option.Composer `yaml:"php-cs-fixer"                       program:"php-cs-fixer"`
	Odd     string          `yaml:"odd"`
	hidden  option.Composer
}

// options are the options of the section of the cases.
type options struct {
	Tools tools `yaml:"tools"`
}

// Validate returns nil.
func (*options) Validate() error {
	return nil
}

// toolless are options without a group of tools.
type toolless struct {
	Paths []string `yaml:"paths"`
}

// Validate returns nil.
func (*toolless) Validate() error {
	return nil
}

// listed are options whose group of tools is a list.
type listed struct {
	Tools []string `yaml:"tools"`
}

// Validate returns nil.
func (*listed) Validate() error {
	return nil
}

// valued are options that are no pointer to a struct.
type valued struct{}

// Validate returns nil.
func (valued) Validate() error {
	return nil
}

// redirect is a transport that sends every request to the server at target, whatever its host.
type redirect struct {
	// target is the address of the server.
	target *url.URL

	// base sends the requests.
	base http.RoundTripper
}

// RoundTrip sends req to the server of r, and returns the error of the transport.
func (r redirect) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme, clone.URL.Host = r.target.Scheme, r.target.Host
	resp, err := r.base.RoundTrip(clone)
	if err != nil {
		return nil, fmt.Errorf("redirect: %w", err)
	}
	return resp, nil
}

// TestMain acts as a fake program when fakeEnv is set. Otherwise it puts the fake toolchains first
// on the PATH of the process, from which exec resolves a program, and builds the archives of the
// cases.
func TestMain(m *testing.M) {
	if os.Getenv(fakeEnv) != "" {
		os.Exit(fake(os.Args))
	}
	name, err := os.Executable()
	if err == nil {
		self, err = os.ReadFile(name)
	}
	dir, err2 := os.MkdirTemp("", "ergon-tool-fakes-")
	if err != nil || err2 != nil {
		fmt.Fprintln(os.Stderr, "tool_test: the fakes do not install:", err, err2)
		os.Exit(2)
	}
	for _, fake := range []string{"go", "cargo", "composer", "npx", "java", "php"} {
		if runtime.GOOS == "windows" {
			fake += exe
		}
		if err := os.WriteFile(filepath.Join(dir, fake), self, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "tool_test: the fakes do not install:", err)
			os.Exit(2)
		}
	}
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	toolTarGz = archive(".tar.gz", "tool-1.0/tool", self)
	toolZip = archive(".zip", "tool.exe", self)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestTool(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("runs a release binary with the arguments and returns its exit status", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.Tool = release(served, "tool.tar.gz", "tool-1.0/tool", toolTarGz)
			r, out, _ := runner(t, served)
			code, err := r.Run(t.Context(), section, o, "tool", []string{"lint", "--exit=3"})
			assert.NoError(t, err, "Run")
			assert.Equal(t, code, 3, "the exit status")
			assert.Equal(t, out.String(), "tool lint --exit=3\n", "the output of the tool")
		})

		t.Run("runs the tool in the directory Dir", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.Tool = release(served, "tool.tar.gz", "tool-1.0/tool", toolTarGz)
			r, out, _ := runner(t, served)
			_, err := r.Run(t.Context(), section, o, "tool", []string{"--pwd"})
			assert.NoError(t, err, "Run")
			dir, err := filepath.EvalSymlinks(r.Dir)
			assert.NoError(t, err, "EvalSymlinks of the directory")
			assert.Equal(t, out.String(), "tool --pwd\n"+dir+"\n", "the output of the tool")
		})

		t.Run("runs a release binary of the cache without a download", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.Tool = release(served, "tool.tar.gz", "tool-1.0/tool", toolTarGz)
			r, out, _ := runner(t, served)
			_, err := r.Run(t.Context(), section, o, "tool", nil)
			assert.NoError(t, err, "the first Run")
			clear(served)
			_, err = r.Run(t.Context(), section, o, "tool", []string{"again"})
			assert.NoError(t, err, "the second Run")
			assert.Equal(t, out.String(), "tool\n"+"tool again\n", "the output of both runs")
		})

		unknown := []struct {
			name string
			give language.Options
			tool string
		}{
			{name: "returns ErrUnknown for a tool that the section does not name", give: demo(), tool: "missing"},
			{name: "returns ErrUnknown for options without tools", give: &toolless{}, tool: "tool"},
			{name: "returns ErrUnknown for tools that are no struct", give: &listed{}, tool: "tool"},
			{name: "returns ErrUnknown for options that are no pointer to a struct", give: valued{}, tool: "tool"},
		}
		for _, tt := range unknown {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _, _ := runner(t, map[string][]byte{})
				_, err := r.Run(t.Context(), section, tt.give, tt.tool, nil)
				assert.ErrorIs(t, err, tool.ErrUnknown, "Run")
			})
		}

		t.Run("returns ErrUnknown that lists the tools of the section", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, demo(), "missing", nil)
			assert.ErrorIs(t, err, tool.ErrUnknown, "Run")
			assert.HasSuffix(t, err.Error(), ": demo.missing, which is none of demo.tool, demo.uv, "+
				"demo.golangci-lint, demo.ruff, demo.mypy, demo.biome, demo.typescript, demo.cargo-audit, demo.pmd, "+
				"demo.ktlint, demo.phpstan, demo.phpstan-strict-rules, demo.php-cs-fixer, demo.odd", "the error")
		})

		t.Run("returns ErrUnknown that names a section without tools", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, &toolless{}, "tool", nil)
			assert.HasSuffix(t, err.Error(), ": demo.tool, because the section demo has no tools", "the error")
		})

		t.Run("returns ErrInstall for a field that is no kind of tool", func(t *testing.T) {
			t.Parallel()
			r, _, _ := runner(t, map[string][]byte{})
			_, err := r.Run(t.Context(), section, demo(), "odd", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})

		t.Run("returns the error of a program that does not start", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.Tool = release(served, "tool", "", []byte("no program\n"))
			r, _, _ := runner(t, served)
			_, err := r.Run(t.Context(), section, o, "tool", nil)
			assert.HasError(t, err, "Run")
			assert.ErrorIsNot(t, err, tool.ErrInstall, "Run")
		})
	})
}

// fake acts as the program that the base name of args[0] states, without .exe, and returns its
// exit status:
//
//   - go install <module>@<version> copies the test binary to GOBIN, named as go install names it,
//     and fails for a module whose path has broken
//   - cargo install --locked --root <root> <crate>@<version> copies it to <root>/bin
//   - composer require ... --working-dir=<dir> <package>:<version>... writes a proxy of each
//     package to <dir>/vendor/bin, and fails for a package whose name has broken
//   - any other program writes its name and its arguments to the standard output, and the working
//     directory for an argument --pwd, and exits N for an argument --exit=N
//
// A toolchain names a program with .exe when its directory is of the platform windows, as go and
// cargo name it for that platform.
func fake(args []string) int {
	name := strings.TrimSuffix(filepath.Base(args[0]), exe)
	switch name {
	case "go":
		pkg, _, _ := strings.Cut(args[2], "@")
		if strings.Contains(pkg, "broken") {
			fmt.Fprintln(os.Stderr, "go: the module does not install")
			return 1
		}
		program := path.Base(pkg)
		if len(program) > 1 && program[0] == 'v' && strings.Trim(program[1:], "0123456789") == "" {
			program = path.Base(path.Dir(pkg))
		}
		return copySelf(filepath.Join(os.Getenv("GOBIN"), program))
	case "cargo":
		crate, _, _ := strings.Cut(args[5], "@")
		return copySelf(filepath.Join(args[4], "bin", crate))
	case "composer":
		var dir string
		for _, arg := range args[1:] {
			if d, ok := strings.CutPrefix(arg, "--working-dir="); ok {
				dir = d
			}
			if pkg, _, ok := strings.Cut(arg, ":"); ok && strings.Contains(pkg, "/") {
				if strings.Contains(pkg, "broken") {
					fmt.Fprintln(os.Stderr, "composer: the package does not install")
					return 1
				}
				bin := filepath.Join(dir, "vendor", "bin")
				_ = os.MkdirAll(bin, 0o755)
				_ = os.WriteFile(filepath.Join(bin, path.Base(pkg)), []byte("<?php\n"), 0o644)
			}
		}
		return 0
	default:
		fmt.Fprintln(os.Stdout, strings.Join(slices.Concat([]string{name}, args[1:]), " "))
		for _, arg := range args[1:] {
			if arg == "--pwd" {
				dir, _ := os.Getwd()
				dir, _ = filepath.EvalSymlinks(dir)
				fmt.Fprintln(os.Stdout, dir)
			}
			if code, ok := strings.CutPrefix(arg, "--exit="); ok {
				n, _ := strconv.Atoi(code)
				return n
			}
		}
		return 0
	}
}

// copySelf copies the test binary to name, with .exe for a directory of the platform windows, and
// returns the exit status of the fake toolchain: 0, and 1 for a copy that fails.
func copySelf(name string) int {
	if strings.Contains(name, "windows-") {
		name += exe
	}
	program, err := os.Executable()
	var content []byte
	if err == nil {
		content, err = os.ReadFile(program)
	}
	if err == nil {
		err = os.MkdirAll(filepath.Dir(name), 0o755)
	}
	if err == nil {
		err = os.WriteFile(name, content, 0o755)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "fake:", err)
		return 1
	}
	return 0
}

// demo returns new options of the section of the cases, with a tool of each kind.
func demo() *options {
	return &options{Tools: tools{
		Lint:    "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0",
		Ruff:    "ruff@0.16.10",
		Mypy:    "mypy@2.4.0",
		Biome:   "@biomejs/biome@2.5.15",
		TSC:     "typescript@7.0.2",
		Audit:   "cargo-audit@0.22.2",
		PMD:     "net.sourceforge.pmd:pmd-java@7.28.0",
		Ktlint:  "com.pinterest.ktlint:ktlint-cli@1.8.0",
		PHPStan: "phpstan/phpstan@2.2.17",
		Strict:  "phpstan/phpstan-strict-rules@2.0.12",
		Fixer:   "php-cs-fixer/shim@3.95.27",
		hidden:  "hidden/package@1.0.0",
	}}
}

// release returns a release binary whose asset is at the address /releases/<file> of the server of
// the cases, with the program at program, and whose pin states the digest of content on the
// platform of the host. It adds content to served.
func release(served map[string][]byte, file, program string, content []byte) binary {
	served["/releases/"+file] = content
	return binary{
		SHA256:  map[option.Platform]string{host(): digest(content)},
		Version: "1.0",
		asset:   option.Asset{URL: "https://example.com/releases/" + file, Program: program},
	}
}

// runner returns a runner of a test, whose client gets the files of served from a server of the
// test, and its standard output and error. The runner installs into a cache of the test, runs in a
// directory of the test on the platform of the host, and runs every program as a fake.
func runner(t *testing.T, served map[string][]byte) (*tool.Runner, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		content, ok := served[req.URL.Path]
		if !ok {
			http.NotFound(w, req)
			return
		}
		_, _ = w.Write(content)
	}))
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	assert.NoError(t, err, "Parse of the address of the server")
	var out, errs bytes.Buffer
	env := slices.DeleteFunc(os.Environ(), func(kv string) bool { return strings.HasPrefix(kv, "GOCOVERDIR=") })
	r := &tool.Runner{
		Client:   &http.Client{Transport: redirect{target: target, base: server.Client().Transport}},
		Stdout:   &out,
		Stderr:   &errs,
		Env:      append(env, fakeEnv+"=1"),
		Cache:    t.TempDir(),
		Dir:      t.TempDir(),
		Platform: host(),
	}
	return r, &out, &errs
}

// host returns the platform of the host, as a platform of a release binary.
func host() option.Platform {
	return option.Platform(runtime.GOOS + "/" + runtime.GOARCH)
}

// digest returns the SHA-256 of content in lowercase hexadecimal.
func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// archive returns an archive of the format of suffix, .tar.gz or .zip, with the entry name of
// content, and an empty archive for content of a failing writer, which no case has.
func archive(suffix, name string, content []byte) []byte {
	var b bytes.Buffer
	if suffix == ".zip" {
		z := zip.NewWriter(&b)
		w, _ := z.Create(name)
		_, _ = w.Write(content)
		_ = z.Close()
		return b.Bytes()
	}
	gz, _ := gzip.NewWriterLevel(&b, gzip.BestSpeed)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))})
	_, _ = io.Copy(tw, bytes.NewReader(content))
	_ = tw.Close()
	_ = gz.Close()
	return b.Bytes()
}
