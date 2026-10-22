// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	pyworkspace "go.dokimi.dev/ergon/lang/python/workspace"
)

// PyPI is the address of the Python Package Index, whose JSON API [Publisher.Published] reads, and
// whose upload API uv publish uploads to.
const PyPI = "https://pypi.org"

// maxIndexBody is the number of bytes of the body of a response of the JSON API that
// [Publisher.Published] reads before it closes the body, so the client can reuse the connection.
const maxIndexBody = 1 << 20

// Publisher is the [language.Publisher] of the toolchain python: it reads the releases of a project
// from the JSON API of PyPI, and uploads the distributions of a project with uv publish.
//
// # Concurrency
//
// A Publisher is safe for concurrent use when its Client and its Tools are.
type Publisher struct {
	// Client sends the requests to the JSON API of PyPI. Published needs it.
	Client *http.Client

	// Tools runs the tool of the section python of .ergon.yaml with args in dir, with env added to
	// the environment of the process, as ergon tool run does. Publish runs uv publish through it.
	Tools func(ctx context.Context, dir, tool string, args, env []string) error
}

var _ language.Publisher = Publisher{}

// Published reports whether PyPI has the release of p at its Version, from the status of the JSON
// API of the release, GET https://pypi.org/pypi/<name>/<version>/json, with the version of PEP 440:
// true for 200, which a yanked release also returns, and false for 404.
//
// It returns an error that wraps [ErrIndex] for any other status. It also returns the error of a
// name that is not valid in an address, of the request and of reading the body. It returns an error
// that wraps [pyworkspace.ErrVersion] for a Version that PEP 440 does not have, before the request.
func (p Publisher) Published(ctx context.Context, pkg *workspace.Package) (bool, error) {
	v, err := pyworkspace.FromSemVer(pkg.Version)
	if err != nil {
		return false, fmt.Errorf("release: %s: %w", pkg.Name, err)
	}
	address := PyPI + "/pypi/" + pkg.Name + "/" + v.String() + "/json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, http.NoBody)
	if err != nil {
		return false, fmt.Errorf("release: %s: %w", address, err)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("release: %s: %w", address, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.Copy(io.Discard, io.LimitReader(resp.Body, maxIndexBody)); err != nil {
		return false, fmt.Errorf("release: read %s: %w", address, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("%w: %s returned %s", ErrIndex, address, resp.Status)
	}
}

// Publish uploads the distributions of each project of pkgs from dir/python/<name>/ to PyPI with uv
// publish in dir, one project at a time in the order of pkgs. It does not read the repository at
// root, as the distributions have every file of their project. uv publish authenticates with the
// trusted publishing of PyPI in a job of GitHub Actions with the permission id-token, and elsewhere
// with the credentials of its environment, such as UV_PUBLISH_TOKEN.
//
// It returns an error that wraps [ErrDistribution] for a project without distributions of its
// Version in dir, before its upload, and the error of uv publish.
func (p Publisher) Publish(ctx context.Context, _ string, pkgs []workspace.Package, dir string) error {
	for i := range pkgs {
		files, err := distributions(filepath.Join(dir, DistDir, pkgs[i].Name), &pkgs[i])
		if err != nil {
			return err
		}
		if err := p.Tools(ctx, dir, uvTool, slices.Concat([]string{"publish"}, files), nil); err != nil {
			return fmt.Errorf("release: publish %s: %w", pkgs[i].Name, err)
		}
	}
	return nil
}
