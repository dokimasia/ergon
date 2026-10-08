// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/pin"
)

// The paths of the registries of the fake below the address of its server.
const (
	goproxyPath    = "/goproxy"
	pypiPath       = "/pypi"
	npmPath        = "/npm"
	cratesPath     = "/crates"
	mavenPath      = "/maven"
	packagistPath  = "/packagist"
	chocolateyPath = "/chocolatey"
)

// userAgent is the user agent of the requests of a resolver.
const userAgent = "ergon (https://github.com/dokimasia/ergon)"

// limit is the largest document that a resolver reads, 64 MiB.
const limit = 67108864

// week is the minimum age of the releases of the cases.
const week = 7 * 24 * time.Hour

// windows is the system of the platforms whose asset the release binary of the cases lacks.
const windows = "windows"

// The hooks of pre-commit, the version of GitHub of the cases: the repository and the key.
const (
	hooksRepo = "pre-commit/pre-commit-hooks"
	hooksKey  = "common.pre-commit-hooks"
)

// The release binary of the cases: its repository, its key, its program, and two digests of its
// assets.
const (
	toolRepo     = "example/tool"
	toolKey      = "demo.tools.tool"
	program      = "tool"
	linuxDigest  = "1111111111111111111111111111111111111111111111111111111111111111"
	darwinDigest = "2222222222222222222222222222222222222222222222222222222222222222"
)

// The action of the cases: its repository, its key, and the commit of its release v7.1.0.
const (
	actionRepo   = "actions/setup-go"
	actionKey    = "demo.ci.actions.setup"
	actionCommit = "b7ad1dad31e06c5925ef5d2fc7ad053ef454303e"
)

// The PyPI package of the cases: its name, its key, and the request of its document.
const (
	pypiName     = "ruff"
	pypiKey      = "python.tools.ruff"
	pypiDocument = "GET /pypi/pypi/ruff/json"
)

// assetPath is the path of the asset of the release binary of the cases at the fake.
const assetPath = "/assets/tool_1.1.0_linux_amd64.tar.gz"

// The times of the cases: the time of the run, a publication more than a week before it, a
// publication less than a week before it, and a publication a week before it.
var (
	now   = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	old   = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	young = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	aged  = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
)

// cest is the time zone of central Europe in summer, two hours ahead of UTC.
var cest = time.FixedZone("CEST", 7200)

// errGitHub is the error of the fake GitHub that fails.
var errGitHub = errors.New("github: the request failed")

// route is the response of the fake registry to a request.
type route struct {
	// header are the headers of the response.
	header http.Header

	// body is the body of the response.
	body string

	// status is the status of the response, or 0 for 200.
	status int

	// short declares a length 10 bytes longer than body, so the response ends before its length.
	short bool
}

// registry is a fake of the registries of the cases. It responds to a request with the route of its
// method, a space and its target, such as GET /pypi/pypi/ruff/json, and with 404 to a request
// without a route. It records the user agent of each request.
type registry struct {
	// routes are the responses by method and target. The fake only reads them.
	routes map[string]route

	// url is the address of the server of the fake.
	url string

	// agents are the user agents of the requests, in their order.
	agents []string

	// mu guards agents.
	mu sync.Mutex
}

// ServeHTTP records the user agent of r, and writes the response of its route.
func (g *registry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	g.agents = append(g.agents, r.UserAgent())
	g.mu.Unlock()
	next, ok := g.routes[r.Method+" "+r.URL.RequestURI()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	maps.Copy(w.Header(), next.header)
	if next.short {
		w.Header().Set("Content-Length", strconv.Itoa(len(next.body)+10))
	}
	if next.status != 0 {
		w.WriteHeader(next.status)
	}
	_, _ = io.WriteString(w, next.body)
}

// recorded returns the user agents of the requests that g received, in their order.
func (g *registry) recorded() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.agents)
}

