// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The files of the repository of a case.
const (
	// manifestFile is the file that the repository of a case has in the directory of each package.
	manifestFile = "manifest"

	// versionFile is the file that a recorder writes the new version of a package into, in the
	// directory of the package.
	versionFile = "VERSION"

	// versionPerm is the mode of a file of versionFile, before the umask.
	versionPerm fs.FileMode = 0o644
)

// The paths of the files of pkg-a and pkg-b, and of the changeset of newState, relative to the root
// of the repository of a case.
var (
	// changelogA and changelogB are the changelogs of pkg-a and pkg-b.
	changelogA = path.Join(packagesDir, "pkg-a", release.ChangelogFile)
	changelogB = path.Join(packagesDir, "pkg-b", release.ChangelogFile)

	// versionA and versionB are the files that a recorder writes the versions of pkg-a and pkg-b
	// into.
	versionA = path.Join(packagesDir, "pkg-a", versionFile)
	versionB = path.Join(packagesDir, "pkg-b", versionFile)

	// changesetA is the file of the changeset of newState.
	changesetA = path.Join(changeset.Dir, "strange-words-combine"+changeset.Ext)
)

// errApply is the error of a versioner whose Apply fails.
var errApply = errors.New("apply failed")

// errRewrite is the error of a versioner that cannot rewrite a requirement.
var errRewrite = errors.New("rewrite failed")

// recorder is a versioner that records the edits of each call of Apply, writes the new version of
// each edit into the file versionFile of its package, and then applies the edits with the versioner
// that it embeds. The embedded versioner also resolves, rewrites and validates.
type recorder struct {
	language.Versioner

	// tb is the test of the case, which a write that fails stops.
	tb testing.TB

	// edits are the edits of every call of Apply, in the order of the calls.
	edits []language.Edit
}

// Apply records edits and writes the version of each into the file versionFile of its package. It
// returns the paths of those writes, then the paths and the error of the Apply of the embedded
// versioner.
func (r *recorder) Apply(ctx context.Context, root string, edits []language.Edit) ([]string, error) {
	r.edits = append(r.edits, edits...)
	var paths []string
	for _, e := range edits {
		file := path.Join(e.Package.Dir, versionFile)
		paths = append(paths, file)
		err := os.WriteFile(filepath.Join(root, filepath.FromSlash(file)), []byte(e.Version.String()+"\n"), versionPerm)
		assert.NoError(r.tb, err, "the write of "+file)
	}
	more, err := r.Versioner.Apply(ctx, root, edits)
	return append(paths, more...), err
}

// failing is a versioner of npm requirements whose Apply fails.
type failing struct{ npm }

// Apply changes no path and returns errApply.
func (failing) Apply(context.Context, string, []language.Edit) ([]string, error) {
	return nil, errApply
}

// unrewritable is a versioner of npm requirements that rewrites no requirement.
type unrewritable struct{ npm }

// Rewrite returns errRewrite.
func (unrewritable) Rewrite(string, version.Version) (string, error) {
	return "", errRewrite
}

// TestMain runs the tests without the variables and the configuration of git of the environment,
// so that git reads the repositories of the tests alone.
func TestMain(m *testing.M) {
	vcstest.Isolate()
	os.Exit(m.Run())
}

