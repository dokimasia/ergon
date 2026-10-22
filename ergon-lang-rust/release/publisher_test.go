// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/rust/release"
)

// The addresses that the cases answer: the file of probe-b in the sparse index of crates.io, the
// OIDC token of a job of GitHub Actions, and the tokens of the trusted publishing of crates.io.
const (
	indexB    = "GET /pr/ob/probe-b"
	oidcURL   = "https://token.actions.example/oidc?api-version=2.0"
	oidcToken = "GET /oidc"
	tokens    = "/api/v1/trusted_publishing/tokens"
	exchange  = "POST " + tokens
	revoke    = "DELETE " + tokens
)

// The values of the token exchange of the cases.
const (
	requestToken = "the request token"
	jwt          = "the.jwt.of-the-job"
	cratesToken  = "the token of crates.io"
)

// The errors of a request that fails, and of a body that fails to read.
var (
	errConnection = errors.New("crates: the connection failed")
	errBody       = errors.New("crates: the body failed")
)

// answer is the status and the body of a response of a fake.
type answer struct {
	body   string
	status int
}

// request is a request that a fake records.
type request struct {
	method        string
	url           string
	authorization string
	agent         string
	contentType   string
	body          string
}

// crates is a fake of crates.io, of its sparse index and of the OIDC endpoint of GitHub Actions.
// The transport records each request. It answers a request with err, or with the answer of its
// method and its path in answers and 404 for any other. The body of a response fails to read with
// bodyErr when it is set.
type crates struct {
	err      error
	bodyErr  error
	answers  map[string]answer
	requests []request
	mu       sync.Mutex
}

// RoundTrip records r and answers it as crates states.
func (f *crates) RoundTrip(r *http.Request) (*http.Response, error) {
	body := ""
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = string(data)
	}
	f.mu.Lock()
	f.requests = append(f.requests, request{
		method:        r.Method,
		url:           r.URL.String(),
		authorization: r.Header.Get("Authorization"),
		agent:         r.Header.Get("User-Agent"),
		contentType:   r.Header.Get("Content-Type"),
		body:          body,
	})
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	a, ok := f.answers[r.Method+" "+r.URL.Path]
	if !ok {
		a = answer{status: http.StatusNotFound}
	}
	var reader io.Reader = strings.NewReader(a.body)
	if f.bodyErr != nil {
		reader = iotest.ErrReader(f.bodyErr)
	}
	return &http.Response{
		StatusCode: a.status,
		Status:     strconv.Itoa(a.status) + " " + http.StatusText(a.status),
		Header:     http.Header{},
		Body:       io.NopCloser(reader),
		Request:    r,
	}, nil
}

// environment returns a function that reads the variables of env, as os.Getenv reads those of the
// process.
func environment(env map[string]string) func(string) string {
	return func(key string) string { return env[key] }
}

// job is the environment of a job of GitHub Actions with the permission id-token.
var job = map[string]string{
	"ACTIONS_ID_TOKEN_REQUEST_URL":   oidcURL,
	"ACTIONS_ID_TOKEN_REQUEST_TOKEN": requestToken,
}

// exchanged are the answers of a token exchange that succeeds.
func exchanged() map[string]answer {
	return map[string]answer{
		oidcToken: {status: http.StatusOK, body: `{"value": "` + jwt + `"}`},
		exchange:  {status: http.StatusOK, body: `{"token": "` + cratesToken + `"}`},
		revoke:    {status: http.StatusNoContent},
	}
}

// packed returns a pack directory with the .crate files of pkgs.
func packed(t *testing.T, pkgs ...workspace.Package) string {
	t.Helper()
	tree := files.Tree{}
	for _, p := range pkgs {
		tree["rust/"+p.Name+"/"+p.Name+"-"+p.Version.String()+".crate"] = files.Text("the crate " + p.Name)
	}
	return files.Workspace(t, tree)
}