// gitHub is a fake of the API of GitHub with the releases and the tags of the cases.
type gitHub struct {
	// releases are the releases of each repository.
	releases map[string][]forge.Release

	// tags are the commits of the tags, by the repository, an @ and the tag, such as
	// actions/setup-go@v7.1.0.
	tags map[string]string

	// err is the error of Releases, or nil.
	err error

	// tagErr is the error of Tag, or nil.
	tagErr error
}

var _ pin.GitHub = (*gitHub)(nil)

// Releases returns the releases of repo, and err.
func (g *gitHub) Releases(_ context.Context, repo string) ([]forge.Release, error) {
	return g.releases[repo], g.err
}

// Tag returns the commit of the tag name of repo and whether g has the tag, and tagErr.
func (g *gitHub) Tag(_ context.Context, repo, name string) (string, bool, error) {
	commit, ok := g.tags[repo+"@"+name]
	return commit, ok, g.tagErr
}

// transport is the transport of a resolver of the cases. It counts the bodies of the responses that
// the resolver has not closed.
type transport struct {
	// base sends the requests.
	base http.RoundTripper

	// open is the number of bodies that the resolver has not closed.
	open atomic.Int64
}

// RoundTrip sends r through base, and counts the body of its response until the resolver closes it.
// It returns the error of base.
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

// tool is the release binary of the cases. Its release publishes <name>_<version>_<os>_<arch>.tar.gz
// for Linux and macOS, and no asset for Windows.
type tool struct {
	option.Binary `yaml:",inline"`

	// Name is the name of the program, which starts the name of each asset.
	Name string `yaml:"name"`
}

var _ option.Release = tool{}

// Asset returns the archive of t for p, and an error that wraps option.ErrNoAsset for Windows.
func (t tool) Asset(p option.Platform) (option.Asset, error) {
	if p.OS() == windows {
		return option.Asset{}, fmt.Errorf("%w: %s", option.ErrNoAsset, p)
	}
	name := t.Name + "_" + t.Version + "_" + p.OS() + "_" + p.Arch() + ".tar.gz"
	return option.Asset{URL: "https://github.com/" + toolRepo + "/releases/download/v" + t.Version + "/" + name}, nil
}

// Repository returns the repository of the releases of the tool.
func (tool) Repository() string {
	return toolRepo
}

// loose is a release binary of the cases whose field Binary is no option.Binary.
type loose struct {
	// Binary is the version of the binary.
	Binary string
}

var _ option.Release = loose{}

// Pin returns the version of l with a digest of Linux on x86-64.
func (l loose) Pin() option.Binary {
	return option.Binary{SHA256: map[option.Platform]string{option.LinuxAMD64: linuxDigest}, Version: l.Binary}
}

// Asset returns the asset of the tool of the cases for p.
func (l loose) Asset(p option.Platform) (option.Asset, error) {
	return tool{Version: l.Binary, Name: program}.Asset(p)
}

// Repository returns the repository of the releases of the tool.
func (loose) Repository() string {
	return toolRepo
}

// bare embeds option.Binary without the methods of a release binary.
type bare struct {
	option.Binary
}

