// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
)

// The toolchains of the cases.
const (
	// npmToolchain is the toolchain of the packages of the cases that changesets tests.
	npmToolchain workspace.Toolchain = "npm"

	// goToolchain is the toolchain of the packages whose requirements are require lines of Go.
	goToolchain workspace.Toolchain = "go"

	// pythonToolchain is a second toolchain, for names that two toolchains share.
	pythonToolchain workspace.Toolchain = "python"

	// bareToolchain is a toolchain that registers no versioner.
	bareToolchain workspace.Toolchain = "bare"
)

// packagesDir is the directory that contains the directory of each package of a state, relative
// to the root of the repository.
const packagesDir = "packages"

// errRefused is the error of a versioner that refuses a version.
var errRefused = errors.New("refused")

// errDiscovery is the error of a discovery that fails.
var errDiscovery = errors.New("discovery failed")

// npm is a versioner with the requirements that the tests of changesets write: an exact version,
// ^ and ~ before a version, and *. A consumer of npm gets the newest version that a range admits,
// so a requirement selects every version that it admits and excludes every other. It reads no
// requirement with a protocol, such as link: or file:.
type npm struct {
	// refuse is the major version from which Validate refuses a release, or 0 for none.
	refuse uint64
}

var _ language.Versioner = npm{}

// Resolve returns ResolutionSelected for a v that req admits, by the ranges of node-semver for
// versions without a pre-release, and ResolutionExcluded for any other v.
func (npm) Resolve(req string, v version.Version) (language.Resolution, error) {
	if req == "*" {
		return language.ResolutionSelected, nil
	}
	op := req[:len(req)-len(strings.TrimLeft(req, "^~"))]
	base, err := version.Parse(strings.TrimPrefix(req, op))
	if err != nil {
		return 0, err
	}
	var admitted bool
	switch {
	case v.Compare(base) < 0:
		admitted = false
	case op == "^" && base.Major > 0:
		admitted = v.Major == base.Major
	case op == "^" && base.Minor > 0, op == "~":
		admitted = v.Major == base.Major && v.Minor == base.Minor
	case op == "^":
		admitted = v.Major == 0 && v.Minor == 0 && v.Patch == base.Patch
	default:
		admitted = v.Compare(base) == 0
	}
	if admitted {
		return language.ResolutionSelected, nil
	}
	return language.ResolutionExcluded, nil
}

// Rewrite returns req with v in place of its version, after the same operator, and * as it is.
func (npm) Rewrite(req string, v version.Version) (string, error) {
	if req == "*" {
		return req, nil
	}
	op := req[:len(req)-len(strings.TrimLeft(req, "^~"))]
	return op + v.String(), nil
}

// Validate returns an error that wraps errRefused for a v at the major version refuse or above.
func (n npm) Validate(p *workspace.Package, v version.Version) error {
	if n.refuse > 0 && v.Major >= n.refuse {
		return fmt.Errorf("%w: %s at %s", errRefused, p.Name, v)
	}
	return nil
}

// Apply changes no path.
func (npm) Apply(context.Context, string, []language.Edit) ([]string, error) {
	return nil, nil
}

// gomod is a versioner with the requirements of go.mod: a require line, such as v1.2.0, names the
// lowest version that it admits, and minimal version selection gives a consumer that version.
type gomod struct{}

var _ language.Versioner = gomod{}

// Resolve returns ResolutionExcluded for a v below the version that req names,
// ResolutionSelected for that version, and ResolutionPinned for a higher v.
func (gomod) Resolve(req string, v version.Version) (language.Resolution, error) {
	base, err := version.Parse(strings.TrimPrefix(req, "v"))
	if err != nil {
		return 0, err
	}
	switch c := v.Compare(base); {
	case c < 0:
		return language.ResolutionExcluded, nil
	case c == 0:
		return language.ResolutionSelected, nil
	}
	return language.ResolutionPinned, nil
}

// Rewrite returns v with a v before it.
func (gomod) Rewrite(_ string, v version.Version) (string, error) {
	return "v" + v.String(), nil
}

// Validate returns nil.
func (gomod) Validate(*workspace.Package, version.Version) error {
	return nil
}

// Apply changes no path.
func (gomod) Apply(context.Context, string, []language.Edit) ([]string, error) {
	return nil, nil
}

// state is the packages and the changesets of a case, as FakeFullState of changesets builds them.
type state struct {
	// tb is the test of the case, which a fixture that the state refuses fails.
	tb testing.TB

	// versioner is the versioner of the toolchain of the packages.
	versioner language.Versioner

	// toolchain is the toolchain of the packages.
	toolchain workspace.Toolchain

	// pkgs are the packages of the case.
	pkgs []workspace.Package

	// sets are the changesets of the case.
	sets []changeset.Changeset

	// roles are the release roles of the toolchain besides its versioner.
	roles []any

	// changelogVersion is the fact of the toolchain that its packages record their versions in
	// their changelogs.
	changelogVersion bool
}