func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("Version", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the paths that it changed in the order of its writes", func(t *testing.T) {
			t.Parallel()
			s := dependentState(t)
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			written, _, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			assert.Equal(t, written, []string{changelogA, changelogB, changesetA, versionA, versionB}, "the paths")
		})

		t.Run("writes an entry into the changelog of each released package", func(t *testing.T) {
			t.Parallel()
			s := dependentState(t)
			dir, commit := repository(t, s, nil)
			cfg := defaultConfig()
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			sha := commit[:7]
			wantA := "# pkg-a\n\n## 1.0.1\n\n### Patch Changes\n\n- " + sha + ": base summary whatever\n"
			wantB := "# pkg-b\n\n## 1.0.1\n\n### Patch Changes\n\n- Updated dependencies [" + sha + "]\n  - pkg-a@1.0.1\n"
			files.HasContent(t, filepath.Join(dir, changelogA), wantA, "the changelog of pkg-a")
			files.HasContent(t, filepath.Join(dir, changelogB), wantB, "the changelog of pkg-b")
		})

		t.Run("removes each changeset of the plan", func(t *testing.T) {
			t.Parallel()
			s := dependentState(t)
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			files.Absent(t, filepath.Join(dir, changesetA), "the changeset")
		})

		t.Run("applies the edits of the plan", func(t *testing.T) {
			t.Parallel()
			s := dependentState(t)
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			_, edits, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			assert.Equal(t, edits, []language.Edit{
				{Version: parse(t, "1.0.1"), Package: s.pkgs[0]},
				{
					Version:      parse(t, "1.0.1"),
					Requirements: []workspace.Dependency{{Name: "pkg-a", Kind: workspace.KindRuntime, Req: "1.0.1"}},
					Package:      s.pkgs[1],
				},
			}, "the edits")
		})

		changelogs := []struct {
			want func(entry string) string
			name string
			give string
		}{
			{
				name: "inserts an entry before the first heading of a version after an introduction",
				give: "# Changelog for pkg-a\n\n## Overview\n\nThis file contains a history\nof changes made to pkg-a. We\n" +
					"hope you enjoy them.\n\n## 1.0.0\n\n- Fixed some bug",
				want: func(entry string) string {
					return "# Changelog for pkg-a\n\n## Overview\n\nThis file contains a history\nof changes made to " +
						"pkg-a. We\nhope you enjoy them.\n\n" + entry + "\n\n## 1.0.0\n\n- Fixed some bug"
				},
			},
			{
				name: "inserts an entry before the first heading of a version without a title",
				give: "## 1.0.0\n\n### Minor Changes\n\n- Initial release\n",
				want: func(entry string) string {
					return entry + "\n\n## 1.0.0\n\n### Minor Changes\n\n- Initial release\n"
				},
			},
			{
				name: "inserts an entry after the first line of a changelog without a version",
				give: "# pkg-a\n\nNotes.\n",
				want: func(entry string) string { return "# pkg-a\n\n" + entry + "\n\nNotes.\n" },
			},
			{
				name: "appends an entry to a changelog of one line",
				give: "# pkg-a",
				want: func(entry string) string { return "# pkg-a\n\n" + entry + "\n" },
			},
			{
				name: "writes the title of the package into an empty changelog",
				give: "",
				want: func(entry string) string { return "# pkg-a\n\n" + entry + "\n" },
			},
		}
		for _, tt := range changelogs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := newState(t)
				dir, commit := repository(t, s, files.Tree{changelogA: files.Text(tt.give)})
				cfg := defaultConfig()
				_, _, err := runVersion(t, s, &cfg, dir)
				assert.NoError(t, err, "Version")
				entry := "## 1.0.1\n\n### Patch Changes\n\n- " + commit[:7] + ": base summary whatever"
				files.HasContent(t, filepath.Join(dir, changelogA), tt.want(entry), "the changelog")
			})
		}

		t.Run("writes no changelog for a configuration without one", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			cfg.Changelog = release.Changelog{}
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			files.Absent(t, filepath.Join(dir, changelogA), "the changelog")
		})

		t.Run("keeps a changeset that names a skipped package", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.add("pkg-b", "1.0.0")
			s.changeset("ignored-fix", r("pkg-b", minor))
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			cfg.Ignore = []string{"pkg-b"}
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			files.IsFile(t, filepath.Join(dir, changeset.Dir, "ignored-fix"+changeset.Ext), "the changeset of pkg-b")
		})

		t.Run("returns no path for a changeset whose file the repository does not have", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.0")
			dir, _ := repository(t, s, nil)
			s.changeset("strange-words-combine", r("pkg-a", patch))
			cfg := defaultConfig()
			cfg.Changelog = release.Changelog{}
			written, _, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			assert.Equal(t, written, []string{versionA}, "the paths")
		})

		requirements := []struct {
			setup  func(s *state)
			config func(c *release.Config)
			name   string
			want   []workspace.Dependency
		}{
			{
				name: "rewrites a requirement that selects a version at updateInternalDependencies",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^1.0.0")
				},
				want: []workspace.Dependency{{Name: "pkg-a", Kind: workspace.KindRuntime, Req: "^1.0.1"}},
			},
			{
				name: "keeps a requirement that selects a version below updateInternalDependencies",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "^1.0.0")
				},
				config: func(c *release.Config) { c.UpdateInternalDependencies = minor },
			},
			{
				name: "keeps a peer requirement that selects a version under onlyUpdatePeerDependentsWhenOutOfRange",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindPeer, "^1.0.0")
				},
				config: func(c *release.Config) { c.OnlyUpdatePeerDependentsWhenOutOfRange = true },
			},
			{
				name: "keeps a requirement that its versioner does not read",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindDev, "link:../pkg-a")
				},
			},
			{
				name: "keeps a requirement that its versioner rewrites to itself",
				setup: func(s *state) {
					s.require("pkg-b", "pkg-a", workspace.KindRuntime, "*")
				},
			},
			{
				name: "keeps a requirement on a package outside the repository",
				setup: func(s *state) {
					s.require("pkg-b", "left-pad", workspace.KindRuntime, "^1.0.0")
				},
			},
			{
				name: "keeps a requirement that selects the version of a package outside the plan",
				setup: func(s *state) {
					s.add("pkg-c", "1.0.0")
					s.require("pkg-b", "pkg-c", workspace.KindRuntime, "1.0.0")
				},
			},
		}
		for _, tt := range requirements {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				s := newState(t)
				s.add("pkg-b", "1.0.0")
				tt.setup(s)
				s.changeset("change-b", r("pkg-b", patch))
				dir, _ := repository(t, s, nil)
				cfg := defaultConfig()
				if tt.config != nil {
					tt.config(&cfg)
				}
				_, edits, err := runVersion(t, s, &cfg, dir)
				assert.NoError(t, err, "Version")
				assert.Length(t, edits, 2, "the edits")
				assert.Equal(t, edits[1].Requirements, tt.want, "the requirements of pkg-b")
			})
		}

		t.Run("rewrites the dev requirement of a dependent that it releases at none", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.add("pkg-b", "1.0.0")
			s.add("pkg-c", "1.0.0")
			s.require("pkg-b", "pkg-a", workspace.KindDev, "1.0.0")
			s.changeset("docs-c", r("pkg-c", none))
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			_, edits, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			assert.Equal(t, edits, []language.Edit{
				{Version: parse(t, "1.0.1"), Package: s.pkgs[0]},
				{
					Version:      parse(t, "1.0.0"),
					Requirements: []workspace.Dependency{{Name: "pkg-a", Kind: workspace.KindDev, Req: "1.0.1"}},
					Package:      s.pkgs[1],
				},
			}, "the edits")
		})

		t.Run("rewrites a requirement that pins an older version of a package outside the plan", func(t *testing.T) {
			t.Parallel()
			s := goState(t)
			s.add("example.com/a", "1.0.0")
			s.add("example.com/b", "1.0.0")
			s.require("example.com/b", "example.com/a", workspace.KindRuntime, "v0.9.0")
			s.changeset("fix-b", r("example.com/b", patch))
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			_, edits, err := runVersion(t, s, &cfg, dir)
			assert.NoError(t, err, "Version")
			reqs := []workspace.Dependency{{Name: "example.com/a", Kind: workspace.KindRuntime, Req: "v1.0.0"}}
			assert.Equal(t, edits, []language.Edit{
				{Version: parse(t, "1.0.1"), Requirements: reqs, Package: s.pkgs[1]},
			}, "the edits")
		})

		t.Run("restores every path that it changed when a versioner fails", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			s.versioner = failing{}
			const history = "# pkg-a\n\n## 1.0.0\n\n- First.\n"
			dir, _ := repository(t, s, files.Tree{changelogA: files.Text(history)})
			cfg := defaultConfig()
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.ErrorIs(t, err, errApply, "Version")
			files.HasContent(t, filepath.Join(dir, changelogA), history, "the changelog")
			files.IsFile(t, filepath.Join(dir, changesetA), "the changeset")
			files.Absent(t, filepath.Join(dir, versionA), "the file of the versioner")
		})

		t.Run("returns ErrNoChangesets for a plan without changesets", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			s.add("pkg-a", "1.0.0")
			cfg := defaultConfig()
			_, _, err := runVersion(t, s, &cfg, t.TempDir())
			assert.ErrorIs(t, err, release.ErrNoChangesets, "Version")
		})

		t.Run("returns the error of a versioner that cannot rewrite a requirement", func(t *testing.T) {
			t.Parallel()
			s := dependentState(t)
			s.versioner = unrewritable{}
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.ErrorIs(t, err, errRewrite, "Version")
			assert.Contains(t, err.Error(), "the requirement 1.0.0 of pkg-b on pkg-a", "the error")
		})

		t.Run("returns ErrGit outside a working tree for a configuration with a changelog", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			cfg := defaultConfig()
			_, _, err := runVersion(t, s, &cfg, t.TempDir())
			assert.ErrorIs(t, err, vcs.ErrGit, "Version")
		})

		t.Run("returns ErrGit outside a working tree for a configuration without a changelog", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			cfg := defaultConfig()
			cfg.Changelog = release.Changelog{}
			_, _, err := runVersion(t, s, &cfg, t.TempDir())
			assert.ErrorIs(t, err, vcs.ErrGit, "Version")
		})

		t.Run("returns the error of a repository that does not exist", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			cfg := defaultConfig()
			cfg.Changelog = release.Changelog{}
			_, _, err := runVersion(t, s, &cfg, filepath.Join(t.TempDir(), "absent"))
			assert.ErrorIs(t, err, fs.ErrNotExist, "Version")
		})

		t.Run("returns ErrChangelog for the changelog of GitHub without a forge", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			dir, _ := repository(t, s, nil)
			cfg := defaultConfig()
			cfg.Changelog = release.Changelog{Format: release.ChangelogGitHub}
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.ErrorIs(t, err, release.ErrChangelog, "Version")
		})

		t.Run("returns the error of a changelog that it cannot read", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			dir, _ := repository(t, s, files.Tree{path.Join(changelogA, "inner"): files.Text("inner\n")})
			cfg := defaultConfig()
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.HasError(t, err, "Version")
			assert.Contains(t, err.Error(), "read "+changelogA, "the error")
		})

		t.Run("returns the error of a changelog whose directory does not exist", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			dir := vcstest.Repository(t, files.Tree{changesetA: files.Bytes(changeset.Format(&s.sets[0]))})
			vcstest.Commit(t, dir, "add the changeset")
			cfg := defaultConfig()
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.ErrorIs(t, err, fs.ErrNotExist, "Version")
			assert.Contains(t, err.Error(), "write "+changelogA, "the error")
		})

		t.Run("returns the error of a changeset that it cannot remove", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			dir := vcstest.Repository(t, files.Tree{
				path.Join(packagesDir, "pkg-a", manifestFile): files.Text("pkg-a\n"),
				path.Join(changesetA, "inner"):                files.Text("inner\n"),
			})
			vcstest.Commit(t, dir, "add a directory in place of the changeset")
			cfg := defaultConfig()
			cfg.Changelog = release.Changelog{}
			_, _, err := runVersion(t, s, &cfg, dir)
			assert.HasError(t, err, "Version")
			assert.Contains(t, err.Error(), "remove "+changesetA, "the error")
		})
	})
}

