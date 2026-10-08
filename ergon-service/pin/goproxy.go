// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"golang.org/x/mod/module"
)

// The paths of the GOPROXY protocol after the escaped path of a module.
const (
	latestPath = "/@latest"
	listPath   = "/@v/list"
	infoPath   = "/@v/"
	infoSuffix = ".info"
)

// info is the document of a version of a module in the GOPROXY protocol.
type info struct {
	// Time is the time of the commit of the version.
	Time time.Time `json:"Time"`

	// Version is the version, such as v2.14.0 or a pseudo-version.
	Version string `json:"Version"`
}

// module returns the releases of the Go module of the package path of p from the module proxy:
// each stable tag newer than the version of p with the time of its commit, and the pseudo-version
// of the newest commit of a module without tags. The module is the longest prefix of the package
// path whose @latest the proxy returns. It returns an error that wraps [ErrRegistry] for a package
// of no module, a request that fails, and a document that does not decode.
func (r *Resolver) module(ctx context.Context, p *Pin) ([]candidate, error) {
	escaped, latest, err := r.latest(ctx, p.Name)
	if err != nil {
		return nil, err
	}
	list, err := r.document(ctx, r.Registries.GoProxy+"/"+escaped+listPath)
	if err != nil {
		return nil, err
	}
	current := order(p.Version)
	var candidates []candidate
	for v := range strings.FieldsSeq(string(list)) {
		n := order(v)
		if !stableVersion.MatchString(v) || !n.above(current) {
			continue
		}
		// A stable version has no upper-case letter, which the protocol would escape.
		i, found, err := r.info(ctx, r.Registries.GoProxy+"/"+escaped+infoPath+v+infoSuffix)
		if err == nil && !found {
			err = fmt.Errorf("%w: %s lists %s without its document", ErrRegistry, p.Name, v)
		}
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate{published: i.Time, rank: n, version: v})
	}
	if module.IsPseudoVersion(latest.Version) {
		candidates = append(candidates, candidate{
			published: latest.Time, rank: order(latest.Version), version: latest.Version,
		})
	}
	return candidates, nil
}

// latest returns the escaped path of the module of the package path pkg, the longest prefix of pkg
// whose @latest the module proxy returns, and its @latest. It returns an error that wraps
// [ErrRegistry] for a package of no module, a request that fails, and a document that does not
// decode.
func (r *Resolver) latest(ctx context.Context, pkg string) (string, info, error) {
	for mod := pkg; strings.Contains(mod, "/"); mod = path.Dir(mod) {
		escaped, err := module.EscapePath(mod)
		if err != nil {
			return "", info{}, fmt.Errorf("%w: the module path %q: %w", ErrRegistry, mod, err)
		}
		i, found, err := r.info(ctx, r.Registries.GoProxy+"/"+escaped+latestPath)
		if err != nil {
			return "", info{}, err
		}
		if found {
			return escaped, i, nil
		}
	}
	return "", info{}, fmt.Errorf("%w: no module of the proxy has the package %s", ErrRegistry, pkg)
}

// info returns the document of a version at address, and reports false for a version or a module
// that the proxy does not have. It returns an error that wraps [ErrRegistry] for a request that
// fails, and a document that does not decode.
func (r *Resolver) info(ctx context.Context, address string) (info, bool, error) {
	data, _, found, err := r.get(ctx, http.MethodGet, address)
	if !found {
		return info{}, false, err
	}
	var i info
	if err := json.Unmarshal(data, &i); err != nil {
		return info{}, false, fmt.Errorf("%w: decode %s: %w", ErrRegistry, address, err)
	}
	return i, true, nil
}