// newState returns the state of changesets' setup for the test tb: pkg-a at 1.0.0 and the
// changeset strange-words-combine with pkg-a at patch.
func newState(tb testing.TB) *state {
	tb.Helper()
	s := blankState(tb)
	s.add("pkg-a", "1.0.0")
	s.changeset("strange-words-combine", changeset.Release{Name: "pkg-a", Bump: version.BumpPatch})
	return s
}

// blankState returns a state of the toolchain npm without packages and changesets for the test
// tb.
func blankState(tb testing.TB) *state {
	tb.Helper()
	return &state{tb: tb, versioner: npm{}, toolchain: npmToolchain}
}

// goState returns a state of the toolchain go without packages and changesets for the test tb.
func goState(tb testing.TB) *state {
	tb.Helper()
	return &state{tb: tb, versioner: gomod{}, toolchain: goToolchain, changelogVersion: true}
}

// add adds the package name of the toolchain of the state at the version v, in the directory name
// of packagesDir.
func (s *state) add(name, v string) {
	s.pkgs = append(s.pkgs, workspace.Package{
		Name: name, Toolchain: s.toolchain, Dir: path.Join(packagesDir, name), Version: parse(s.tb, v),
	})
}

// setVersion sets the version of the package name to v.
func (s *state) setVersion(name, v string) {
	s.pkgs[s.at(name)].Version = parse(s.tb, v)
}

// require adds the requirement req of the package dependent on the package dependency, in the
// section kind.
func (s *state) require(dependent, dependency string, kind workspace.Kind, req string) {
	i := s.at(dependent)
	s.pkgs[i].Deps = append(s.pkgs[i].Deps, workspace.Dependency{Name: dependency, Kind: kind, Req: req})
}

// changeset adds the changeset id with releases, each on its line of a file after the opening
// line of the front matter.
func (s *state) changeset(id string, releases ...changeset.Release) {
	for k := range releases {
		releases[k].Line = k + 2
	}
	s.sets = append(s.sets, changeset.Changeset{ID: id, Summary: "base summary whatever", Releases: releases})
}

// graph returns the graph of the packages, with the versioner, the other roles and the facts of the
// state's toolchain.
func (s *state) graph(tb testing.TB) *release.Graph {
	tb.Helper()
	var c language.Catalog
	toolchain := language.Toolchain{Name: s.toolchain, ChangelogVersion: s.changelogVersion}
	roles := append([]any{s.versioner}, s.roles...)
	assert.NoError(tb, language.RegisterToolchain(&c, toolchain, roles...), "RegisterToolchain")
	g, err := release.NewGraph(&c, s.pkgs)
	assert.NoError(tb, err, "NewGraph")
	return g
}

// at returns the index of the package name, and fails the test for a name that the state does not
// have.
func (s *state) at(name string) int {
	at := -1
	for i := range s.pkgs {
		if s.pkgs[i].Name == name {
			at = i
		}
	}
	assert.NotEqual(s.tb, at, -1, "the index of the package "+name)
	return at
}

