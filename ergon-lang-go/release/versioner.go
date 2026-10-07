// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"golang.org/x/mod/module"
)

// ErrRequirement is the error for a requirement that is no require line of go.mod: v and a version
// of Semantic Versioning 2.0.0.
var ErrRequirement = errors.New("release: invalid require line")

// ErrMajor is the error for a release whose major version the module path does not state.
var ErrMajor = errors.New("release: major version without its module path")

// ErrCycle is the error for modules that require each other without a directory replace, whose
// go.sum files have no fixed point.
var ErrCycle = errors.New("release: modules that require each other")

// Versioner is the [language.Versioner] of the toolchain go.
//
// # Concurrency
//
// A Versioner is safe for concurrent use when its Snapshot is. Two calls of Apply must not write
// one repository at once.
type Versioner struct {
	// Snapshot returns the tree of the working tree of a directory as git would commit it, which
	// Apply writes the zips of the released modules from. It must not be nil.
	Snapshot func(ctx context.Context, dir string) (string, error)
}

var _ language.Versioner = Versioner{}

// Resolve returns how a consumer that resolves the require line req alone treats v. A require line
// names the lowest version of the module that it admits, and minimal version selection gives a
// consumer that version: Resolve returns [language.ResolutionExcluded] for a v below it,
// [language.ResolutionSelected] for it, and [language.ResolutionPinned] for a higher v. It returns
// an error that wraps [ErrRequirement] for a req that is no require line.
func (Versioner) Resolve(req string, v version.Version) (language.Resolution, error) {
	named, err := requirement(req)
	if err != nil {
		return 0, err
	}
	switch c := v.Compare(named); {
	case c < 0:
		return language.ResolutionExcluded, nil
	case c == 0:
		return language.ResolutionSelected, nil
	}
	return language.ResolutionPinned, nil
}

// Rewrite returns the require line of v: v with a v before it, such as v1.3.0. It returns an error
// that wraps [ErrRequirement] for a req that is no require line.
func (Versioner) Rewrite(req string, v version.Version) (string, error) {
	if _, err := requirement(req); err != nil {
		return "", err
	}
	return "v" + v.String(), nil
}

// Validate returns an error that wraps [ErrMajor] for a v whose major version the module path of p
// does not state, as the go command refuses it: the path of a module ends in /vN for a major
// version N of 2 and above, and in no such element for 0 and 1. It returns nil for any other
// release.
func (Versioner) Validate(p *workspace.Package, v version.Version) error {
	_, major, _ := module.SplitPathVersion(p.Name)
	if err := module.CheckPathMajor("v"+v.String(), major); err != nil {
		return fmt.Errorf("%w: %s at %s: %w", ErrMajor, p.Name, v, err)
	}
	return nil
}

// Apply writes the rewritten requirements of edits into the go.mod of their modules, and refreshes
// the go.sum of each module that requires a released module without a directory replace, as the
// package documentation states. Each edit names a module of the repository, and its Version is
// the new version of the module, which a tag records: go.mod has no version field. Apply returns
// the go.mod and go.sum files whose content it changed, relative to root and slash-separated, in
// the order of the first change. It runs go mod tidy with GOWORK=off, GOFLAGS=-mod=mod
// -modcacherw, a module cache in a temporary directory, a GOPROXY that tries the proxy of the
// released modules and the download cache of the module cache of the go command before the
// configured GOPROXY, and a GONOSUMDB that adds the module paths of the repository.
//
// It returns an error that wraps [ErrCycle] for modules that require each other without a
// directory replace, with their paths, and an error for an edit of a module that the repository
// does not have. It returns the error of the go command with its output, the error of git, and the
// error of the file system. On an error it also returns every go.mod and go.sum that it began to
// change.
func (v Versioner) Apply(ctx context.Context, root string, edits []language.Edit) ([]string, error) {
	r, err := newRun(v, root, edits)
	if err != nil {
		return nil, err
	}
	err = r.release(ctx, edits)
	if err != nil {
		return r.files.touched, err
	}
	return r.files.changed(), nil
}

// requirement returns the version that the require line req names. It returns an error that wraps
// [ErrRequirement] for a req that is no v and a version.
func requirement(req string) (version.Version, error) {
	rest, ok := strings.CutPrefix(req, "v")
	named, err := version.Parse(rest)
	if !ok || err != nil {
		return version.Version{}, fmt.Errorf("%w: %q", ErrRequirement, req)
	}
	return named, nil
}
