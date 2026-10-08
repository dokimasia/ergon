// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The parts of a version pull request of the cases.
const (
	proposalRepo   = "dokimasia/ergon"
	proposalBranch = "ergon-release/main"
)

// dirPerm is the mode of a directory that a case creates in a working tree.
const dirPerm fs.FileMode = 0o755

// errHub is the error of a host of pull requests that fails.
var errHub = errors.New("host failed")

// hub is a [release.Proposer] that records its calls, each as a line, and has the pull request open
// when open is set. It returns failAt from the call whose line starts with failOn.
type hub struct {
	// failAt is the error of the call that failOn names.
	failAt error

	// failOn is the start of the line of the call that fails.
	failOn string

	// calls are the calls, each as a line.
	calls []string

	// commits counts the commits.
	commits int

	// open reports that the pull request is open, as number 7.
	open bool
}

var _ release.Proposer = (*hub)(nil)

// record appends call to the calls and returns failAt for a call that starts with failOn.
func (h *hub) record(call string) error {
	h.calls = append(h.calls, call)
	if h.failOn != "" && strings.HasPrefix(call, h.failOn) {
		return h.failAt
	}
	return nil
}

// SetBranch records the branch.
func (h *hub) SetBranch(_ context.Context, repo, name, sha string) error {
	return h.record("branch " + repo + " " + name + " " + sha)
}

// Commit records the files and the deletions of the commit, and returns the number of the commit
// as its hash.
func (h *hub) Commit(
	_ context.Context, repo, branch, head, message string, files map[string][]byte, deleted []string,
) (string, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	h.commits++
	call := "commit " + repo + " " + branch + " " + head + " " + message + " [" + strings.Join(names, " ") + "] -[" +
		strings.Join(deleted, " ") + "]"
	return strings.Repeat(string(rune('0'+h.commits)), 3), h.record(call)
}

// PullRequest returns 7 when the pull request is open.
func (h *hub) PullRequest(_ context.Context, repo, head, base string) (int, bool, error) {
	return 7, h.open, h.record("pull " + repo + " " + head + " " + base)
}

// CreatePullRequest records the pull request and returns 8.
func (h *hub) CreatePullRequest(_ context.Context, repo, head, base, title, body string) (int, error) {
	return 8, h.record("create " + repo + " " + head + " " + base + " " + title + " " + body)
}

// UpdatePullRequest records the update.
func (h *hub) UpdatePullRequest(_ context.Context, repo string, number int, title, body string) error {
	return h.record("update " + repo + " " + string(rune('0'+number)) + " " + title + " " + body)
}