func TestResolver(t *testing.T) {
	t.Parallel()

	t.Run("PublicRegistries", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the addresses of the public registries", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, pin.PublicRegistries(), pin.Registries{
				GoProxy:    "https://proxy.golang.org",
				PyPI:       "https://pypi.org",
				NPM:        "https://registry.npmjs.org",
				Crates:     "https://index.crates.io",
				Maven:      "https://repo1.maven.org/maven2",
				Packagist:  "https://repo.packagist.org",
				Chocolatey: "https://community.chocolatey.org/api/v2",
			}, "the registries")
		})
	})

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		selections := []struct {
			name string
			give []forge.Release
			want pin.Resolution
		}{
			{
				name: "returns the newest release of the major version of the pin",
				give: []forge.Release{
					{Tag: "v6.1.0", Published: old},
					{Tag: "v6.2.0", Published: old},
					{Tag: "v6.1.5", Published: old},
					{Tag: "v5.9.0", Published: old},
				},
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "v6.2.0"}},
			},
			{
				name: "returns the newest release of a later major version beside it",
				give: []forge.Release{
					{Tag: "v7.0.0", Published: old}, {Tag: "v8.1.0", Published: old}, {Tag: "v8.0.0", Published: old},
				},
				want: pin.Resolution{Major: &pin.Release{Published: old, Version: "v8.1.0"}},
			},
			{
				name: "leaves out a release younger than the minimum age",
				give: []forge.Release{{Tag: "v6.2.0", Published: young}, {Tag: "v6.1.0", Published: old}},
				want: pin.Resolution{Next: &pin.Release{Published: old, Version: "v6.1.0"}},
			},
			{
				name: "takes a release published the minimum age before the run",
				give: []forge.Release{{Tag: "v6.1.0", Published: aged}},
				want: pin.Resolution{Next: &pin.Release{Published: aged, Version: "v6.1.0"}},
			},
			{
				name: "leaves out a later major version younger than the minimum age",
				give: []forge.Release{{Tag: "v7.0.0", Published: young}},
			},
			{
				name: "returns no release for a pin at its newest release",
				give: []forge.Release{{Tag: "v6.0.0", Published: old}, {Tag: "v5.0.0", Published: old}},
			},
			{
				name: "returns the times of publication in UTC",
				give: []forge.Release{
					{Tag: "v6.1.0", Published: old.In(cest)},
					{Tag: "v7.0.0", Published: old.In(cest)},
				},
				want: pin.Resolution{
					Next:  &pin.Release{Published: old, Version: "v6.1.0"},
					Major: &pin.Release{Published: old, Version: "v7.0.0"},
				},
			},
		}
		for _, tt := range selections {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, nil)
				r.GitHub = &gitHub{releases: map[string][]forge.Release{hooksRepo: tt.give}}
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindGitHub, Key: hooksKey, Name: hooksRepo,
					Version: "v6.0.0",
				})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		exemptions := []struct {
			name   string
			exempt []string
			want   pin.Resolution
		}{
			{
				name:   "takes a young release for a name that starts with a prefix of Exempt",
				exempt: []string{"example/", "pre-commit/"},
				want:   pin.Resolution{Next: &pin.Release{Published: young, Version: "v6.1.0"}},
			},
			{
				name:   "leaves out a young release for a name that starts with no prefix of Exempt",
				exempt: []string{"example/", "commit/"},
			},
		}
		for _, tt := range exemptions {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, nil)
				r.GitHub = &gitHub{releases: map[string][]forge.Release{hooksRepo: {{Tag: "v6.1.0", Published: young}}}}
				r.Exempt = tt.exempt
				got, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindGitHub, Key: hooksKey, Name: hooksRepo,
					Version: "v6.0.0",
				})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}

		t.Run("takes the newest later major version with Major", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{releases: map[string][]forge.Release{hooksRepo: {
				{Tag: "v6.1.0", Published: old}, {Tag: "v8.0.0", Published: old}, {Tag: "v7.0.0", Published: old},
			}}}
			r.Major = true
			got, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindGitHub, Key: hooksKey, Name: hooksRepo,
				Version: "v6.0.0",
			})
			assert.NoError(t, err, "Resolve")
			assert.Equal(
				t,
				got,
				pin.Resolution{Next: &pin.Release{Published: old, Version: "v8.0.0"}},
				"the resolution",
			)
		})

		t.Run("returns ErrRegistry with the key for a pin of no kind", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			_, err := r.Resolve(t.Context(), &pin.Pin{Key: hooksKey, Name: hooksRepo, Version: "v6.0.0"})
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.HasPrefix(t, err.Error(), "pin: resolve "+hooksKey+": ", "the error")
		})

		t.Run("returns the digests that GitHub states for the platforms of a release binary", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{
				Tag: "v1.1.0", Published: old,
				Assets: []forge.Asset{
					{Name: "tool_1.1.0_linux_amd64.tar.gz", Digest: linuxDigest},
					{Name: "tool_1.1.0_darwin_arm64.tar.gz", Digest: darwinDigest},
					{Name: "tool_1.1.0_linux_arm64.tar.gz", Digest: darwinDigest},
				},
			}}}}
			got, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindBinary, Key: toolKey, Name: toolRepo, Version: "1.0.0", Value: tool{
					Version: "1.0.0",
					SHA256: map[option.Platform]string{
						option.LinuxAMD64:  linuxDigest,
						option.DarwinARM64: darwinDigest,
					},
					Name: program,
				},
			})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, got, pin.Resolution{Next: &pin.Release{
				Published: old, Version: "1.1.0",
				Digests: map[option.Platform]string{option.LinuxAMD64: linuxDigest, option.DarwinARM64: darwinDigest},
			}}, "the resolution")
		})

		t.Run("returns a later major version of a release binary without its digests", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{Tag: "v2.0.0", Published: old}}}}
			got, err := r.Resolve(t.Context(), linuxTool())
			assert.NoError(t, err, "Resolve")
			assert.Equal(
				t,
				got,
				pin.Resolution{Major: &pin.Release{Published: old, Version: "2.0.0"}},
				"the resolution",
			)
		})

		t.Run("downloads and hashes the asset of a platform without a digest", func(t *testing.T) {
			t.Parallel()
			r, fake := serve(t, map[string]route{"GET " + assetPath: {body: "archive"}})
			r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{
				Tag: "v1.1.0", Published: old,
				Assets: []forge.Asset{{Name: "tool_1.1.0_linux_amd64.tar.gz", URL: fake.url + assetPath}},
			}}}}
			got, err := r.Resolve(t.Context(), linuxTool())
			assert.NoError(t, err, "Resolve")
			sum := sha256.Sum256([]byte("archive"))
			assert.NotNil(t, got.Next, "the next release")
			assert.Equal(t, got.Next.Digests, map[option.Platform]string{option.LinuxAMD64: hex.EncodeToString(sum[:])},
				"the digests")
		})

		t.Run("sends the user agent of ergon with the download of an asset", func(t *testing.T) {
			t.Parallel()
			r, fake := serve(t, map[string]route{"GET " + assetPath: {body: "archive"}})
			r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{
				Tag: "v1.1.0", Published: old,
				Assets: []forge.Asset{{Name: "tool_1.1.0_linux_amd64.tar.gz", URL: fake.url + assetPath}},
			}}}}
			_, err := r.Resolve(t.Context(), linuxTool())
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, fake.recorded(), []string{userAgent}, "the user agents")
		})

		downloads := []struct {
			name string
			fail route
		}{
			{name: "returns ErrRegistry for an asset whose download fails", fail: route{status: http.StatusBadGateway}},
			{
				name: "returns ErrRegistry for an asset that ends before its length",
				fail: route{body: "arch", short: true},
			},
		}
		for _, tt := range downloads {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, fake := serve(t, map[string]route{"GET " + assetPath: tt.fail})
				r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{
					Tag: "v1.1.0", Published: old,
					Assets: []forge.Asset{{Name: "tool_1.1.0_linux_amd64.tar.gz", URL: fake.url + assetPath}},
				}}}}
				_, err := r.Resolve(t.Context(), linuxTool())
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			})
		}

		gone := httptest.NewServer(http.NotFoundHandler())
		gone.Close()
		addresses := []struct {
			name string
			give string
		}{
			{name: "returns ErrRegistry for an asset whose address is no URL", give: "http://bad\x7fhost/tool.tar.gz"},
			{name: "returns ErrRegistry for an asset whose server does not respond", give: gone.URL + assetPath},
		}
		for _, tt := range addresses {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, nil)
				r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{
					Tag: "v1.1.0", Published: old,
					Assets: []forge.Asset{{Name: "tool_1.1.0_linux_amd64.tar.gz", URL: tt.give}},
				}}}}
				_, err := r.Resolve(t.Context(), linuxTool())
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			})
		}

		t.Run("returns ErrAsset for a release without the asset of a platform of the pin", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{
				Tag: "v1.1.0", Published: old,
				Assets: []forge.Asset{{Name: "tool_1.0.0_linux_amd64.tar.gz", Digest: linuxDigest}},
			}}}}
			_, err := r.Resolve(t.Context(), linuxTool())
			assert.ErrorIs(t, err, pin.ErrAsset, "Resolve")
		})

		t.Run("returns ErrAsset for a platform whose asset the type of the pin does not name", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{Tag: "v1.1.0", Published: old}}}}
			_, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindBinary, Key: toolKey, Name: toolRepo, Version: "1.0.0", Value: tool{
					Version: "1.0.0",
					SHA256:  map[option.Platform]string{option.WindowsAMD64: linuxDigest},
					Name:    program,
				},
			})
			assert.ErrorIs(t, err, pin.ErrAsset, "Resolve")
			assert.ErrorIs(t, err, option.ErrNoAsset, "Resolve")
		})

		values := []struct {
			name string
			give any
		}{
			{name: "returns ErrAsset for a release binary of a value that is no release binary", give: "1.0.0"},
			{name: "returns ErrAsset for a release binary of a struct that is no release binary", give: bare{}},
			{
				name: "returns ErrAsset for a release binary of a pointer",
				give: &tool{Version: "1.0.0", Name: program},
			},
			{
				name: "returns ErrAsset for a release binary whose field Binary is no option.Binary",
				give: loose{"1.0.0"},
			},
		}
		for _, tt := range values {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, nil)
				r.GitHub = &gitHub{releases: map[string][]forge.Release{toolRepo: {{Tag: "v1.1.0", Published: old}}}}
				_, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindBinary, Key: toolKey, Name: toolRepo,
					Version: "1.0.0", Value: tt.give,
				})
				assert.ErrorIs(t, err, pin.ErrAsset, "Resolve")
			})
		}

		t.Run("returns the commit of the tag of the release of an action", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{
				releases: map[string][]forge.Release{actionRepo: {{Tag: "v7.1.0", Published: old}}},
				tags:     map[string]string{actionRepo + "@v7.1.0": actionCommit},
			}
			got, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindAction, Key: actionKey, Name: actionRepo,
				Version: "v7.0.0",
			})
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, got, pin.Resolution{Next: &pin.Release{
				Published: old, Version: "v7.1.0",
				Commit: actionCommit,
			}}, "the resolution")
		})

		t.Run("returns ErrRegistry with the key for a release of an action without its tag", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{releases: map[string][]forge.Release{actionRepo: {{Tag: "v7.1.0", Published: old}}}}
			_, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindAction, Key: actionKey, Name: actionRepo,
				Version: "v7.0.0",
			})
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.HasPrefix(t, err.Error(), "pin: resolve "+actionKey+": ", "the error")
		})

		t.Run("returns ErrRegistry for the tag of an action that GitHub does not return", func(t *testing.T) {
			t.Parallel()
			r, _ := serve(t, nil)
			r.GitHub = &gitHub{
				releases: map[string][]forge.Release{actionRepo: {{Tag: "v7.1.0", Published: old}}},
				tagErr:   errGitHub,
			}
			_, err := r.Resolve(t.Context(), &pin.Pin{
				Kind: pin.KindAction, Key: actionKey, Name: actionRepo,
				Version: "v7.0.0",
			})
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
			assert.ErrorIs(t, err, errGitHub, "Resolve")
		})

		t.Run("sends the user agent of ergon with the request of a document", func(t *testing.T) {
			t.Parallel()
			r, fake := serve(t, map[string]route{pypiDocument: {body: `{"releases":{}}`}})
			_, err := r.Resolve(
				t.Context(),
				&pin.Pin{Kind: pin.KindPyPI, Key: pypiKey, Name: pypiName, Version: "1.0.0"},
			)
			assert.NoError(t, err, "Resolve")
			assert.Equal(t, fake.recorded(), []string{userAgent}, "the user agents")
		})

		failures := []struct {
			name string
			fail route
			want string
		}{
			{
				name: "returns ErrRegistry with the status of a response other than 200",
				fail: route{status: http.StatusInternalServerError},
				want: "500 Internal Server Error",
			},
			{
				name: "returns ErrRegistry for a response that ends before its length",
				fail: route{body: `{"releases":`, short: true},
				want: "unexpected EOF",
			},
			{
				name: "returns ErrRegistry for a project that the registry does not have",
				fail: route{status: http.StatusNotFound},
				want: "the registry does not have it",
			},
			{
				name: "returns ErrRegistry for a project that the registry removed",
				fail: route{status: http.StatusGone},
				want: "the registry does not have it",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, map[string]route{pypiDocument: tt.fail})
				_, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindPyPI, Key: pypiKey, Name: pypiName,
					Version: "1.0.0",
				})
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}

		registries := []struct {
			name string
			give string
		}{
			{
				name: "returns ErrRegistry with the request for a registry whose address is no URL",
				give: "http://bad\x7fhost",
			},
			{name: "returns ErrRegistry with the request for a registry that does not respond", give: gone.URL},
		}
		for _, tt := range registries {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _ := serve(t, nil)
				r.Registries.PyPI = tt.give
				_, err := r.Resolve(t.Context(), &pin.Pin{
					Kind: pin.KindPyPI, Key: pypiKey, Name: pypiName,
					Version: "1.0.0",
				})
				assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
				assert.Contains(t, err.Error(), "GET "+tt.give+"/pypi/"+pypiName+"/json: ", "the error")
			})
		}

		const prefix, suffix = `{"releases":{},"pad":"`, `"}`

		t.Run("reads a document of 64 MiB", func(t *testing.T) {
			t.Parallel()
			document := prefix + strings.Repeat("x", limit-len(prefix)-len(suffix)) + suffix
			r, _ := serve(t, map[string]route{pypiDocument: {body: document}})
			_, err := r.Resolve(
				t.Context(),
				&pin.Pin{Kind: pin.KindPyPI, Key: pypiKey, Name: pypiName, Version: "1.0.0"},
			)
			assert.NoError(t, err, "Resolve")
		})

		t.Run("returns ErrRegistry for a document longer than 64 MiB", func(t *testing.T) {
			t.Parallel()
			document := prefix + strings.Repeat("x", limit+1-len(prefix)-len(suffix)) + suffix
			r, _ := serve(t, map[string]route{pypiDocument: {body: document}})
			_, err := r.Resolve(
				t.Context(),
				&pin.Pin{Kind: pin.KindPyPI, Key: pypiKey, Name: pypiName, Version: "1.0.0"},
			)
			assert.ErrorIs(t, err, pin.ErrRegistry, "Resolve")
		})
	})
}

