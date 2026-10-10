// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/internal/cli"
)

// fakeEnv is the variable of the environment that makes the test binary act as the program that the
// base name of its path states, as [fake] states.
const fakeEnv = "ERGON_CLI_FAKE"

// The variables of the environment that the fake program reads: the exit status, which exitFlag
// overrides, and the release that an upgrade of ergon started, which the fake writes.
const (
	fakeExitEnv = "ERGON_CLI_FAKE_EXIT"
	upgradeEnv  = "ERGON_UPGRADE"
)

// The flags of the fake program.
const (
	// exitFlag makes the fake exit with the status after the equals sign.
	exitFlag = "--exit="

	// pwdFlag makes the fake write its working directory.
	pwdFlag = "--pwd"

	// stdinFlag makes the fake copy its standard input to its standard output.
	stdinFlag = "--stdin"
)

// biome is the line that the fake npx writes for the Biome of the section js at the baseline,
// before the arguments of the tool.
const biome = "npx --yes --package=@biomejs/biome@2.5.15 -- biome"

// toolHelp is the help of ergon tool, pinned because a person reads it.
const toolHelp = `ergon tool runs the tools that the sections of .ergon.yaml name, at the
versions that they name. Every target of the Makefile and every hook of the
baseline of ergon init runs its tools this way.

Usage:
  ergon tool [flags]
  ergon tool [command]

Available Commands:
  ci          Run the jobs that keep the tools of ergon in GitHub Actions
  prune       Remove the tools that no section of .ergon.yaml names
  run         Install a tool of a section and run it

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command

Use "ergon tool [command] --help" for more information about a command.
`

// pruneHelp is the help of ergon tool prune, pinned because a person reads it.
const pruneHelp = `ergon tool prune removes each file from ergon/tools in the cache directory of
the user that installs no tool of the sections of .ergon.yaml at the version
that they name: the earlier versions of the tools, the programs of a Go module
for another version of the go command, the programs of golangci-lint for other
plugins, and the files of an install that did not finish. It also removes the
tools of other repositories that share the directory. Each job of the managed
workflows that keeps the tools of ergon in the cache of GitHub Actions runs it
before the cache saves the directory.

Usage:
  ergon tool prune [flags]

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// toolCIHelp is the help of ergon tool ci, pinned because a person reads it.
const toolCIHelp = `ergon tool ci runs the steps of the jobs of the managed workflows that keep the
tool directory of ergon in the cache of GitHub Actions.

Usage:
  ergon tool ci [flags]
  ergon tool ci [command]

Available Commands:
  prune       Delete the tool caches that newer caches replaced

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command

Use "ergon tool ci [command] --help" for more information about a command.
`

// ciPruneHelp is the help of ergon tool ci prune, pinned because a person reads it.
const ciPruneHelp = `ergon tool ci prune deletes each cache of the tools of ergon in GitHub Actions
that a newer cache of the same job replaced, through the API of GitHub with the
token of GITHUB_TOKEN, which needs the permission actions: write. A job saves a
cache under a new key for each change of the files of its key, and restores
only the newest cache of its own when its key misses. The caches of a branch and
of a pull request are separate.

Usage:
  ergon tool ci prune [flags]

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// The caches of the prune cases, as the API of GitHub lists them: two caches of the job check-go of
// main, of which the second is newer, and a cache of the job commits.
const (
	olderCache = `{"id":11,"ref":"refs/heads/main","key":"ergon-tools-Linux-X64-check-go-aaaa",` +
		`"created_at":"2026-10-09T13:51:00Z"}`
	newerCache = `{"id":12,"ref":"refs/heads/main","key":"ergon-tools-Linux-X64-check-go-bbbb",` +
		`"created_at":"2026-10-10T09:56:00Z"}`
	commitsCache = `{"id":13,"ref":"refs/heads/main","key":"ergon-tools-Linux-X64-commits-aaaa",` +
		`"created_at":"2026-10-08T20:20:00Z"}`
)

// cachesRoute is the request of the first page of the caches of the tools of ergon of hubRepo.
const cachesRoute = "GET /repos/" + hubRepo + "/actions/caches?key=ergon-tools-&page=1&per_page=100"

