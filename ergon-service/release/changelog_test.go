// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
)

// The repository, the commit, the pull request and the author of the tests of
// @changesets/changelog-github, and the links that its mock of GitHub returns.
const (
	emotion    = "emotion-js/emotion"
	commitSHA  = "a085003"
	commitLink = "[`a085003`](https://github.com/emotion-js/emotion/commit/a085003)"
	pullLink   = "[#1613](https://github.com/emotion-js/emotion/pull/1613)"
	authorLink = "[@Andarist](https://github.com/Andarist)"

	// loneSHA is a commit that no pull request merged.
	loneSHA = "c0ffee1"

	// loneLink is the link of loneSHA.
	loneLink = "[`c0ffee1`](https://github.com/emotion-js/emotion/commit/c0ffee1)"

	// failingSHA is a commit whose links the host fails to return.
	failingSHA = "deadbee"
)

// errHost is the error of a host that fails.
var errHost = errors.New("host failed")

// gitHub is the host of the tests of @changesets/changelog-github: the commit a085003, which the
// pull request 1613 of Andarist merged, and the commit c0ffee1 without a pull request. It fails
// for the commit deadbee, and for every call when fail is set.
type gitHub struct {
	// fail makes every call return errHost.
	fail bool
}

var _ release.Forge = gitHub{}

// CommitLinks returns the links of a085003 and c0ffee1 in emotion-js/emotion, and empty links for
// any other commit.
func (h gitHub) CommitLinks(_ context.Context, repo, sha string) (string, string, string, error) {
	switch {
	case h.fail || sha == failingSHA:
		return "", "", "", errHost
	case repo == emotion && sha == commitSHA:
		return commitLink, pullLink, authorLink, nil
	case repo == emotion && sha == loneSHA:
		return loneLink, "", "", nil
	}
	return "", "", "", nil
}

// PullLinks returns the links of the pull request 1613 in emotion-js/emotion, and empty links for
// any other pull request.
func (h gitHub) PullLinks(_ context.Context, repo string, number int) (string, string, string, error) {
	switch {
	case h.fail:
		return "", "", "", errHost
	case repo == emotion && number == 1613:
		return pullLink, commitLink, authorLink, nil
	}
	return "", "", "", nil
}

