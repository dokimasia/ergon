// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/cli"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The releases of ergon in the upgrade cases: the newest release of the major version of the
// running ergon, and a release of a later major version. The running ergon is version.
const (
	newest = "1.3.0"
	later  = "2.0.0"
)

// published is the time at which the module proxy of the upgrade cases published each release.
const published = "2026-09-01T12:00:00Z"

// The paths of the fake upstream of the upgrade cases: the module of ergon in its module proxy, and
// the releases of ergon on its server of GitHub.
const (
	proxyPath    = "/goproxy"
	modulePrefix = proxyPath + "/go.dokimi.dev/ergon/"
	downloads    = "/dokimasia/ergon/releases/download/"
	checksums    = "checksums.txt"
)

// The variables of the environment of an upgrade.
const (
	proxyEnv  = "GOPROXY"
	serverEnv = "GITHUB_SERVER_URL"
)

// changesetOfUpgrade is the changeset that ergon init ci upgrade of the running ergon writes, after
// the summary Upgrade ergon to 1.2.3. and the random digits ab and cd of the cases.
const changesetOfUpgrade = ".changeset/upgrade-ergon-to-1-2-abcd.md"

// overridden is the line of .ergon.yaml that sets actions/checkout to another release than the
// baseline v7.0.1.
const overridden = "release: v7.0.0"

// The parts of the body of the pull request of ergon init ci upgrade 1.2.3, pinned because a person
// reads them: the files of a sync that wrote the LICENSE, the override of overridden, and the
// conflict of a .gitignore that was edited by hand.
const (
	proposedFiles = "This pull request was opened by `ergon init ci upgrade`. Merging it moves the managed " +
		"files to the baseline of ergon 1.2.3.\n\n# Files\n\n- wrote `LICENSE`"
	proposedOverrides = "\n\n# Overrides\n\n.ergon.yaml sets these pins to another version than the " +
		"baseline. Remove a key to take the baseline.\n\n" +
		"- `github.ci.actions.checkout`: v7.0.0, where the baseline has v7.0.1"
	proposedConflicts = "\n\n# Conflicts\n\nergon init sync left these managed files, which were edited by " +
		"hand. Move each edit into the local file of its path under .ergon/local, and run ergon init sync " +
		"--force.\n\n- `.gitignore`"
)

// dev is the version of a build of ergon without a release.
var dev = cli.Version{Release: "dev", Full: "dev"}

// native is the platform of the test, which the process of each case states.
var native = option.Platform(runtime.GOOS + "/" + runtime.GOARCH)

// upgradeHelp is the help of ergon init upgrade, pinned because a person reads it.
const upgradeHelp = `ergon init upgrade moves the repository to the baseline of the newest release
of ergon: the highest release of go.dokimi.dev/ergon that the first module
proxy of GOPROXY that is a URL lists, of the major version of this ergon, or of
any major version with --major. When it is newer than this ergon, the command
installs it into the cache of ergon tool run, checks its archive against the
checksums.txt of the release, and runs its ergon init upgrade, whose exit
status it returns.

The newest release runs ergon init sync, and then prints each pin of
.ergon.yaml whose version differs from the baseline. A build of ergon without a
release runs the upgrade itself, and refuses a lock that a release wrote. A
release refuses a lock that such a build wrote.

Usage:
  ergon init upgrade [flags]

Flags:
      --force   overwrite the managed files that were edited by hand
      --major   take a release of a later major version

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// initCIHelp is the help of ergon init ci, pinned because a person reads it.
const initCIHelp = `ergon init ci runs the job of the workflow baseline.yml in GitHub Actions.

Usage:
  ergon init ci [flags]
  ergon init ci [command]

Available Commands:
  upgrade     Upgrade ergon and open the pull request of the change

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command

Use "ergon init ci [command] --help" for more information about a command.
`

// ciUpgradeHelp is the help of ergon init ci upgrade, pinned because a person reads it.
const ciUpgradeHelp = `ergon init ci upgrade upgrades the repository as ergon init upgrade does. When
the sync changes a file, it writes a changeset without packages, commits the
changes on the branch ergon-baseline/<base> through the API of GitHub, which
signs the commits, and opens or updates the pull request into the base branch
of .changeset/config.json. The exit status is 1 after it proposed the files of a
sync that left a managed file that was edited by hand.

Usage:
  ergon init ci upgrade [flags]

Flags:
      --force   overwrite the managed files that were edited by hand
      --major   take a release of a later major version

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// The archives of ergon of the cases, each built once: one with the program ergon, and one with the
// program ergon.exe of Windows.
var (
	ergonArchive   = sync.OnceValues(func() ([]byte, error) { return archiveOf("ergon") })
	windowsArchive = sync.OnceValues(func() ([]byte, error) { return archiveOf("ergon.exe") })
)

