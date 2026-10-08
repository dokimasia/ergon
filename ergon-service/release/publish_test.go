// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"cmp"
	"context"
	"errors"
	"path"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The commits of the cases.
const (
	commitA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	commitB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// The output of git that a terminal of a [release.GitReleaser] receives.
const (
	// tagExists is the error of git tag for a tag that the repository has.
	tagExists = "already exists"

	// newTag is the mark of git push for a tag that the remote did not have.
	newTag = "[new tag]"
)

// The errors of a releaser and of a host that fail.
var (
	errTag     = errors.New("tag failed")
	errRelease = errors.New("release failed")
)

// released is a release that a releaser created.
type released struct {
	// name is the tag of the release.
	name string

	// commit is the commit of the tag.
	commit string

	// notes are the notes of the release.
	notes string

	// prerelease reports that the release is a pre-release.
	prerelease bool
}

// releaser is a [release.Releaser] that has the tags of tags, and records each release and each
// Finish. Tag returns failTag, and Release failRelease.
type releaser struct {
	// failTag is the error of Tag, or nil.
	failTag error

	// failRelease is the error of Release, or nil.
	failRelease error

	// tags are the tags of the repository and their commits.
	tags map[string]string

	// releases are the releases that Release created, in their order.
	releases []released

	// finished counts the calls of Finish.
	finished int
}

var _ release.Releaser = (*releaser)(nil)

// Tag returns the commit of the tag name, and reports whether r has it.
func (r *releaser) Tag(_ context.Context, name string) (string, bool, error) {
	commit, ok := r.tags[name]
	return commit, ok, r.failTag
}

// Release records the release of the tag name.
func (r *releaser) Release(_ context.Context, name, commit, notes string, prerelease bool) error {
	if r.failRelease != nil {
		return r.failRelease
	}
	r.releases = append(r.releases, released{name: name, commit: commit, notes: notes, prerelease: prerelease})
	return nil
}

// Finish counts the call.
func (r *releaser) Finish(context.Context) error {
	r.finished++
	return nil
}

// tagForge is a [release.TagForge] that has the tags of tags and records each tag and each release
// that it creates. CreateTag returns failCreate.
type tagForge struct {
	// failCreate is the error of CreateTag, or nil.
	failCreate error

	// tags are the tags of the repository and their commits.
	tags map[string]string

	// calls are the calls of CreateTag and CreateRelease, each as a line.
	calls []string
}

var _ release.TagForge = (*tagForge)(nil)

// Tag returns the commit of the tag name of repo, and reports whether f has it.
func (f *tagForge) Tag(_ context.Context, repo, name string) (string, bool, error) {
	commit, ok := f.tags[repo+" "+name]
	return commit, ok, nil
}

// CreateTag records the tag name of repo at sha.
func (f *tagForge) CreateTag(_ context.Context, repo, name, sha string) error {
	if f.failCreate != nil {
		return f.failCreate
	}
	f.calls = append(f.calls, "tag "+repo+" "+name+" "+sha)
	return nil
}

// CreateRelease records the release of the tag of repo.
func (f *tagForge) CreateRelease(_ context.Context, repo, tag, body string, prerelease bool) error {
	kind := "release"
	if prerelease {
		kind = "prerelease"
	}
	f.calls = append(f.calls, kind+" "+repo+" "+tag+" "+body)
	return nil
}

func TestPublish(t *testing.T) {
	t.Parallel()

	t.Run("Publish", func(t *testing.T) {
		t.Parallel()

		t.Run("tags each entry at the commit with the section of its changelog as its notes", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			root := files.Workspace(t, files.Tree{
				path.Join(packagesDir, "pkg-a", release.ChangelogFile): files.Text("# pkg-a\n\n## 1.0.0\n\n- First.\n"),
			})
			r := &releaser{}
			got, err := release.Publish(t.Context(), root, s.graph(t), plan, commitA, "", r)
			assert.NoError(t, err, "Publish")
			assert.Equal(
				t,
				got,
				[]release.Published{{Name: "pkg-a", Version: "1.0.0"}, {Name: "pkg-b", Version: "2.0.0-rc.1"}},
				"the released packages",
			)
			assert.Equal(t, r.releases, []released{
				{name: "pkg-a@1.0.0", commit: commitA, notes: "- First."},
				{name: "pkg-b@2.0.0-rc.1", commit: commitA, prerelease: true},
			}, "the releases")
			assert.Equal(t, r.finished, 1, "the calls of Finish")
		})

		t.Run("lists a package whose tag the repository has at the commit without a new release", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			r := &releaser{tags: map[string]string{"pkg-a@1.0.0": commitA}}
			got, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, "", r)
			assert.NoError(t, err, "Publish")
			assert.Length(t, got, 2, "the released packages")
			assert.Equal(t, r.releases, []released{{name: "pkg-b@2.0.0-rc.1", commit: commitA, prerelease: true}},
				"the releases")
		})

		t.Run("returns ErrTag for a tag at another commit", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			r := &releaser{tags: map[string]string{"pkg-a@1.0.0": commitB}}
			got, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, "", r)
			assert.ErrorIs(t, err, release.ErrTag, "Publish")
			assert.Contains(t, err.Error(), "pkg-a@1.0.0 is at "+commitB+", and the release of pkg-a is at "+commitA,
				"the error")
			assert.Empty(t, got, "the released packages")
			assert.Equal(t, r.finished, 1, "the calls of Finish")
		})

		t.Run("uploads the packages of a chunk before it tags them", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.add("pkg-b", "1.0.0")
			reg := newRegistry("pkg-b@1.0.0")
			s.roles = []any{reg}
			cfg := defaultConfig()
			g := s.graph(t)
			plan, err := release.NewPublishPlan(t.Context(), g, &cfg, nil)
			assert.NoError(t, err, "NewPublishPlan")
			plan.Plan[0] = append(plan.Plan[0], release.PublishEntry{
				Kind:      release.KindPublish,
				Toolchain: npmToolchain,
				Name:      "pkg-b",
				Tag:       "pkg-b@1.0.0",
				Version:   parse(t, "1.0.0"),
			})
			r := &releaser{}
			_, err = release.Publish(t.Context(), t.TempDir(), g, &plan, commitA, "pack", r)
			assert.NoError(t, err, "Publish")
			assert.Equal(t, uploaded(reg), [][]string{{"pkg-a"}}, "the uploads")
			assert.Length(t, r.releases, 2, "the releases")
		})

		failures := []struct {
			want  error
			setup func(s *state, r *releaser)
			name  string
			kind  string
			entry string
		}{
			{
				name: "returns the error of the registry of a package",
				setup: func(s *state, _ *releaser) {
					failing := newRegistry()
					failing.failPublished = "pkg-a"
					s.roles = []any{failing}
				},
				kind: release.KindPublish,
				want: errRegistry,
			},
			{
				name: "returns the error of an upload",
				setup: func(s *state, _ *releaser) {
					failing := newRegistry()
					failing.failPublish = true
					s.roles = []any{failing}
				},
				kind: release.KindPublish,
				want: errRegistry,
			},
			{
				name:  "returns the error of the tags of the releaser",
				setup: func(_ *state, r *releaser) { r.failTag = errTag },
				kind:  release.KindTagOnly,
				want:  errTag,
			},
			{
				name:  "returns the error of a release",
				setup: func(_ *state, r *releaser) { r.failRelease = errRelease },
				kind:  release.KindTagOnly,
				want:  errRelease,
			},
			{
				name:  "returns ErrPublishPlan for an entry of a package that the repository does not have",
				setup: func(*state, *releaser) {},
				kind:  release.KindTagOnly,
				entry: "pkg-z",
				want:  release.ErrPublishPlan,
			},
			{
				name:  "returns ErrPublishPlan for an upload of a package whose toolchain has no registry",
				setup: func(*state, *releaser) {},
				kind:  release.KindPublish,
				want:  release.ErrPublishPlan,
			},
			{
				name:  "returns ErrStale for lockfiles that record other content of a package",
				setup: func(s *state, _ *releaser) { s.roles = []any{&lockfiles{stale: []string{lockA}}} },
				kind:  release.KindTagOnly,
				want:  release.ErrStale,
			},
			{
				name:  "returns the error of the lockfiles of a toolchain",
				setup: func(s *state, _ *releaser) { s.roles = []any{&lockfiles{failStale: errLockfiles}} },
				kind:  release.KindTagOnly,
				want:  errLockfiles,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := newState(t)
				r := &releaser{}
				tt.setup(s, r)
				plan := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{{{
					Kind: tt.kind, Toolchain: npmToolchain, Name: cmp.Or(tt.entry, "pkg-a"), Tag: "v1.0.0",
					Version: parse(t, "1.0.0"),
				}}}}
				_, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, "", r)
				assert.ErrorIs(t, err, tt.want, "Publish")
			})
		}

		t.Run("names the stale lockfiles and the command that rewrites them", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			s.roles = []any{&lockfiles{stale: []string{lockA}}}
			r := &releaser{}
			_, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, "", r)
			assert.ErrorIs(t, err, release.ErrStale, "Publish")
			assert.Contains(t, err.Error(), "stale lockfiles: "+lockA+", which ergon release version rewrites",
				"the error")
			assert.Empty(t, r.releases, "the releases")
		})

		t.Run("returns the error of a changelog that it cannot read", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			root := files.Workspace(t, files.Tree{
				path.Join(packagesDir, "pkg-a", release.ChangelogFile, "inner"): files.Text("x\n"),
			})
			_, err := release.Publish(t.Context(), root, s.graph(t), plan, commitA, "", &releaser{})
			assert.HasError(t, err, "Publish")
			assert.Contains(t, err.Error(), "read packages/pkg-a/CHANGELOG.md", "the error")
		})
	})

	t.Run("GitReleaser", func(t *testing.T) {
		t.Parallel()

		t.Run("creates annotated tags with their notes and pushes them in one push", func(t *testing.T) {
			t.Parallel()
			root, remote := remoteRepository(t)
			head := vcstest.Git(t, root, "rev-parse", "HEAD")
			r := &release.GitReleaser{Root: root, Remote: "origin"}
			assert.NoError(t, r.Release(t.Context(), "v1.0.0", strings.TrimSpace(head), "### Minor Changes", false),
				"Release")
			assert.NoError(t, r.Finish(t.Context()), "Finish")
			assert.Equal(t, vcstest.Git(t, root, "tag", "--list", "--format=%(contents)", "v1.0.0"),
				"### Minor Changes\n\n", "the annotation with its newline, and the newline of git tag --list")
			assert.Equal(t, vcstest.Git(t, remote, "tag", "--list"), "v1.0.0\n", "the tags of the remote")
		})

		t.Run("writes the name of the tag as the annotation of a release without notes", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			r := &release.GitReleaser{Root: root, Remote: "origin"}
			assert.NoError(t, r.Release(t.Context(), "v1.0.0", head, "", false), "Release")
			assert.Equal(t, vcstest.Git(t, root, "tag", "--list", "--format=%(contents)", "v1.0.0"), "v1.0.0\n\n",
				"the annotation with its newline, and the newline of git tag --list")
		})

		t.Run("pushes nothing without a release", func(t *testing.T) {
			t.Parallel()
			r := &release.GitReleaser{Root: t.TempDir(), Remote: "origin"}
			assert.NoError(t, r.Finish(t.Context()), "Finish")
		})

		t.Run("returns the commit of a tag of the repository", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			vcstest.Git(t, root, "tag", "v0.9.0", head)
			r := &release.GitReleaser{Root: root, Remote: "origin"}
			commit, found, err := r.Tag(t.Context(), "v0.9.0")
			assert.NoError(t, err, "Tag")
			assert.True(t, found, "whether the repository has the tag")
			assert.Equal(t, commit, head, "the commit")
		})

		t.Run("returns ErrGit for a directory outside a working tree", func(t *testing.T) {
			t.Parallel()
			r := &release.GitReleaser{Root: t.TempDir(), Remote: "origin"}
			_, _, err := r.Tag(t.Context(), "v1.0.0")
			assert.ErrorIs(t, err, vcs.ErrGit, "Tag")
		})

		t.Run("returns ErrGit for a tag that the repository has", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			vcstest.Git(t, root, "tag", "v1.0.0", head)
			r := &release.GitReleaser{Root: root, Remote: "origin"}
			assert.ErrorIs(t, r.Release(t.Context(), "v1.0.0", head, "notes", false), vcs.ErrGit, "Release")
		})

		t.Run("runs git tag with its terminal", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			vcstest.Git(t, root, "tag", "v1.0.0", head)
			var stderr strings.Builder
			r := &release.GitReleaser{Terminal: vcs.Terminal{Stderr: &stderr}, Root: root, Remote: "origin"}
			assert.ErrorIs(t, r.Release(t.Context(), "v1.0.0", head, "notes", false), vcs.ErrGit, "Release")
			assert.Contains(t, stderr.String(), tagExists, "the standard error of git tag")
		})

		t.Run("runs git push with its terminal", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			var stderr strings.Builder
			r := &release.GitReleaser{Terminal: vcs.Terminal{Stderr: &stderr}, Root: root, Remote: "origin"}
			assert.NoError(t, r.Release(t.Context(), "v1.0.0", head, "notes", false), "Release")
			assert.NoError(t, r.Finish(t.Context()), "Finish")
			assert.Contains(t, stderr.String(), newTag, "the standard error of git push")
		})
	})

	t.Run("ForgeReleaser", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the commit of a tag of the host", func(t *testing.T) {
			t.Parallel()
			r := release.ForgeReleaser{Forge: &tagForge{tags: map[string]string{"o/r v1.0.0": commitA}}, Repo: "o/r"}
			commit, found, err := r.Tag(t.Context(), "v1.0.0")
			assert.NoError(t, err, "Tag")
			assert.True(t, found, "whether the host has the tag")
			assert.Equal(t, commit, commitA, "the commit")
		})

		t.Run("creates the tag and then its release with the notes", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			assert.NoError(t, r.Release(t.Context(), "v1.0.0-rc.1", commitA, "notes", true), "Release")
			assert.NoError(t, r.Finish(t.Context()), "Finish")
			assert.Equal(t, f.calls, []string{"tag o/r v1.0.0-rc.1 " + commitA, "prerelease o/r v1.0.0-rc.1 notes"},
				"the calls")
		})

		t.Run("returns the error of the tag without a release", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{failCreate: errTag}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			assert.ErrorIs(t, r.Release(t.Context(), "v1.0.0", commitA, "notes", false), errTag, "Release")
			assert.Empty(t, f.calls, "the calls")
		})
	})
}

// publishable returns the state of newState with pkg-b at 2.0.0-rc.1, and the publish plan of tags
// of both at their versions, for the test tb.
func publishable(tb testing.TB) (*state, *release.PublishPlan) {
	tb.Helper()
	s := newState(tb)
	s.add("pkg-b", "2.0.0-rc.1")
	return s, &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{
		{tagOnly(tb, "pkg-a", "pkg-a@1.0.0", "1.0.0")},
		{tagOnly(tb, "pkg-b", "pkg-b@2.0.0-rc.1", "2.0.0-rc.1")},
	}}
}

// remoteRepository returns a working tree of git with one commit and its remote origin, a bare
// repository, for the test tb.
func remoteRepository(tb testing.TB) (string, string) {
	tb.Helper()
	root := vcstest.Repository(tb, files.Tree{"README.md": files.Text("# r\n")})
	vcstest.Commit(tb, root, "first")
	remote := tb.TempDir()
	vcstest.Git(tb, remote, "init", "--bare")
	vcstest.Git(tb, root, "remote", "add", "origin", remote)
	return root, remote
}
