// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Index is the sparse index of crates.io, whose file of a crate [Publisher.Published] reads.
const Index = "https://index.crates.io"

// Registry is crates.io, whose trusted publishing exchanges the OIDC token of a job for a token of
// the registry, and to which cargo publish uploads.
const Registry = "https://crates.io"

// The variables of the environment of a job of GitHub Actions with the permission id-token, from
// which a process requests the OIDC token of the job.
const (
	requestURLEnv   = "ACTIONS_ID_TOKEN_REQUEST_URL"
	requestTokenEnv = "ACTIONS_ID_TOKEN_REQUEST_TOKEN"
)

// audience is the audience of the OIDC token that crates.io exchanges, which is its host.
const audience = "crates.io"

// exchangePath is the path of the API of crates.io that exchanges an OIDC token for a token of
// trusted publishing, and that revokes such a token.
const exchangePath = "/api/v1/trusted_publishing/tokens"

// tokenEnv is the variable of the environment from which cargo reads the token of crates.io.
const tokenEnv = "CARGO_REGISTRY_TOKEN"

// userAgent identifies the requests of a Publisher, as crates.io asks of a client.
const userAgent = "ergon (https://github.com/dokimasia/ergon)"

// The command and the flag of cargo that upload the crates.
const (
	publishCommand = "publish"
	noVerifyFlag   = "--no-verify"
)

// jsonType is the media type of the body of the token exchange.
const jsonType = "application/json"

// The limits of the bytes of a response that a Publisher reads.
const (
	// maxIndexBody is the most bytes of the file of a crate in the index.
	maxIndexBody = 64 << 20

	// maxTokenBody is the most bytes of an answer of GitHub or of crates.io.
	maxTokenBody = 1 << 20

	// maxQuotedBody is the most bytes of an answer that the error of a failed exchange quotes.
	maxQuotedBody = 512
)

// ErrIndex is the error of [Publisher.Published] for a status of the index other than 200 and 404,
// and for a file of a crate that does not decode.
var ErrIndex = errors.New("release: unexpected answer of the index")

// ErrToken is the error of [Publisher.Publish] for a token exchange that fails: a status of GitHub
// or of crates.io outside 2xx, a response without a token or that does not decode, and a revoke
// that fails.
var ErrToken = errors.New("release: the token exchange failed")

// Publisher is the [language.Publisher] of the toolchain rust: it reads the versions of a crate
// from the sparse index of crates.io, and uploads the crates with cargo publish.
//
// # Concurrency
//
// A Publisher is safe for concurrent use when its Client, its Command and its Getenv are.
type Publisher struct {
	// Client sends the requests to the index, to GitHub and to crates.io. It must not be nil.
	Client *http.Client

	// Command runs the program name of the PATH with args in dir, with env added to the environment
	// of the process. Publish runs cargo publish through it. It must not be nil.
	Command func(ctx context.Context, dir, name string, args, env []string) error

	// Getenv returns the variable key of the environment of the process, as [os.Getenv] does.
	// Publish reads the variables of the OIDC token of a job of GitHub Actions through it. It must
	// not be nil.
	Getenv func(key string) string
}

var _ language.Publisher = Publisher{}