func TestPublisher(t *testing.T) {
	t.Parallel()

	t.Run("Published", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want bool
		}{
			{
				name: "reports true for a version that the index has",
				give: `{"name":"probe-b","vers":"0.2.0","yanked":false}` + "\n" + `{"name":"probe-b","vers":"0.2.1"}`,
				want: true,
			},
			{
				name: "reports true for a yanked version",
				give: `{"name":"probe-b","vers":"0.2.1","yanked":true}`,
				want: true,
			},
			{
				name: "reports true for the version with other build metadata",
				give: `{"name":"probe-b","vers":"0.2.1+3"}`,
				want: true,
			},
			{
				name: "reports false for a version that the index does not have",
				give: `{"name":"probe-b","vers":"0.2.0"}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f := &crates{answers: map[string]answer{indexB: {status: http.StatusOK, body: tt.give + "\n"}}}
				got, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &releasedB)
				assert.NoError(t, err, "Published")
				assert.Equal(t, got, tt.want, "Published")
			})
		}

		t.Run("reports false for a crate that the index does not have", func(t *testing.T) {
			t.Parallel()
			f := &crates{}
			got, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &releasedB)
			assert.NoError(t, err, "Published")
			assert.False(t, got, "Published")
		})

		paths := []struct {
			name     string
			giveName string
			wantURL  string
		}{
			{name: "reads a name of one character under 1", giveName: "a", wantURL: "https://index.crates.io/1/a"},
			{name: "reads a name of two characters under 2", giveName: "ab", wantURL: "https://index.crates.io/2/ab"},
			{
				name:     "reads a name of three characters under 3 and its first character",
				giveName: "abc",
				wantURL:  "https://index.crates.io/3/a/abc",
			},
			{
				name:     "reads a longer name under its first two and its next two characters in lower case",
				giveName: "Probe-B",
				wantURL:  "https://index.crates.io/pr/ob/probe-b",
			},
		}
		for _, tt := range paths {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f := &crates{}
				p := releasedB
				p.Name = tt.giveName
				_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &p)
				assert.NoError(t, err, "Published")
				assert.Length(t, f.requests, 1, "the requests")
				expect.Equal(t, f.requests[0].url, tt.wantURL, "the address of the request")
				expect.Equal(t, f.requests[0].agent, "ergon (https://github.com/dokimasia/ergon)", "the user agent")
			})
		}

		invalid := []struct {
			name     string
			giveFake *crates
		}{
			{
				name:     "returns ErrIndex for another status of the index",
				giveFake: &crates{answers: map[string]answer{indexB: {status: http.StatusServiceUnavailable}}},
			},
			{
				name:     "returns ErrIndex for a line that does not decode",
				giveFake: &crates{answers: map[string]answer{indexB: {status: http.StatusOK, body: "{\n"}}},
			},
			{
				name:     "returns ErrIndex for a version that does not parse",
				giveFake: &crates{answers: map[string]answer{indexB: {status: http.StatusOK, body: `{"vers":"one"}`}}},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := release.Publisher{
					Client: &http.Client{Transport: tt.giveFake},
				}.Published(
					t.Context(),
					&releasedB,
				)
				assert.ErrorIs(t, err, release.ErrIndex, "Published")
			})
		}

		failures := []struct {
			name     string
			giveFake *crates
			want     error
		}{
			{name: "returns the error of the request", giveFake: &crates{err: errConnection}, want: errConnection},
			{
				name: "returns the error of reading the body",
				giveFake: &crates{
					bodyErr: errBody,
					answers: map[string]answer{indexB: {status: http.StatusOK}},
				},
				want: errBody,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := release.Publisher{
					Client: &http.Client{Transport: tt.giveFake},
				}.Published(
					t.Context(),
					&releasedB,
				)
				assert.ErrorIs(t, err, tt.want, "Published")
			})
		}

		t.Run("returns the error of a name that is not valid in an address", func(t *testing.T) {
			t.Parallel()
			f := &crates{}
			p := releasedB
			p.Name = "probe%zz"
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &p)
			ue := assert.ErrorAs[*url.Error](t, err, "Published")
			assert.Equal(t, ue.Op, "parse", "the operation of the error")
			assert.Empty(t, f.requests, "the requests")
		})
	})

	t.Run("Publish", func(t *testing.T) {
		t.Parallel()

		t.Run("runs cargo publish without a verification for the crates in the workspace", func(t *testing.T) {
			t.Parallel()
			root, pkgs := t.TempDir(), []workspace.Package{releasedA, releasedB}
			c, f := &commands{}, &crates{}
			p := release.Publisher{Client: &http.Client{Transport: f}, Command: c.run, Getenv: environment(nil)}
			assert.NoError(t, p.Publish(t.Context(), root, pkgs, packed(t, pkgs...)), "Publish")
			expect.Equal(t, c.calls, []call{{
				dir:  root,
				name: "cargo",
				args: []string{"publish", "--no-verify", "--package", "probe-a", "--package", "probe-b"},
			}}, "the runs of cargo")
			expect.Empty(t, f.requests, "the requests")
		})

		t.Run("gives cargo publish the token that the trusted publishing of crates.io exchanges", func(t *testing.T) {
			t.Parallel()
			root, pkgs := t.TempDir(), []workspace.Package{releasedB}
			c, f := &commands{}, &crates{answers: exchanged()}
			p := release.Publisher{Client: &http.Client{Transport: f}, Command: c.run, Getenv: environment(job)}
			assert.NoError(t, p.Publish(t.Context(), root, pkgs, packed(t, pkgs...)), "Publish")
			expect.Equal(t, c.calls, []call{{
				dir:  root,
				name: "cargo",
				args: []string{"publish", "--no-verify", "--package", "probe-b"},
				env:  []string{"CARGO_REGISTRY_TOKEN=" + cratesToken},
			}}, "the runs of cargo")
			agent := "ergon (https://github.com/dokimasia/ergon)"
			expect.Equal(t, f.requests, []request{
				{
					method:        http.MethodGet,
					url:           oidcURL + "&audience=crates.io",
					authorization: "Bearer " + requestToken,
					agent:         agent,
				},
				{
					method: http.MethodPost, url: "https://crates.io" + tokens, agent: agent,
					contentType: "application/json", body: `{"jwt":"` + jwt + `"}`,
				},
				{
					method:        http.MethodDelete,
					url:           "https://crates.io" + tokens,
					authorization: "Bearer " + cratesToken,
					agent:         agent,
				},
			}, "the requests")
		})

		t.Run("revokes the token after a cargo publish that fails", func(t *testing.T) {
			t.Parallel()
			pkgs := []workspace.Package{releasedB}
			c, f := &commands{do: func(call) error { return errProgram }}, &crates{answers: exchanged()}
			p := release.Publisher{Client: &http.Client{Transport: f}, Command: c.run, Getenv: environment(job)}
			err := p.Publish(t.Context(), t.TempDir(), pkgs, packed(t, pkgs...))
			assert.ErrorIs(t, err, errProgram, "Publish")
			assert.Length(t, f.requests, 3, "the requests")
			assert.Equal(t, f.requests[2].method, http.MethodDelete, "the method of the last request")
		})

		t.Run("runs no cargo publish for no crate", func(t *testing.T) {
			t.Parallel()
			c, f := &commands{}, &crates{answers: exchanged()}
			p := release.Publisher{Client: &http.Client{Transport: f}, Command: c.run, Getenv: environment(job)}
			assert.NoError(t, p.Publish(t.Context(), t.TempDir(), nil, t.TempDir()), "Publish")
			expect.Empty(t, c.calls, "the runs of cargo")
			expect.Empty(t, f.requests, "the requests")
		})

		t.Run("returns ErrCrate for a crate without its .crate file in the pack directory", func(t *testing.T) {
			t.Parallel()
			c, f := &commands{}, &crates{answers: exchanged()}
			p := release.Publisher{Client: &http.Client{Transport: f}, Command: c.run, Getenv: environment(job)}
			err := p.Publish(t.Context(), t.TempDir(), []workspace.Package{releasedA, releasedB}, packed(t, releasedA))
			assert.ErrorIs(t, err, release.ErrCrate, "Publish")
			expect.Empty(t, c.calls, "the runs of cargo")
			expect.Empty(t, f.requests, "the requests")
		})

		tokenErrors := []struct {
			name     string
			give     map[string]answer
			wantRuns int
		}{
			{
				name: "returns ErrToken for a status of GitHub outside 2xx",
				give: map[string]answer{oidcToken: {status: http.StatusForbidden}},
			},
			{
				name: "returns ErrToken for a response of GitHub without a token",
				give: map[string]answer{oidcToken: {status: http.StatusOK, body: `{"value": ""}`}},
			},
			{
				name: "returns ErrToken for a response of GitHub that does not decode",
				give: map[string]answer{oidcToken: {status: http.StatusOK, body: "{"}},
			},
			{
				name: "returns ErrToken for a status of crates.io outside 2xx",
				give: map[string]answer{
					oidcToken: exchanged()[oidcToken],
					exchange:  {status: http.StatusBadRequest, body: `{"errors":[{"detail":"no trusted publisher"}]}`},
				},
			},
			{
				name: "returns ErrToken for a response of crates.io without a token",
				give: map[string]answer{
					oidcToken: exchanged()[oidcToken],
					exchange:  {status: http.StatusOK, body: `{"token": ""}`},
				},
			},
			{
				name: "returns ErrToken for a revoke that fails",
				give: map[string]answer{
					oidcToken: exchanged()[oidcToken],
					exchange:  exchanged()[exchange],
					revoke:    {status: http.StatusInternalServerError},
				},
				wantRuns: 1,
			},
		}
		for _, tt := range tokenErrors {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				pkgs := []workspace.Package{releasedB}
				c, f := &commands{}, &crates{answers: tt.give}
				p := release.Publisher{Client: &http.Client{Transport: f}, Command: c.run, Getenv: environment(job)}
				err := p.Publish(t.Context(), t.TempDir(), pkgs, packed(t, pkgs...))
				assert.ErrorIs(t, err, release.ErrToken, "Publish")
				assert.Length(t, c.calls, tt.wantRuns, "the runs of cargo")
			})
		}

		t.Run("returns the detail of crates.io in the error of the exchange", func(t *testing.T) {
			t.Parallel()
			pkgs := []workspace.Package{releasedB}
			f := &crates{answers: map[string]answer{
				oidcToken: exchanged()[oidcToken],
				exchange:  {status: http.StatusBadRequest, body: `{"errors":[{"detail":"no trusted publisher"}]}`},
			}}
			p := release.Publisher{
				Client:  &http.Client{Transport: f},
				Command: (&commands{}).run,
				Getenv:  environment(job),
			}
			err := p.Publish(t.Context(), t.TempDir(), pkgs, packed(t, pkgs...))
			assert.HasError(t, err, "Publish")
			assert.Contains(t, err.Error(), "no trusted publisher", "the text of the error")
		})

		failures := []struct {
			name     string
			giveEnv  map[string]string
			giveFake *crates
		}{
			{
				name:     "returns the error of an address of the OIDC token that does not parse",
				giveEnv:  map[string]string{"ACTIONS_ID_TOKEN_REQUEST_URL": "%zz"},
				giveFake: &crates{},
			},
			{name: "returns the error of a request that fails", giveEnv: job, giveFake: &crates{err: errConnection}},
			{
				name:     "returns the error of a body that fails to read",
				giveEnv:  job,
				giveFake: &crates{bodyErr: errBody, answers: exchanged()},
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				pkgs := []workspace.Package{releasedB}
				c := &commands{}
				p := release.Publisher{
					Client:  &http.Client{Transport: tt.giveFake},
					Command: c.run,
					Getenv:  environment(tt.giveEnv),
				}
				err := p.Publish(t.Context(), t.TempDir(), pkgs, packed(t, pkgs...))
				assert.HasError(t, err, "Publish")
				assert.Empty(t, c.calls, "the runs of cargo")
			})
		}
	})
}
