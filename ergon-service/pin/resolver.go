// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"path"
	"reflect"
	"slices"
	"strings"
	"time"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/forge"
)

// The errors of [Resolver.Resolve].
var (
	// ErrRegistry is the error for a registry that fails, or that responds with a document that the
	// resolver cannot read.
	ErrRegistry = errors.New("pin: the registry failed")

	// ErrAsset is the error for a release that lacks the asset of a platform of its pin.
	ErrAsset = errors.New("pin: the release lacks an asset of a platform of the pin")
)

// maxDocument is the largest document of a registry that a Resolver reads, 64 MiB. The document of
// the npm package typescript had 15,770,879 bytes on 2026-10-08.
const maxDocument = 67108864

// userAgent identifies the requests of a Resolver, as crates.io asks of a client.
const userAgent = "ergon (https://github.com/dokimasia/ergon)"

// The addresses of the public registries, which [PublicRegistries] returns.
const (
	publicGoProxy    = "https://proxy.golang.org"
	publicPyPI       = "https://pypi.org"
	publicNPM        = "https://registry.npmjs.org"
	publicCrates     = "https://index.crates.io"
	publicMaven      = "https://repo1.maven.org/maven2"
	publicPackagist  = "https://repo.packagist.org"
	publicChocolatey = "https://community.chocolatey.org/api/v2"
)

// GitHub is the API of GitHub that a [Resolver] reads. *forge.Client implements it.
type GitHub interface {
	// Releases returns the published releases of repo, as owner/name.
	Releases(ctx context.Context, repo string) ([]forge.Release, error)

	// Tag returns the commit of the tag name of repo, through the tag object of an annotated tag,
	// and reports whether repo has the tag.
	Tag(ctx context.Context, repo, name string) (string, bool, error)
}

var _ GitHub = (*forge.Client)(nil)

// Registries are the addresses of the registries of a [Resolver], each without a slash at the end.
type Registries struct {
	// GoProxy is a module proxy of the GOPROXY protocol, such as https://proxy.golang.org.
	GoProxy string

	// PyPI is the host of the JSON API of PyPI, such as https://pypi.org.
	PyPI string

	// NPM is the npm registry, such as https://registry.npmjs.org.
	NPM string

	// Crates is the sparse index of crates.io, such as https://index.crates.io.
	Crates string

	// Maven is a Maven repository, such as https://repo1.maven.org/maven2.
	Maven string

	// Packagist is the repository of Composer, such as https://repo.packagist.org.
	Packagist string

	// Chocolatey is the OData API of the Chocolatey community repository, such as
	// https://community.chocolatey.org/api/v2.
	Chocolatey string
}

// PublicRegistries returns the addresses of the public registries.
func PublicRegistries() Registries {
	return Registries{
		GoProxy:    publicGoProxy,
		PyPI:       publicPyPI,
		NPM:        publicNPM,
		Crates:     publicCrates,
		Maven:      publicMaven,
		Packagist:  publicPackagist,
		Chocolatey: publicChocolatey,
	}
}

// Release is a release of a pin in its registry.
type Release struct {
	// Published is the time at which the registry published the release, in UTC.
	Published time.Time

	// Digests are the SHA-256 digests of the assets of a release binary, by platform, in lowercase
	// hexadecimal, and nil for every other kind.
	Digests map[option.Platform]string

	// Version is the version of the release, in the form of the version of the pin.
	Version string

	// Commit is the commit of the tag of an action, and empty for every other kind.
	Commit string
}

// Resolution is what a [Resolver] finds for a pin.
type Resolution struct {
	// Next is the newest release that an update of the pin takes, and nil when the pin is at it or
	// no release qualifies.
	Next *Release

	// Major is the newest stable release of a later major version, which the update leaves to a
	// person, and nil when the resolver takes later major versions or there is none.
	Major *Release
}

// candidate is a release of a pin that a registry lists.
type candidate struct {
	// published is the time at which the registry published the release.
	published time.Time

	// release is the GitHub release of a release binary, an action or a version of GitHub, and nil
	// for every other kind.
	release *forge.Release

	// version is the version as the registry states it, such as the tag v7.1.0.
	version string

	// rank is the rank of the version among the versions of the project.
	rank rank
}

