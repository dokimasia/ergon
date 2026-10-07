// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
)

// The levels of the cases, by their spelling in changesets.
const (
	none  = version.BumpNone
	patch = version.BumpPatch
	minor = version.BumpMinor
	major = version.BumpMajor
)

// planCase is a case of the planner: the changes to the state and to the configuration of
// changesets' setup, and the releases of the plan.
type planCase struct {
	// setup changes the state of changesets' setup with pkg-b, pkg-c and pkg-d at 1.0.0.
	setup func(s *state)

	// config changes the default configuration, or is nil.
	config func(c *release.Config)

	// name is the name of the case.
	name string

	// want are the releases of the plan.
	want []release.Release
}

// The kinds of requirement of the matrix of dependents, as changesets names them.
var matrixKinds = []struct {
	name string
	kind workspace.Kind
}{
	{name: "dep", kind: workspace.KindRuntime},
	{name: "dev", kind: workspace.KindDev},
	{name: "peer", kind: workspace.KindPeer},
}

// matrixRanges are the operators of the matrix of dependents: = writes the version alone.
var matrixRanges = []string{"^", "~", "="}

// defaultBumps are the versions of the dependent pkg-a of the matrix of dependents under the
// default configuration, by the level of its dependency and the operator of the requirement, as
// changesets' test of dependent bumping states them for a requirement outside the dev section.
var defaultBumps = map[version.Bump]map[string]string{
	none:  {"^": "1.0.0", "~": "1.0.0", "=": "1.0.0"},
	patch: {"^": "1.0.0", "~": "1.0.0", "=": "1.0.1"},
	minor: {"^": "1.0.0", "~": "1.0.1", "=": "1.0.1"},
	major: {"^": "1.0.1", "~": "1.0.1", "=": "1.0.1"},
}