// runHelp is the help of ergon tool run, pinned because a person reads it.
const runHelp = `ergon tool run installs the tool that a section of .ergon.yaml names, unless
the cache has it, and runs it in the working directory with the arguments that
follow the tool. It installs the tool under ergon/tools in the cache directory
of the user, and checks a release binary against the digest that the section
states for the platform. It reads the options of the repository of the working
directory or of the nearest of its parents with .ergon/init.lock.

The exit status is the exit status of the tool. It is 1 when the tool does not
install, and 2 when the section or the tool does not exist.

Usage:
  ergon tool run <section>.<tool> [-- <argument>...] [flags]

Examples:
  ergon tool run go.golangci-lint -- run ./...

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// errCache is the error of a cache directory that cannot be found.
var errCache = errors.New("cache: $HOME is not defined")

// cacheHub is a fake API of GitHub for the prune cases. It lists the caches of its field caches on
// one page for cachesRoute. It responds with 403 to the deletion of the cache of its field refused,
// and deletes every other cache. It records the method and the path with the query of each
// request. It is safe for concurrent use, as the goroutines of a server use it.
type cacheHub struct {
	// caches are the caches of the list, as a JSON array.
	caches string

	// requests are the method and the path with the query of each request, in their order.
	requests []string

	// refused is the cache whose deletion fails, or 0 for none.
	refused int64

	// mu guards requests.
	mu sync.Mutex
}

// ServeHTTP records the request and responds to it.
func (h *cacheHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	call := r.Method + " " + r.URL.RequestURI()
	h.mu.Lock()
	h.requests = append(h.requests, call)
	h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case call == cachesRoute:
		_, _ = io.WriteString(w, `{"total_count":3,"actions_caches":`+h.caches+`}`)
	case call == "DELETE /repos/"+hubRepo+"/actions/caches/"+strconv.FormatInt(h.refused, 10):
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"message":"Resource not accessible by integration"}`)
	case r.Method == http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
	}
}

// start serves h for the test t, and returns the variables of GitHub Actions that point ergon tool
// ci prune at it, with a token and the repository hubRepo.
func (h *cacheHub) start(t *testing.T) map[string]string {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return map[string]string{"GITHUB_TOKEN": "token", "GITHUB_REPOSITORY": hubRepo, "GITHUB_API_URL": server.URL}
}

// calls returns the requests that h received, in their order.
func (h *cacheHub) calls() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.requests)
}