// Resolver resolves the newest releases of pins. Each field is required, and Exempt may be empty.
//
// # Concurrency
//
// A Resolver is safe for concurrent use when its Client, its GitHub and its Now are.
type Resolver struct {
	// Client reads the registries other than GitHub, and downloads an asset without a digest.
	Client *http.Client

	// GitHub reads the releases of GitHub.
	GitHub GitHub

	// Now returns the current time, against which MinAge counts.
	Now func() time.Time

	// Registries are the addresses of the registries.
	Registries Registries

	// Exempt are the prefixes of the names whose releases qualify at once, such as
	// go.dokimi.dev/ergon/.
	Exempt []string

	// MinAge is the time since the publication of a release before an update takes it.
	MinAge time.Duration

	// Major lets an update take a release of a later major version.
	Major bool
}

// Resolve returns the newest release of p that is stable, newer than the version of p, published
// at least MinAge before Now, and of the major version of p, as [Resolution] states. A version is
// stable when it is one to four numbers of at most 19 digits separated by dots, with an optional
// leading v. Versions compare by their numbers, and a release ranks above a pre-release of the same
// numbers, so a pin at v2.0.0-rc.1 takes v2.0.0. A pin whose name starts with a prefix of Exempt takes a release at
// once. A Go module without tags takes the pseudo-version of its newest commit. The release of a
// release binary has the digest of the asset of each platform of p, which GitHub states or which
// Resolve computes from a download, and the release of an action has the commit of its tag.
//
// It returns an error that wraps [ErrRegistry] for a registry that fails, for a project that the
// registry does not have, and for a pin of no kind, and [ErrAsset] for a release binary whose new
// release lacks the asset of a platform of p.
func (r *Resolver) Resolve(ctx context.Context, p *Pin) (Resolution, error) {
	var candidates []candidate
	var err error
	switch p.Kind {
	case KindModule:
		candidates, err = r.module(ctx, p)
	case KindPyPI:
		candidates, err = r.pypi(ctx, p.Name)
	case KindNPM:
		candidates, err = r.npm(ctx, p.Name)
	case KindCrate:
		candidates, err = r.crate(ctx, p.Name)
	case KindMaven:
		candidates, err = r.maven(ctx, p)
	case KindComposer:
		candidates, err = r.packagist(ctx, p.Name)
	case KindChocolatey:
		candidates, err = r.chocolatey(ctx, p.Name)
	case KindBinary, KindAction, KindGitHub:
		candidates, err = r.github(ctx, p.Name)
	default:
		err = fmt.Errorf("%w: the kind %d", ErrRegistry, p.Kind)
	}
	if err != nil {
		return Resolution{}, fmt.Errorf("pin: resolve %s: %w", p.Key, err)
	}
	current := order(p.Version)
	exempt := slices.ContainsFunc(r.Exempt, func(prefix string) bool { return strings.HasPrefix(p.Name, prefix) })
	now := r.Now()
	var next, major *candidate
	for k := range candidates {
		c := &candidates[k]
		if !c.rank.above(current) || (!exempt && now.Sub(c.published) < r.MinAge) {
			continue
		}
		if r.Major || c.rank.numbers[0] == current.numbers[0] {
			if next == nil || c.rank.above(next.rank) {
				next = c
			}
		} else if major == nil || c.rank.above(major.rank) {
			major = c
		}
	}
	var res Resolution
	if next != nil {
		release, err := r.release(ctx, p, next)
		if err != nil {
			return Resolution{}, fmt.Errorf("pin: resolve %s: %w", p.Key, err)
		}
		res.Next = &release
	}
	if major != nil {
		res.Major = &Release{Published: major.published.UTC(), Version: like(p.Version, major.version)}
	}
	return res, nil
}

// release returns the release of the candidate c of p: its time and its version, with the digests
// of a release binary and the commit of an action. It returns an error that wraps [ErrAsset] for a
// release binary without the asset of a platform of p, and [ErrRegistry] for a tag that GitHub
// does not return.
func (r *Resolver) release(ctx context.Context, p *Pin, c *candidate) (Release, error) {
	release := Release{Published: c.published.UTC(), Version: like(p.Version, c.version)}
	if p.Kind == KindBinary {
		digests, err := r.digests(ctx, p, c, release.Version)
		release.Digests = digests
		return release, err
	}
	if p.Kind == KindAction {
		commit, ok, err := r.GitHub.Tag(ctx, p.Name, c.version)
		if err != nil {
			return Release{}, fmt.Errorf("%w: %w", ErrRegistry, err)
		}
		if !ok {
			return Release{}, fmt.Errorf("%w: %s has no tag %s", ErrRegistry, p.Name, c.version)
		}
		release.Commit = commit
	}
	return release, nil
}