func TestChangelog(t *testing.T) {
	t.Parallel()

	t.Run("Entries", func(t *testing.T) {
		t.Parallel()

		github := release.Changelog{Format: release.ChangelogGitHub, Repo: emotion}
		templated := release.Changelog{Format: release.ChangelogGitHub, Repo: emotion, Template: "\n- {summary} {ref}"}
		host := release.Host{Forge: gitHub{}}
		standard := "- " + pullLink + " " + commitLink + " Thanks " + authorLink + "! - something"

		lines := []struct {
			host    release.Host
			name    string
			summary string
			commit  string
			want    string
			format  release.Changelog
		}{
			{
				name:    "writes the summary of a changeset without a commit",
				format:  release.Changelog{Format: release.ChangelogGit},
				summary: "something",
				want:    "- something",
			},
			{
				name:    "writes the short commit that added a changeset before its summary",
				format:  release.Changelog{Format: release.ChangelogGit},
				summary: "something",
				commit:  commitSHA + "9f2b5e6c1d4e2a7b8c9d0e1f2a3b4c5d6e7f",
				want:    "- a085003: something",
			},
			{
				name:    "indents each further line of a summary by two spaces",
				format:  release.Changelog{Format: release.ChangelogGit},
				summary: "Random stuff\n\nget it while it's hot!",
				want:    "- Random stuff\n  \n  get it while it's hot!",
			},
			{
				name:    "reads the repository of GitHub from the host for options without one",
				format:  release.Changelog{Format: release.ChangelogGitHub},
				host:    release.Host{Forge: gitHub{}, Repo: emotion},
				summary: "something",
				commit:  commitSHA,
				want:    standard,
			},
			{
				name:    "reads the repository of GitHub from the options before the host",
				format:  github,
				host:    release.Host{Forge: gitHub{}, Repo: "other/repo"},
				summary: "something",
				commit:  commitSHA,
				want:    standard,
			},
			{
				name:    "links the commit of a summary that names one",
				format:  github,
				host:    host,
				summary: "something\ncommit: a085003",
				commit:  "wrongcommit",
				want:    standard,
			},
			{
				name:    "links the commit of a summary that names a pull request and a commit",
				format:  github,
				host:    host,
				summary: "something\npr: #1613\ncommit: c0ffee1abcdef",
				commit:  commitSHA,
				want: "- " + pullLink + " [`c0ffee1`](https://github.com/emotion-js/emotion/commit/c0ffee1abcdef) Thanks " +
					authorLink + "! - something",
			},
			{
				name:    "thanks the author that a summary names with an at sign",
				format:  github,
				host:    host,
				summary: "something\nauthor: @other",
				commit:  commitSHA,
				want:    "- " + pullLink + " " + commitLink + " Thanks [@other](https://github.com/other)! - something",
			},
			{
				name:    "thanks the user that a summary names without an at sign",
				format:  github,
				host:    host,
				summary: "something\nuser: other",
				commit:  commitSHA,
				want:    "- " + pullLink + " " + commitLink + " Thanks [@other](https://github.com/other)! - something",
			},
			{
				name:    "thanks every author that a summary names",
				format:  github,
				host:    host,
				summary: "something\nauthor: @Andarist\nauthor: @mitchellhamilton",
				commit:  commitSHA,
				want: "- " + pullLink + " " + commitLink + " Thanks " + authorLink +
					", [@mitchellhamilton](https://github.com/mitchellhamilton)! - something",
			},
			{
				name:    "thanks nobody under disableThanks",
				format:  release.Changelog{Format: release.ChangelogGitHub, Repo: emotion, DisableThanks: true},
				host:    host,
				summary: "something\nauthor: @Andarist",
				commit:  commitSHA,
				want:    "- " + pullLink + " " + commitLink + " - something",
			},
			{
				name:    "writes the summary alone for a changeset without a commit",
				format:  github,
				host:    host,
				summary: "something",
				want:    "- something",
			},
			{
				name:    "links each reference to an issue",
				format:  github,
				host:    host,
				summary: "something\nfixes #1234 and #5678",
				commit:  commitSHA,
				want: standard + "\n  fixes [#1234](https://github.com/emotion-js/emotion/issues/1234) and " +
					"[#5678](https://github.com/emotion-js/emotion/issues/5678)",
			},
			{
				name:    "keeps a link to an issue as it is",
				format:  github,
				host:    host,
				summary: "something\nsee [#1234](https://github.com/emotion-js/emotion/issues/1234)",
				commit:  commitSHA,
				want:    standard + "\n  see [#1234](https://github.com/emotion-js/emotion/issues/1234)",
			},
			{
				name:    "keeps a reference to an issue inside the text of a link",
				format:  github,
				host:    host,
				summary: "something\nsee [fix for #99](https://example.com)",
				commit:  commitSHA,
				want:    standard + "\n  see [fix for #99](https://example.com)",
			},
			{
				name:    "keeps a reference after a word character",
				format:  github,
				host:    host,
				summary: "something\nfoo#123",
				commit:  commitSHA,
				want:    standard + "\n  foo#123",
			},
			{
				name:    "keeps the reference #0",
				format:  github,
				host:    host,
				summary: "something\nsee #0",
				commit:  commitSHA,
				want:    standard + "\n  see #0",
			},
			{
				name:    "links a reference at the start of a line",
				format:  github,
				host:    host,
				summary: "something\n#42 was fixed",
				commit:  commitSHA,
				want:    standard + "\n  [#42](https://github.com/emotion-js/emotion/issues/42) was fixed",
			},
			{
				name:    "links a reference after punctuation",
				format:  github,
				host:    host,
				summary: "something\nfixed (#99)",
				commit:  commitSHA,
				want:    standard + "\n  fixed ([#99](https://github.com/emotion-js/emotion/issues/99))",
			},
			{
				name:    "links a bare reference beside a link",
				format:  github,
				host:    host,
				summary: "something\nfixes [#1](https://github.com/emotion-js/emotion/issues/1) and #2",
				commit:  commitSHA,
				want: standard + "\n  fixes [#1](https://github.com/emotion-js/emotion/issues/1) and " +
					"[#2](https://github.com/emotion-js/emotion/issues/2)",
			},
			{
				name:    "links a reference before a full stop",
				format:  github,
				host:    host,
				summary: "something\nthis fixes #42.",
				commit:  commitSHA,
				want:    standard + "\n  this fixes [#42](https://github.com/emotion-js/emotion/issues/42).",
			},
			{
				name:    "renders the summary and the pull request of a template",
				format:  templated,
				host:    host,
				summary: "fix the thing",
				commit:  commitSHA,
				want:    "- fix the thing (" + pullLink + ")",
			},
			{
				name:    "indents the further lines of a summary after a template",
				format:  templated,
				host:    host,
				summary: "first line\nsecond line",
				commit:  commitSHA,
				want:    "- first line (" + pullLink + ")\n  second line",
			},
			{
				name:    "renders the commit as the reference of a template without a pull request",
				format:  templated,
				host:    host,
				summary: "fix the thing",
				commit:  loneSHA,
				want:    "- fix the thing (" + loneLink + ")",
			},
			{
				name:    "trims the space that an empty token leaves at the end of a template",
				format:  templated,
				host:    host,
				summary: "fix the thing",
				want:    "- fix the thing",
			},
			{
				name: "renders the authors of a template",
				format: release.Changelog{
					Format: release.ChangelogGitHub, Repo: emotion, Template: "\n- {summary} Thanks {authors}!",
				},
				host:    host,
				summary: "fix the thing",
				commit:  commitSHA,
				want:    "- fix the thing Thanks " + authorLink + "!",
			},
			{
				name: "renders every token of a template",
				format: release.Changelog{
					Format: release.ChangelogGitHub, Repo: emotion,
					Template: "\n- {pull} {commit} Thanks {authors}! - {summary}",
				},
				host:    host,
				summary: "fix the thing",
				commit:  commitSHA,
				want:    "- " + pullLink + " " + commitLink + " Thanks " + authorLink + "! - fix the thing",
			},
			{
				name:    "links every reference of the summary of a template",
				format:  release.Changelog{Format: release.ChangelogGitHub, Repo: emotion, Template: "\n- {summary}"},
				host:    host,
				summary: "did a thing (fixes #99) and also #1234",
				commit:  commitSHA,
				want: "- did a thing (fixes [#99](https://github.com/emotion-js/emotion/issues/99)) and also " +
					"[#1234](https://github.com/emotion-js/emotion/issues/1234)",
			},
		}
		for _, tt := range lines {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := blankState(t)
				s.add("pkg", "1.0.0")
				s.sets = []changeset.Changeset{{ID: "some-id", Summary: tt.summary, Releases: []changeset.Release{
					{Name: "pkg", Bump: minor},
				}}}
				cfg := defaultConfig()
				cfg.Changelog = tt.format
				got := entries(t, s, &cfg, map[string]string{"some-id": tt.commit}, tt.host)
				want := []release.Entry{{Name: "pkg", Text: "## 1.1.0\n\n### Minor Changes\n\n" + tt.want}}
				assert.Equal(t, got, want, "the entries")
			})
		}

		for _, keyword := range []string{"pr", "pull request", "pull"} {
			for _, hash := range []string{"#", ""} {
				for _, commit := range []string{commitSHA, "wrongcommit", ""} {
					name := "links the pull request of a summary that names it with " + keyword + " " + hash +
						" after the commit " + commit
					t.Run(name, func(t *testing.T) {
						t.Parallel()
						s := blankState(t)
						s.add("pkg", "1.0.0")
						s.sets = []changeset.Changeset{{
							ID: "some-id", Summary: "something\n" + keyword + ": " + hash + "1613",
							Releases: []changeset.Release{{Name: "pkg", Bump: minor}},
						}}
						cfg := defaultConfig()
						cfg.Changelog = github
						got := entries(t, s, &cfg, map[string]string{"some-id": commit}, host)
						want := []release.Entry{{Name: "pkg", Text: "## 1.1.0\n\n### Minor Changes\n\n" + standard}}
						assert.Equal(t, got, want, "the entries")
					})
				}
			}
		}

		dependencies := []struct {
			host   release.Host
			name   string
			commit string
			want   []release.Entry
			format release.Changelog
			minor  bool
		}{
			{
				name:   "lists the requirements that a release rewrites under Patch Changes",
				format: release.Changelog{Format: release.ChangelogGit},
				want: []release.Entry{
					{Name: "pkg-a", Text: "## 1.0.4\n\n### Patch Changes\n\n- Hey, let's have fun with testing!\n" +
						"- Updated dependencies\n  - pkg-b@1.2.1"},
					{Name: "pkg-b", Text: "## 1.2.1\n\n### Patch Changes\n\n- Hey, let's have fun with testing!\n" +
						"- Updated dependencies\n  - pkg-a@1.0.4"},
				},
			},
			{
				name:   "lists no requirement that still selects a version below updateInternalDependencies",
				format: release.Changelog{Format: release.ChangelogGit},
				minor:  true,
				want: []release.Entry{
					{Name: "pkg-a", Text: "## 1.0.4\n\n### Patch Changes\n\n- Hey, let's have fun with testing!"},
					{Name: "pkg-b", Text: "## 1.2.1\n\n### Patch Changes\n\n- Hey, let's have fun with testing!"},
				},
			},
			{
				name:   "writes the short commit of each changeset of a rewritten requirement",
				format: release.Changelog{Format: release.ChangelogGit},
				commit: commitSHA,
				want: []release.Entry{
					{
						Name: "pkg-a",
						Text: "## 1.0.4\n\n### Patch Changes\n\n- a085003: Hey, let's have fun with testing!\n" +
							"- Updated dependencies [a085003]\n  - pkg-b@1.2.1",
					},
					{
						Name: "pkg-b",
						Text: "## 1.2.1\n\n### Patch Changes\n\n- a085003: Hey, let's have fun with testing!\n" +
							"- Updated dependencies [a085003]\n  - pkg-a@1.0.4",
					},
				},
			},
			{
				name:   "links the commit of each changeset of a rewritten requirement on GitHub",
				format: github,
				host:   host,
				commit: commitSHA,
				want: []release.Entry{
					{
						Name: "pkg-a",
						Text: "## 1.0.4\n\n### Patch Changes\n\n- " + pullLink + " " + commitLink + " Thanks " +
							authorLink + "! - Hey, let's have fun with testing!\n- Updated dependencies [" + commitLink +
							"]:\n  - pkg-b@1.2.1",
					},
					{
						Name: "pkg-b",
						Text: "## 1.2.1\n\n### Patch Changes\n\n- " + pullLink + " " + commitLink + " Thanks " +
							authorLink + "! - Hey, let's have fun with testing!\n- Updated dependencies [" + commitLink +
							"]:\n  - pkg-a@1.0.4",
					},
				},
			},
			{
				name:   "links no commit of a changeset without one on GitHub",
				format: github,
				host:   host,
				want: []release.Entry{
					{Name: "pkg-a", Text: "## 1.0.4\n\n### Patch Changes\n\n- Hey, let's have fun with testing!\n" +
						"- Updated dependencies []:\n  - pkg-b@1.2.1"},
					{Name: "pkg-b", Text: "## 1.2.1\n\n### Patch Changes\n\n- Hey, let's have fun with testing!\n" +
						"- Updated dependencies []:\n  - pkg-a@1.0.4"},
				},
			},
			{
				name:   "writes the short commit of a changeset that GitHub does not have",
				format: release.Changelog{Format: release.ChangelogGitHub, Repo: emotion, Template: "\n- {summary}"},
				host:   host,
				commit: "abc1234ffff",
				want: []release.Entry{
					{Name: "pkg-a", Text: "## 1.0.4\n\n### Patch Changes\n\n- Hey, let's have fun with testing!\n" +
						"- Updated dependencies [`abc1234`]:\n  - pkg-b@1.2.1"},
					{Name: "pkg-b", Text: "## 1.2.1\n\n### Patch Changes\n\n- Hey, let's have fun with testing!\n" +
						"- Updated dependencies [`abc1234`]:\n  - pkg-a@1.0.4"},
				},
			},
		}
		for _, tt := range dependencies {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := blankState(t)
				s.add("pkg-a", "1.0.3")
				s.add("pkg-b", "1.2.0")
				s.require("pkg-a", "pkg-b", workspace.KindRuntime, "~1.2.0")
				s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^1.0.3")
				s.changeset("quick-lions-devour", r("pkg-a", patch), r("pkg-b", patch))
				s.sets[0].Summary = "Hey, let's have fun with testing!"
				cfg := defaultConfig()
				cfg.Changelog = tt.format
				if tt.minor {
					cfg.UpdateInternalDependencies = minor
				}
				got := entries(t, s, &cfg, map[string]string{"quick-lions-devour": tt.commit}, tt.host)
				assert.Equal(t, got, tt.want, "the entries")
			})
		}

		t.Run("lists a requirement that leaves its range below updateInternalDependencies", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.3")
			s.add("pkg-b", "1.2.0")
			s.add("pkg-c", "2.0.0")
			s.require("pkg-a", "pkg-b", workspace.KindRuntime, "~1.2.0")
			s.require("pkg-b", "pkg-c", workspace.KindRuntime, "2.0.0")
			s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^1.0.3")
			s.require("pkg-c", "pkg-a", workspace.KindRuntime, "^1.0.3")
			s.changeset("quick-lions-devour", r("pkg-a", patch), r("pkg-b", patch), r("pkg-c", patch))
			s.sets[0].Summary = "Hey, let's have fun with testing!"
			cfg := defaultConfig()
			cfg.UpdateInternalDependencies = minor
			got := entries(t, s, &cfg, nil, release.Host{})
			assert.Equal(t, got[1], release.Entry{Name: "pkg-b", Text: "## 1.2.1\n\n### Patch Changes\n\n" +
				"- Hey, let's have fun with testing!\n- Updated dependencies\n  - pkg-c@2.0.1"}, "the entry of pkg-b")
		})

		t.Run("lists a rewritten requirement without changesets by itself", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.0")
			s.add("pkg-b", "1.0.0")
			s.require("pkg-a", "pkg-b", workspace.KindRuntime, "1.0.0")
			cfg := defaultConfig()
			plan := release.Plan{
				Changesets: []changeset.Changeset{{
					ID: "quick-lions-devour", Summary: "Hey, let's have fun with testing!",
					Releases: []changeset.Release{{Name: "pkg-a", Bump: minor}},
				}},
				Releases: []release.Release{
					rel(t, "pkg-a", minor, "1.0.0", "1.1.0", "quick-lions-devour"),
					rel(t, "pkg-b", major, "1.0.0", "2.0.0"),
				},
			}
			got, err := release.Entries(t.Context(), s.graph(t), &cfg, &plan, nil, release.Host{})
			assert.NoError(t, err, "Entries")
			assert.Equal(t, got, []release.Entry{
				{Name: "pkg-a", Text: "## 1.1.0\n\n### Minor Changes\n\n- Hey, let's have fun with testing!\n\n" +
					"### Patch Changes\n\n- pkg-b@2.0.0"},
				{Name: "pkg-b", Text: "## 2.0.0\n\nNo changes in this release."},
			}, "the entries")
		})

		peers := []struct {
			name string
			want string
			only bool
		}{
			{
				name: "lists a peer requirement that selects the new version",
				want: "## 1.0.1\n\n### Patch Changes\n\n- Use b.\n- Updated dependencies\n  - pkg-b@1.2.1",
			},
			{
				name: "lists no peer requirement that selects the new version under onlyUpdatePeerDependentsWhenOutOfRange",
				only: true,
				want: "## 1.0.1\n\n### Patch Changes\n\n- Use b.",
			},
		}
		for _, tt := range peers {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := blankState(t)
				s.add("pkg-a", "1.0.0")
				s.add("pkg-b", "1.2.0")
				s.require("pkg-a", "pkg-b", workspace.KindPeer, "^1.2.0")
				s.changeset("use-b", r("pkg-a", patch), r("pkg-b", patch))
				s.sets[0].Summary = "Use b."
				cfg := defaultConfig()
				cfg.OnlyUpdatePeerDependentsWhenOutOfRange = tt.only
				got := entries(t, s, &cfg, nil, release.Host{})
				assert.Equal(t, got[0], release.Entry{Name: "pkg-a", Text: tt.want}, "the entry of pkg-a")
			})
		}

		t.Run("lists a require line that pins an older version of a module outside the plan", func(t *testing.T) {
			t.Parallel()
			s := goState(t)
			s.add("example.com/a", "1.0.0")
			s.add("example.com/b", "1.0.0")
			s.add("example.com/c", "1.0.0")
			s.require("example.com/b", "example.com/a", workspace.KindRuntime, "v0.9.0")
			s.changeset("fix-it", r("example.com/b", patch))
			s.sets[0].Summary = "Fix it."
			cfg := defaultConfig()
			got := entries(t, s, &cfg, nil, release.Host{})
			assert.Equal(t, got, []release.Entry{{
				Name: "example.com/b", Text: "## 1.0.1\n\n### Patch Changes\n\n- Fix it.\n- example.com/a@1.0.0",
			}}, "the entries")
		})

		t.Run("returns no entry for a configuration without a changelog", func(t *testing.T) {
			t.Parallel()
			cfg := defaultConfig()
			cfg.Changelog = release.Changelog{}
			got := entries(t, newState(t), &cfg, nil, release.Host{})
			assert.Nil(t, got, "the entries")
		})

		t.Run("returns no entry for a release at none", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.0")
			s.changeset("docs", r("pkg-a", none))
			cfg := defaultConfig()
			got := entries(t, s, &cfg, nil, release.Host{})
			assert.Nil(t, got, "the entries")
		})

		failures := []struct {
			want   error
			host   release.Host
			name   string
			commit string
			format release.Changelog
			text   string
		}{
			{
				name:   "returns ErrChangelog for the changelog of GitHub without a host",
				format: github,
				want:   release.ErrChangelog,
				text:   "the changelog of GitHub, which needs the host of the repository",
			},
			{
				name:   "returns ErrChangelog for a repository that is not owner/name",
				format: release.Changelog{Format: release.ChangelogGitHub},
				host:   release.Host{Forge: gitHub{}},
				want:   release.ErrChangelog,
				text:   `the repository "", which is not owner/name`,
			},
			{
				name: "returns ErrChangelog for a token that a template does not define",
				format: release.Changelog{
					Format: release.ChangelogGitHub, Repo: emotion, Template: "- {summary} {nope}",
				},
				host: host,
				want: release.ErrChangelog,
				text: "the token {nope} of the template",
			},
			{
				name:   "returns the error of the host for the links of a commit",
				format: github,
				host:   release.Host{Forge: gitHub{fail: true}},
				commit: commitSHA,
				want:   errHost,
				text:   "the changelog of pkg-a",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := newState(t)
				cfg := defaultConfig()
				cfg.Changelog = tt.format
				plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
				assert.NoError(t, err, "NewPlan")
				_, err = release.Entries(t.Context(), s.graph(t), &cfg, &plan,
					map[string]string{"strange-words-combine": tt.commit}, tt.host)
				assert.ErrorIs(t, err, tt.want, "Entries")
				assert.Contains(t, err.Error(), tt.text, "the error")
			})
		}

		t.Run("returns ErrChangelog for a repository that is not owner/name in a release without changesets",
			func(t *testing.T) {
				t.Parallel()
				s := blankState(t)
				s.add("pkg-a", "1.0.0")
				cfg := defaultConfig()
				cfg.Changelog = release.Changelog{Format: release.ChangelogGitHub}
				plan := release.Plan{Releases: []release.Release{rel(t, "pkg-a", patch, "1.0.0", "1.0.1")}}
				_, err := release.Entries(t.Context(), s.graph(t), &cfg, &plan, nil, host)
				assert.ErrorIs(t, err, release.ErrChangelog, "Entries")
			})

		t.Run("returns the error of the host for the links of a pull request", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.sets[0].Summary = "something\npr: #1613"
			cfg := defaultConfig()
			cfg.Changelog = github
			plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			_, err = release.Entries(t.Context(), s.graph(t), &cfg, &plan, nil, release.Host{Forge: gitHub{fail: true}})
			assert.ErrorIs(t, err, errHost, "Entries")
		})

		t.Run("returns ErrChangelog for a pull request past the range of an int", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.sets[0].Summary = "something\npr: #99999999999999999999"
			cfg := defaultConfig()
			cfg.Changelog = github
			plan, err := release.NewPlan(s.graph(t), &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			_, err = release.Entries(t.Context(), s.graph(t), &cfg, &plan, nil, host)
			assert.ErrorIs(t, err, release.ErrChangelog, "Entries")
		})

		t.Run("returns the error of the host for the links of a rewritten requirement", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.0")
			s.add("pkg-b", "1.0.0")
			s.require("pkg-a", "pkg-b", workspace.KindRuntime, "1.0.0")
			s.changeset("break-b", r("pkg-b", major))
			cfg := defaultConfig()
			cfg.Changelog = release.Changelog{Format: release.ChangelogGitHub, Repo: emotion, Template: "- {summary}"}
			plan := release.Plan{Changesets: s.sets, Releases: []release.Release{
				rel(t, "pkg-a", patch, "1.0.0", "1.0.1"),
				rel(t, "pkg-b", major, "1.0.0", "2.0.0", "break-b"),
			}}
			_, err := release.Entries(t.Context(), s.graph(t), &cfg, &plan, map[string]string{"break-b": failingSHA},
				host)
			assert.ErrorIs(t, err, errHost, "Entries")
			assert.Contains(t, err.Error(), "the changelog of pkg-a", "the error")
		})
	})

	t.Run("Section", func(t *testing.T) {
		t.Parallel()

		const changelog = "# pkg\n\n## 1.1.0\n\n### Minor Changes\n\n- a\n\n## 1.0.0\n\n### Patch Changes\n\n- b\n"
		tests := []struct {
			name    string
			give    string
			version string
			want    string
			highest version.Bump
			found   bool
		}{
			{
				name: "returns the section of a version up to the next heading of its depth", give: changelog,
				version: "1.1.0", want: "### Minor Changes\n\n- a", highest: minor, found: true,
			},
			{
				name: "returns the section of the last version up to the end", give: changelog,
				version: "1.0.0", want: "### Patch Changes\n\n- b", highest: minor, found: true,
			},
			{
				name:    "returns the highest level of the headings up to the end of the section",
				give:    "## 2.0.0\n\n### Major Changes\n\n- x\n\n## 1.0.0\n\n### Patch Changes\n\n- y\n",
				version: "1.0.0", want: "### Patch Changes\n\n- y", highest: major, found: true,
			},
			{
				name:    "skips the headings inside a fenced code block",
				give:    "## 1.1.0\n\n```md\n## 1.0.0\n```\n\n## 1.0.0\n\n- b\n",
				version: "1.1.0", want: "```md\n## 1.0.0\n```", highest: none, found: true,
			},
			{
				name:    "returns the rest of a changelog after a code block that does not close",
				give:    "## 1.1.0\n\n- a\n\n```\n## 1.0.0\n",
				version: "1.1.0", want: "- a\n\n```\n## 1.0.0", highest: none, found: true,
			},
			{
				name: "reports false for a version without a heading", give: changelog,
				version: "2.0.0", highest: minor, found: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, highest, found := release.Section([]byte(tt.give), parse(t, tt.version))
				assert.Equal(t, got, tt.want, "the section")
				assert.Equal(t, highest, tt.highest, "the highest level")
				assert.Equal(t, found, tt.found, "whether the changelog has the version")
			})
		}
	})
}

// entries returns the changelog entries of the plan of the changesets of s under c, with commits
// and h, for the test tb.
func entries(tb testing.TB, s *state, c *release.Config, commits map[string]string, h release.Host) []release.Entry {
	tb.Helper()
	g := s.graph(tb)
	plan, err := release.NewPlan(g, c, s.sets)
	assert.NoError(tb, err, "NewPlan")
	got, err := release.Entries(tb.Context(), g, c, &plan, commits, h)
	assert.NoError(tb, err, "Entries")
	return got
}