func TestGraph(t *testing.T) {
	t.Parallel()

	t.Run("NewGraph", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrPackages for a package whose toolchain registers no versioner", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: bareToolchain}),
				"RegisterToolchain of bare")
			_, err := release.NewGraph(&c, []workspace.Package{{Name: "a", Toolchain: bareToolchain}})
			assert.ErrorIs(t, err, release.ErrPackages, "NewGraph")
		})

		t.Run("returns ErrPackages for two packages of one toolchain with the same name", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.0")
			s.add("pkg-a", "2.0.0")
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: npmToolchain}, npm{}),
				"RegisterToolchain of npm")
			_, err := release.NewGraph(&c, s.pkgs)
			assert.ErrorIs(t, err, release.ErrPackages, "NewGraph")
		})

		t.Run("names a package that two toolchains share after its toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			for _, name := range []workspace.Toolchain{npmToolchain, pythonToolchain} {
				assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: name}, npm{}),
					"RegisterToolchain of "+string(name))
			}
			g, err := release.NewGraph(&c, []workspace.Package{
				{Name: "dokimi-assert", Toolchain: npmToolchain, Version: parse(t, "1.0.0")},
				{Name: "dokimi-assert", Toolchain: pythonToolchain, Version: parse(t, "1.0.0")},
			})
			assert.NoError(t, err, "NewGraph")
			cfg := defaultConfig()
			plan, err := release.NewPlan(g, &cfg, []changeset.Changeset{{ID: "x", Releases: []changeset.Release{
				{Name: "python:dokimi-assert", Bump: version.BumpMinor},
			}}})
			assert.NoError(t, err, "NewPlan")
			assert.Equal(t, plan.Releases, []release.Release{{
				Name: "python:dokimi-assert", Bump: version.BumpMinor, Old: parse(t, "1.0.0"),
				New: parse(t, "1.1.0"), Changesets: []string{"x"},
			}}, "the releases")
		})

		t.Run("makes no dependent of a package of another toolchain with the required name", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			for _, name := range []workspace.Toolchain{npmToolchain, pythonToolchain} {
				assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: name}, npm{}),
					"RegisterToolchain of "+string(name))
			}
			g, err := release.NewGraph(&c, []workspace.Package{
				{Name: "core", Toolchain: npmToolchain, Version: parse(t, "1.0.0")},
				{Name: "app", Toolchain: pythonToolchain, Version: parse(t, "1.0.0"), Deps: []workspace.Dependency{
					{Name: "core", Kind: workspace.KindRuntime, Req: "1.0.0"},
				}},
			})
			assert.NoError(t, err, "NewGraph")
			cfg := defaultConfig()
			plan, err := release.NewPlan(g, &cfg, []changeset.Changeset{{ID: "x", Releases: []changeset.Release{
				{Name: "core", Bump: version.BumpMajor},
			}}})
			assert.NoError(t, err, "NewPlan")
			assert.Length(t, plan.Releases, 1, "the releases")
		})

		t.Run("makes no dependent of a package that requires itself", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.require("pkg-a", "pkg-a", workspace.KindRuntime, "1.0.0")
			cfg := defaultConfig()
			plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			assert.Length(t, plan.Releases, 1, "the releases")
		})

		t.Run("decides a dependent by the requirement of the last section that names the package", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.add("pkg-b", "1.0.0")
			s.require("pkg-b", "pkg-a", workspace.KindPeer, "^2.0.0")
			s.require("pkg-b", "pkg-a", workspace.KindRuntime, "1.0.0")
			cfg := defaultConfig()
			plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			assert.Length(t, plan.Releases, 1, "the releases")
		})

		t.Run("makes a dependent of a package whose require line pins an older version", func(t *testing.T) {
			t.Parallel()
			s := goState(t)
			s.add("example.com/a", "1.0.0")
			s.add("example.com/b", "1.0.0")
			s.require("example.com/b", "example.com/a", workspace.KindRuntime, "v0.9.0")
			s.changeset("fix", r("example.com/a", version.BumpPatch))
			cfg := defaultConfig()
			plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			assert.Equal(t, plan.Releases, []release.Release{
				rel(t, "example.com/a", version.BumpPatch, "1.0.0", "1.0.1", "fix"),
				rel(t, "example.com/b", version.BumpPatch, "1.0.0", "1.0.1"),
			}, "the releases")
		})

		t.Run("makes no dependent of a package whose require line excludes its current version", func(t *testing.T) {
			t.Parallel()
			s := goState(t)
			s.add("example.com/a", "1.0.0")
			s.add("example.com/b", "1.0.0")
			s.require("example.com/b", "example.com/a", workspace.KindRuntime, "v1.1.0")
			s.changeset("fix", r("example.com/a", version.BumpMinor))
			cfg := defaultConfig()
			plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			assert.Length(t, plan.Releases, 1, "the releases")
		})
	})

	t.Run("Discover", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the packages of each toolchain in the order of the catalog", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			discover := func(name string) func(context.Context, string) ([]workspace.Package, error) {
				return func(_ context.Context, root string) ([]workspace.Package, error) {
					return []workspace.Package{{Name: name, Toolchain: npmToolchain, Dir: root}}, nil
				}
			}
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{Name: bareToolchain}),
				"RegisterToolchain of bare")
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{
				Name: npmToolchain, Discover: discover("first"),
			}, npm{}), "RegisterToolchain of npm")
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{
				Name: pythonToolchain, Discover: discover("second"),
			}, npm{}), "RegisterToolchain of python")
			g, err := release.Discover(t.Context(), &c, "root")
			assert.NoError(t, err, "Discover")
			assert.Equal(t, g.Packages(), []workspace.Package{
				{Name: "first", Toolchain: npmToolchain, Dir: "root"},
				{Name: "second", Toolchain: npmToolchain, Dir: "root"},
			}, "the packages")
		})

		t.Run("returns the error of a discovery with the name of its toolchain", func(t *testing.T) {
			t.Parallel()
			var c language.Catalog
			assert.NoError(t, language.RegisterToolchain(&c, language.Toolchain{
				Name: npmToolchain,
				Discover: func(context.Context, string) ([]workspace.Package, error) {
					return nil, errDiscovery
				},
			}, npm{}), "RegisterToolchain of npm")
			_, err := release.Discover(t.Context(), &c, "root")
			assert.ErrorIs(t, err, errDiscovery, "Discover")
			assert.Contains(t, err.Error(), "the toolchain npm", "the error")
		})
	})
}

// parse returns the version that s states, and fails the test tb for an s that version.Parse
// refuses.
func parse(tb testing.TB, s string) version.Version {
	tb.Helper()
	v, err := version.Parse(s)
	assert.NoError(tb, err, "Parse of "+s)
	return v
}