// dependentState returns the state of newState with pkg-b at 1.0.0, whose requirement 1.0.0 on pkg-a
// excludes the new version of pkg-a, for the test tb.
func dependentState(tb testing.TB) *state {
	tb.Helper()
	s := newState(tb)
	s.add("pkg-b", "1.0.0")
	s.require("pkg-b", "pkg-a", workspace.KindRuntime, "1.0.0")
	return s
}

// repository returns a working tree of git with the file manifestFile in the directory of each
// package of s, the file of each changeset of s and the files of extra, all in one commit, and that
// commit, for the test tb.
func repository(tb testing.TB, s *state, extra files.Tree) (string, string) {
	tb.Helper()
	tree := files.Tree{}
	for _, p := range s.pkgs {
		tree[path.Join(p.Dir, manifestFile)] = files.Text(p.Name + "\n")
	}
	for k := range s.sets {
		tree[path.Join(changeset.Dir, s.sets[k].ID+changeset.Ext)] = files.Bytes(changeset.Format(&s.sets[k]))
	}
	maps.Copy(tree, extra)
	dir := vcstest.Repository(tb, tree)
	return dir, vcstest.Commit(tb, dir, "add the packages and the changesets")
}

// runVersion plans the changesets of s under c and writes the plan into dir through a recorder of
// the versioner of s, for the test tb. It returns the paths and the error of Version, and the edits
// that the recorder recorded.
func runVersion(tb testing.TB, s *state, c *release.Config, dir string) ([]string, []language.Edit, error) {
	tb.Helper()
	rec := &recorder{Versioner: s.versioner, tb: tb}
	s.versioner = rec
	g := s.graph(tb)
	plan, err := release.NewPlan(g, c, s.sets)
	assert.NoError(tb, err, "NewPlan")
	written, err := release.Version(tb.Context(), dir, g, c, &plan, release.Host{})
	return written, rec.edits, err
}