// gamma is the language of the case whose baseline has a version with a source tag of no registry.
const gamma workspace.Language = "gamma"

// sourced are the options of gamma, with a version whose source tag names no registry.
type sourced struct {
	// Make has a source tag of no registry.
	Make option.Version `yaml:"make" doc:"The version of make." source:"make"`
}

// Validate returns nil.
func (*sourced) Validate() error {
	return nil
}

// unsourced is the producer of gamma: the files of fixed, and the options sourced.
type unsourced struct {
	fixed
}

// Options returns the options of gamma at the baseline.
func (unsourced) Options() language.Options {
	return &sourced{Make: "4.4.1"}
}

// upstream is a fake of the module proxy and of the server of GitHub for the upgrade cases. Below
// the module of ergon in the proxy, it returns the last of versions for @latest, versions for
// @v/list, and the document of each version. Below the releases of ergon, it returns checksums for
// checksums.txt and archive for any other file. It responds with 404 to every other request, and to
// checksums.txt for empty checksums, and with 502 to the requests of the proxy for fail.
type upstream struct {
	// versions are the versions of ergon that the proxy lists, the highest last.
	versions []string

	// checksums is the content of checksums.txt.
	checksums string

	// archive is the content of each archive.
	archive []byte

	// fail makes every request of the proxy fail.
	fail bool
}