func TestTool(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		help := []struct {
			name   string
			args   []string
			stdout string
		}{
			{name: "writes the help of tool", args: []string{"tool", "--help"}, stdout: toolHelp},
			{name: "writes the help of tool run", args: []string{"tool", "run", "--help"}, stdout: runHelp},
			{name: "writes the help of tool prune", args: []string{"tool", "prune", "--help"}, stdout: pruneHelp},
			{name: "writes the help of tool ci", args: []string{"tool", "ci", "--help"}, stdout: toolCIHelp},
			{
				name: "writes the help of tool ci prune", args: []string{"tool", "ci", "prune", "--help"},
				stdout: ciPruneHelp,
			},
		}
		for _, tt := range help {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				status, stdout, stderr := run(t, t.TempDir(), tt.args...)
				assert.Equal(t, status, statusOK, "the exit status")
				assert.Equal(t, stdout, tt.stdout, "the standard output")
				assert.Empty(t, stderr, "the standard error")
			})
		}

		usage := []struct {
			name   string
			args   []string
			stderr string
		}{
			{
				name: "returns 2 for tool without a subcommand", args: []string{"tool"},
				stderr: "ergon: cli: tool needs a subcommand: ci, prune, run\nRun 'ergon tool --help' for usage.\n",
			},
			{
				name: "returns 2 for tool prune with an argument", args: []string{"tool", "prune", "main"},
				stderr: "ergon: unknown command \"main\" for \"ergon tool prune\"\n" +
					"Run 'ergon tool prune --help' for usage.\n",
			},
			{
				name: "returns 2 for tool ci without a subcommand", args: []string{"tool", "ci"},
				stderr: "ergon: cli: ci needs a subcommand: prune\nRun 'ergon tool ci --help' for usage.\n",
			},
			{
				name: "returns 2 for tool ci prune with an argument", args: []string{"tool", "ci", "prune", "main"},
				stderr: "ergon: unknown command \"main\" for \"ergon tool ci prune\"\n" +
					"Run 'ergon tool ci prune --help' for usage.\n",
			},
			{
				name: "returns 2 for tool run without a tool", args: []string{"tool", "run"},
				stderr: "ergon: requires at least 1 arg(s), only received 0\nRun 'ergon tool run --help' for usage.\n",
			},
			{
				name:   "returns 2 for a tool without a section",
				args:   []string{"tool", "run", "biome"},
				stderr: "ergon: cli: \"biome\", which is not <section>.<tool>\nRun 'ergon tool run --help' for usage.\n",
			},
			{
				name: "returns 2 for a tool that the section does not name",
				args: []string{"tool", "run", "js.prettier"},
				stderr: "ergon: tool: the section names no such tool: js.prettier, which is none of js.biome\n" +
					"Run 'ergon tool run --help' for usage.\n",
			},
			{
				name: "returns 2 for a section of no producer of the repository",
				args: []string{"tool", "run", "go.vet"},
				stderr: "ergon: baseline: no producer of the repository has the section: \"go\"\n" +
					"Run 'ergon tool run --help' for usage.\n",
			},
		}
		for _, tt := range usage {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				status, stdout, stderr := runWith(t, app.Register, javascriptRepository(t), tt.args...)
				assert.Equal(t, status, statusUsage, "the exit status")
				assert.Empty(t, stdout, "the standard output")
				assert.Equal(t, stderr, tt.stderr, "the standard error")
			})
		}

		t.Run("runs the tool of the section with the arguments after the tool", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runWith(t, app.Register, javascriptRepository(t),
				"tool", "run", "js.biome", "--", "check", "--write")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, biome+" check --write\n", "the output of the tool")
		})

		t.Run("returns the exit status of the tool without an error", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runWith(t, app.Register, javascriptRepository(t),
				"tool", "run", "js.biome", "--", exitFlag+"3")
			assert.Equal(t, status, 3, "the exit status")
			assert.Equal(t, stdout, biome+" "+exitFlag+"3\n", "the output of the tool")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("runs the tool in the working directory", func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(javascriptRepository(t), "packages", "web")
			assert.NoError(t, os.MkdirAll(dir, 0o755), "MkdirAll of the package")
			resolved, err := filepath.EvalSymlinks(dir)
			assert.NoError(t, err, "EvalSymlinks of the package")
			status, stdout, stderr := runWith(t, app.Register, resolved, "tool", "run", "js.biome", "--", pwdFlag)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, biome+" "+pwdFlag+"\n"+resolved+"\n", "the output of the tool")
		})

		t.Run("passes the standard input of the process to the tool", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			p := process(javascriptRepository(t), &stdout, &stderr, "tool", "run", "js.biome", "--", stdinFlag)
			p.Stdin = strings.NewReader("src/index.js\n")
			assert.Equal(t, cli.Run(t.Context(), p, app.Register, version), statusOK, "the exit status")
			assert.Equal(t, stdout.String(), biome+" "+stdinFlag+"\nsrc/index.js\n", "the output of the tool")
		})

		t.Run("returns 1 for a tool without a digest for the platform", func(t *testing.T) {
			t.Parallel()
			dir := javascriptRepository(t)
			host := option.Platform(runtime.GOOS + "/" + runtime.GOARCH)
			other := option.LinuxARM64
			if host == other {
				other = option.LinuxAMD64
			}
			config := read(t, dir, ".ergon.yaml")
			start := strings.Index(config, "      sha256:\n")
			end := strings.Index(config, "      version: 0.12.0\n")
			assert.InRange(t, start, 0, float64(end-1), "the digests of commitlint before its version in .ergon.yaml")
			digests := "      sha256:\n        " + string(other) + ": " + strings.Repeat("ab", 32) + "\n"
			write(t, dir, ".ergon.yaml", config[:start]+digests+config[end:])
			status, stdout, stderr := runWith(t, app.Register, dir, "tool", "run", "common.commitlint")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: tool: the tool does not install: commitlint 0.12.0 has no digest for "+
				string(host)+"\n", "the standard error")
		})

		t.Run("returns 1 for a cache directory that cannot be found", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			p := process(javascriptRepository(t), &stdout, &stderr, "tool", "run", "js.biome")
			// The error comes with a directory of the test, which a run that went on would install into.
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, errCache }
			assert.Equal(t, cli.Run(t.Context(), p, app.Register, version), statusFailure, "the exit status")
			assert.Equal(t, stderr.String(), "ergon: cli: find the cache directory: "+errCache.Error()+"\n",
				"the standard error")
		})

		t.Run("returns 1 for a .ergon.yaml that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := javascriptRepository(t)
			write(t, dir, ".ergon.yaml", "js: [\n")
			status, _, stderr := runWith(t, app.Register, dir, "tool", "run", "js.biome")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: options: invalid .ergon.yaml: ", "the standard error")
		})

		t.Run("ci prune deletes the tool caches that newer caches replaced", func(t *testing.T) {
			t.Parallel()
			h := &cacheHub{caches: "[" + olderCache + "," + newerCache + "," + commitsCache + "]"}
			status, stdout, stderr := runRelease(t, t.TempDir(), h.start(t), "tool", "ci", "prune")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "deleted ergon-tools-Linux-X64-check-go-aaaa of refs/heads/main\n",
				"the standard output")
			assert.Equal(t, h.calls(), []string{cachesRoute, "DELETE /repos/" + hubRepo + "/actions/caches/11"},
				"the requests")
		})

		t.Run("ci prune returns 1 for a deletion that GitHub refuses", func(t *testing.T) {
			t.Parallel()
			h := &cacheHub{caches: "[" + olderCache + "," + newerCache + "]", refused: 11}
			status, stdout, stderr := runRelease(t, t.TempDir(), h.start(t), "tool", "ci", "prune")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Contains(t, stderr, "403 Forbidden: Resource not accessible by integration", "the standard error")
		})

		t.Run("prune removes the tools that no section of the repository names", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			p := process(javascriptRepository(t), &stdout, &stderr, "tool", "prune")
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, nil }
			tools := filepath.Join(cache, "ergon", "tools")
			assert.NoError(t, os.MkdirAll(tools, 0o755), "MkdirAll of the tool directory")
			files.Write(t, tools, files.Tree{
				"module/example.com/gone/v1.0.0/gone": files.Text("stale"),
				"release/gone/1.0/gone":               files.Text("stale"),
			})
			status := cli.Run(t.Context(), p, app.Register, version)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr.String())
			assert.Equal(t, stdout.String(), "removed module\nremoved "+filepath.Join("release", "gone")+"\n",
				"the standard output")
			files.Absent(t, filepath.Join(tools, "release", "gone"), "the release binary that no section names")
		})

		t.Run("prune returns 1 for a cache directory that cannot be found", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			p := process(javascriptRepository(t), &stdout, &stderr, "tool", "prune")
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, errCache }
			assert.Equal(t, cli.Run(t.Context(), p, app.Register, version), statusFailure, "the exit status")
			assert.Equal(t, stderr.String(), "ergon: cli: find the cache directory: "+errCache.Error()+"\n",
				"the standard error")
		})

		t.Run("ci prune returns 1 without GITHUB_REPOSITORY", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runRelease(t, t.TempDir(), map[string]string{"GITHUB_TOKEN": "token"},
				"tool", "ci", "prune")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: cli: set GITHUB_REPOSITORY to the repository on GitHub, as owner/name\n",
				"the standard error")
		})
	})
}