func TestProposal(t *testing.T) {
	t.Parallel()

	t.Run("Propose", func(t *testing.T) {
		t.Parallel()

		t.Run("commits the files on the branch of the base and opens a pull request", func(t *testing.T) {
			t.Parallel()
			h := &hub{}
			number, err := release.Propose(t.Context(), h, proposal(nil))
			assert.NoError(t, err, "Propose")
			assert.Equal(t, number, 8, "the number of the pull request")
			assert.Equal(t, h.calls, []string{
				"branch " + proposalRepo + " " + proposalBranch + " " + commitA,
				"commit " + proposalRepo + " " + proposalBranch + " " + commitA +
					" Version Packages [a/CHANGELOG.md b/go.mod] -[.changeset/strange-words-combine.md]",
				"pull " + proposalRepo + " " + proposalBranch + " main",
				"create " + proposalRepo + " " + proposalBranch + " main Version Packages Body.",
			}, "the calls")
		})

		t.Run("commits on the branch of the proposal", func(t *testing.T) {
			t.Parallel()
			h := &hub{}
			p := proposal(nil)
			p.Branch = "ergon-baseline/main"
			_, err := release.Propose(t.Context(), h, p)
			assert.NoError(t, err, "Propose")
			assert.Equal(t, h.calls, []string{
				"branch " + proposalRepo + " ergon-baseline/main " + commitA,
				"commit " + proposalRepo + " ergon-baseline/main " + commitA +
					" Version Packages [a/CHANGELOG.md b/go.mod] -[.changeset/strange-words-combine.md]",
				"pull " + proposalRepo + " ergon-baseline/main main",
				"create " + proposalRepo + " ergon-baseline/main main Version Packages Body.",
			}, "the calls")
		})

		t.Run("updates the open pull request", func(t *testing.T) {
			t.Parallel()
			h := &hub{open: true}
			number, err := release.Propose(t.Context(), h, proposal(nil))
			assert.NoError(t, err, "Propose")
			assert.Equal(t, number, 7, "the number of the pull request")
			assert.Equal(t, h.calls[len(h.calls)-1], "update "+proposalRepo+" 7 Version Packages Body.", "the update")
		})

		t.Run("commits the files over the bound of a request in commits within it", func(t *testing.T) {
			t.Parallel()
			h := &hub{}
			large := bytes.Repeat([]byte("a"), 4_000_000)
			p := proposal(map[string][]byte{"a/lock": large, "b/lock": large, "c/lock": []byte("c\n")})
			_, err := release.Propose(t.Context(), h, p)
			assert.NoError(t, err, "Propose")
			assert.Equal(t, h.calls[1:3], []string{
				"commit " + proposalRepo + " " + proposalBranch + " " + commitA +
					" Version Packages [a/lock] -[.changeset/strange-words-combine.md]",
				"commit " + proposalRepo + " " + proposalBranch + " 111 Version Packages [b/lock c/lock] -[]",
			}, "the commits")
		})

		t.Run("commits the deletions of a version without files", func(t *testing.T) {
			t.Parallel()
			h := &hub{}
			p := proposal(nil)
			p.Files = nil
			_, err := release.Propose(t.Context(), h, p)
			assert.NoError(t, err, "Propose")
			assert.Equal(t, h.calls[1], "commit "+proposalRepo+" "+proposalBranch+" "+commitA+
				" Version Packages [] -[.changeset/strange-words-combine.md]", "the commit")
		})

		failures := []struct {
			name   string
			failOn string
			files  bool
		}{
			{name: "returns the error of the branch", failOn: "branch", files: true},
			{name: "returns the error of a commit of files", failOn: "commit", files: true},
			{name: "returns the error of a commit of deletions", failOn: "commit"},
			{name: "returns the error of the lookup of the pull request", failOn: "pull", files: true},
			{name: "returns the error of a new pull request", failOn: "create", files: true},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				h := &hub{failOn: tt.failOn, failAt: errHub}
				p := proposal(nil)
				if !tt.files {
					p.Files = nil
				}
				_, err := release.Propose(t.Context(), h, p)
				assert.ErrorIs(t, err, errHub, "Propose")
			})
		}
	})

	t.Run("NewProposal", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the changed files with their content and the removed files", func(t *testing.T) {
			t.Parallel()
			root := committed(t, files.Tree{"a.md": files.Text("a\n"), "b.md": files.Text("b\n")})
			assert.NoError(t, os.WriteFile(filepath.Join(root, "a.md"), []byte("changed\n"), versionPerm), "the change")
			assert.NoError(t, os.WriteFile(filepath.Join(root, "c.md"), []byte("new\n"), versionPerm), "the new file")
			assert.NoError(t, os.Remove(filepath.Join(root, "b.md")), "the removal")
			got, err := release.NewProposal(t.Context(), root, "main", newState(t).graph(t), &release.Plan{})
			assert.NoError(t, err, "NewProposal")
			expect.That(t, got.Files).Equal(map[string][]byte{"a.md": []byte("changed\n"), "c.md": []byte("new\n")},
				"the files")
			expect.That(t, got.Deleted).Equal([]string{"b.md"}, "the removed files")
			expect.That(t, got.Base).Equal("main", "the base branch")
			expect.That(t, got.Branch).Equal(proposalBranch, "the branch of the version pull request")
		})

		t.Run("lists the changed lockfiles of a plan without releases", func(t *testing.T) {
			t.Parallel()
			root := committed(t, files.Tree{"b/go.sum": files.Text("b\n"), "c/go.sum": files.Text("c\n")})
			for _, name := range []string{"c/go.sum", "b/go.sum"} {
				err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte("locked\n"), versionPerm)
				assert.NoError(t, err, "the change of "+name)
			}
			got, err := release.NewProposal(t.Context(), root, "main", newState(t).graph(t), &release.Plan{})
			assert.NoError(t, err, "NewProposal")
			assert.Equal(t, got.Body, "This pull request was opened by `ergon release ci version`. A package of a "+
				"release that waits for its publish changed after its version commit, and the lockfiles below still "+
				"record its earlier content. Merging this pull request rewrites them, so that the release workflow "+
				"publishes and tags the content that they record. A changeset that is merged into main before this pull "+
				"request updates it.\n# Lockfiles\n\n- `b/go.sum`\n- `c/go.sum`", "the body")
		})

		t.Run("returns ErrGit for a directory outside a working tree", func(t *testing.T) {
			t.Parallel()
			_, err := release.NewProposal(t.Context(), t.TempDir(), "main", newState(t).graph(t), &release.Plan{})
			assert.ErrorIs(t, err, vcs.ErrGit, "NewProposal")
		})

		t.Run("returns the error of a file that it cannot read", func(t *testing.T) {
			t.Parallel()
			root := committed(t, files.Tree{"a.md": files.Text("a\n")})
			assert.NoError(t, os.Remove(filepath.Join(root, "a.md")), "the removal")
			assert.NoError(t, os.MkdirAll(filepath.Join(root, "a.md", "inner"), dirPerm), "the directory in its place")
			_, err := release.NewProposal(t.Context(), root, "main", newState(t).graph(t), &release.Plan{})
			assert.HasError(t, err, "NewProposal")
			assert.Contains(t, err.Error(), "read a.md", "the error")
		})

		t.Run(
			"lists the releases of public packages from the highest level before the private ones",
			func(t *testing.T) {
				t.Parallel()
				s := blankState(t)
				for _, name := range []string{"pkg-a", "pkg-b", "pkg-c", "pkg-d", "pkg-e"} {
					s.add(name, "1.0.0")
				}
				s.pkgs[3].Private = true
				s.changeset("all", r("pkg-b", minor), r("pkg-a", patch), r("pkg-e", major), r("pkg-c", none),
					r("pkg-d", major))
				root := committed(t, files.Tree{
					changelogA: files.Text("# pkg-a\n\n## 1.0.1\n\n### Patch Changes\n\n- a\n"),
					changelogB: files.Text("# pkg-b\n\n## 1.1.0\n\n### Minor Changes\n\n- b\n"),
					path.Join(packagesDir, "pkg-d", release.ChangelogFile): files.Text(
						"# pkg-d\n\n## 2.0.0\n\n### Major Changes\n\n- d\n"),
					path.Join(packagesDir, "pkg-e", release.ChangelogFile): files.Text(
						"# pkg-e\n\n## 2.0.0\n\n### Major Changes\n\n- e\n"),
				})
				cfg := defaultConfig()
				cfg.PrivateVersion = true
				g := s.graph(t)
				plan, err := release.NewPlan(g, &cfg, s.sets)
				assert.NoError(t, err, "NewPlan")
				got, err := release.NewProposal(t.Context(), root, "main", g, &plan)
				assert.NoError(t, err, "NewProposal")
				assert.Equal(
					t,
					got.Body,
					"This pull request was opened by `ergon release ci version`. Merging it releases "+
						"the packages below: the release workflow publishes each package to its registry and tags it. A "+
						"changeset that is merged into main before this pull request updates it.\n# Releases\n"+
						"## pkg-e@2.0.0\n\n### Major Changes\n\n- e\n## pkg-b@1.1.0\n\n### Minor Changes\n\n- b\n"+
						"## pkg-a@1.0.1\n\n### Patch Changes\n\n- a\n## pkg-d@2.0.0\n\n### Major Changes\n\n- d",
					"the body",
				)
			},
		)

		t.Run("leaves out the changelogs of a body over the bound", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			root := committed(t, files.Tree{
				changelogA: files.Text("# pkg-a\n\n## 1.0.1\n\n" + strings.Repeat("- a\n", 20_000)),
			})
			cfg := defaultConfig()
			g := s.graph(t)
			plan, err := release.NewPlan(g, &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			got, err := release.NewProposal(t.Context(), root, "main", g, &plan)
			assert.NoError(t, err, "NewProposal")
			assert.HasSuffix(t, got.Body, "\n> The changelog of each package is left out, because the body would "+
				"exceed the size limit.\n\n## pkg-a@1.0.1\n\n", "the body")
		})

		t.Run("leaves out the releases of a body that is still over the bound", func(t *testing.T) {
			t.Parallel()
			s := blankState(t)
			for k := range 7 {
				letter := string(rune('a' + k))
				s.add(strings.Repeat(letter, 10_000), "1.0.0")
				s.pkgs[k].Dir = path.Join(packagesDir, letter)
				s.changeset(letter, r(s.pkgs[k].Name, patch))
			}
			cfg := defaultConfig()
			g := s.graph(t)
			plan, err := release.NewPlan(g, &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			got, err := release.NewProposal(t.Context(), committed(t, files.Tree{}), "main", g, &plan)
			assert.NoError(t, err, "NewProposal")
			assert.HasSuffix(t, got.Body, "# Releases\n\n> The releases are left out, because the body would exceed "+
				"the size limit.", "the body")
		})

		t.Run("returns the error of a changelog that it cannot read", func(t *testing.T) {
			t.Parallel()
			s := newState(t)
			root := committed(t, files.Tree{path.Join(changelogA, "inner"): files.Text("x\n")})
			cfg := defaultConfig()
			g := s.graph(t)
			plan, err := release.NewPlan(g, &cfg, s.sets)
			assert.NoError(t, err, "NewPlan")
			_, err = release.NewProposal(t.Context(), root, "main", g, &plan)
			assert.HasError(t, err, "NewProposal")
			assert.Contains(t, err.Error(), "read packages/pkg-a/CHANGELOG.md", "the error")
		})
	})

	t.Run("ChangedFiles", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the changed files with their content and the removed files", func(t *testing.T) {
			t.Parallel()
			root := committed(t, files.Tree{"a.md": files.Text("a\n"), "d/b.md": files.Text("b\n")})
			assert.NoError(t, os.WriteFile(filepath.Join(root, "a.md"), []byte("changed\n"), versionPerm), "the change")
			assert.NoError(t, os.Remove(filepath.Join(root, "d", "b.md")), "the removal")
			changed, deleted, err := release.ChangedFiles(t.Context(), root)
			assert.NoError(t, err, "ChangedFiles")
			expect.That(t, changed).Equal(map[string][]byte{"a.md": []byte("changed\n")}, "the changed files")
			expect.That(t, deleted).Equal([]string{"d/b.md"}, "the removed files")
		})

		t.Run("returns ErrGit for a directory outside a working tree", func(t *testing.T) {
			t.Parallel()
			_, _, err := release.ChangedFiles(t.Context(), t.TempDir())
			assert.ErrorIs(t, err, vcs.ErrGit, "ChangedFiles")
		})
	})
}

// committed returns a working tree of git with tree in one commit, for the test tb.
func committed(tb testing.TB, tree files.Tree) string {
	tb.Helper()
	root := vcstest.Repository(tb, tree)
	vcstest.Commit(tb, root, "first")
	return root
}

// proposal returns the version pull request of the cases with files, or with a changelog and a
// go.mod for nil files.
func proposal(files map[string][]byte) *release.Proposal {
	if files == nil {
		files = map[string][]byte{"b/go.mod": []byte("module b\n"), "a/CHANGELOG.md": []byte("# a\n")}
	}
	return &release.Proposal{
		Files: files, Repo: proposalRepo, Base: "main", Branch: proposalBranch, Head: commitA,
		Title: "Version Packages", Body: "Body.", Deleted: []string{".changeset/strange-words-combine.md"},
	}
}