func TestPlan(t *testing.T) {
	t.Parallel()

	t.Run("NewPlan", func(t *testing.T) {
		t.Parallel()

		tests := []planCase{
			{
				name: "releases the package of a changeset at its level",
				want: []release.Release{rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine")},
			},
			{
				name: "releases the packages of several changesets in their order",
				setup: func(s *state) {
					s.changeset("big-cats-delight", r("pkg-b", patch), r("pkg-c", patch), r("pkg-d", major))
				},
				want: []release.Release{
					rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine"),
					rel(t, "pkg-b", patch, "1.0.0", "1.0.1", "big-cats-delight"),
					rel(t, "pkg-c", patch, "1.0.0", "1.0.1", "big-cats-delight"),
					rel(t, "pkg-d", major, "1.0.0", "2.0.0", "big-cats-delight"),
				},
			},
			{
				name: "releases a package of two changesets at the higher level",
				setup: func(s *state) {
					s.changeset("big-cats-delight", r("pkg-a", major))
				},
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine", "big-cats-delight"),
				},
			},
			{
				name: "keeps a level above none against changesets at none",
				setup: func(s *state) {
					s.changeset("big-cats-delight", r("pkg-a", none), r("pkg-b", none), r("pkg-c", none))
					s.changeset("big-cats-wonder", r("pkg-a", patch), r("pkg-b", minor), r("pkg-c", major))
					s.changeset("big-cats-yelp", r("pkg-a", none), r("pkg-b", none), r("pkg-c", none))
				},
				want: []release.Release{
					rel(t, "pkg-a", patch, "1.0.0", "1.0.1",
						"strange-words-combine", "big-cats-delight", "big-cats-wonder", "big-cats-yelp"),
					rel(t, "pkg-b", minor, "1.0.0", "1.1.0", "big-cats-delight", "big-cats-wonder", "big-cats-yelp"),
					rel(t, "pkg-c", major, "1.0.0", "2.0.0", "big-cats-delight", "big-cats-wonder", "big-cats-yelp"),
				},
			},
			{
				name: "releases every dependent of a package whose requirement leaves the range",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "1.0.0")
					s.require("pkg-c", "pkg-a", workspace.KindRuntime, "1.0.0")
				},
				want: []release.Release{
					rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine"),
					rel(t, "pkg-b", patch, "1.0.0", "1.0.1"),
					rel(t, "pkg-c", patch, "1.0.0", "1.0.1"),
				},
			},
			{
				name: "releases the dependents of a released dependent down the tree",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "1.0.0")
					s.require("pkg-c", "pkg-b", workspace.KindRuntime, "1.0.0")
					s.require("pkg-d", "pkg-c", workspace.KindRuntime, "1.0.0")
				},
				want: []release.Release{
					rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine"),
					rel(t, "pkg-b", patch, "1.0.0", "1.0.1"),
					rel(t, "pkg-c", patch, "1.0.0", "1.0.1"),
					rel(t, "pkg-d", patch, "1.0.0", "1.0.1"),
				},
			},
			{
				name: "releases no dependent through a requirement of every version",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "*")
					s.changeset("big-cats-delight", r("pkg-a", major))
				},
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine", "big-cats-delight"),
				},
			},
			{
				name: "releases no dependent through a link: requirement",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindDev, "link:../pkg-a")
					s.changeset("big-cats-delight", r("pkg-a", major))
				},
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine", "big-cats-delight"),
				},
			},
			{
				name: "releases no dependent through a file: requirement",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindDev, "file:../pkg-a")
					s.changeset("big-cats-delight", r("pkg-a", major))
				},
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine", "big-cats-delight"),
				},
			},
			{
				name: "releases no ignored package that a changeset names",
				setup: func(s *state) {
					s.changeset("big-cats-delight", r("pkg-a", major))
					s.changeset("small-dogs-sad", r("pkg-b", minor))
				},
				config: ignore("pkg-b"),
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine", "big-cats-delight"),
				},
			},
			{
				name:   "releases an ignored dependent at none",
				setup:  ignoredDependent(workspace.KindRuntime),
				config: ignore("pkg-b"),
				want:   ignoredDependentReleases(t),
			},
			{
				name:   "releases an ignored peer dependent at none",
				setup:  ignoredDependent(workspace.KindPeer),
				config: ignore("pkg-b"),
				want:   ignoredDependentReleases(t),
			},
			{
				name:   "releases an ignored dev dependent at none",
				setup:  ignoredDependent(workspace.KindDev),
				config: ignore("pkg-b"),
				want:   ignoredDependentReleases(t),
			},
			{
				name: "releases every member of a fixed group together",
				setup: func(s *state) {
					s.changeset("just-some-umbrellas", r("pkg-a", minor))
				},
				config: func(c *release.Config) { c.Fixed = [][]string{{"pkg-a", "pkg-b"}} },
				want: []release.Release{
					rel(t, "pkg-a", minor, "1.0.0", "1.1.0", "strange-words-combine", "just-some-umbrellas"),
					rel(t, "pkg-b", minor, "1.0.0", "1.1.0"),
				},
			},
			{
				name: "releases no ignored member of a fixed group",
				config: func(c *release.Config) {
					c.Fixed = [][]string{{"pkg-a", "pkg-b"}}
					c.Ignore = []string{"pkg-b"}
				},
				want: []release.Release{rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine")},
			},
			{
				name: "releases a fixed group from the version of an unreleased member",
				setup: func(s *state) {
					s.changeset("just-some-umbrellas", r("pkg-b", minor), r("pkg-a", patch))
					s.setVersion("pkg-c", "2.0.0")
				},
				config: func(c *release.Config) { c.Fixed = [][]string{{"pkg-a", "pkg-b", "pkg-c"}} },
				want: []release.Release{
					rel(t, "pkg-a", minor, "2.0.0", "2.1.0", "strange-words-combine", "just-some-umbrellas"),
					rel(t, "pkg-b", minor, "2.0.0", "2.1.0", "just-some-umbrellas"),
					rel(t, "pkg-c", minor, "2.0.0", "2.1.0"),
				},
			},
			{
				name: "releases two fixed groups in a chain when one requires the other",
				setup: func(s *state) {
					s.changeset("just-some-umbrellas", r("pkg-b", major))
					s.changeset("totally-average-verbiage", r("pkg-d", minor))
					s.require("pkg-c", "pkg-a", workspace.KindRuntime, "^1.0.0")
				},
				config: func(c *release.Config) { c.Fixed = [][]string{{"pkg-a", "pkg-b"}, {"pkg-c", "pkg-d"}} },
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine"),
					rel(t, "pkg-b", major, "1.0.0", "2.0.0", "just-some-umbrellas"),
					rel(t, "pkg-d", minor, "1.0.0", "1.1.0", "totally-average-verbiage"),
					rel(t, "pkg-c", minor, "1.0.0", "1.1.0"),
				},
			},
			{
				name: "releases two fixed groups in a chain when the unreleased member requires the other",
				setup: func(s *state) {
					s.changeset("just-some-umbrellas", r("pkg-a", major))
					s.changeset("totally-average-verbiage", r("pkg-d", minor))
					s.require("pkg-c", "pkg-b", workspace.KindRuntime, "^1.0.0")
				},
				config: func(c *release.Config) { c.Fixed = [][]string{{"pkg-a", "pkg-b"}, {"pkg-c", "pkg-d"}} },
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine", "just-some-umbrellas"),
					rel(t, "pkg-d", minor, "1.0.0", "1.1.0", "totally-average-verbiage"),
					rel(t, "pkg-b", major, "1.0.0", "2.0.0"),
					rel(t, "pkg-c", minor, "1.0.0", "1.1.0"),
				},
			},
			{
				name: "releases a peer dependent of a member that a fixed group releases",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-c", workspace.KindPeer, "1.0.0")
					s.changeset("some-id", r("pkg-a", minor))
				},
				config: func(c *release.Config) { c.Fixed = [][]string{{"pkg-a", "pkg-c"}} },
				want: []release.Release{
					rel(t, "pkg-a", minor, "1.0.0", "1.1.0", "strange-words-combine", "some-id"),
					rel(t, "pkg-c", minor, "1.0.0", "1.1.0"),
					rel(t, "pkg-b", patch, "1.0.0", "1.0.1"),
				},
			},
			{
				name: "releases the members of a linked group at the highest level",
				setup: func(s *state) {
					s.changeset("just-some-umbrellas", r("pkg-b", major))
				},
				config: func(c *release.Config) { c.Linked = [][]string{{"pkg-a", "pkg-b"}} },
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine"),
					rel(t, "pkg-b", major, "1.0.0", "2.0.0", "just-some-umbrellas"),
				},
			},
			{
				name: "releases a linked group from the version of an unreleased member",
				setup: func(s *state) {
					s.changeset("just-some-umbrellas", r("pkg-b", minor), r("pkg-a", patch))
					s.setVersion("pkg-c", "2.0.0")
				},
				config: func(c *release.Config) { c.Linked = [][]string{{"pkg-a", "pkg-b", "pkg-c"}} },
				want: []release.Release{
					rel(t, "pkg-a", minor, "2.0.0", "2.1.0", "strange-words-combine", "just-some-umbrellas"),
					rel(t, "pkg-b", minor, "2.0.0", "2.1.0", "just-some-umbrellas"),
				},
			},
			{
				name: "releases two linked groups in a chain when one requires the other",
				setup: func(s *state) {
					s.changeset("just-some-umbrellas", r("pkg-b", major))
					s.changeset("totally-average-verbiage", r("pkg-d", minor))
					s.require("pkg-c", "pkg-a", workspace.KindRuntime, "^1.0.0")
				},
				config: func(c *release.Config) { c.Linked = [][]string{{"pkg-a", "pkg-b"}, {"pkg-c", "pkg-d"}} },
				want: []release.Release{
					rel(t, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine"),
					rel(t, "pkg-b", major, "1.0.0", "2.0.0", "just-some-umbrellas"),
					rel(t, "pkg-d", minor, "1.0.0", "1.1.0", "totally-average-verbiage"),
					rel(t, "pkg-c", minor, "1.0.0", "1.1.0"),
				},
			},
			{
				name: "releases a peer dependent of a member that a linked group raises",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindPeer, "1.0.0")
					s.changeset("some-id", r("pkg-c", minor))
				},
				config: func(c *release.Config) { c.Linked = [][]string{{"pkg-a", "pkg-c"}} },
				want: []release.Release{
					rel(t, "pkg-a", minor, "1.0.0", "1.1.0", "strange-words-combine"),
					rel(t, "pkg-c", minor, "1.0.0", "1.1.0", "some-id"),
					rel(t, "pkg-b", patch, "1.0.0", "1.0.1"),
				},
			},
			{
				name: "releases a transitive dependent under updateInternalDependents always",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^1.0.0")
					s.require("pkg-c", "pkg-b", workspace.KindRuntime, "^1.0.0")
				},
				config: func(c *release.Config) { c.UpdateInternalDependents = release.DependentsAlways },
				want: []release.Release{
					rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine"),
					rel(t, "pkg-b", patch, "1.0.0", "1.0.1"),
					rel(t, "pkg-c", patch, "1.0.0", "1.0.1"),
				},
			},
			{
				name: "releases no dependent of a release at none under updateInternalDependents always",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-c", workspace.KindRuntime, "^1.0.0")
					s.changeset("stuff-and-nonsense", r("pkg-c", none))
				},
				config: func(c *release.Config) { c.UpdateInternalDependents = release.DependentsAlways },
				want: []release.Release{
					rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine"),
					rel(t, "pkg-c", none, "1.0.0", "1.0.0", "stuff-and-nonsense"),
				},
			},
			{
				name: "releases an ignored dependent whose range selects the new version at none",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^1.0.0")
				},
				config: ignore("pkg-b"),
				want: []release.Release{
					rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine"),
					rel(t, "pkg-b", none, "1.0.0", "1.0.0"),
				},
			},
			{
				name: "releases a private package that privatePackages.version releases",
				setup: func(s *state) {
					s.pkgs[s.at("pkg-b")].Private = true
					s.changeset("private-fix", r("pkg-b", patch))
				},
				config: func(c *release.Config) { c.PrivateVersion = true },
				want: []release.Release{
					rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine"),
					rel(t, "pkg-b", patch, "1.0.0", "1.0.1", "private-fix"),
				},
			},
			{
				name: "releases no private package without privatePackages.version",
				setup: func(s *state) {
					s.pkgs[s.at("pkg-b")].Private = true
					s.changeset("private-fix", r("pkg-b", patch))
				},
				want: []release.Release{rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "strange-words-combine")},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := newState(t)
				s.add("pkg-b", "1.0.0")
				s.add("pkg-c", "1.0.0")
				s.add("pkg-d", "1.0.0")
				if tt.setup != nil {
					tt.setup(s)
				}
				cfg := defaultConfig()
				if tt.config != nil {
					tt.config(&cfg)
				}
				plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
				assert.NoError(t, err, "NewPlan")
				assert.Equal(t, plan.Releases, tt.want, "the releases")
			})
		}

		t.Run("returns no release for no changeset", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.add("pkg-b", "1.0.0")
			cfg := defaultConfig()
			cfg.Fixed = [][]string{{"pkg-a", "pkg-b"}}
			cfg.Linked = [][]string{{"pkg-c"}}
			plan, err := release.NewPlan(s.graph(t), &cfg, nil)
			assert.NoError(t, err, "NewPlan")
			assert.Empty(t, plan.Releases, "the releases")
		})

		t.Run("releases a fixed group whose member a peer requires at its own version", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.0")
			for _, name := range []string{"@ex/core", "@ex/errors", "@ex/api", "some-peer", "@ex/components"} {
				s.add(name, "0.1.0")
			}
			s.require("@ex/api", "@ex/core", workspace.KindRuntime, "0.1.0")
			s.require("@ex/api", "@ex/errors", workspace.KindRuntime, "0.1.0")
			s.require("@ex/api", "some-peer", workspace.KindPeer, "0.1.0")
			s.require("@ex/components", "@ex/api", workspace.KindRuntime, "0.1.0")
			s.require("@ex/components", "some-peer", workspace.KindRuntime, "0.1.0")
			s.changeset("strange-words-combine", r("@ex/core", minor))
			cfg := defaultConfig()
			cfg.Fixed = [][]string{{"@ex/core", "@ex/errors", "@ex/api", "some-peer", "@ex/components"}}
			plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			assert.Equal(t, plan.Releases, []release.Release{
				rel(t, "@ex/core", minor, "0.1.0", "0.2.0", "strange-words-combine"),
				rel(t, "@ex/api", minor, "0.1.0", "0.2.0"),
				rel(t, "@ex/components", minor, "0.1.0", "0.2.0"),
				rel(t, "@ex/errors", minor, "0.1.0", "0.2.0"),
				rel(t, "some-peer", minor, "0.1.0", "0.2.0"),
			}, "the releases")
		})

		t.Run("releases a dependent with a changed runtime and dev requirement at patch", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.0")
			s.add("pkg-b", "1.0.0")
			s.add("pkg-c", "1.0.0")
			s.require("pkg-b", "pkg-a", workspace.KindDev, "^1.0.0")
			s.require("pkg-b", "pkg-c", workspace.KindRuntime, "^1.0.0")
			s.changeset("big-cats-delight", r("pkg-a", major), r("pkg-c", major))
			cfg := defaultConfig()
			plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			assert.Equal(t, plan.Releases, []release.Release{
				rel(t, "pkg-a", major, "1.0.0", "2.0.0", "big-cats-delight"),
				rel(t, "pkg-c", major, "1.0.0", "2.0.0", "big-cats-delight"),
				rel(t, "pkg-b", patch, "1.0.0", "1.0.1"),
			}, "the releases")
		})

		atNone := []struct {
			name string
			kind workspace.Kind
			req  string
		}{
			{name: "releases no dependent of a release at none", kind: workspace.KindRuntime, req: "^1.0.0"},
			{
				name: "releases no peer dependent through ~ of a release at none",
				kind: workspace.KindPeer,
				req:  "~1.0.0",
			},
			{
				name: "releases no peer dependent through ^ of a release at none",
				kind: workspace.KindPeer,
				req:  "^1.0.0",
			},
		}
		for _, tt := range atNone {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := blankState(t)
				s.add("pkg-a", "1.0.0")
				s.add("pkg-b", "1.0.0")
				s.require("pkg-b", "pkg-a", tt.kind, tt.req)
				s.changeset("anyway-the-windblows", r("pkg-a", none))
				cfg := defaultConfig()
				plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
				assert.NoError(t, err, "NewPlan")
				assert.Equal(t, plan.Releases, []release.Release{
					rel(t, "pkg-a", none, "1.0.0", "1.0.0", "anyway-the-windblows"),
				}, "the releases")
			})
		}

		configs := []struct {
			name   string
			config func(c *release.Config)
			always bool
		}{
			{name: "the default configuration"},
			{
				name:   "updateInternalDependents always",
				config: func(c *release.Config) { c.UpdateInternalDependents = release.DependentsAlways },
				always: true,
			},
			{
				name:   "onlyUpdatePeerDependentsWhenOutOfRange",
				config: func(c *release.Config) { c.OnlyUpdatePeerDependentsWhenOutOfRange = true },
			},
			{
				name: "onlyUpdatePeerDependentsWhenOutOfRange and updateInternalDependents always",
				config: func(c *release.Config) {
					c.OnlyUpdatePeerDependentsWhenOutOfRange = true
					c.UpdateInternalDependents = release.DependentsAlways
				},
				always: true,
			},
		}
		for _, cc := range configs {
			for _, k := range matrixKinds {
				for _, bump := range []version.Bump{none, patch, minor, major} {
					for _, op := range matrixRanges {
						want := defaultBumps[bump][op]
						if cc.always && bump != none {
							want = "1.0.1"
						}
						if k.kind == workspace.KindDev {
							want = "1.0.0"
						}
						name := fmt.Sprintf(
							"gives the dependent %s through %s %s at %s under %s",
							want,
							op,
							k.name,
							bump,
							cc.name,
						)
						t.Run(name, func(t *testing.T) {
							t.Parallel()
							s := blankState(t)
							s.add("pkg-a", "1.0.0")
							s.add("pkg-a-b", "1.0.0")
							s.changeset("dependency", r("pkg-a-b", bump))
							req := "1.0.0"
							if op != "=" {
								req = op + req
							}
							s.require("pkg-a", "pkg-a-b", k.kind, req)
							cfg := defaultConfig()
							if cc.config != nil {
								cc.config(&cfg)
							}
							plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
							assert.NoError(t, err, "NewPlan")
							got := "1.0.0"
							for _, rr := range plan.Releases {
								if rr.Name == "pkg-a" {
									got = rr.New.String()
								}
							}
							assert.Equal(t, got, want, "the version of the dependent")
						})
					}
				}
			}
		}

		for _, cc := range configs[:2] {
			for _, bump := range []version.Bump{none, patch, minor, major} {
				want := "1.0.1"
				if bump == none {
					want = "1.0.0"
				}
				t.Run(fmt.Sprintf("gives the Go dependent %s at %s under %s", want, bump, cc.name), func(t *testing.T) {
					t.Parallel()
					s := goState(t)
					s.add("example.com/a", "1.0.0")
					s.add("example.com/b", "1.0.0")
					s.require("example.com/b", "example.com/a", workspace.KindRuntime, "v1.0.0")
					s.changeset("dependency", r("example.com/a", bump))
					cfg := defaultConfig()
					if cc.config != nil {
						cc.config(&cfg)
					}
					plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
					assert.NoError(t, err, "NewPlan")
					got := "1.0.0"
					for _, rr := range plan.Releases {
						if rr.Name == "example.com/b" {
							got = rr.New.String()
						}
					}
					assert.Equal(t, got, want, "the version of the dependent")
				})
			}
		}

		t.Run("releases the Go modules that require a released module, directly or through other modules",
			func(t *testing.T) {
				t.Parallel()
				s := goState(t)
				for _, name := range []string{
					"example.com/core", "example.com/service", "example.com/lang",
					"example.com/root", "example.com/other",
				} {
					s.add(name, "0.1.0")
				}
				s.require("example.com/service", "example.com/core", workspace.KindRuntime, "v0.1.0")
				s.require("example.com/lang", "example.com/service", workspace.KindRuntime, "v0.1.0")
				s.require("example.com/root", "example.com/lang", workspace.KindRuntime, "v0.1.0")
				s.require("example.com/root", "example.com/other", workspace.KindRuntime, "v0.1.0")
				s.changeset("add-the-unit-type", r("example.com/core", minor))
				cfg := defaultConfig()
				plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
				assert.NoError(t, err, "NewPlan")
				assert.Equal(t, plan.Releases, []release.Release{
					rel(t, "example.com/core", minor, "0.1.0", "0.2.0", "add-the-unit-type"),
					rel(t, "example.com/service", patch, "0.1.0", "0.1.1"),
					rel(t, "example.com/lang", patch, "0.1.0", "0.1.1"),
					rel(t, "example.com/root", patch, "0.1.0", "0.1.1"),
				}, "the releases")
			})

		t.Run("returns ErrChangeset with the file and the line of a package that the repository does not have",
			func(t *testing.T) {
				t.Parallel()
				s := newState(t)
				s.changeset("big-cats-delight", r("pkg-a", major))
				s.changeset("small-dogs-sad", r("pkg-a", minor), r("pkg-z", minor))
				cfg := defaultConfig()
				_, err := release.NewPlan(s.graph(t), &cfg, s.sets)
				assert.ErrorIs(t, err, release.ErrChangeset, "NewPlan")
				assert.Contains(t, err.Error(), ".changeset/small-dogs-sad.md:3", "the error")
			})

		t.Run("returns ErrChangeset for a changeset that names an ignored and a released package", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.add("pkg-b", "1.0.0")
			s.changeset("big-cats-delight", r("pkg-a", major), r("pkg-b", minor))
			cfg := defaultConfig()
			cfg.Ignore = []string{"pkg-b"}
			_, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.ErrorIs(t, err, release.ErrChangeset, "NewPlan")
			assert.Contains(t, err.Error(), "the skipped packages pkg-b and the released packages pkg-a", "the error")
		})

		t.Run("returns the error of a versioner that refuses a new version", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.versioner = npm{refuse: 2}
			s.changeset("break-it", r("pkg-a", major))
			cfg := defaultConfig()
			_, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.ErrorIs(t, err, errRefused, "NewPlan")
			assert.Contains(t, err.Error(), "pkg-a at 2.0.0", "the error")
		})

		t.Run("validates no release at none", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.versioner = npm{refuse: 1}
			s.add("pkg-a", "1.0.0")
			s.changeset("docs", r("pkg-a", none))
			cfg := defaultConfig()
			_, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
		})

		t.Run("returns ErrOverflow for a release past the bound of a component", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", fmt.Sprintf("%d.0.0", version.MaxComponent))
			s.changeset("too-far", r("pkg-a", major))
			cfg := defaultConfig()
			_, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.ErrorIs(t, err, version.ErrOverflow, "NewPlan")
		})

		t.Run("returns ErrOverflow for a dependency whose release passes the bound of a component", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", fmt.Sprintf("%d.0.0", version.MaxComponent))
			s.add("pkg-b", "1.0.0")
			s.require("pkg-b", "pkg-a", workspace.KindRuntime, "*")
			s.changeset("too-far", r("pkg-a", major))
			cfg := defaultConfig()
			_, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.ErrorIs(t, err, version.ErrOverflow, "NewPlan")
			assert.Contains(t, err.Error(), "release: pkg-a", "the error")
		})
	})

	t.Run("Plan", func(t *testing.T) {
		t.Parallel()

		t.Run("encodes to the release plan of changesets", func(t *testing.T) {
			t.Parallel()
			plan := release.Plan{
				Changesets: []changeset.Changeset{{ID: "x", Summary: "Fix it.", Releases: []changeset.Release{
					{Name: "pkg-a", Bump: patch, Line: 2},
				}}},
				Releases: []release.Release{rel(t, "pkg-a", patch, "1.0.0", "1.0.1", "x")},
			}
			got, err := json.Marshal(plan)
			assert.NoError(t, err, "Marshal")
			assert.Equal(t, string(got), `{"changesets":[{"id":"x","summary":"Fix it.","releases":[{"name":"pkg-a",`+
				`"type":"patch"}]}],"releases":[{"name":"pkg-a","type":"patch","changesets":["x"],`+
				`"oldVersion":"1.0.0","newVersion":"1.0.1"}]}`, "the JSON")
		})
	})
}