// serve starts the fake registry with routes for the test t. It returns the fake, and a resolver
// of the cases at now with a minimum age of a week, which reads the registries of the fake, each
// below a path of its own, and a fake GitHub without releases.
func serve(t *testing.T, routes map[string]route) (*pin.Resolver, *registry) {
	t.Helper()
	fake := &registry{routes: routes}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	fake.url = srv.URL
	client := &transport{base: srv.Client().Transport}
	t.Cleanup(func() { expect.Equal(t, client.open.Load(), int64(0), "the bodies that the resolver left open") })
	return &pin.Resolver{
		Client: &http.Client{Transport: client},
		GitHub: &gitHub{},
		Now:    func() time.Time { return now },
		Registries: pin.Registries{
			GoProxy:    srv.URL + goproxyPath,
			PyPI:       srv.URL + pypiPath,
			NPM:        srv.URL + npmPath,
			Crates:     srv.URL + cratesPath,
			Maven:      srv.URL + mavenPath,
			Packagist:  srv.URL + packagistPath,
			Chocolatey: srv.URL + chocolateyPath,
		},
		MinAge: week,
	}, fake
}

// linuxTool returns the pin of the release binary of the cases at 1.0.0, with a digest of Linux on
// x86-64.
func linuxTool() *pin.Pin {
	b := option.Binary{Version: "1.0.0", SHA256: map[option.Platform]string{option.LinuxAMD64: linuxDigest}}
	return &pin.Pin{
		Kind: pin.KindBinary, Key: toolKey, Name: toolRepo, Version: b.Version,
		Value: tool{Binary: b, Name: program},
	}
}