// ServeHTTP responds to r as u states.
func (u *upstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	module, ofProxy := strings.CutPrefix(r.URL.Path, modulePrefix)
	file, ofRelease := strings.CutPrefix(r.URL.Path, downloads)
	switch {
	case ofProxy && u.fail:
		w.WriteHeader(http.StatusBadGateway)
	case ofProxy && module == "@latest":
		_, _ = io.WriteString(w, `{"Version":"`+u.versions[len(u.versions)-1]+`","Time":"`+published+`"}`)
	case ofProxy && module == "@v/list":
		_, _ = io.WriteString(w, strings.Join(u.versions, "\n")+"\n")
	case ofProxy && strings.HasSuffix(module, ".info"):
		v := strings.TrimSuffix(strings.TrimPrefix(module, "@v/"), ".info")
		_, _ = io.WriteString(w, `{"Version":"`+v+`","Time":"`+published+`"}`)
	case ofRelease && path.Base(file) == checksums && u.checksums != "":
		_, _ = io.WriteString(w, u.checksums)
	case ofRelease && path.Base(file) != checksums:
		_, _ = w.Write(u.archive)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// transport is the transport of the process of an upgrade case. It counts the bodies of the
// responses that ergon has not closed.
type transport struct {
	// base sends the requests.
	base http.RoundTripper

	// open is the number of bodies that ergon has not closed.
	open atomic.Int64
}

// RoundTrip sends r through base, and counts the body of its response until ergon closes it. It
// returns the error of base.
func (c *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := c.base.RoundTrip(r)
	if err != nil {
		return nil, fmt.Errorf("transport: %w", err)
	}
	c.open.Add(1)
	resp.Body = &body{ReadCloser: resp.Body, open: &c.open}
	return resp, nil
}

// body is the body of a response, which counts down the open bodies of its transport when it closes
// for the first time.
type body struct {
	io.ReadCloser

	// open is the number of open bodies of the transport.
	open *atomic.Int64

	// once counts the body down once.
	once sync.Once
}

// Close counts the body down once, and closes it. It returns the error of the close.
func (b *body) Close() error {
	b.once.Do(func() { b.open.Add(-1) })
	if err := b.ReadCloser.Close(); err != nil {
		return fmt.Errorf("body: %w", err)
	}
	return nil
}

func TestUpgrade(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		help := []struct {
			name   string
			args   []string
			stdout string
		}{
			{name: "writes the help of init upgrade", args: []string{"init", "upgrade", "--help"}, stdout: upgradeHelp},
			{name: "writes the help of init ci", args: []string{"init", "ci", "--help"}, stdout: initCIHelp},
			{
				name:   "writes the help of init ci upgrade",
				args:   []string{"init", "ci", "upgrade", "--help"},
				stdout: ciUpgradeHelp,
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

		t.Run("starts the newest release of the major version with the arguments of the process", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v1.2.3", "v" + newest, "v" + later}}, native)
			status, stdout, stderr := upgrade(t.Context(), t, version, initialized(t), env, "init", "upgrade")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "ergon init upgrade\n"+upgradeEnv+"="+newest+"\n", "the output of the release")
			assert.Equal(
				t,
				stderr,
				"ergon: ergon "+later+" is a release of a later major version, which --major takes\n",
				"the standard error",
			)
		})

		t.Run("leaves the sync to the release that it starts", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v" + newest}}, native)
			dir := initialized(t)
			assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
			status, _, stderr := upgrade(t.Context(), t, version, dir, env, "init", "upgrade")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			files.Absent(t, filepath.Join(dir, licensePath), "the LICENSE, which the fake release does not write")
		})

		t.Run("starts the newest release of a later major version with --major", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v" + newest, "v" + later}}, native)
			status, stdout, stderr := upgrade(
				t.Context(),
				t,
				version,
				initialized(t),
				env,
				"init",
				"upgrade",
				"--major",
			)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(
				t,
				stdout,
				"ergon init upgrade --major\n"+upgradeEnv+"="+later+"\n",
				"the output of the release",
			)
		})

		t.Run("returns the exit status of the release that it starts", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v" + newest}}, native)
			env[fakeExitEnv] = "3"
			status, _, _ := upgrade(t.Context(), t, version, initialized(t), env, "init", "upgrade")
			assert.Equal(t, status, 3, "the exit status")
		})

		t.Run("starts the newest release of the major version for init ci upgrade", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v" + newest, "v" + later}}, native)
			status, stdout, stderr := upgrade(t.Context(), t, version, initialized(t), env, "init", "ci", "upgrade")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "ergon init ci upgrade\n"+upgradeEnv+"="+newest+"\n", "the output of the release")
		})

		t.Run("returns 1 for a module proxy that fails under init ci upgrade", func(t *testing.T) {
			t.Parallel()
			env := serveUpstream(t, &upstream{versions: []string{"v" + newest}, fail: true})
			status, _, stderr := upgrade(t.Context(), t, version, initialized(t), env, "init", "ci", "upgrade")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Contains(t, stderr, "502 Bad Gateway", "the standard error")
		})

		t.Run("starts the program ergon.exe of the archive of Windows", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v" + newest}}, option.WindowsAMD64)
			var stdout, stderr bytes.Buffer
			p := upgradeProcess(t, initialized(t), env, &stdout, &stderr, "init", "upgrade")
			p.Platform = option.WindowsAMD64
			assert.Equal(t, cli.Run(t.Context(), p, register, version), statusOK, "the exit status: "+stderr.String())
			assert.Equal(
				t,
				stdout.String(),
				"ergon init upgrade\n"+upgradeEnv+"="+newest+"\n",
				"the output of the release",
			)
		})

		t.Run("returns 1 for a baseline whose overrides do not resolve", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			status, _, stderr := runWith(t, registerGamma, dir,
				slices.Concat([]string{"init", "new", "--language", string(gamma)}, required)...)
			assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
			var stdout, errs bytes.Buffer
			p := upgradeProcess(t, dir, map[string]string{upgradeEnv: newest}, &stdout, &errs, "init", "upgrade")
			assert.Equal(t, cli.Run(t.Context(), p, registerGamma, version), statusFailure, "the exit status")
			assert.HasPrefix(t, errs.String(), "ergon: pin: invalid source tag: gamma.make", "the standard error")
		})

		t.Run("reads the newest release from the first module proxy of GOPROXY that is a URL", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v" + newest}}, native)
			env[proxyEnv] = "off,direct|" + env[proxyEnv] + "/,https://proxy.example.com"
			status, stdout, stderr := upgrade(t.Context(), t, version, initialized(t), env, "init", "upgrade")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "ergon init upgrade\n"+upgradeEnv+"="+newest+"\n", "the output of the release")
		})

		proxies := []struct {
			name    string
			goproxy string
			want    string
		}{
			{
				name:    "reads the newest release from the first module proxy of GOPROXY that is a URL of https",
				goproxy: "off|https://proxy.example.com/,https://other.example.com",
				want:    "GET https://proxy.example.com/go.dokimi.dev/ergon/@latest: ",
			},
			{
				name:    "reads the newest release from proxy.golang.org for a GOPROXY without a URL",
				goproxy: "direct",
				want:    "GET https://proxy.golang.org/go.dokimi.dev/ergon/@latest: ",
			},
		}
		for _, tt := range proxies {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				env := map[string]string{proxyEnv: tt.goproxy}
				status, _, stderr := upgrade(ctx, t, version, initialized(t), env, "init", "upgrade")
				assert.Equal(t, status, statusFailure, "the exit status")
				assert.Contains(t, stderr, tt.want, "the standard error")
			})
		}

		syncs := []struct {
			name     string
			version  cli.Version
			versions []string
			env      map[string]string
			args     []string
			want     string
		}{
			{
				name:     "upgrades the repository itself at the newest release",
				version:  version,
				versions: []string{"v1.2.3"},
				want:     "wrote " + licensePath + "\n",
			},
			{
				name:     "upgrades the repository itself in the release that an upgrade started",
				version:  version,
				versions: []string{"v1.2.3", "v" + newest},
				env:      map[string]string{upgradeEnv: newest},
				want:     "wrote " + licensePath + "\n",
			},
		}
		for _, tt := range syncs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				env := released(t, &upstream{versions: tt.versions}, native)
				maps.Copy(env, tt.env)
				dir := initialized(t)
				assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
				args := slices.Concat([]string{"init", "upgrade"}, tt.args)
				status, stdout, stderr := upgrade(t.Context(), t, tt.version, dir, env, args...)
				assert.Equal(t, status, statusOK, "the exit status: "+stderr)
				expect.Equal(t, stdout, tt.want, "the files of the sync")
				expect.Empty(t, stderr, "the standard error")
			})
		}

		t.Run("returns 1 for a lock of a release in a build without a release", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v1.2.3", "v" + newest, "v" + later}}, native)
			dir := initialized(t)
			status, stdout, stderr := upgrade(t.Context(), t, dev, dir, env, "init", "upgrade", "--major")
			assert.Equal(t, status, statusFailure, "the exit status")
			expect.Empty(t, stdout, "the standard output")
			expect.HasPrefix(t, stderr,
				"ergon: baseline: a build of ergon without a release writes no lock of a release", "the standard error")
		})

		t.Run("upgrades a repository with the lock of a build without a release in such a build", func(t *testing.T) {
			t.Parallel()
			dir := developed(t)
			assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
			status, stdout, stderr := upgrade(t.Context(), t, dev, dir, nil, "init", "upgrade")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+licensePath+"\n", "the files of the sync")
		})

		t.Run("writes each pin of .ergon.yaml whose version differs from the baseline", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t)
			write(t, dir, ".ergon.yaml", strings.Replace(read(t, dir, ".ergon.yaml"), "release: v7.0.1", overridden, 1))
			status, stdout, stderr := upgrade(t.Context(), t, version, dir, map[string]string{upgradeEnv: newest},
				"init", "upgrade")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.HasSuffix(t, stdout, "override github.ci.actions.checkout v7.0.0, the baseline is v7.0.1\n",
				"the output of the upgrade")
		})

		t.Run("writes no override after a sync that fails", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t)
			write(t, dir, ".ergon.yaml", strings.Replace(read(t, dir, ".ergon.yaml"), "release: v7.0.1", overridden, 1))
			assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
			assert.NoError(t, os.Mkdir(filepath.Join(dir, licensePath), 0o755), "Mkdir of the LICENSE")
			status, stdout, stderr := upgrade(t.Context(), t, version, dir, map[string]string{upgradeEnv: newest},
				"init", "upgrade")
			assert.Equal(t, status, statusFailure, "the exit status")
			expect.Empty(t, stdout, "the standard output")
			expect.HasPrefix(t, stderr, "ergon: baseline: read "+licensePath+": ", "the standard error")
		})

		t.Run("returns 1 for a lock that a build without a release wrote", func(t *testing.T) {
			t.Parallel()
			dir := developed(t)
			status, _, stderr := upgrade(
				t.Context(),
				t,
				version,
				dir,
				map[string]string{upgradeEnv: newest},
				"init",
				"upgrade",
			)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: cli: a build of ergon without a release wrote .ergon/init.lock, so run "+
				"ergon init sync with that build\n", "the standard error")
		})

		refusals := []struct {
			name   string
			change func(t *testing.T, dir string)
			want   string
		}{
			{
				name: "returns 1 for a lock that does not read",
				change: func(t *testing.T, dir string) {
					t.Helper()
					lock := filepath.Join(dir, filepath.FromSlash(lockPath))
					assert.NoError(t, os.Remove(lock), "Remove of the lock")
					assert.NoError(t, os.Mkdir(lock, 0o755), "Mkdir of the lock")
				},
				want: "ergon: cli: read .ergon/init.lock: ",
			},
			{
				name:   "returns 1 for a lock that does not decode",
				change: func(t *testing.T, dir string) { t.Helper(); write(t, dir, lockPath, "not JSON\n") },
				want:   "ergon: lock: invalid .ergon/init.lock",
			},
			{
				name: "returns 1 for a repository without a lock",
				change: func(t *testing.T, dir string) {
					t.Helper()
					assert.NoError(t, os.Remove(filepath.Join(dir, filepath.FromSlash(lockPath))), "Remove of the lock")
				},
				want: "ergon: baseline: the repository has no lock",
			},
			{
				name:   "returns 1 for a .ergon.yaml that the sync does not accept",
				change: func(t *testing.T, dir string) { t.Helper(); write(t, dir, ".ergon.yaml", "github: [\n") },
				want:   "ergon: options: invalid .ergon.yaml",
			},
			{
				name:   "returns 1 after the sync of a managed file that was edited by hand",
				change: func(t *testing.T, dir string) { t.Helper(); write(t, dir, licensePath, "edited\n") },
				want:   conflict + licensePath + "\n",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := initialized(t)
				tt.change(t, dir)
				status, _, stderr := upgrade(t.Context(), t, version, dir, map[string]string{upgradeEnv: newest},
					"init", "upgrade")
				assert.Equal(t, status, statusFailure, "the exit status")
				assert.HasPrefix(t, stderr, tt.want, "the standard error")
			})
		}

		upstreams := []struct {
			name string
			give upstream
			want string
		}{
			{
				name: "returns 1 for a module proxy that fails",
				give: upstream{versions: []string{"v" + newest}, fail: true},
				want: "502 Bad Gateway",
			},
			{
				name: "returns 1 for a release without checksums.txt",
				give: upstream{versions: []string{"v" + newest}},
				want: checksums + ": 404 Not Found",
			},
			{
				name: "returns 1 for a checksums.txt without the archive",
				give: upstream{
					versions:  []string{"v" + newest},
					checksums: strings.Repeat("0", 64) + "  other.tar.gz\n",
				},
				want: checksums + " has no line for ergon_" + newest + "_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz",
			},
			{
				name: "returns 1 for a checksums.txt with a line too long to read",
				give: upstream{versions: []string{"v" + newest}, checksums: strings.Repeat("0", 70000) + "\n"},
				want: "cli: read ",
			},
		}
		for _, tt := range upstreams {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				env := serveUpstream(t, &tt.give)
				status, _, stderr := upgrade(t.Context(), t, version, initialized(t), env, "init", "upgrade")
				assert.Equal(t, status, statusFailure, "the exit status")
				assert.Contains(t, stderr, tt.want, "the standard error")
			})
		}

		t.Run("returns 1 for an archive that differs from its digest", func(t *testing.T) {
			t.Parallel()
			env := serveUpstream(t, &upstream{
				versions: []string{"v" + newest},
				archive:  []byte("archive"),
				checksums: strings.Repeat(
					"0",
					64,
				) + "  ergon_" + newest + "_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz\n",
			})
			status, _, stderr := upgrade(t.Context(), t, version, initialized(t), env, "init", "upgrade")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Contains(t, stderr, "ergon: tool: the tool does not install: ", "the standard error")
		})

		servers := []struct {
			name   string
			server func(t *testing.T) string
		}{
			{
				name:   "returns 1 for a server of GitHub that is no URL",
				server: func(*testing.T) string { return "http://bad\x7fhost" },
			},
			{
				name: "returns 1 for a server of GitHub that does not respond",
				server: func(t *testing.T) string {
					t.Helper()
					gone := httptest.NewServer(http.NotFoundHandler())
					gone.Close()
					return gone.URL
				},
			},
		}
		for _, tt := range servers {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				env := serveUpstream(t, &upstream{versions: []string{"v" + newest}})
				env[serverEnv] = tt.server(t)
				status, _, stderr := upgrade(t.Context(), t, version, initialized(t), env, "init", "upgrade")
				assert.Equal(t, status, statusFailure, "the exit status")
				assert.Contains(t, stderr, "ergon: cli: GET ", "the standard error")
			})
		}

		t.Run("returns 1 for a cache directory that cannot be found", func(t *testing.T) {
			t.Parallel()
			env := released(t, &upstream{versions: []string{"v" + newest}}, native)
			var stdout, stderr bytes.Buffer
			p := upgradeProcess(t, initialized(t), env, &stdout, &stderr, "init", "upgrade")
			// The error comes with a directory of the test, which a run that went on would install into.
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, errCache }
			assert.Equal(t, cli.Run(t.Context(), p, register, version), statusFailure, "the exit status")
			assert.Equal(t, stderr.String(), "ergon: cli: find the cache directory: "+errCache.Error()+"\n",
				"the standard error")
		})

		t.Run("proposes the synced files and a changeset on the branch ergon-baseline/main", func(t *testing.T) {
			t.Parallel()
			dir := committed(t, register, func(dir string) {
				assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
			})
			h := &hub{head: strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))}
			status, stdout, stderr := upgrade(t.Context(), t, version, dir, h.start(t), "init", "ci", "upgrade")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+licensePath+"\nwrote "+changesetOfUpgrade+"\npull request 7\n",
				"the standard output")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/git/ref/heads/main",
				"GET /repos/" + hubRepo + "/git/ref/heads/ergon-baseline/main",
				"POST /repos/" + hubRepo + "/git/refs",
				"POST " + graphqlPath,
				"GET /repos/" + hubRepo + "/pulls?base=main&head=o%3Aergon-baseline%2Fmain&state=open",
				"POST /repos/" + hubRepo + "/pulls",
			}, "the requests to GitHub")
			commits := h.commits()
			assert.Length(t, commits, 1, "the commits")
			expect.Contains(t, commits[0], `"headline":"build: upgrade ergon to 1.2.3"`, "the message of the commit")
			expect.Contains(t, commits[0], `"path":"`+licensePath+`"`, "the LICENSE in the commit")
			expect.Contains(t, commits[0], `"path":"`+changesetOfUpgrade+`"`, "the changeset in the commit")
			expect.Equal(t, h.opened(), []string{proposedFiles}, "the bodies of the pull requests")
		})

		t.Run("skips a commit that is no longer the head of the base branch", func(t *testing.T) {
			t.Parallel()
			dir := committed(t, register, func(dir string) {
				assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
			})
			head := strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))
			h := &hub{head: laterCommit}
			status, stdout, stderr := upgrade(t.Context(), t, version, dir, h.start(t), "init", "ci", "upgrade")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+licensePath+"\nwrote "+changesetOfUpgrade+"\nskipped "+head+
				", which is no longer the head of main\n", "the standard output")
			assert.Equal(t, h.calls(), []string{"GET /repos/" + hubRepo + "/git/ref/heads/main"},
				"the requests to GitHub, which change nothing")
		})

		t.Run("returns the conflict of the sync when the base branch moved past the commit", func(t *testing.T) {
			t.Parallel()
			dir := committed(t, register, func(dir string) {
				assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
				write(t, dir, ignorePath, "edited\n")
			})
			h := &hub{head: laterCommit}
			status, stdout, stderr := upgrade(t.Context(), t, version, dir, h.start(t), "init", "ci", "upgrade")
			assert.Equal(t, status, statusFailure, "the exit status")
			expect.HasPrefix(t, stderr, conflict+ignorePath, "the standard error")
			expect.Contains(t, stdout, ", which is no longer the head of main\n", "the standard output")
			expect.Empty(t, h.opened(), "the pull requests")
		})

		t.Run("lists the overrides and the conflicts in the pull request", func(t *testing.T) {
			t.Parallel()
			dir := committed(t, register, func(dir string) {
				write(
					t,
					dir,
					".ergon.yaml",
					strings.Replace(read(t, dir, ".ergon.yaml"), "release: v7.0.1", overridden, 1),
				)
				status, _, stderr := run(t, dir, "init", "sync")
				assert.Equal(t, status, statusOK, "the exit status of init sync: "+stderr)
				assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
				write(t, dir, ignorePath, "edited\n")
			})
			h := &hub{head: strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))}
			status, _, stderr := upgrade(t.Context(), t, version, dir, h.start(t), "init", "ci", "upgrade")
			assert.Equal(t, status, statusFailure, "the exit status")
			expect.HasPrefix(t, stderr, conflict+ignorePath, "the standard error")
			expect.Equal(t, h.opened(), []string{proposedFiles + proposedOverrides + proposedConflicts},
				"the bodies of the pull requests")
		})

		t.Run("opens no pull request for a repository at the baseline", func(t *testing.T) {
			t.Parallel()
			h := &hub{}
			status, stdout, stderr := upgrade(
				t.Context(),
				t,
				version,
				committed(t, register, nil),
				h.start(t),
				"init",
				"ci",
				"upgrade",
			)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Empty(t, stdout, "the standard output")
			assert.Empty(t, h.calls(), "the requests to GitHub")
		})

		unproposed := []struct {
			name      string
			register  func(*language.Catalog) error
			languages []workspace.Language
			change    func(t *testing.T, dir string)
			want      string
		}{
			{
				name:     "opens no pull request for a sync that only leaves a managed file",
				register: register,
				change:   func(t *testing.T, dir string) { t.Helper(); write(t, dir, ignorePath, "edited\n") },
				want:     conflict + ignorePath,
			},
			{
				name:      "opens no pull request for overrides that do not resolve",
				register:  registerGamma,
				languages: []workspace.Language{gamma},
				change: func(t *testing.T, dir string) {
					t.Helper()
					assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
				},
				want: "ergon: pin: invalid source tag: gamma.make",
			},
		}
		for _, tt := range unproposed {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := &hub{}
				dir := committed(t, tt.register, func(dir string) { tt.change(t, dir) }, tt.languages...)
				var stdout, stderr bytes.Buffer
				p := upgradeProcess(t, dir, h.start(t), &stdout, &stderr, "init", "ci", "upgrade")
				assert.Equal(t, cli.Run(t.Context(), p, tt.register, version), statusFailure, "the exit status")
				expect.HasPrefix(t, stderr.String(), tt.want, "the standard error")
				expect.Empty(t, h.calls(), "the requests to GitHub")
			})
		}

		ci := []struct {
			name   string
			change func(t *testing.T, dir string, env map[string]string)
			want   string
		}{
			{
				name:   "returns 1 for an environment without GITHUB_REPOSITORY",
				change: func(_ *testing.T, _ string, env map[string]string) { env["GITHUB_REPOSITORY"] = "" },
				want:   "ergon: cli: set GITHUB_REPOSITORY",
			},
			{
				name: "returns 1 for a repository without .changeset/config.json",
				change: func(t *testing.T, dir string, _ map[string]string) {
					t.Helper()
					assert.NoError(
						t,
						os.Remove(filepath.Join(dir, filepath.FromSlash(configPath))),
						"Remove of the configuration",
					)
				},
				want: "ergon: cli: read .changeset/config.json, which ergon init seeds: ",
			},
			{
				name: "returns 1 for a .changeset/config.json that does not parse",
				change: func(t *testing.T, dir string, _ map[string]string) {
					t.Helper()
					write(t, dir, configPath, "[\n")
				},
				want: "ergon: release: invalid .changeset/config.json",
			},
			{
				name: "returns 1 outside a working tree of git",
				change: func(t *testing.T, dir string, _ map[string]string) {
					t.Helper()
					assert.NoError(t, os.RemoveAll(filepath.Join(dir, ".git")), "RemoveAll of .git")
				},
				want: "ergon: vcs: ",
			},
			{
				name: "returns 1 for a sync that fails",
				change: func(t *testing.T, dir string, _ map[string]string) {
					t.Helper()
					write(t, dir, ".ergon.yaml", "github: [\n")
				},
				want: "ergon: options: invalid .ergon.yaml",
			},
			{
				name: "returns 1 for a changeset that exists",
				change: func(t *testing.T, dir string, _ map[string]string) {
					t.Helper()
					assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
					write(t, dir, changesetOfUpgrade, "---\n---\n")
				},
				want: "ergon: release: create " + changesetOfUpgrade + ": ",
			},
			{
				name: "returns 1 for changes that git does not list",
				change: func(t *testing.T, dir string, _ map[string]string) {
					t.Helper()
					assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
					write(t, dir, ".git/index", "corrupt\n")
				},
				want: "ergon: vcs: ",
			},
			{
				name: "returns 1 for a proposal that GitHub refuses",
				change: func(t *testing.T, dir string, _ map[string]string) {
					t.Helper()
					assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
				},
				want: "ergon: forge: ",
			},
		}
		for _, tt := range ci {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := committed(t, register, nil)
				h := &hub{
					fail: "POST /repos/" + hubRepo + "/git/refs",
					head: strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD")),
				}
				env := h.start(t)
				tt.change(t, dir, env)
				status, _, stderr := upgrade(t.Context(), t, version, dir, env, "init", "ci", "upgrade")
				assert.Equal(t, status, statusFailure, "the exit status")
				assert.HasPrefix(t, stderr, tt.want, "the standard error")
			})
		}

		t.Run("returns 1 for random digits that do not read", func(t *testing.T) {
			t.Parallel()
			h := &hub{}
			dir := committed(t, register, func(dir string) {
				assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
			})
			var stdout, stderr bytes.Buffer
			p := upgradeProcess(t, dir, h.start(t), &stdout, &stderr, "init", "ci", "upgrade")
			p.Random = iotest.ErrReader(errRandom)
			assert.Equal(t, cli.Run(t.Context(), p, register, version), statusFailure, "the exit status")
			assert.Contains(t, stderr.String(), errRandom.Error(), "the standard error")
		})
	})
}

