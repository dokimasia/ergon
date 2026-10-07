// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// The repository and the token of the cases.
const (
	repo  = "dokimasia/ergon"
	token = "secret"
)

// graphqlPath is the path of the GraphQL API of the fake GitHub.
const graphqlPath = "/graphql"

// request is a request that the fake GitHub received.
type request struct {
	// header are the headers of the request.
	header http.Header

	// method is the method of the request.
	method string

	// target is the path and the query of the request.
	target string

	// body is the body of the request.
	body string
}

// response is the response of a route of the fake GitHub.
type response struct {
	// body is the body of the response.
	body string

	// status is the status of the response, or 0 for 200.
	status int
}

// gitHub is a fake GitHub that answers each request with the next response of its route, and
// records the requests.
type gitHub struct {
	// routes are the responses of each method and target, such as GET /repos/a/b/pulls, in the
	// order of the requests.
	routes map[string][]response

	// requests are the requests that the fake received, in their order.
	requests []request

	// mu guards routes and requests.
	mu sync.Mutex
}

// ServeHTTP records r and writes the next response of its route, or 404 for a route without one.
func (g *gitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	g.mu.Lock()
	g.requests = append(g.requests, request{
		header: r.Header, method: r.Method, target: r.URL.RequestURI(),
		body: string(body),
	})
	key := r.Method + " " + r.URL.RequestURI()
	next := response{status: http.StatusNotFound, body: `{"message":"Not Found"}`}
	if queue := g.routes[key]; len(queue) > 0 {
		next, g.routes[key] = queue[0], queue[1:]
	}
	g.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if next.status != 0 {
		w.WriteHeader(next.status)
	}
	_, _ = io.WriteString(w, next.body)
}

// recorded returns the requests that g received.
func (g *gitHub) recorded() []request {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]request(nil), g.requests...)
}

// serve returns a client of a fake GitHub with routes, and the fake, for the test tb. Each key of
// routes is a method, a space and a target.
func serve(tb testing.TB, routes map[string][]response) (*forge.Client, *gitHub) {
	tb.Helper()
	fake := &gitHub{routes: routes}
	srv := httptest.NewServer(fake)
	tb.Cleanup(srv.Close)
	c, err := forge.New(srv.Client(), forge.Config{Token: token, API: srv.URL + "/", GraphQL: srv.URL + graphqlPath})
	assert.NoError(tb, err, "New")
	return c, fake
}

func TestClient(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrToken for a configuration without a token", func(t *testing.T) {
			t.Parallel()
			_, err := forge.New(&http.Client{}, forge.Config{})
			assert.ErrorIs(t, err, forge.ErrToken, "New")
		})
	})

	t.Run("Server", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the address of github.com for a configuration without one", func(t *testing.T) {
			t.Parallel()
			c, err := forge.New(&http.Client{}, forge.Config{Token: token})
			assert.NoError(t, err, "New")
			assert.Equal(t, c.Server(), forge.DefaultServer, "the server")
		})

		t.Run("returns the address of the configuration without a slash at the end", func(t *testing.T) {
			t.Parallel()
			c, err := forge.New(&http.Client{}, forge.Config{Token: token, Server: "https://git.example.com/"})
			assert.NoError(t, err, "New")
			assert.Equal(t, c.Server(), "https://git.example.com", "the server")
		})
	})

	t.Run("Branch", func(t *testing.T) {
		t.Parallel()

		t.Run("sends the headers that GitHub requires", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, nil)
			_, _, err := c.Branch(t.Context(), repo, "main")
			assert.NoError(t, err, "Branch")
			header := fake.recorded()[0].header
			assert.Equal(t, header.Get("Authorization"), "Bearer "+token, "Authorization")
			assert.Equal(t, header.Get("X-GitHub-Api-Version"), "2022-11-28", "X-GitHub-Api-Version")
			assert.Equal(t, header.Get("Accept"), "application/vnd.github+json", "Accept")
			assert.Equal(t, header.Get("User-Agent"), "ergon", "User-Agent")
		})

		t.Run("returns ErrGitHub with the message of GitHub for a request that it refuses", func(t *testing.T) {
			t.Parallel()
			refused := response{status: http.StatusUnauthorized, body: `{"message":"Bad credentials"}`}
			c, _ := serve(t, map[string][]response{"GET /repos/" + repo + "/git/ref/heads/main": {refused}})
			_, _, err := c.Branch(t.Context(), repo, "main")
			assert.ErrorIs(t, err, forge.ErrGitHub, "Branch")
			assert.Contains(t, err.Error(), "401 Unauthorized: Bad credentials", "the error")
		})

		t.Run("returns ErrGitHub for a response that is no JSON", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{"GET /repos/" + repo + "/git/ref/heads/main": {{body: "<html>"}}})
			_, _, err := c.Branch(t.Context(), repo, "main")
			assert.ErrorIs(t, err, forge.ErrGitHub, "Branch")
		})

		t.Run("returns ErrGitHub for a response shorter than its length", func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", strconv.Itoa(100))
				_, _ = io.WriteString(w, "{")
			}))
			t.Cleanup(srv.Close)
			c, err := forge.New(srv.Client(), forge.Config{Token: token, API: srv.URL})
			assert.NoError(t, err, "New")
			_, _, err = c.Branch(t.Context(), repo, "main")
			assert.ErrorIs(t, err, forge.ErrGitHub, "Branch")
			assert.Contains(t, err.Error(), "read GET", "the error")
		})

		t.Run("returns ErrGitHub for a host that does not answer", func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.NotFoundHandler())
			srv.Close()
			c, err := forge.New(srv.Client(), forge.Config{Token: token, API: srv.URL})
			assert.NoError(t, err, "New")
			_, _, err = c.Branch(t.Context(), repo, "main")
			assert.ErrorIs(t, err, forge.ErrGitHub, "Branch")
		})

		t.Run("returns the error of an address that is no URL", func(t *testing.T) {
			t.Parallel()
			c, err := forge.New(&http.Client{}, forge.Config{Token: token, API: "http://bad\x7fhost"})
			assert.NoError(t, err, "New")
			_, _, err = c.Branch(t.Context(), repo, "main")
			assert.HasError(t, err, "Branch")
			assert.ErrorIsNot(t, err, forge.ErrGitHub, "the class of the error")
		})
	})

	t.Run("CommitLinks", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrGitHub with the messages of the errors of a query", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{
				"POST " + graphqlPath: {{body: `{"errors":[{"message":"first"},{"message":"second"}]}`}},
			})
			_, _, _, err := c.CommitLinks(t.Context(), repo, "a085003")
			assert.ErrorIs(t, err, forge.ErrGitHub, "CommitLinks")
			assert.Contains(t, err.Error(), "GraphQL: first; second", "the error")
		})

		t.Run("returns ErrGitHub for data that does not decode", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{"POST " + graphqlPath: {{body: `{"data":"text"}`}}})
			_, _, _, err := c.CommitLinks(t.Context(), repo, "a085003")
			assert.ErrorIs(t, err, forge.ErrGitHub, "CommitLinks")
			assert.Contains(t, err.Error(), "decode the data of a query", "the error")
		})

		t.Run("returns the error of a request that fails", func(t *testing.T) {
			t.Parallel()
			c, _ := serve(t, map[string][]response{"POST " + graphqlPath: {{status: http.StatusBadGateway}}})
			_, _, _, err := c.CommitLinks(t.Context(), repo, "a085003")
			assert.ErrorIs(t, err, forge.ErrGitHub, "CommitLinks")
		})
	})
}
