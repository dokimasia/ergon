// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The commits of the cases.
const (
	commitA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	commitB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// rootDir is the directory of a package at the root of the repository.
const rootDir = "."

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

	// latest is the mark of the latest release that the publish of the release sets.
	latest forge.Latest

	// assets are the base names of the files of the assets of the release.
	assets []string

	// prerelease reports that the release is a pre-release.
	prerelease bool

	// draft reports that the release remains a draft.
	draft bool
}

// releaser is a [release.Releaser] that has the tags of tags, and records each release, each draft
// and each Finish. Tag returns failTag, and Release and Draft return failRelease.
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

// Release records the release of the tag name, with the base names of its assets and its mark of
// the latest release.
func (r *releaser) Release(
	_ context.Context, name, commit, notes string, prerelease bool, assets []string, latest forge.Latest,
) error {
	return r.record(released{name: name, commit: commit, notes: notes, latest: latest, prerelease: prerelease}, assets)
}

// Draft records the release of the tag name as a draft, with the base names of its assets.
func (r *releaser) Draft(_ context.Context, name, commit, notes string, prerelease bool, assets []string) error {
	return r.record(released{name: name, commit: commit, notes: notes, prerelease: prerelease, draft: true}, assets)
}

// record appends rel with the base names of assets to the releases of r, and returns failRelease
// in place of the record.
func (r *releaser) record(rel released, assets []string) error {
	if r.failRelease != nil {
		return r.failRelease
	}
	for _, a := range assets {
		rel.assets = append(rel.assets, filepath.Base(a))
	}
	r.releases = append(r.releases, rel)
	return nil
}

// Finish counts the call.
func (r *releaser) Finish(context.Context) error {
	r.finished++
	return nil
}

// tagForge is a [release.TagForge] that has the tags of tags and the releases of releases, and
// records each tag, draft, upload and publish that it receives. Tag returns failTag, CreateTag
// failCreate, ReleaseOf failReleaseOf, CreateDraft failDraft, and UploadAsset failUpload.
type tagForge struct {
	// failTag is the error of Tag, or nil.
	failTag error

	// failCreate is the error of CreateTag, or nil.
	failCreate error

	// failReleaseOf is the error of ReleaseOf, or nil.
	failReleaseOf error

	// failDraft is the error of CreateDraft, or nil.
	failDraft error

	// failUpload is the error of UploadAsset, or nil.
	failUpload error

	// tags are the tags of the repository and their commits, each keyed as <repo> <tag>.
	tags map[string]string

	// releases are the releases of the repository, each keyed as <repo> <tag>.
	releases map[string]forge.Release

	// calls are the calls of CreateTag, CreateDraft, UploadAsset and Publish, each as a line.
	calls []string
}

var _ release.TagForge = (*tagForge)(nil)