// digests returns the digest of the asset of each platform of the release binary p at the version
// of the candidate c: the asset that the type of p names for the version, which GitHub states the
// digest of, or which digests downloads and hashes. It returns an error that wraps [ErrAsset] for a
// value of p that is no struct that implements [option.Release] and embeds [option.Binary], for a
// platform without an asset, and the error of a download.
func (r *Resolver) digests(ctx context.Context, p *Pin, c *candidate, version string) (
	map[option.Platform]string, error,
) {
	earlier, ok := p.Value.(option.Release)
	t := reflect.TypeOf(p.Value)
	binaryType := reflect.TypeFor[option.Binary]()
	var binary, embedded reflect.Value
	if ok && t.Kind() == reflect.Struct {
		binary = reflect.New(t).Elem()
		binary.Set(reflect.ValueOf(p.Value))
		embedded = binary.FieldByName(binaryType.Name())
	}
	if !embedded.CanSet() || embedded.Type() != binaryType {
		return nil, fmt.Errorf("%w: %s is no release binary that embeds option.Binary", ErrAsset, p.Key)
	}
	pin := earlier.Pin()
	pin.Version = version
	embedded.Set(reflect.ValueOf(pin))
	next, _ := reflect.TypeAssert[option.Release](binary)
	digests := map[option.Platform]string{}
	for _, platform := range slices.Sorted(maps.Keys(earlier.Pin().SHA256)) {
		asset, err := next.Asset(platform)
		if err != nil {
			return nil, fmt.Errorf("%w: %s %s on %s: %w", ErrAsset, p.Key, version, platform, err)
		}
		name := path.Base(asset.URL)
		i := slices.IndexFunc(c.release.Assets, func(a forge.Asset) bool { return a.Name == name })
		if i < 0 {
			return nil, fmt.Errorf("%w: %s %s has no asset %s for %s", ErrAsset, p.Key, version, name, platform)
		}
		digest := c.release.Assets[i].Digest
		if digest == "" {
			if digest, err = r.download(ctx, c.release.Assets[i].URL); err != nil {
				return nil, err
			}
		}
		digests[platform] = digest
	}
	return digests, nil
}

// get returns the body of the response to a request of method to address, and its headers. It
// reads at most 64 MiB of the body, so the decoder of a longer document fails. It reports true for
// a response of the status 200 alone, and false with no error for a response of the status 404 or
// 410, which a registry returns for a project that it does not have. It returns an error that wraps
// [ErrRegistry] for a request that fails, and any other status.
func (r *Resolver) get(ctx context.Context, method, address string) ([]byte, http.Header, bool, error) {
	req, err := http.NewRequestWithContext(ctx, method, address, nil)
	if err != nil {
		return nil, nil, false, fmt.Errorf("%w: %s %s: %w", ErrRegistry, method, address, err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, nil, false, fmt.Errorf("%w: %s %s: %w", ErrRegistry, method, address, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, false, fmt.Errorf("%w: %s %s: %s", ErrRegistry, method, address, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDocument))
	if err != nil {
		return nil, nil, false, fmt.Errorf("%w: %s %s: %w", ErrRegistry, method, address, err)
	}
	return data, resp.Header, true, nil
}

// document returns the body of the response to a GET of address, as [Resolver.get] states. It
// returns an error that wraps [ErrRegistry] for a project that the registry does not have.
func (r *Resolver) document(ctx context.Context, address string) ([]byte, error) {
	data, _, found, err := r.get(ctx, http.MethodGet, address)
	if err == nil && !found {
		err = fmt.Errorf("%w: %s: the registry does not have it", ErrRegistry, address)
	}
	return data, err
}

// download returns the SHA-256 of the file at address in lowercase hexadecimal, which it hashes as
// it reads, without a bound of size. It returns an error that wraps [ErrRegistry] for a request
// that fails, and for any other status than 200.
func (r *Resolver) download(ctx context.Context, address string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", fmt.Errorf("%w: GET %s: %w", ErrRegistry, address, err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := r.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: GET %s: %w", ErrRegistry, address, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: GET %s: %s", ErrRegistry, address, resp.Status)
	}
	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", fmt.Errorf("%w: GET %s: %w", ErrRegistry, address, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
