// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/internal/app"
	"go.stealthscale.io/cli/clitest"
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

// toolRuns are the command lines of the transcript of ergon tool, which run in a repository after
// init new with JavaScript.
var toolRuns = []clitest.Input{
	{Args: []string{"tool", "--help"}},
	{Args: []string{"tool"}},
	{Args: []string{"tool", "ci", "--help"}},
	{Args: []string{"tool", "ci"}},
	{Args: []string{"tool", "run", "--help"}},
	{Args: []string{"tool", "path", "--help"}},
	{Args: []string{"tool", "prune", "--help"}},
	{Args: []string{"tool", "ci", "prune", "--help"}},
	{Args: []string{"tool", "run"}},
	{Args: []string{"tool", "run", "biome"}},
	{Args: []string{"tool", "run", "js.prettier"}},
	{Args: []string{"tool", "run", "go.vet"}},
	{Args: []string{"tool", "path", "js.biome"}},
	{Args: []string{"tool", "path", "js.prettier"}},
	{Args: []string{"tool", "prune", "main"}},
	{Args: []string{"tool", "ci", "prune", "main"}},
}

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

	t.Run("Main", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "ergon", program(javascriptRepository(t), app.Register).Main,
				toolRuns...)
			golden.Match(t, "tool.txt", got, golden.ShouldUpdate())
		})

		t.Run("returns 2 for a tool that the section does not name", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runWith(t, app.Register, javascriptRepository(t), "tool", "run", "js.prettier")
			assert.Equal(t, status, statusUsage, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.HasPrefix(t, stderr, "ergon: tool: the section names no such tool: ", "the standard error")
		})

		t.Run("runs the tool of the section with the arguments after the tool", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runWith(t, app.Register, javascriptRepository(t),
				"tool", "run", "js.biome", "--", "check", "--write")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, biome+" check --write\n", "the output of the tool")
		})

		t.Run("runs an npm package with pnpm dlx for the package manager pnpm of the section js", func(t *testing.T) {
			t.Parallel()
			dir := javascriptRepository(t)
			config := read(t, dir, ".ergon.yaml")
			assert.Contains(t, config, "\n  package-manager: npm\n", "the package manager of .ergon.yaml")
			write(t, dir, ".ergon.yaml", strings.Replace(config, "\n  package-manager: npm\n",
				"\n  package-manager: pnpm\n", 1))
			status, stdout, stderr := runWith(t, app.Register, dir, "tool", "run", "js.biome", "--", "check")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "pnpm --package=@biomejs/biome@2.5.15 dlx biome check\n", "the output of the tool")
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
			res := clitest.Main(t.Context(), program(javascriptRepository(t), app.Register).Main, &clitest.Input{
				Args: []string{"tool", "run", "js.biome", "--", stdinFlag}, Vars: environ(), Stdin: "src/index.js\n",
			})
			assert.Equal(t, res.Status, statusOK, "the exit status: "+res.Stderr)
			assert.Equal(t, res.Stdout, biome+" "+stdinFlag+"\nsrc/index.js\n", "the output of the tool")
		})

		t.Run("returns 1 for a tool without a digest for the platform", func(t *testing.T) {
			t.Parallel()
			dir := javascriptRepository(t)
			host := native
			other := option.LinuxARM64
			if host == other {
				other = option.LinuxAMD64
			}
			pinCommitlint(t, dir, other, strings.Repeat("ab", 32))
			status, stdout, stderr := runWith(t, app.Register, dir, "tool", "run", "common.commitlint")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: tool: the tool does not install: commitlint 0.12.0 has no digest for "+
				string(host)+"\n", "the standard error")
		})

		t.Run("returns 1 for a cache directory that cannot be found", func(t *testing.T) {
			t.Parallel()
			p := program(javascriptRepository(t), app.Register)
			// The error comes with a directory of the test, which a run that went on would install into.
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, errCache }
			res := clitest.Main(t.Context(), p.Main, &clitest.Input{
				Args: []string{"tool", "run", "js.biome"}, Vars: environ(),
			})
			assert.Equal(t, res.Status, statusFailure, "the exit status")
			assert.Equal(t, res.Stderr, "ergon: cli: find the cache directory: "+errCache.Error()+"\n",
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

		t.Run("path writes the program of a release binary of the cache", func(t *testing.T) {
			t.Parallel()
			dir := javascriptRepository(t)
			digest := strings.Repeat("ab", 32)
			pinCommitlint(t, dir, native, digest)
			name := "commitlint"
			if native.OS() == "windows" {
				name += ".exe"
			}
			install := path.Join("ergon", "tools", "release", "commitlint", "0.12.0",
				strings.ReplaceAll(string(native), "/", "-"), digest, name)
			p := program(dir, app.Register)
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, nil }
			files.Write(t, cache, files.Tree{install: files.Executable("")})
			res := clitest.Main(t.Context(), p.Main, &clitest.Input{
				Args: []string{"tool", "path", "common.commitlint"}, Vars: environ(),
			})
			assert.Equal(t, res.Status, statusOK, "the exit status: "+res.Stderr)
			assert.Equal(t, res.Stdout, filepath.Join(cache, filepath.FromSlash(install))+"\n", "the standard output")
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
			p := program(javascriptRepository(t), app.Register)
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, nil }
			tools := filepath.Join(cache, "ergon", "tools")
			assert.NoError(t, os.MkdirAll(tools, 0o755), "MkdirAll of the tool directory")
			files.Write(t, tools, files.Tree{
				"module/example.com/gone/v1.0.0/gone": files.Text("stale"),
				"release/gone/1.0/gone":               files.Text("stale"),
			})
			res := clitest.Main(t.Context(), p.Main, &clitest.Input{Args: []string{"tool", "prune"}, Vars: environ()})
			assert.Equal(t, res.Status, statusOK, "the exit status: "+res.Stderr)
			assert.Equal(t, res.Stdout, "removed module\nremoved "+filepath.Join("release", "gone")+"\n",
				"the standard output")
			files.Absent(t, filepath.Join(tools, "release", "gone"), "the release binary that no section names")
		})

		t.Run("prune returns 1 for a .ergon.yaml that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := javascriptRepository(t)
			write(t, dir, ".ergon.yaml", "js: [\n")
			status, stdout, stderr := runWith(t, app.Register, dir, "tool", "prune")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.HasPrefix(t, stderr, "ergon: options: invalid .ergon.yaml: ", "the standard error")
		})

		t.Run("prune returns 1 for a cache directory that cannot be found", func(t *testing.T) {
			t.Parallel()
			p := program(javascriptRepository(t), app.Register)
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, errCache }
			res := clitest.Main(t.Context(), p.Main, &clitest.Input{Args: []string{"tool", "prune"}, Vars: environ()})
			assert.Equal(t, res.Status, statusFailure, "the exit status")
			assert.Equal(t, res.Stderr, "ergon: cli: find the cache directory: "+errCache.Error()+"\n",
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

// pinCommitlint gives commitlint in the .ergon.yaml of the repository dir the digest of platform
// alone, in place of the digests of the baseline.
func pinCommitlint(t *testing.T, dir string, platform option.Platform, digest string) {
	t.Helper()
	config := read(t, dir, ".ergon.yaml")
	start := strings.Index(config, "      sha256:\n")
	end := strings.Index(config, "      version: 0.12.0\n")
	assert.InRange(t, start, 0, float64(end-1), "the digests of commitlint before its version in .ergon.yaml")
	digests := "      sha256:\n        " + string(platform) + ": " + digest + "\n"
	write(t, dir, ".ergon.yaml", config[:start]+digests+config[end:])
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