// javascriptRepository returns a new repository after init new with JavaScript and the answers of
// the cases.
func javascriptRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	args := slices.Concat([]string{"init", "new", "--language", "javascript"}, required)
	status, _, stderr := runWith(t, app.Register, dir, args...)
	assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
	return dir
}

// fake acts as the program of the base name of args[0], without the suffix .exe. It writes the
// name and the other arguments to stdout, on one line, and the line upgradeEnv=<release> when the
// environment sets upgradeEnv. Then it writes its working directory for pwdFlag, copies stdin to
// stdout for stdinFlag, and exits with the status of exitFlag, or of fakeExitEnv, or 0.
func fake(args []string, stdin io.Reader, stdout io.Writer) int {
	name := strings.TrimSuffix(filepath.Base(args[0]), ".exe")
	fmt.Fprintln(stdout, strings.Join(append([]string{name}, args[1:]...), " "))
	if release := os.Getenv(upgradeEnv); release != "" {
		fmt.Fprintln(stdout, upgradeEnv+"="+release)
	}
	status, _ := strconv.Atoi(os.Getenv(fakeExitEnv))
	for _, arg := range args[1:] {
		switch {
		case arg == pwdFlag:
			dir, _ := os.Getwd()
			fmt.Fprintln(stdout, dir)
		case arg == stdinFlag:
			_, _ = io.Copy(stdout, stdin)
		case strings.HasPrefix(arg, exitFlag):
			status, _ = strconv.Atoi(strings.TrimPrefix(arg, exitFlag))
		}
	}
	return status
}
