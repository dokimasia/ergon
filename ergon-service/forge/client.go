// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// The addresses of github.com, which a [Config] without addresses takes.
const (
	// DefaultAPI is the address of the REST API of github.com.
	DefaultAPI = "https://api.github.com"

	// DefaultGraphQL is the address of the GraphQL API of github.com.
	DefaultGraphQL = "https://api.github.com/graphql"

	// DefaultServer is the address of github.com.
	DefaultServer = "https://github.com"
)

// The headers of a request.
const (
	// apiVersion is the version of the REST API of the requests.
	apiVersion = "2022-11-28"

	// mediaType is the media type of the responses that a request accepts.
	mediaType = "application/vnd.github+json"

	// jsonType is the media type of the body of a request of the API.
	jsonType = "application/json"

	// userAgent names ergon in each request, which GitHub requires.
	userAgent = "ergon"
)

// maxResponse is the largest response that a Client reads, in bytes.
const maxResponse = 10 << 20

// ErrToken is the error of [New] for a configuration without a token.
var ErrToken = errors.New("forge: no token")

// ErrGitHub is the error for a request that GitHub refuses or that fails.
var ErrGitHub = errors.New("forge: GitHub failed")

// Config is the access to a repository host: the token and the addresses, as GitHub Actions states
// them in GITHUB_TOKEN, GITHUB_API_URL, GITHUB_GRAPHQL_URL and GITHUB_SERVER_URL.
type Config struct {
	// Token authorizes every request.
	Token string

	// API is the address of the REST API, or empty for [DefaultAPI].
	API string

	// GraphQL is the address of the GraphQL API, or empty for [DefaultGraphQL].
	GraphQL string

	// Server is the address of the web pages of the host, or empty for [DefaultServer].
	Server string
}

// Client is the client of the GitHub API of one repository host.
//
// # Concurrency
//
// A Client is safe for concurrent use, as its [http.Client] is.
type Client struct {
	// http sends the requests.
	http *http.Client

	// config is the configuration, with the addresses without a slash at the end.
	config Config
}

// New returns a client of the host that c configures, which sends its requests through h. It
// returns an error that wraps [ErrToken] for a c without a token.
func New(h *http.Client, c Config) (*Client, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("%w: set GITHUB_TOKEN", ErrToken)
	}
	for _, field := range []struct {
		address  *string
		fallback string
	}{{&c.API, DefaultAPI}, {&c.GraphQL, DefaultGraphQL}, {&c.Server, DefaultServer}} {
		if *field.address == "" {
			*field.address = field.fallback
		}
		*field.address = strings.TrimRight(*field.address, "/")
	}
	return &Client{http: h, config: c}, nil
}

// Server returns the address of the web pages of the host, without a slash at the end.
func (c *Client) Server() string {
	return c.config.Server
}

// rest sends a request of method to path of the REST API, with in as its JSON body when in is not
// nil, and decodes the JSON of a response into out when out is not nil. It returns the status of
// the response. It returns an error that wraps [ErrGitHub] for a status outside 2xx, except for
// the statuses of allowed, with the message of GitHub, and the error of the request and of the
// decode.
func (c *Client) rest(ctx context.Context, method, path string, in, out any, allowed ...int) (int, error) {
	if in == nil {
		return c.send(ctx, method, c.config.API+path, nil, "", out, allowed)
	}
	// Every body of the package is a struct of strings, numbers and bools, which encodes.
	data, _ := json.Marshal(in)
	return c.send(ctx, method, c.config.API+path, bytes.NewReader(data), jsonType, out, allowed)
}

// graphql sends query with variables to the GraphQL API, and decodes the data of the response into
// out. It returns an error that wraps [ErrGitHub] for a response with errors, and the error of
// rest.
func (c *Client) graphql(ctx context.Context, query string, variables map[string]any, out any) error {
	// The variables of every query of the package are strings, numbers and lists of objects of
	// strings, which encode.
	data, _ := json.Marshal(map[string]any{"query": query, "variables": variables})
	var response struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	_, err := c.send(ctx, http.MethodPost, c.config.GraphQL, bytes.NewReader(data), jsonType, &response, nil)
	if err != nil {
		return err
	}
	if len(response.Errors) > 0 {
		messages := make([]string, 0, len(response.Errors))
		for _, e := range response.Errors {
			messages = append(messages, e.Message)
		}
		return fmt.Errorf("%w: GraphQL: %s", ErrGitHub, strings.Join(messages, "; "))
	}
	if err := json.Unmarshal(response.Data, out); err != nil {
		return fmt.Errorf("%w: decode the data of a query: %w", ErrGitHub, err)
	}
	return nil
}

// send sends a request of method to address with body of the media type contentType, and decodes
// the JSON of a response into out when out is not nil, as rest states. A body is nil or a
// *bytes.Reader, whose length the request states.
func (c *Client) send(
	ctx context.Context, method, address string, body io.Reader, contentType string, out any, allowed []int,
) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, address, body)
	if err != nil {
		return 0, fmt.Errorf("forge: %s %s: %w", method, address, err)
	}
	req.Header.Set("Accept", mediaType)
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("User-Agent", userAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %s %s: %w", ErrGitHub, method, address, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("%w: read %s %s: %w", ErrGitHub, method, address, err)
	}
	ok := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	if !ok {
		if slices.Contains(allowed, resp.StatusCode) {
			return resp.StatusCode, nil
		}
		var refusal struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &refusal)
		return resp.StatusCode, fmt.Errorf(
			"%w: %s %s: %s: %s",
			ErrGitHub,
			method,
			address,
			resp.Status,
			refusal.Message,
		)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, fmt.Errorf("%w: decode %s %s: %w", ErrGitHub, method, address, err)
		}
	}
	return resp.StatusCode, nil
}
