// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
)

// errRegistry is the error of a registry that fails.
var errRegistry = errors.New("registry failed")

// registry is a publisher whose registry has the releases of published, by <name>@<version>, and
// that records the packages of each Publish. It returns errRegistry from Published for the name
// failPublished, and from Publish once failPublish is set.
type registry struct {
	// published are the releases that the registry has.
	published map[string]bool

	// uploads are the names of the packages of each Publish, in the order of the calls.
	uploads *[][]string

	// mu guards uploads and published.
	mu *sync.Mutex

	// failPublished is the name of a package whose release Published fails for.
	failPublished string

	// failPublish makes Publish fail.
	failPublish bool
}

var _ language.Publisher = registry{}

// newRegistry returns a registry that has releases, each as <name>@<version>.
func newRegistry(releases ...string) registry {
	published := map[string]bool{}
	for _, r := range releases {
		published[r] = true
	}
	return registry{published: published, uploads: new([][]string), mu: new(sync.Mutex)}
}

// Published reports whether the registry has p at its version.
func (r registry) Published(_ context.Context, p *workspace.Package) (bool, error) {
	if p.Name == r.failPublished {
		return false, errRegistry
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.published[p.Name+"@"+p.Version.String()], nil
}

// Publish records the names of pkgs and adds their releases to the registry.
func (r registry) Publish(_ context.Context, _ string, pkgs []workspace.Package) error {
	if r.failPublish {
		return errRegistry
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var names []string
	for _, p := range pkgs {
		names = append(names, p.Name)
		r.published[p.Name+"@"+p.Version.String()] = true
	}
	*r.uploads = append(*r.uploads, names)
	return nil
}

// prefixTagger names the tag of a package after its directory, as Go names it.
type prefixTagger struct{}

var _ language.Tagger = prefixTagger{}

// Tag returns the directory of p, a slash, v and the version.
func (prefixTagger) Tag(p *workspace.Package, v version.Version) string {
	return p.Dir + "/v" + v.String()
}

func TestPublishPlan(t *testing.T) {
	t.Parallel()

	t.Run("NewPublishPlan", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			setup func(s *state, c *release.Config)
			name  string
			tags  []string
			want  [][]release.PublishEntry
		}{
			{
				name: "returns an entry of tag-only for a released package without its tag",
				setup: func(s *state, _ *release.Config) {
					s.add("pkg-b", "0.0.0")
				},
				want: [][]release.PublishEntry{{tagOnly(t, "pkg-a", "pkg-a@1.0.0", "1.0.0")}},
			},
			{
				name: "returns no entry for a package with its tag",
				setup: func(s *state, _ *release.Config) {
					s.add("pkg-b", "0.0.0")
				},
				tags: []string{"pkg-a@1.0.0"},
			},
			{
				name: "names the tag v and the version in a repository with one package",
				want: [][]release.PublishEntry{{tagOnly(t, "pkg-a", "v1.0.0", "1.0.0")}},
			},
			{
				name: "names the tag of a package with the tagger of its toolchain",
				setup: func(s *state, _ *release.Config) {
					s.roles = []any{prefixTagger{}}
				},
				want: [][]release.PublishEntry{{tagOnly(t, "pkg-a", "packages/pkg-a/v1.0.0", "1.0.0")}},
			},
			{
				name: "returns an entry of publish for a package whose registry does not have its version",
				setup: func(s *state, _ *release.Config) {
					s.roles = []any{newRegistry()}
				},
				want: [][]release.PublishEntry{{{
					Kind: release.KindPublish, Toolchain: npmToolchain, Name: "pkg-a", Tag: "v1.0.0",
					Version: parse(t, "1.0.0"),
				}}},
			},
			{
				name: "returns no entry for a package whose registry has its version",
				setup: func(s *state, _ *release.Config) {
					s.roles = []any{newRegistry("pkg-a@1.0.0")}
				},
			},
			{
				name: "returns an entry of tag-only for a private package under privatePackages.tag",
				setup: func(s *state, c *release.Config) {
					s.pkgs[0].Private = true
					s.roles = []any{newRegistry()}
					c.PrivateVersion, c.PrivateTag = true, true
				},
				want: [][]release.PublishEntry{{tagOnly(t, "pkg-a", "v1.0.0", "1.0.0")}},
			},
			{
				name: "returns no entry for a private package without privatePackages.tag",
				setup: func(s *state, _ *release.Config) {
					s.pkgs[0].Private = true
				},
			},
			{
				name: "puts a package in the chunk after the packages that it requires",
				setup: func(s *state, _ *release.Config) {
					s.add("pkg-b", "1.0.0")
					s.add("pkg-c", "1.0.0")
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^1.0.0")
					s.require("pkg-c", "pkg-a", workspace.KindDev, "^1.0.0")
				},
				want: [][]release.PublishEntry{
					{
						tagOnly(t, "pkg-a", "pkg-a@1.0.0", "1.0.0"),
						tagOnly(t, "pkg-c", "pkg-c@1.0.0", "1.0.0"),
					},
					{tagOnly(t, "pkg-b", "pkg-b@1.0.0", "1.0.0")},
				},
			},
			{
				name: "puts packages that require each other in one chunk",
				setup: func(s *state, _ *release.Config) {
					s.add("pkg-b", "1.0.0")
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^1.0.0")
					s.require("pkg-a", "pkg-b", workspace.KindRuntime, "^1.0.0")
				},
				want: [][]release.PublishEntry{{
					tagOnly(t, "pkg-a", "pkg-a@1.0.0", "1.0.0"),
					tagOnly(t, "pkg-b", "pkg-b@1.0.0", "1.0.0"),
				}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := newState(t)
				cfg := defaultConfig()
				if tt.setup != nil {
					tt.setup(s, &cfg)
				}
				tags := map[string]string{}
				for _, name := range tt.tags {
					tags[name] = commitA
				}
				plan, err := release.NewPublishPlan(t.Context(), s.graph(t), &cfg, tags)
				assert.NoError(t, err, "NewPublishPlan")
				assert.Equal(t, plan.Plan, tt.want, "the chunks", assert.EquateEmpty())
			})
		}

		t.Run("returns a plan of the version 1 of the format", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			cfg := defaultConfig()
			plan, err := release.NewPublishPlan(t.Context(), s.graph(t), &cfg, nil)
			assert.NoError(t, err, "NewPublishPlan")
			assert.Equal(t, plan.Version, 1, "the version of the format")
		})

		t.Run("encodes a plan without entries as an empty list", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			cfg := defaultConfig()
			plan, err := release.NewPublishPlan(t.Context(), s.graph(t), &cfg, nil)
			assert.NoError(t, err, "NewPublishPlan")
			data, err := json.Marshal(&plan)
			assert.NoError(t, err, "Marshal")
			assert.Equal(t, string(data), `{"plan":[],"version":1}`, "the JSON")
		})

		t.Run("returns the error of a registry", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			failing := newRegistry()
			failing.failPublished = "pkg-a"
			s.roles = []any{failing}
			cfg := defaultConfig()
			_, err := release.NewPublishPlan(t.Context(), s.graph(t), &cfg, nil)
			assert.ErrorIs(t, err, errRegistry, "NewPublishPlan")
			assert.Contains(t, err.Error(), "the registry of pkg-a", "the error")
		})
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give [][]release.PublishEntry
			want bool
		}{
			{name: "reports true for a plan without chunks", want: true},
			{name: "reports true for a plan of empty chunks", give: [][]release.PublishEntry{{}}, want: true},
			{name: "reports false for a plan with an entry", give: [][]release.PublishEntry{{{Name: "pkg-a"}}}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				plan := release.PublishPlan{Version: 1, Plan: tt.give}
				assert.Equal(t, plan.Empty(), tt.want, "Empty")
			})
		}
	})
}

// tagOnly returns the entry of tag-only of the package name of the toolchain npm with the tag and
// the version v, for the test tb.
func tagOnly(tb testing.TB, name, tag, v string) release.PublishEntry {
	tb.Helper()
	return release.PublishEntry{
		Kind:      release.KindTagOnly,
		Toolchain: npmToolchain,
		Name:      name,
		Tag:       tag,
		Version:   parse(tb, v),
	}
}

// uploaded returns the names of the packages of each Publish of r.
func uploaded(r registry) [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(*r.uploads)
}