// Published reports whether crates.io has p at its Version. It reads the file of the crate in the
// sparse index of crates.io, which has a line of JSON for each version. It reports true for a
// version with the numbers and the pre-release of the Version of p, a yanked one included. It
// reports false for the status 404, which the index returns for a crate that crates.io does not
// have. Cargo lays the file of a crate out by its name in lower case:
//
//   - under 1/ or 2/ for a name of one or two characters
//   - under 3/ and its first character for a name of three characters
//   - under its first two and its next two characters for a longer name
//
// It returns an error that wraps [ErrIndex] for any other status and for a line that does not
// decode. It also returns the error of a name that is not valid in an address, of the request and
// of reading the body.
func (p Publisher) Published(ctx context.Context, pkg *workspace.Package) (bool, error) {
	name := strings.ToLower(pkg.Name)
	var dir string
	switch {
	case len(name) <= 2:
		dir = strconv.Itoa(len(name))
	case len(name) == 3:
		dir = "3/" + name[:1]
	default:
		dir = name[:2] + "/" + name[2:4]
	}
	address := Index + "/" + dir + "/" + name
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, http.NoBody)
	if err != nil {
		return false, fmt.Errorf("release: %s: %w", address, err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("release: %s: %w", address, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxIndexBody))
	if err != nil {
		return false, fmt.Errorf("release: read %s: %w", address, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("%w: %s returned %s", ErrIndex, address, resp.Status)
	}
	want := pkg.Version
	want.Build = ""
	lines := bufio.NewScanner(bytes.NewReader(data))
	lines.Buffer(nil, maxIndexBody)
	for lines.Scan() {
		var line struct {
			Version string `json:"vers"`
		}
		if err = json.Unmarshal(lines.Bytes(), &line); err != nil {
			return false, fmt.Errorf("%w: %s: %w", ErrIndex, address, err)
		}
		var v version.Version
		if v, err = version.Parse(line.Version); err != nil {
			return false, fmt.Errorf("%w: %s: %w", ErrIndex, address, err)
		}
		v.Build = ""
		if v == want {
			return true, nil
		}
	}
	return false, nil
}

// Publish uploads the crates of pkgs from the workspace at root with one run of cargo publish
// --no-verify, with a flag --package for each crate. cargo publish orders the crates and waits for
// each in the index before the crates that require it. It does not build the crates again: the job
// pack verified them with cargo package, and the job publish has the tokens of the release, which
// no build script may read. Publish first checks that dir has the .crate file of each crate, which
// [Packer.Pack] writes. It runs nothing for no crate.
//
// In a job of GitHub Actions with the permission id-token, Publish requests the OIDC token of the
// job for the audience crates.io and exchanges it with the trusted publishing of crates.io for a
// token, as rust-lang/crates-io-auth-action does. cargo publish reads that token as
// CARGO_REGISTRY_TOKEN, and Publish revokes it after the run. Elsewhere cargo publish authenticates
// with the token of its own environment or configuration.
//
// It returns an error that wraps [ErrCrate] for a crate without its .crate file in dir, before the
// upload, and an error that wraps [ErrToken] for a token exchange or a revoke that fails. It also
// returns the errors of the requests and of cargo publish, with the error of a revoke after a
// cargo publish that fails.
func (p Publisher) Publish(ctx context.Context, root string, pkgs []workspace.Package, dir string) error {
	if len(pkgs) == 0 {
		return nil
	}
	args := []string{publishCommand, noVerifyFlag}
	for i := range pkgs {
		name := pkgs[i].Name + "-" + pkgs[i].Version.String() + crateSuffix
		if _, err := os.Stat(filepath.Join(dir, DistDir, pkgs[i].Name, name)); err != nil {
			return fmt.Errorf("%w: the pack directory has no %s: %w", ErrCrate, name, err)
		}
		args = append(args, packageFlag, pkgs[i].Name)
	}
	var env []string
	token := ""
	if p.Getenv(requestURLEnv) != "" {
		var err error
		if token, err = p.token(ctx); err != nil {
			return err
		}
		env = []string{tokenEnv + "=" + token}
	}
	err := p.Command(ctx, root, cargoProgram, args, env)
	if err != nil {
		err = fmt.Errorf("release: publish the crates: %w", err)
	}
	if token != "" {
		// The token is valid for 30 minutes, so the revoke runs also after the end of ctx.
		err = errors.Join(err, p.send(context.WithoutCancel(ctx), http.MethodDelete, Registry+exchangePath,
			"the revoke of the token of crates.io", token, nil, nil))
	}
	return err
}

// token returns a token of crates.io for the OIDC token of the job: the OIDC token for the audience
// crates.io, from the address of the environment with the request token of the environment, and
// then the token of crates.io that the trusted publishing of crates.io exchanges it for. It returns
// an error that wraps [ErrToken] for a response without a token, and the errors of [Publisher.send].
func (p Publisher) token(ctx context.Context) (string, error) {
	// The address of the request has a query, which the audience joins.
	address := p.Getenv(requestURLEnv) + "&audience=" + url.QueryEscape(audience)
	var oidc struct {
		Value string `json:"value"`
	}
	if err := p.send(ctx, http.MethodGet, address, "the OIDC token of the job", p.Getenv(requestTokenEnv), nil,
		&oidc); err != nil {
		return "", err
	}
	if oidc.Value == "" {
		return "", fmt.Errorf("%w: GitHub returned no OIDC token", ErrToken)
	}
	var exchanged struct {
		Token string `json:"token"`
	}
	in := struct {
		JWT string `json:"jwt"`
	}{JWT: oidc.Value}
	if err := p.send(ctx, http.MethodPost, Registry+exchangePath, "the token exchange of crates.io", "", in,
		&exchanged); err != nil {
		return "", err
	}
	if exchanged.Token == "" {
		return "", fmt.Errorf("%w: crates.io returned no token", ErrToken)
	}
	return exchanged.Token, nil
}

// send sends a request of method to address, and decodes the JSON of the response into out when out
// is not nil. The request has the bearer token auth when auth is not empty, and the JSON of in as
// its body when in is not nil. The errors of send include what, the description of the request.
//
// send returns an error that wraps [ErrToken] with the start of the body of a response for a status
// outside 2xx. It also returns one for a body that does not decode, and the error of the request and
// of reading the body.
func (p Publisher) send(ctx context.Context, method, address, what, auth string, in, out any) error {
	var body []byte
	var encode error
	if in != nil {
		body, encode = json.Marshal(in)
	}
	req, err := http.NewRequestWithContext(ctx, method, address, bytes.NewReader(body))
	if err = errors.Join(encode, err); err != nil {
		return fmt.Errorf("release: %s: %w", what, err)
	}
	req.Header.Set("User-Agent", userAgent)
	if auth != "" {
		req.Header.Set("Authorization", "Bearer "+auth)
	}
	if in != nil {
		req.Header.Set("Content-Type", jsonType)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("release: %s: %w", what, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxTokenBody))
	if err != nil {
		return fmt.Errorf("release: read %s: %w", what, err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: %s returned %s: %s", ErrToken, what, resp.Status,
			strings.TrimSpace(string(data[:min(len(data), maxQuotedBody)])))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%w: decode %s: %w", ErrToken, what, err)
	}
	return nil
}
