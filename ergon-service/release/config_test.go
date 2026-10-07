// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("ParseConfig", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			want   func() release.Config
			setup  func(s *state)
			name   string
			config string
		}{
			{
				name:   "returns the defaults of changesets for an empty object",
				config: `{}`,
				want:   defaultConfig,
			},
			{
				name: "reads every key of changesets that ergon reads",
				config: `{
					"$schema": "https://unpkg.com/@changesets/config@4.0.1/schema.json",
					"baseBranch": "trunk",
					"access": "public",
					"format": "prettier",
					"changedFilePatterns": ["src/**"],
					"ignore": ["pkg-d"],
					"fixed": [["pkg-a", "pkg-b"]],
					"linked": [["pkg-c"]],
					"updateInternalDependencies": "minor",
					"privatePackages": {"version": true, "tag": true},
					"commit": false,
					"changelog": ["@changesets/changelog-github", {"repo": "dokimasia/ergon", "disableThanks": true}],
					"___experimentalUnsafeOptions_WILL_CHANGE_IN_PATCH": {
						"onlyUpdatePeerDependentsWhenOutOfRange": true,
						"updateInternalDependents": "always"
					}
				}`,
				want: func() release.Config {
					return release.Config{
						Changelog: release.Changelog{
							Format: release.ChangelogGitHub, Repo: "dokimasia/ergon", DisableThanks: true,
						},
						BaseBranch:                 "trunk",
						Access:                     "public",
						UpdateInternalDependencies: version.BumpMinor,
						UpdateInternalDependents:   release.DependentsAlways,
						ChangedFilePatterns:        []string{"src/**"},
						Ignore:                     []string{"pkg-d"},
						Fixed:                      [][]string{{"pkg-a", "pkg-b"}},
						Linked:                     [][]string{{"pkg-c"}},
						PrivateVersion:             true,
						PrivateTag:                 true,

						OnlyUpdatePeerDependentsWhenOutOfRange: true,
					}
				},
			},
			{
				name:   "writes no changelog for changelog false",
				config: `{"changelog": false}`,
				want:   withConfig(func(c *release.Config) { c.Changelog = release.Changelog{} }),
			},
			{
				name:   "writes the changelog of git for the module of changesets that the CLI exports",
				config: `{"changelog": "@changesets/changelog-git"}`,
				want:   defaultConfig,
			},
			{
				name:   "writes the changelog of git for the module of the CLI with null options",
				config: `{"changelog": ["@changesets/cli/changelog", null]}`,
				want:   defaultConfig,
			},
			{
				name:   "writes the changelog of GitHub with the repository of the workflow for no options",
				config: `{"changelog": "@changesets/changelog-github"}`,
				want: withConfig(func(c *release.Config) {
					c.Changelog = release.Changelog{Format: release.ChangelogGitHub}
				}),
			},
			{
				name:   "reads the template of the changelog of GitHub",
				config: `{"changelog": ["@changesets/changelog-github", {"template": "{summary} {ref}"}]}`,
				want: withConfig(func(c *release.Config) {
					c.Changelog = release.Changelog{Format: release.ChangelogGitHub, Template: "{summary} {ref}"}
				}),
			},
			{
				name:   "versions and tags the private packages for privatePackages true",
				config: `{"privatePackages": true}`,
				want: withConfig(func(c *release.Config) {
					c.PrivateVersion, c.PrivateTag = true, true
				}),
			},
			{
				name:   "versions the private packages for privatePackages.version alone",
				config: `{"privatePackages": {"version": true}}`,
				want:   withConfig(func(c *release.Config) { c.PrivateVersion = true }),
			},
			{
				name:   "ignores the packages that a glob matches and a later glob after ! does not remove",
				config: `{"ignore": ["pkg-*", "!pkg-b"]}`,
				want:   withConfig(func(c *release.Config) { c.Ignore = []string{"pkg-a", "pkg-c", "pkg-d"} }),
			},
			{
				name:   "ignores a package that a glob after a glob after ! admits again",
				config: `{"ignore": ["pkg-*", "!pkg-b", "pkg-b"]}`,
				want: withConfig(func(c *release.Config) {
					c.Ignore = []string{"pkg-a", "pkg-b", "pkg-c", "pkg-d"}
				}),
			},
			{
				name:   "warns of a name and a glob of a group that match no package",
				config: `{"fixed": [["pkg-a", "pkg-z"]], "linked": [["@scope/*", "!pkg-*"]]}`,
				want: withConfig(func(c *release.Config) {
					c.Fixed = [][]string{{"pkg-a"}}
					c.Linked = [][]string{nil}
					c.Warnings = []string{
						`fixed: the package or glob "pkg-z" matches no package`,
						`linked: the package or glob "@scope/*" matches no package`,
						`linked: the package or glob "!pkg-*" matches no package`,
					}
				}),
			},
			{
				name:   "adds the packages that share the source of their version as a fixed group",
				config: `{"fixed": [["pkg-a"]]}`,
				setup: func(s *state) {
					s.pkgs[s.at("pkg-b")].Source = "Cargo.toml"
					s.pkgs[s.at("pkg-c")].Source = "Cargo.toml"
					s.pkgs[s.at("pkg-d")].Source = "pkg-d/Cargo.toml"
				},
				want: withConfig(func(c *release.Config) { c.Fixed = [][]string{{"pkg-a"}, {"pkg-b", "pkg-c"}} }),
			},
			{
				name:   "accepts an ignored package that a released private package requires",
				config: `{"ignore": ["pkg-a"], "privatePackages": {"version": true}}`,
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "1.0.0")
					s.pkgs[s.at("pkg-b")].Private = true
				},
				want: withConfig(func(c *release.Config) {
					c.Ignore = []string{"pkg-a"}
					c.PrivateVersion = true
				}),
			},
			{
				name:   "accepts a skipped package that another skipped package requires",
				config: `{"ignore": ["pkg-a", "pkg-b"]}`,
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "1.0.0")
				},
				want: withConfig(func(c *release.Config) { c.Ignore = []string{"pkg-a", "pkg-b"} }),
			},
			{
				name:   "accepts a skipped package that a package requires in the dev section",
				config: `{"ignore": ["pkg-a"]}`,
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindDev, "1.0.0")
				},
				want: withConfig(func(c *release.Config) { c.Ignore = []string{"pkg-a"} }),
			},
			{
				name:   "accepts a private package that a package requires while privatePackages.version is true",
				config: `{"privatePackages": {"version": true}}`,
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "1.0.0")
					s.pkgs[s.at("pkg-a")].Private = true
				},
				want: withConfig(func(c *release.Config) { c.PrivateVersion = true }),
			},
			{
				name:   "accepts a changelog with a toolchain that records versions in changelogs",
				config: `{}`,
				setup:  func(s *state) { s.changelogVersion = true },
				want:   defaultConfig,
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := configState(t)
				if tt.setup != nil {
					tt.setup(s)
				}
				got, err := release.ParseConfig([]byte(tt.config), s.graph(t))
				assert.NoError(t, err, "ParseConfig")
				assert.Equal(t, got, tt.want(), "the configuration")
			})
		}

		invalid := []struct {
			setup  func(s *state)
			name   string
			config string
			want   string
		}{
			{name: "returns ErrConfig for no JSON", config: `{`, want: "unexpected end of JSON input"},
			{name: "returns ErrConfig for data after the object", config: `{} {}`, want: "invalid character"},
			{name: "returns ErrConfig for a key of another type", config: `{"baseBranch": 5}`, want: "baseBranch"},
			{name: "returns ErrConfig for an unknown access", config: `{"access": "secret"}`, want: `access "secret"`},
			{
				name:   "returns ErrConfig for updateInternalDependencies major",
				config: `{"updateInternalDependencies": "major"}`,
				want:   `updateInternalDependencies "major"`,
			},
			{
				name: "returns ErrConfig for an unknown updateInternalDependents",
				config: `{"___experimentalUnsafeOptions_WILL_CHANGE_IN_PATCH": ` +
					`{"updateInternalDependents": "sometimes"}}`,
				want: `updateInternalDependents "sometimes"`,
			},
			{
				name:   "returns ErrConfig for a changelog module of JavaScript",
				config: `{"changelog": "./changelog.js"}`,
				want:   `changelog "./changelog.js", which is none of`,
			},
			{name: "returns ErrConfig for a changelog of a number", config: `{"changelog": 5}`, want: "changelog 5"},
			{name: "returns ErrConfig for a changelog of null", config: `{"changelog": null}`, want: "changelog null"},
			{
				name:   "returns ErrConfig for a changelog without options",
				config: `{"changelog": ["@changesets/cli/changelog"]}`,
				want:   "neither false, a module nor a module and its options",
			},
			{
				name:   "returns ErrConfig for options of GitHub of another type",
				config: `{"changelog": ["@changesets/changelog-github", {"repo": 5}]}`,
				want:   "the options of changelog",
			},
			{name: "returns ErrConfig for commit true", config: `{"commit": true}`, want: "commit true"},
			{
				name:   "returns ErrConfig for a commit module",
				config: `{"commit": "@changesets/cli/commit"}`,
				want:   "ergon release version commits nothing",
			},
			{
				name:   "returns ErrConfig for privatePackages of a string",
				config: `{"privatePackages": "yes"}`,
				want:   `privatePackages "yes"`,
			},
			{
				name:   "returns ErrConfig for privatePackages of null",
				config: `{"privatePackages": null}`,
				want:   "privatePackages null",
			},
			{
				name:   "returns ErrConfig for a package of two fixed groups",
				config: `{"fixed": [["pkg-a", "pkg-b"], ["pkg-*"]]}`,
				want:   `the package "pkg-a", which two fixed groups name`,
			},
			{
				name:   "returns ErrConfig for a package of two linked groups",
				config: `{"linked": [["pkg-a"], ["pkg-a", "pkg-c"]]}`,
				want:   `the package "pkg-a", which two linked groups name`,
			},
			{
				name:   "returns ErrConfig for a package of a fixed and a linked group",
				config: `{"fixed": [["pkg-a"]], "linked": [["pkg-a"]]}`,
				want:   `the package "pkg-a", which a fixed and a linked group name`,
			},
			{
				name:   "returns ErrConfig for a package of a group that shares the source of its version",
				config: `{"linked": [["pkg-b"]]}`,
				setup: func(s *state) {
					s.pkgs[s.at("pkg-b")].Source = "Cargo.toml"
					s.pkgs[s.at("pkg-c")].Source = "Cargo.toml"
				},
				want: `the package "pkg-b", which shares the version of its manifest`,
			},
			{
				name:   "returns ErrConfig for a released package that requires an ignored package",
				config: `{"ignore": ["pkg-a"]}`,
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindPeer, "1.0.0")
				},
				want: `"pkg-b" requires the skipped package "pkg-a"`,
			},
			{
				name:   "returns ErrConfig for a released package that requires a private package",
				config: `{}`,
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "1.0.0")
					s.pkgs[s.at("pkg-a")].Private = true
				},
				want: `"pkg-b" requires the skipped package "pkg-a"`,
			},
			{
				name:   "returns ErrConfig for privatePackages.tag without privatePackages.version",
				config: `{"privatePackages": {"tag": true}}`,
				want:   "privatePackages.tag, which privatePackages.version requires",
			},
			{
				name:   "returns ErrConfig for changelog false with a toolchain that records versions in changelogs",
				config: `{"changelog": false}`,
				setup:  func(s *state) { s.changelogVersion = true },
				want:   `changelog false, which "pkg-a" cannot release under`,
			},
			{
				name:   "returns ErrConfig with every rule that the configuration breaks",
				config: `{"fixed": [["pkg-a"], ["pkg-a"]], "privatePackages": {"tag": true}}`,
				want:   "two fixed groups name\nrelease: invalid .changeset/config.json: privatePackages.tag",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := configState(t)
				if tt.setup != nil {
					tt.setup(s)
				}
				_, err := release.ParseConfig([]byte(tt.config), s.graph(t))
				assert.ErrorIs(t, err, release.ErrConfig, "ParseConfig")
				assert.Contains(t, err.Error(), tt.want, "the error")
			})
		}
	})
}

// configState returns the packages of the cases of the configuration for the test tb: pkg-a to
// pkg-d at 1.0.0.
func configState(tb testing.TB) *state {
	tb.Helper()
	s := blankState(tb)
	for _, name := range []string{"pkg-a", "pkg-b", "pkg-c", "pkg-d"} {
		s.add(name, "1.0.0")
	}
	return s
}

// defaultConfig returns the configuration that changesets 4.0.1 reads from an empty object.
func defaultConfig() release.Config {
	return release.Config{
		Changelog:                  release.Changelog{Format: release.ChangelogGit},
		BaseBranch:                 "main",
		Access:                     "restricted",
		UpdateInternalDependencies: version.BumpPatch,
		UpdateInternalDependents:   release.DependentsOutOfRange,
		ChangedFilePatterns:        []string{"**"},
	}
}

// withConfig returns a function that returns the default configuration with change applied.
func withConfig(change func(c *release.Config)) func() release.Config {
	return func() release.Config {
		c := defaultConfig()
		change(&c)
		return c
	}
}