// registerGamma adds the toolchain tool, the languages alpha and beta, and the language gamma of the
// producer unsourced to c.
func registerGamma(c *language.Catalog) error {
	if err := register(c); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: gamma, Toolchain: tool},
		unsourced{producer(gamma, "gamma.txt")})
}

// developed returns a new repository whose lock records the version of a build of ergon without a
// release. A release writes the lock, because such a build writes no new lock, and the version in
// the lock is then replaced.
func developed(t *testing.T) string {
	t.Helper()
	dir := initialized(t)
	recorded := `"ergon": "` + version.Release + `"`
	write(t, dir, lockPath, strings.Replace(read(t, dir, lockPath), recorded, `"ergon": "`+dev.Release+`"`, 1))
	assert.Contains(t, read(t, dir, lockPath), `"ergon": "`+dev.Release+`"`, "the lock")
	return dir
}

// released serves u for the test t with the archive of ergon of platform, and with the checksums.txt
// of the releases newest and later for platform, and returns the variables of the environment that
// point an upgrade at it.
func released(t *testing.T, u *upstream, platform option.Platform) map[string]string {
	t.Helper()
	build := ergonArchive
	if platform.OS() == "windows" {
		build = windowsArchive
	}
	archive, err := build()
	assert.NoError(t, err, "the archive of the release")
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	u.archive = archive
	for _, release := range []string{newest, later} {
		u.checksums += digest + "  ergon_" + release + "_" + platform.OS() + "_" + platform.Arch() + ".tar.gz\n"
	}
	return serveUpstream(t, u)
}