// Tag returns the commit of the tag name of repo, and reports whether f has it.
func (f *tagForge) Tag(_ context.Context, repo, name string) (string, bool, error) {
	if f.failTag != nil {
		return "", false, f.failTag
	}
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

// ReleaseOf returns the release of the tag of repo, and reports whether f has it.
func (f *tagForge) ReleaseOf(_ context.Context, repo, tag string) (forge.Release, bool, error) {
	if f.failReleaseOf != nil {
		return forge.Release{}, false, f.failReleaseOf
	}
	r, ok := f.releases[repo+" "+tag]
	return r, ok, nil
}

// CreateDraft records the draft of the tag of repo, and returns it with the ID 7.
func (f *tagForge) CreateDraft(_ context.Context, repo, tag, body string, prerelease bool) (forge.Release, error) {
	if f.failDraft != nil {
		return forge.Release{}, f.failDraft
	}
	kind := "draft"
	if prerelease {
		kind = "prerelease draft"
	}
	f.calls = append(f.calls, kind+" "+repo+" "+tag+" "+body)
	return forge.Release{ID: 7, Tag: tag, Draft: true}, nil
}

// UploadAsset records the upload of the base name of path to the release r.
func (f *tagForge) UploadAsset(_ context.Context, r *forge.Release, path string) error {
	if f.failUpload != nil {
		return f.failUpload
	}
	f.calls = append(f.calls, "upload "+r.Tag+" "+filepath.Base(path))
	return nil
}

// Publish records the publish of the release id of repo with its mark of the latest release.
func (f *tagForge) Publish(_ context.Context, repo string, id int64, latest forge.Latest) error {
	f.calls = append(f.calls, "publish "+repo+" "+strconv.FormatInt(id, 10)+" make_latest="+string(latest))
	return nil
}

func TestPublish(t *testing.T) {
	t.Parallel()

	t.Run("Published", func(t *testing.T) {
		t.Parallel()

		t.Run("encodes the name and the version of a draft alone", func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(release.Published{Name: "pkg-a", Version: "1.0.0", Draft: true})
			assert.NoError(t, err, "Marshal")
			assert.Equal(t, string(got), `{"name":"pkg-a","version":"1.0.0"}`, "the JSON")
		})
	})

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

		t.Run("marks the release of the root package alone as the latest release", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.add("pkg-b", "1.1.0")
			s.pkgs[s.at("pkg-a")].Dir = rootDir
			plan := &release.PublishPlan{Version: 1, Plan: [][]release.PublishEntry{
				{tagOnly(t, "pkg-a", "v1.0.0", "1.0.0")},
				{tagOnly(t, "pkg-b", "pkg-b@1.1.0", "1.1.0")},
			}}
			r := &releaser{}
			_, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, "", r)
			assert.NoError(t, err, "Publish")
			assert.Equal(t, r.releases, []released{
				{name: "v1.0.0", commit: commitA, latest: forge.LatestTrue},
				{name: "pkg-b@1.1.0", commit: commitA, latest: forge.LatestFalse},
			}, "the releases")
		})

		t.Run("does not mark a prerelease of the root package as the latest release", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			s.pkgs[s.at("pkg-b")].Dir = rootDir
			r := &releaser{}
			_, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, "", r)
			assert.NoError(t, err, "Publish")
			assert.Equal(t, r.releases, []released{
				{name: "pkg-a@1.0.0", commit: commitA, latest: forge.LatestFalse},
				{name: "pkg-b@2.0.0-rc.1", commit: commitA, latest: forge.LatestFalse, prerelease: true},
			}, "the releases")
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

		t.Run("attaches the assets of the pack directory to the release of their tag", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			dir := files.Workspace(t, files.Tree{
				path.Join(language.AssetsDir, "pkg-a@1.0.0", "pkg-a_1.0.0_linux_amd64.tar.gz"): files.Text("archive\n"),
				path.Join(language.AssetsDir, "pkg-a@1.0.0", "checksums.txt"):                  files.Text("sums\n"),
			})
			r := &releaser{}
			_, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, dir, r)
			assert.NoError(t, err, "Publish")
			assert.Equal(t, r.releases, []released{
				{
					name:   "pkg-a@1.0.0",
					commit: commitA,
					assets: []string{"checksums.txt", "pkg-a_1.0.0_linux_amd64.tar.gz"},
				},
				{name: "pkg-b@2.0.0-rc.1", commit: commitA, prerelease: true},
			}, "the releases")
		})

		t.Run("completes the release of a tag at the commit that has assets", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			dir := files.Workspace(t, files.Tree{
				path.Join(language.AssetsDir, "pkg-a@1.0.0", "checksums.txt"): files.Text("sums\n"),
			})
			r := &releaser{tags: map[string]string{"pkg-a@1.0.0": commitA, "pkg-b@2.0.0-rc.1": commitA}}
			_, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, dir, r)
			assert.NoError(t, err, "Publish")
			want := []released{{name: "pkg-a@1.0.0", commit: commitA, assets: []string{"checksums.txt"}}}
			assert.Equal(t, r.releases, want, "the releases")
		})

		t.Run("leaves the release of a tag with a file of drafts as a draft", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			dir := files.Workspace(t, files.Tree{
				path.Join(language.AssetsDir, "pkg-a@1.0.0", "pkg-a_1.0.0_SHA256SUMS"): files.Text("sums\n"),
				path.Join(language.DraftsDir, "pkg-a@1.0.0"):                           files.Text(""),
			})
			r := &releaser{}
			got, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, dir, r)
			assert.NoError(t, err, "Publish")
			assert.Equal(t, got, []release.Published{
				{Name: "pkg-a", Version: "1.0.0", Draft: true},
				{Name: "pkg-b", Version: "2.0.0-rc.1"},
			}, "the released packages")
			assert.Equal(t, r.releases, []released{
				{name: "pkg-a@1.0.0", commit: commitA, assets: []string{"pkg-a_1.0.0_SHA256SUMS"}, draft: true},
				{name: "pkg-b@2.0.0-rc.1", commit: commitA, prerelease: true},
			}, "the releases")
		})

		t.Run("returns the error of a draft that it cannot read", func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == "windows" {
				t.Skip("Windows does not deny the read of a directory by its mode")
			}
			s, plan := publishable(t)
			dir := files.Workspace(t, files.Tree{language.DraftsDir: files.Dir().WithMode(0o000)})
			_, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, dir, &releaser{})
			assert.HasError(t, err, "Publish")
			assert.Contains(t, err.Error(), "read the draft of pkg-a@1.0.0", "the error")
		})

		t.Run("returns the error of assets that it cannot read", func(t *testing.T) {
			t.Parallel()
			s, plan := publishable(t)
			dir := files.Workspace(t, files.Tree{path.Join(language.AssetsDir, "pkg-a@1.0.0"): files.Text("a file\n")})
			_, err := release.Publish(t.Context(), t.TempDir(), s.graph(t), plan, commitA, dir, &releaser{})
			assert.HasError(t, err, "Publish")
			assert.Contains(t, err.Error(), "read the assets of pkg-a@1.0.0", "the error")
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
			root := t.TempDir()
			_, err = release.Publish(t.Context(), root, g, &plan, commitA, "pack", r)
			assert.NoError(t, err, "Publish")
			assert.Equal(t, uploaded(reg), [][]string{{root, "pack", "pkg-a"}}, "the uploads")
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
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			r := &release.GitReleaser{Root: root, Remote: "origin"}
			err := r.Release(t.Context(), "v1.0.0", head, "### Minor Changes", false, nil, forge.LatestDefault)
			assert.NoError(t, err, "Release")
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
			assert.NoError(t, r.Release(t.Context(), "v1.0.0", head, "", false, nil, forge.LatestDefault), "Release")
			assert.Equal(t, vcstest.Git(t, root, "tag", "--list", "--format=%(contents)", "v1.0.0"), "v1.0.0\n\n",
				"the annotation with its newline, and the newline of git tag --list")
		})

		t.Run("returns ErrAssets for a release with assets before it creates the tag", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			r := &release.GitReleaser{Root: root, Remote: "origin"}
			assets := []string{"checksums.txt"}
			err := r.Release(t.Context(), "v1.0.0", head, "notes", false, assets, forge.LatestDefault)
			assert.ErrorIs(t, err, release.ErrAssets, "Release")
			assert.Empty(t, vcstest.Git(t, root, "tag", "--list"), "the tags of the repository")
		})

		t.Run("pushes nothing without a release", func(t *testing.T) {
			t.Parallel()
			r := &release.GitReleaser{Root: t.TempDir(), Remote: "origin"}
			assert.NoError(t, r.Finish(t.Context()), "Finish")
		})

		t.Run("returns ErrDraft for a draft and creates no tag", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			r := &release.GitReleaser{Root: root, Remote: "origin"}
			err := r.Draft(t.Context(), "v1.0.0", head, "notes", false, []string{"SHA256SUMS"})
			assert.ErrorIs(t, err, release.ErrDraft, "Draft")
			assert.NoError(t, r.Finish(t.Context()), "Finish")
			assert.Empty(t, vcstest.Git(t, root, "tag", "--list"), "the tags of the repository")
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
			err := r.Release(t.Context(), "v1.0.0", head, "notes", false, nil, forge.LatestDefault)
			assert.ErrorIs(t, err, vcs.ErrGit, "Release")
		})

		t.Run("runs git tag with its terminal", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			vcstest.Git(t, root, "tag", "v1.0.0", head)
			var stderr strings.Builder
			r := &release.GitReleaser{Terminal: vcs.Terminal{Stderr: &stderr}, Root: root, Remote: "origin"}
			err := r.Release(t.Context(), "v1.0.0", head, "notes", false, nil, forge.LatestDefault)
			assert.ErrorIs(t, err, vcs.ErrGit, "Release")
			assert.Contains(t, stderr.String(), tagExists, "the standard error of git tag")
		})

		t.Run("runs git push with its terminal", func(t *testing.T) {
			t.Parallel()
			root, _ := remoteRepository(t)
			head := strings.TrimSpace(vcstest.Git(t, root, "rev-parse", "HEAD"))
			var stderr strings.Builder
			r := &release.GitReleaser{Terminal: vcs.Terminal{Stderr: &stderr}, Root: root, Remote: "origin"}
			err := r.Release(t.Context(), "v1.0.0", head, "notes", false, nil, forge.LatestDefault)
			assert.NoError(t, err, "Release")
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

		t.Run("creates the tag, then the draft with the notes, then its assets, and then publishes it",
			func(t *testing.T) {
				t.Parallel()
				f := &tagForge{}
				r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
				assets := []string{filepath.Join("dist", "assets", "checksums.txt"), "ergon_1.0.0_linux_amd64.tar.gz"}
				err := r.Release(t.Context(), "v1.0.0-rc.1", commitA, "notes", true, assets, forge.LatestFalse)
				assert.NoError(t, err, "Release")
				assert.NoError(t, r.Finish(t.Context()), "Finish")
				assert.Equal(t, f.calls, []string{
					"tag o/r v1.0.0-rc.1 " + commitA,
					"prerelease draft o/r v1.0.0-rc.1 notes",
					"upload v1.0.0-rc.1 checksums.txt",
					"upload v1.0.0-rc.1 ergon_1.0.0_linux_amd64.tar.gz",
					"publish o/r 7 make_latest=false",
				}, "the calls")
			})

		t.Run("creates the tag, then the draft, then its assets, and publishes nothing for a draft",
			func(t *testing.T) {
				t.Parallel()
				f := &tagForge{}
				r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
				assets := []string{"demo_1.0.0_SHA256SUMS", "demo_1.0.0_linux_amd64.zip"}
				assert.NoError(t, r.Draft(t.Context(), "v1.0.0", commitA, "notes", false, assets), "Draft")
				assert.Equal(t, f.calls, []string{
					"tag o/r v1.0.0 " + commitA,
					"draft o/r v1.0.0 notes",
					"upload v1.0.0 demo_1.0.0_SHA256SUMS",
					"upload v1.0.0 demo_1.0.0_linux_amd64.zip",
				}, "the calls")
			})

		t.Run("returns the error of an upload of a draft", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{failUpload: errRelease}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			err := r.Draft(t.Context(), "v1.0.0", commitA, "notes", false, []string{"demo_1.0.0_SHA256SUMS"})
			assert.ErrorIs(t, err, errRelease, "Draft")
		})

		t.Run("completes the draft of an earlier publish with the assets that it lacks", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{
				tags: map[string]string{"o/r v1.0.0": commitA},
				releases: map[string]forge.Release{"o/r v1.0.0": {
					ID: 9, Tag: "v1.0.0", Draft: true, Assets: []forge.Asset{{Name: "checksums.txt"}},
				}},
			}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			assets := []string{"checksums.txt", "ergon.tar.gz"}
			err := r.Release(t.Context(), "v1.0.0", commitA, "notes", false, assets, forge.LatestTrue)
			assert.NoError(t, err, "Release")
			want := []string{"upload v1.0.0 ergon.tar.gz", "publish o/r 9 make_latest=true"}
			assert.Equal(t, f.calls, want, "the calls")
		})

		t.Run("changes nothing of a published release with every asset", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{
				tags: map[string]string{"o/r v1.0.0": commitA},
				releases: map[string]forge.Release{"o/r v1.0.0": {
					ID: 9, Tag: "v1.0.0", Assets: []forge.Asset{{Name: "checksums.txt"}},
				}},
			}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			assets := []string{"checksums.txt"}
			err := r.Release(t.Context(), "v1.0.0", commitA, "notes", false, assets, forge.LatestTrue)
			assert.NoError(t, err, "Release")
			assert.Empty(t, f.calls, "the calls")
		})

		t.Run("returns the error of the tag without a release", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{failCreate: errTag}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			err := r.Release(t.Context(), "v1.0.0", commitA, "notes", false, nil, forge.LatestTrue)
			assert.ErrorIs(t, err, errTag, "Release")
			assert.Empty(t, f.calls, "the calls")
		})

		t.Run("returns the error of the lookup of the tag", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{failTag: errTag}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			err := r.Release(t.Context(), "v1.0.0", commitA, "notes", false, nil, forge.LatestTrue)
			assert.ErrorIs(t, err, errTag, "Release")
			assert.Empty(t, f.calls, "the calls")
		})

		t.Run("returns the error of the lookup of the release after the tag", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{failReleaseOf: errRelease}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			err := r.Release(t.Context(), "v1.0.0", commitA, "notes", false, nil, forge.LatestTrue)
			assert.ErrorIs(t, err, errRelease, "Release")
			assert.Equal(t, f.calls, []string{"tag o/r v1.0.0 " + commitA}, "the calls")
		})

		t.Run("returns the error of the draft after the tag", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{failDraft: errRelease}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			err := r.Release(t.Context(), "v1.0.0", commitA, "notes", false, nil, forge.LatestTrue)
			assert.ErrorIs(t, err, errRelease, "Release")
			assert.Equal(t, f.calls, []string{"tag o/r v1.0.0 " + commitA}, "the calls")
		})

		t.Run("returns the error of an upload and leaves the draft unpublished", func(t *testing.T) {
			t.Parallel()
			f := &tagForge{failUpload: errRelease}
			r := release.ForgeReleaser{Forge: f, Repo: "o/r"}
			assets := []string{"checksums.txt"}
			err := r.Release(t.Context(), "v1.0.0", commitA, "notes", false, assets, forge.LatestTrue)
			assert.ErrorIs(t, err, errRelease, "Release")
			assert.Equal(t, f.calls, []string{"tag o/r v1.0.0 " + commitA, "draft o/r v1.0.0 notes"}, "the calls")
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