// ignoredDependentReleases returns the releases of [ignoredDependent] for the test tb.
func ignoredDependentReleases(tb testing.TB) []release.Release {
	tb.Helper()
	return []release.Release{
		rel(tb, "pkg-a", major, "1.0.0", "2.0.0", "strange-words-combine", "big-cats-delight"),
		rel(tb, "pkg-b", none, "1.0.0", "1.0.0"),
	}
}

// ignoredDependent returns the setup of a case whose pkg-b requires pkg-a at 1.0.0 in the section
// kind, and a changeset of each at major and minor.
func ignoredDependent(kind workspace.Kind) func(s *state) {
	return func(s *state) {
		s.require("pkg-b", "pkg-a", kind, "1.0.0")
		s.changeset("big-cats-delight", r("pkg-a", major))
		s.changeset("small-dogs-sad", r("pkg-b", minor))
	}
}

// ignore returns a change of the configuration that ignores names.
func ignore(names ...string) func(c *release.Config) {
	return func(c *release.Config) { c.Ignore = names }
}

// r returns the release of name at bump in a changeset.
func r(name string, bump version.Bump) changeset.Release {
	return changeset.Release{Name: name, Bump: bump}
}

// rel returns the release of the plan of name at bump from old to next, by changesets, for the
// test tb.
func rel(tb testing.TB, name string, bump version.Bump, old, next string, changesets ...string) release.Release {
	tb.Helper()
	if changesets == nil {
		changesets = []string{}
	}
	return release.Release{
		Name: name, Bump: bump, Changesets: changesets, Old: parse(tb, old), New: parse(tb, next),
	}
}
