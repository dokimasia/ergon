// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Registry is the address of the npm registry, whose API [Publisher.Published] reads, and to which
// npm publish uploads.
const Registry = "https://registry.npmjs.org"

// maxRegistryBody is the number of bytes of the body of a response of the registry that
// [Publisher.Published] reads before it closes the body, so the client can reuse the connection.
const maxRegistryBody = 1 << 20

// npmTool is the tool of the section js that uploads the tarballs.
const npmTool = "npm"

// escapedSlash is the / of the name of a package with a scope in an address of the registry.
const escapedSlash = "%2f"

// nextTag is the dist-tag of a pre-release whose first identifier npm refuses as a dist-tag.
const nextTag = "next"

// Publisher is the [language.Publisher] of the js toolchain: it reads the versions of a package
// from the npm registry, and uploads the tarball of a package with npm publish.
//
// # Concurrency
//
// A Publisher is safe for concurrent use when its Client and its Tools are.
type Publisher struct {
	// Client sends the requests to the npm registry. Published needs it.
	Client *http.Client

	// Tools runs the tool of the section js of .ergon.yaml with args in dir, with env added to the
	// environment of the process, as ergon tool run does. Publish runs npm publish through it.
	Tools func(ctx context.Context, dir, tool string, args, env []string) error
}

var _ language.Publisher = Publisher{}

// Published reports whether the npm registry has p at its Version, without build metadata, from
// the status of GET https://registry.npmjs.org/<name>/<version>, with the / of a scope as %2f:
// true for 200, and false for 404.
//
// It returns an error that wraps [ErrRegistry] for any other status. It also returns the error of a
// name that is not valid in an address, of the request and of reading the body.
func (p Publisher) Published(ctx context.Context, pkg *workspace.Package) (bool, error) {
	v := version.Version{
		Major: pkg.Version.Major,
		Minor: pkg.Version.Minor,
		Patch: pkg.Version.Patch,
		Pre:   pkg.Version.Pre,
	}
	address := Registry + "/" + strings.Replace(pkg.Name, "/", escapedSlash, 1) + "/" + v.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, http.NoBody)
	if err != nil {
		return false, fmt.Errorf("release: %s: %w", address, err)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("release: %s: %w", address, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.Copy(io.Discard, io.LimitReader(resp.Body, maxRegistryBody)); err != nil {
		return false, fmt.Errorf("release: read %s: %w", address, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("%w: %s returned %s", ErrRegistry, address, resp.Status)
	}
}

// Publish uploads the tarball of each package of pkgs from dir/js/<name>/ to the npm registry with
// npm publish of the section js in dir, one package at a time in the order of pkgs. It does not
// read the repository at root, as a tarball has every file of its package. A pre-release
// gets the dist-tag of the first identifier of its pre-release, such as beta for 2.0.0-beta.1. npm
// refuses a dist-tag that is a range, such as 0, and such a pre-release gets the dist-tag next. In
// a job of GitHub Actions with the permission id-token, npm publish authenticates through the
// trusted publishing of npm, and elsewhere with the token of its configuration.
//
// It returns an error that wraps [ErrTarball] for a package without its tarball in dir, before its
// upload, and the error of npm publish.
func (p Publisher) Publish(ctx context.Context, _ string, pkgs []workspace.Package, dir string) error {
	for i := range pkgs {
		file, err := tarball(dir, &pkgs[i])
		if err != nil {
			return err
		}
		args := []string{"publish", file}
		if pre := pkgs[i].Version.Pre; pre != "" {
			tag, _, _ := strings.Cut(pre, ".")
			if _, isRange := parseRange(tag); isRange {
				tag = nextTag
			}
			args = append(args, "--tag", tag)
		}
		if err := p.Tools(ctx, dir, npmTool, args, nil); err != nil {
			return fmt.Errorf("release: publish %s: %w", pkgs[i].Name, err)
		}
	}
	return nil
}