// serveUpstream serves u for the test t, and returns the variables of the environment that point an
// upgrade at it.
func serveUpstream(t *testing.T, u *upstream) map[string]string {
	t.Helper()
	server := httptest.NewServer(u)
	t.Cleanup(server.Close)
	return map[string]string{proxyEnv: server.URL + proxyPath, serverEnv: server.URL}
}

// committed returns a working tree of git on the branch main with one commit. The commit has the
// repository that init new writes under the registration register with a flag --language for each
// of languages, after change modified it. A nil change modifies nothing.
func committed(
	t *testing.T, register func(*language.Catalog) error, change func(dir string), languages ...workspace.Language,
) string {
	t.Helper()
	dir := vcstest.Repository(t, files.Tree{})
	vcstest.Git(t, dir, "symbolic-ref", "HEAD", "refs/heads/main")
	args := slices.Concat([]string{"init", "new"}, required)
	for _, l := range languages {
		args = append(args, "--language", string(l))
	}
	status, _, stderr := runWith(t, register, dir, args...)
	assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
	if change != nil {
		change(dir)
	}
	vcstest.Commit(t, dir, "first")
	return dir
}

// upgrade runs the command line args of the release v of ergon in the working directory dir under
// ctx, with the variables of env, as upgradeProcess states, and returns the exit status, the
// standard output and the standard error.
func upgrade(
	ctx context.Context, t *testing.T, v cli.Version, dir string, env map[string]string, args ...string,
) (status int, stdout, stderr string) {
	t.Helper()
	var out, errs bytes.Buffer
	status = cli.Run(ctx, upgradeProcess(t, dir, env, &out, &errs, args...), register, v)
	return status, out.String(), errs.String()
}

// upgradeProcess returns the process of an upgrade case: the process of a release case with the
// variables of an upgrade set to empty values and then the variables of env, a cache directory of
// the test t, and a transport that the test checks for bodies that ergon left open when it ends.
func upgradeProcess(
	t *testing.T, dir string, env map[string]string, stdout, stderr io.Writer, args ...string,
) *cli.Process {
	t.Helper()
	p := releaseProcess(dir, nil, stdout, stderr, args...)
	p.Env = append(p.Env, proxyEnv+"=", upgradeEnv+"=", fakeExitEnv+"=")
	for name, value := range env {
		p.Env = append(p.Env, name+"="+value)
	}
	cache := t.TempDir()
	p.CacheDir = func() (string, error) { return cache, nil }
	counted := &transport{base: p.Transport}
	t.Cleanup(func() { expect.Equal(t, counted.open.Load(), int64(0), "the bodies that ergon left open") })
	p.Transport = counted
	return p
}

// archiveOf returns an archive of ergon whose program name is a copy of the test binary, which acts
// as the fake program for fakeEnv. It returns the error of the read of the test binary. The writers
// of the archive write into memory, so they return no error.
func archiveOf(name string) ([]byte, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}
	content, err := os.ReadFile(self)
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}
	var b bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&b, gzip.NoCompression)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg})
	_, _ = tw.Write(content)
	_ = tw.Close()
	_ = gz.Close()
	return b.Bytes(), nil
}
