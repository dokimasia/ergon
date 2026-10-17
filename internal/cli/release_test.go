// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
	"go.stealthscale.io/cli/clitest"
)

// The module of the release cases, at the root of its repository, and its files.
const (
	modulePath    = "example.com/demo"
	moduleFile    = "demo.go"
	changelogPath = "CHANGELOG.md"
	configPath    = ".changeset/config.json"
)

// The changeset of the release cases, which releases the module at minor.
const (
	changesetID   = "add-the-unit-type-abcd"
	changesetPath = ".changeset/" + changesetID + ".md"
	changesetText = "---\n\"" + modulePath + "\": minor\n---\n\nAdd the Unit type.\n"
)

// The changesets of the release cases that do not release a package: a changeset without packages
// and a changeset of the module at none.
const (
	emptyChangeset = "---\n---\n\nFix a typo of the README.\n"
	noneChangeset  = "---\n\"" + modulePath + "\": none\n---\n\nFix a typo of the README.\n"
)

// The configurations of changesets of the release cases: the changelog of git, and the changelog of
// GitHub. Neither makes a commit.
const (
	gitConfig    = `{"changelog": "@changesets/cli/changelog", "commit": false}` + "\n"
	githubConfig = `{"changelog": ["@changesets/changelog-github", {"repo": "o/r"}], "commit": false}` + "\n"
)

// firstChangelog is the changelog of the module, at the version 1.0.0.
const firstChangelog = "# " + modulePath + "\n\n## 1.0.0\n\n### Major Changes\n\n- Release the first version.\n"

// staleSum is the go.sum of the module b of the cases with a second module.
const staleSum = "b/go.sum"

// newTag is the mark of git push for a tag that the remote did not have.
const newTag = "[new tag]"

// The publish plan of the module at 1.0.0 without its tag, and the packages that its publish
// releases, as the release commands write them.
const (
	publishPlan = `{
  "plan": [
    [
      {
        "kind": "tag-only",
        "toolchain": "go",
        "name": "example.com/demo",
        "tag": "v1.0.0",
        "version": "1.0.0"
      }
    ]
  ],
  "version": 1
}
`
	releasedPackages = `[{"name":"example.com/demo","version":"1.0.0"}]`
)

// The fake API of GitHub: the repository of the cases, the path of GraphQL, the commit of each
// commit that the API makes, the tree of every commit, and a commit that the branch main of a case
// points at after a later push.
const (
	hubRepo     = "o/r"
	graphqlPath = "/graphql"
	hubCommit   = "c0ffeec0ffeec0ffeec0ffeec0ffeec0ffeec0ff"
	hubTree     = "7ee7ee7ee7ee7ee7ee7ee7ee7ee7ee7ee7ee7ee7"
	laterCommit = "1a7e21a7e21a7e21a7e21a7e21a7e21a7e21a7e2"
)

// The passed runs of the gate that the fake API of GitHub returns: one run on the page runPage, and
// none.
const (
	runPage   = "https://github.com/o/r/actions/runs/9"
	passedRun = `{"workflow_runs":[{"status":"completed","conclusion":"success","html_url":"` + runPage + `"}]}`
	noRun     = `{"workflow_runs":[]}`
)

// releaseRuns are the command lines of the transcript of ergon release: the help of each
// subcommand, and the command lines that the subcommands refuse.
var releaseRuns = []clitest.Input{
	{Args: []string{"release", "--help"}},
	{Args: []string{"release"}},
	{Args: []string{"release", "ci", "--help"}},
	{Args: []string{"release", "ci"}},
	{Args: []string{"release", "add", "--help"}},
	{Args: []string{"release", "status", "--help"}},
	{Args: []string{"release", "version", "--help"}},
	{Args: []string{"release", "publish-plan", "--help"}},
	{Args: []string{"release", "pack", "--help"}},
	{Args: []string{"release", "publish", "--help"}},
	{Args: []string{"release", "git-tag", "--help"}},
	{Args: []string{"release", "ci", "select-mode", "--help"}},
	{Args: []string{"release", "ci", "skip", "--help"}},
	{Args: []string{"release", "ci", "verify", "--help"}},
	{Args: []string{"release", "ci", "version", "--help"}},
	{Args: []string{"release", "ci", "homebrew", "--help"}},
	{Args: []string{"release", "add"}},
	{Args: []string{"release", "add", "--bump", modulePath, "-m", "x"}},
	{Args: []string{"release", "add", "--bump", modulePath + "=huge", "-m", "x"}},
	{Args: []string{"release", "ci", "homebrew"}},
}

// tapRepo is the Homebrew tap of the release cases.
const tapRepo = "o/homebrew-tap"

// releaseVariables are the variables of GitHub Actions and of the editors that the release
// commands read, which every release case sets to an empty value before its own.
var releaseVariables = []string{
	"GITHUB_TOKEN", "GITHUB_REPOSITORY", "GITHUB_API_URL", "GITHUB_GRAPHQL_URL", "GITHUB_SERVER_URL",
	"GITHUB_OUTPUT", "GITHUB_ACTIONS", "GITHUB_RUN_ID", "VISUAL", "EDITOR",
}

// errRandom is the error of a source of random digits that fails.
var errRandom = errors.New("random: failed")

// hub is a fake API of GitHub for the release cases. It responds as GitHub does for a repository
// whose branch main points at head, without the branch of the version pull request until a request
// creates it, without tags and without pull requests, and makes each commit as hubCommit, whose
// branch then points at it. Every commit has the tree tree, hubTree when tree is empty, and the
// parent parent, none when parent is empty. The commit marked has the status ergon/version in the
// state success, and no other commit has a status. It creates each release as the draft 7, and has
// the tap tapRepo with the default branch main and no file. It responds with runs to a request for
// the runs of a workflow, and with the status 500 to a request whose method and path start with
// fail. It is safe for concurrent use, as the goroutines of a server use it.
type hub struct {
	// fail is the start of the method and the path of the requests that fail, or empty for none.
	fail string

	// head is the commit of the branch main, or empty for a repository without the branch.
	head string

	// runs is the body of the response to a request for the runs of a workflow.
	runs string

	// tree is the tree of every commit, or empty for hubTree.
	tree string

	// parent is the first parent of every commit, or empty for none.
	parent string

	// marked is the commit with the status ergon/version, or empty for none.
	marked string

	// requests are the method and the path with the query of each request, in their order.
	requests []string

	// graphql are the bodies of the requests of GraphQL, in their order.
	graphql []string

	// pulls are the bodies of the pull requests that the requests open, in their order.
	pulls []string

	// branches are the names of the branches that the requests created.
	branches []string

	// statuses are the bodies of the statuses that the requests set, in their order.
	statuses []string

	// mu guards requests, graphql, pulls, branches and statuses.
	mu sync.Mutex
}

// ServeHTTP records the request and responds to it. It records an empty body for a request to open
// a pull request whose JSON does not decode.
func (h *hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	call := r.Method + " " + r.URL.RequestURI()
	h.mu.Lock()
	h.requests = append(h.requests, call)
	if r.URL.Path == graphqlPath {
		h.graphql = append(h.graphql, string(body))
	}
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pulls") {
		var pull struct {
			Body string `json:"body"`
		}
		_ = json.Unmarshal(body, &pull)
		h.pulls = append(h.pulls, pull.Body)
	}
	if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/statuses/") {
		h.statuses = append(h.statuses, string(body))
	}
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/refs") {
		var ref struct {
			Ref string `json:"ref"`
		}
		_ = json.Unmarshal(body, &ref)
		h.branches = append(h.branches, strings.TrimPrefix(ref.Ref, "refs/heads/"))
	}
	created := slices.ContainsFunc(h.branches, func(name string) bool {
		return strings.HasSuffix(r.URL.Path, "/git/ref/heads/"+name)
	})
	h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case h.fail != "" && strings.HasPrefix(call, h.fail):
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"message":"failed"}`)
	case r.Method == http.MethodGet && h.head != "" && strings.HasSuffix(r.URL.Path, "/git/ref/heads/main"):
		_, _ = io.WriteString(w, `{"ref":"refs/heads/main","object":{"type":"commit","sha":"`+h.head+`"}}`)
	case r.Method == http.MethodGet && created:
		_, _ = io.WriteString(w, `{"object":{"type":"commit","sha":"`+hubCommit+`"}}`)
	case r.Method == http.MethodGet && (strings.Contains(r.URL.Path, "/git/ref/") ||
		strings.Contains(r.URL.Path, "/contents/")):
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
	case r.Method == http.MethodGet && r.URL.Path == "/repos/"+tapRepo:
		_, _ = io.WriteString(w, `{"default_branch":"main"}`)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/releases"):
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":7,"draft":true,"assets":[]}`)
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/commits/"):
		_, _ = io.WriteString(w, h.commit())
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/status"):
		_, _ = io.WriteString(w, h.status(r.URL.Path))
	case r.URL.Path == graphqlPath:
		_, _ = io.WriteString(w, `{"data":{"createCommitOnBranch":{"commit":{"oid":"`+hubCommit+`"}}}}`)
	case strings.Contains(r.URL.Path, "/actions/workflows/"):
		_, _ = io.WriteString(w, h.runs)
	case r.Method == http.MethodGet:
		_, _ = io.WriteString(w, "[]")
	default:
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"number":7}`)
	}
}

// commit returns the body of the response to a request for a commit of the Git database: the tree
// and the parent of every commit of h.
func (h *hub) commit() string {
	tree := h.tree
	if tree == "" {
		tree = hubTree
	}
	parents := ""
	if h.parent != "" {
		parents = `{"sha":"` + h.parent + `"}`
	}
	return `{"tree":{"sha":"` + tree + `"},"parents":[` + parents + `]}`
}

// status returns the body of the response to a request for the combined status of a commit at
// path: the status ergon/version in the state success for the commit h.marked, and no status for
// any other commit.
func (h *hub) status(path string) string {
	if h.marked != "" && strings.HasSuffix(path, "/commits/"+h.marked+"/status") {
		return `{"state":"success","statuses":[{"state":"success","context":"` + release.VersionStatus + `"}]}`
	}
	return `{"state":"pending","statuses":[]}`
}

// start serves h for the test t, and returns the variables of GitHub Actions that point the release
// commands at it, with a token and the repository hubRepo.
func (h *hub) start(t *testing.T) map[string]string {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return map[string]string{
		"GITHUB_TOKEN": "token", "GITHUB_REPOSITORY": hubRepo, "GITHUB_API_URL": server.URL,
		"GITHUB_GRAPHQL_URL": server.URL + graphqlPath, "GITHUB_SERVER_URL": server.URL,
	}
}

// calls returns the requests that h received, in their order.
func (h *hub) calls() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.requests)
}

// commits returns the bodies of the requests of GraphQL that h received, in their order.
func (h *hub) commits() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.graphql)
}

// opened returns the bodies of the pull requests that the requests to h opened, in their order.
func (h *hub) opened() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.pulls)
}

// marks returns the bodies of the statuses that the requests to h set, in their order.
func (h *hub) marks() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.statuses)
}

func TestRelease(t *testing.T) {
	t.Parallel()

	t.Run("Main", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "ergon", program(t.TempDir(), app.Register).Main, releaseRuns...)
			golden.Match(t, "release.txt", got, golden.ShouldUpdate())
		})

		t.Run("add writes a changeset of the packages of --bump", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "add", "--bump", modulePath+"=minor",
				"-m", "Add the Unit type.")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+changesetPath+"\n", "the standard output")
			files.HasContent(t, filepath.Join(dir, changesetPath), changesetText, "the changeset")
		})

		t.Run("add writes a changeset without packages with --empty", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "add", "--empty", "-m", "Fix a typo.")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote .changeset/fix-a-typo-abcd.md\n", "the standard output")
			files.HasContent(t, filepath.Join(dir, ".changeset", "fix-a-typo-abcd.md"), "---\n---\n\nFix a typo.\n",
				"the changeset")
		})

		t.Run("add returns 1 for a package that the repository does not have", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "add", "--bump", "example.com/other=patch",
				"-m", "Add the Unit type.")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: release: invalid changeset: "+changesetPath+`:2: the package `+
				`"example.com/other", which the repository does not have`+"\n", "the standard error")
			files.Absent(t, filepath.Join(dir, changesetPath), "the changeset")
		})

		t.Run("add returns 1 when the random digits cannot be read", func(t *testing.T) {
			t.Parallel()
			p := program(goModule(t, gitConfig, nil), app.Register)
			p.Random = iotest.ErrReader(errRandom)
			res := clitest.Main(t.Context(), p.Main, &clitest.Input{
				Args: []string{"release", "add", "--bump", modulePath + "=minor", "-m", "Add the Unit type."},
				Vars: releaseVars(nil),
			})
			assert.Equal(t, res.Status, statusFailure, "the exit status")
			assert.Equal(t, res.Stderr, "ergon: release: read the random digits of a changeset: "+
				errRandom.Error()+"\n", "the standard error")
		})

		t.Run("add returns 1 for a changeset that exists", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{changesetPath: files.Text(changesetText)})
			status, stdout, stderr := runRelease(t, dir, nil, "release", "add", "--bump", modulePath+"=patch",
				"-m", "Add the Unit type.")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: release: create "+changesetPath+": file already exists\n",
				"the standard error")
			files.HasContent(t, filepath.Join(dir, changesetPath), changesetText, "the changeset")
		})

		editors := []struct {
			name string
			env  map[string]string
		}{
			{
				name: "add opens the changeset in the editor of VISUAL with --open",
				env:  map[string]string{"VISUAL": "npx"},
			},
			{
				name: "add opens the changeset in the editor of EDITOR without VISUAL",
				env:  map[string]string{"EDITOR": "npx"},
			},
		}
		for _, tt := range editors {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := goModule(t, gitConfig, nil)
				status, stdout, stderr := runRelease(t, dir, tt.env, "release", "add", "--bump", modulePath+"=minor",
					"-m", "Add the Unit type.", "--open")
				assert.Equal(t, status, statusOK, "the exit status: "+stderr)
				assert.Equal(t, stdout, "wrote "+changesetPath+"\nnpx "+filepath.Join(dir, ".changeset",
					changesetID+".md")+"\n", "the standard output")
			})
		}

		t.Run("add returns 1 with --open without an editor", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "add", "--bump",
				modulePath+"=minor", "-m", "Add the Unit type.", "--open")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stdout, "wrote "+changesetPath+"\n", "the standard output")
			assert.Equal(t, stderr, "ergon: cli: set VISUAL or EDITOR to open the changeset\n", "the standard error")
		})

		t.Run("add returns 1 for an editor that fails", func(t *testing.T) {
			t.Parallel()
			env := map[string]string{"VISUAL": "npx " + exitFlag + "3"}
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), env, "release", "add", "--bump",
				modulePath+"=minor", "-m", "Add the Unit type.", "--open")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: cli: open the changeset in npx: exit status 3\n", "the standard error")
		})

		unconfigured := []struct {
			env  map[string]string
			name string
			args []string
		}{
			{
				name: "add returns 1 for a repository without .changeset/config.json",
				args: []string{"release", "add", "--bump", modulePath + "=minor", "-m", "x"},
			},
			{
				name: "status returns 1 for a repository without .changeset/config.json",
				args: []string{"release", "status"},
			},
			{
				name: "version returns 1 for a repository without .changeset/config.json",
				args: []string{"release", "version"},
			},
			{
				name: "publish-plan returns 1 for a repository without .changeset/config.json",
				args: []string{"release", "publish-plan"},
			},
			{name: "pack returns 1 for a repository without .changeset/config.json", args: []string{"release", "pack"}},
			{
				name: "pack returns 1 for a repository without .changeset/config.json with --from-publish-plan",
				args: []string{"release", "pack", "--from-publish-plan", "plan.json"},
			},
			{
				name: "publish returns 1 for a repository without .changeset/config.json",
				args: []string{"release", "publish"},
			},
			{
				name: "git-tag returns 1 for a repository without .changeset/config.json",
				args: []string{"release", "git-tag"},
			},
			{
				name: "select-mode returns 1 for a repository without .changeset/config.json",
				args: []string{"release", "ci", "select-mode"},
			},
			{
				name: "ci version returns 1 for a repository without .changeset/config.json",
				args: []string{"release", "ci", "version"},
				env:  map[string]string{"GITHUB_REPOSITORY": hubRepo, "GITHUB_TOKEN": "token"},
			},
		}
		for _, tt := range unconfigured {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := goModule(t, gitConfig, nil)
				config := filepath.Join(dir, filepath.FromSlash(configPath))
				assert.NoError(t, os.Remove(config), "Remove of the configuration")
				status, stdout, stderr := runRelease(t, dir, tt.env, tt.args...)
				assert.Equal(t, status, statusFailure, "the exit status")
				assert.Empty(t, stdout, "the standard output")
				assert.HasPrefix(t, stderr, "ergon: cli: read .changeset/config.json, which ergon init seeds: ",
					"the standard error")
			})
		}

		t.Run("returns 1 for a .changeset/config.json that ergon does not read", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, `{"commit": true}`+"\n", nil)
			status, _, stderr := runRelease(t, dir, nil, "release", "status")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: release: invalid .changeset/config.json: commit true, which is not false: "+
				"ergon release version commits nothing\n", "the standard error")
		})

		t.Run("returns 1 for packages that the discovery refuses", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{"go.work": files.Text("go 1.27\n\nuse ./absent\n")})
			status, _, stderr := runRelease(t, dir, nil, "release", "status")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: release: discover the packages of the toolchain go: workspace: invalid "+
				"modules: the directory absent, which go.work uses, has no go.mod\n", "the standard error")
		})

		t.Run("writes each warning of the configuration", func(t *testing.T) {
			t.Parallel()
			config := `{"changelog": "@changesets/cli/changelog", "commit": false, "fixed": [["example.com/none"]]}`
			status, stdout, stderr := runRelease(t, goModule(t, config+"\n", nil), nil, "release", "status")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "no release\n", "the standard output")
			assert.Equal(t, stderr, `ergon: warning: fixed: the package or glob "example.com/none" matches no package`+
				"\n", "the standard error")
		})

		t.Run("status writes no release for a branch without changes", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "status")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "no release\n", "the standard output")
		})

		t.Run("status returns 1 for a changed package without a changeset", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, moduleFile, "// Package demo changed.\npackage demo\n")
			status, stdout, stderr := runRelease(t, dir, nil, "release", "status")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stdout, "no release\n", "the standard output")
			assert.Equal(t, stderr, "ergon: cli: example.com/demo changed without a changeset that names it; add a "+
				"changeset with ergon release add, or with --empty for a change that releases nothing\n",
				"the standard error")
		})

		t.Run("status writes the plan of the changesets that the branch added", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, moduleFile, "// Package demo changed.\npackage demo\n")
			write(t, dir, changesetPath, changesetText)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "status")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, modulePath+" minor 1.0.0 -> 1.1.0\n", "the standard output")
		})

		t.Run("status lists the changesets of each release with --verbose", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{})
			write(t, dir, changesetPath, changesetText)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "status", "--verbose")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, modulePath+" minor 1.0.0 -> 1.1.0\n  "+changesetID+"\n", "the standard output")
		})

		t.Run("status writes the plan as JSON into --output", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, changesetPath, changesetText)
			output := filepath.Join(t.TempDir(), "plan.json")
			status, _, stderr := runRelease(t, dir, nil, "release", "status", "--output", output)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			var plan struct {
				Releases []struct {
					Name       string   `json:"name"`
					Type       string   `json:"type"`
					OldVersion string   `json:"oldVersion"`
					NewVersion string   `json:"newVersion"`
					Changesets []string `json:"changesets"`
				} `json:"releases"`
			}
			assert.NoError(t, json.Unmarshal([]byte(files.Read(t, output)), &plan), "Unmarshal of the plan")
			assert.Length(t, plan.Releases, 1, "the releases of the plan")
			expect.That(t, plan.Releases[0].Name).Equal(modulePath, "the name of the release")
			expect.That(t, plan.Releases[0].Type).Equal("minor", "the type of the release")
			expect.That(t, plan.Releases[0].OldVersion).Equal("1.0.0", "the old version of the release")
			expect.That(t, plan.Releases[0].NewVersion).Equal("1.1.0", "the new version of the release")
			expect.That(t, plan.Releases[0].Changesets).Equal([]string{changesetID}, "the changesets of the release")
		})

		t.Run("status returns 1 for a --since that names no commit", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := runRelease(
				t,
				goModule(t, gitConfig, nil),
				nil,
				"release",
				"status",
				"--since",
				"bogus",
			)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
		})

		t.Run("status returns 1 for an --output that cannot be written", func(t *testing.T) {
			t.Parallel()
			output := filepath.Join(t.TempDir(), "absent", "plan.json")
			status, _, stderr := runRelease(
				t,
				goModule(t, gitConfig, nil),
				nil,
				"release",
				"status",
				"--output",
				output,
			)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: write "+output+": ", "the standard error")
		})

		t.Run("status returns 1 for a changeset that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{".changeset/broken.md": files.Text("no front matter\n")})
			status, _, stderr := runRelease(t, dir, nil, "release", "status")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: release: changeset: invalid changeset: broken.md:1: the file does not "+
				"open with the line ---\n", "the standard error")
		})

		t.Run("version writes the changelog and removes the changeset", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, changesetPath, changesetText)
			commit := vcstest.Commit(t, dir, "add a changeset")
			status, stdout, stderr := runRelease(t, dir, nil, "release", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+changelogPath+"\nremoved "+changesetPath+"\n", "the standard output")
			files.HasContent(
				t,
				filepath.Join(dir, changelogPath),
				"# "+modulePath+"\n\n## 1.1.0\n\n### Minor Changes\n\n- "+
					commit[:7]+": Add the Unit type.\n\n## 1.0.0\n\n### Major Changes\n\n- Release the first version.\n",
				"the changelog",
			)
			files.Absent(t, filepath.Join(dir, changesetPath), "the changeset")
		})

		t.Run("version writes the plan with --dry-run and changes nothing", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{changesetPath: files.Text(changesetText)})
			var status int
			var stdout, stderr string
			files.Unchanged(t, os.DirFS(dir), func() {
				status, stdout, stderr = runRelease(t, dir, nil, "release", "version", "--dry-run")
			}, "the repository")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, modulePath+" minor 1.0.0 -> 1.1.0\n  "+changesetID+"\n", "the standard output")
		})

		t.Run("version writes the message of changesets without changesets", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "No unreleased changesets found.\n", "the standard output")
		})

		t.Run("version rewrites a stale go.sum without changesets", func(t *testing.T) {
			t.Parallel()
			dir := staleModules(t)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+staleSum+"\n", "the standard output")
			vcstest.Commit(t, dir, "rewrite the lockfiles")
			status, stdout, stderr = runRelease(t, dir, nil, "release", "ci", "select-mode")
			assert.Equal(t, status, statusOK, "the exit status of select-mode: "+stderr)
			assert.Equal(t, stdout, "mode publish\n", "the standard output of select-mode")
		})

		t.Run("version returns 1 for a go.sum that it cannot read", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := runRelease(t, unreadableSum(t), nil, "release", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: release: the lockfiles of the toolchain go: release: read "+staleSum+
				": ", "the standard error")
		})

		t.Run("version returns 1 outside a working tree of git without changesets", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{configPath: files.Text(gitConfig)})
			status, _, stderr := runRelease(t, dir, nil, "release", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
		})

		t.Run("version returns 1 for GITHUB_TOKEN without GITHUB_REPOSITORY", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{changesetPath: files.Text(changesetText)})
			status, _, stderr := runRelease(t, dir, map[string]string{"GITHUB_TOKEN": "token"}, "release", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: cli: set GITHUB_REPOSITORY to the repository on GitHub, as owner/name\n",
				"the standard error")
		})

		t.Run("version returns 1 for the changelog of GitHub without GITHUB_TOKEN", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, githubConfig, files.Tree{changesetPath: files.Text(changesetText)})
			status, _, stderr := runRelease(t, dir, nil, "release", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: release: invalid changelog: the changelog of GitHub, which needs the host "+
				"of the repository\n", "the standard error")
			files.HasContent(t, filepath.Join(dir, changelogPath), firstChangelog, "the changelog")
		})

		t.Run("version returns 1 for a changeset that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{".changeset/broken.md": files.Text("no front matter\n")})
			status, _, stderr := runRelease(t, dir, nil, "release", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: release: changeset: invalid changeset: broken.md:1: ",
				"the standard error")
		})

		t.Run("version returns 1 for a changeset of a package that the repository does not have", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{
				changesetPath: files.Text("---\n\"example.com/other\": minor\n---\n\nAdd the Unit type.\n"),
			})
			status, _, stderr := runRelease(t, dir, nil, "release", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: release: invalid changeset: "+changesetPath+`:2: the package `+
				`"example.com/other", which the repository does not have`+"\n", "the standard error")
		})

		t.Run("publish-plan writes the publish plan as JSON", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "publish-plan")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, publishPlan, "the standard output")
		})

		t.Run("publish-plan writes the publish plan into --output", func(t *testing.T) {
			t.Parallel()
			output := filepath.Join(t.TempDir(), "plan.json")
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "publish-plan",
				"--output", output)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Empty(t, stdout, "the standard output")
			files.HasContent(t, output, publishPlan, "the publish plan")
		})

		t.Run("publish-plan returns 1 when the plan cannot be written", func(t *testing.T) {
			t.Parallel()
			res := clitest.Main(t.Context(), program(goModule(t, gitConfig, nil), app.Register).Main, &clitest.Input{
				Args: []string{"release", "publish-plan"}, Vars: releaseVars(nil), FailStdout: true,
			})
			assert.Equal(t, res.Status, statusFailure, "the exit status")
			assert.Equal(t, res.Stderr, "ergon: cli: write the JSON: "+clitest.ErrStdout.Error()+"\n",
				"the standard error")
		})

		t.Run("publish-plan returns 1 outside a working tree of git", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{configPath: files.Text(gitConfig)})
			status, stdout, stderr := runRelease(t, dir, nil, "release", "publish-plan")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
		})

		t.Run("pack builds no artifact for a plan of tags", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			out := filepath.Join(t.TempDir(), "dist")
			status, stdout, stderr := runRelease(t, dir, nil, "release", "pack", "--out-dir", out)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Empty(t, stdout, "the standard output")
			files.Absent(t, out, "the directory of the artifacts")
		})

		t.Run("pack reads the publish plan of --from-publish-plan", func(t *testing.T) {
			t.Parallel()
			plan := filepath.Join(t.TempDir(), "plan.json")
			assert.NoError(t, os.WriteFile(plan, []byte(publishPlan), 0o600), "WriteFile of the plan")
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "pack",
				"--from-publish-plan", plan)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
		})

		t.Run("pack returns 1 for a plan file that does not exist", func(t *testing.T) {
			t.Parallel()
			plan := filepath.Join(t.TempDir(), "absent.json")
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "pack",
				"--from-publish-plan", plan)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: read the publish plan: ", "the standard error")
		})

		t.Run("pack returns 1 for a plan file that is not JSON", func(t *testing.T) {
			t.Parallel()
			plan := filepath.Join(t.TempDir(), "plan.json")
			assert.NoError(t, os.WriteFile(plan, []byte("plan\n"), 0o600), "WriteFile of the plan")
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "pack",
				"--from-publish-plan", plan)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: read the publish plan "+plan+": ", "the standard error")
		})

		t.Run("pack returns 1 for a plan of a package that the repository does not have", func(t *testing.T) {
			t.Parallel()
			plan := filepath.Join(t.TempDir(), "plan.json")
			other := `{"plan": [[{"kind": "publish", "toolchain": "go", "name": "example.com/other", "tag": "v1.0.0", ` +
				`"version": "1.0.0"}]], "version": 1}`
			assert.NoError(t, os.WriteFile(plan, []byte(other), 0o600), "WriteFile of the plan")
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "pack",
				"--from-publish-plan", plan)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: release: invalid publish plan: the package example.com/other, which the "+
				"repository does not have\n", "the standard error")
		})

		t.Run("publish tags the packages and pushes the tags to origin", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			remote := withOrigin(t, dir)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "publish")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "released "+modulePath+"@1.0.0\n", "the standard output")
			assert.Contains(t, stderr, newTag, "the standard error, which shows the output of git push")
			assert.Equal(t, vcstest.Git(t, remote, "tag", "--list"), "v1.0.0\n", "the tags of origin")
			assert.Equal(t, vcstest.Git(t, dir, "tag", "--list", "--format=%(contents)", "v1.0.0"),
				"### Major Changes\n\n- Release the first version.\n\n",
				"the annotation of the tag with its newline, and the newline of git tag --list")
		})

		t.Run("publish writes the released packages into --output", func(t *testing.T) {
			t.Parallel()
			output := filepath.Join(t.TempDir(), "released.json")
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "publish", "--no-git-tag",
				"--output", output)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			files.HasContent(t, output, releasedPackages+"\n", "the released packages")
		})

		t.Run("publish creates no tag with --no-git-tag", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "publish", "--no-git-tag")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "released "+modulePath+"@1.0.0\n", "the standard output")
			assert.Empty(t, vcstest.Git(t, dir, "tag", "--list"), "the tags")
		})

		t.Run("publish writes the outputs published and published-packages", func(t *testing.T) {
			t.Parallel()
			output := filepath.Join(t.TempDir(), "output")
			env := map[string]string{"GITHUB_OUTPUT": output}
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), env, "release", "publish", "--no-git-tag")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			files.HasContent(t, output, "published=true\npublished-packages="+releasedPackages+"\n", "the outputs")
		})

		t.Run("publish writes published false for a plan without packages", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			vcstest.Git(t, dir, "tag", "v1.0.0")
			output := filepath.Join(t.TempDir(), "output")
			env := map[string]string{"GITHUB_OUTPUT": output}
			status, stdout, stderr := runRelease(t, dir, env, "release", "publish")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Empty(t, stdout, "the standard output")
			files.HasContent(t, output, "published=false\npublished-packages=[]\n", "the outputs")
		})

		t.Run("publish creates the tags and the releases through GitHub in GitHub Actions", func(t *testing.T) {
			t.Parallel()
			h := &hub{}
			env := h.start(t)
			env["GITHUB_ACTIONS"] = "true"
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), env, "release", "publish")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "released "+modulePath+"@1.0.0\n", "the standard output")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/git/ref/tags/v1.0.0",
				"GET /repos/" + hubRepo + "/git/ref/tags/v1.0.0",
				"POST /repos/" + hubRepo + "/git/refs",
				"GET /repos/" + hubRepo + "/releases?per_page=100&page=1",
				"POST /repos/" + hubRepo + "/releases",
				"PATCH /repos/" + hubRepo + "/releases/7",
			}, "the requests to GitHub")
		})

		t.Run("ci homebrew commits the casks of the publish plan to the tap", func(t *testing.T) {
			t.Parallel()
			h := &hub{head: laterCommit}
			env := h.start(t)
			dir := goModule(t, gitConfig, nil)
			plan := filepath.Join(t.TempDir(), "plan.json")
			assert.NoError(t, os.WriteFile(plan, []byte(publishPlan), 0o600), "WriteFile of the plan")
			pack := files.Workspace(t, files.Tree{"casks/v1.0.0/demo.rb": files.Text("cask \"demo\" do\nend\n")})
			status, stdout, stderr := runRelease(t, dir, env, "release", "ci", "homebrew", "--tap", tapRepo,
				"--from-publish-plan", plan, "--from-pack-dir", pack)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "committed Casks/demo.rb to "+tapRepo+"\n", "the standard output")
			assert.Contains(t, h.commits()[0], `"headline":"demo 1.0.0"`, "the message of the commit")
		})

		t.Run("ci homebrew returns 1 for a plan file that does not exist", func(t *testing.T) {
			t.Parallel()
			plan := filepath.Join(t.TempDir(), "absent.json")
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "ci", "homebrew",
				"--tap", tapRepo, "--from-publish-plan", plan)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.HasPrefix(t, stderr, "ergon: cli: read the publish plan: ", "the standard error")
		})

		t.Run("ci homebrew returns 1 without GITHUB_REPOSITORY", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "ci", "homebrew",
				"--tap", tapRepo)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: cli: set GITHUB_REPOSITORY to the repository on GitHub, as owner/name\n",
				"the standard error")
		})

		t.Run("publish returns 1 in GitHub Actions without GITHUB_REPOSITORY", func(t *testing.T) {
			t.Parallel()
			env := map[string]string{"GITHUB_ACTIONS": "true"}
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), env, "release", "publish")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: cli: set GITHUB_REPOSITORY to the repository on GitHub, as owner/name\n",
				"the standard error")
		})

		t.Run("publish returns 1 for a repository without a commit", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{
				"go.mod":      files.Text("module " + modulePath + "\n\ngo 1.27\n"),
				changelogPath: files.Text(firstChangelog),
				configPath:    files.Text(gitConfig),
			})
			status, stdout, stderr := runRelease(t, dir, nil, "release", "publish")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
		})

		t.Run("publish returns 1 for a tag at another commit", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			plan := filepath.Join(t.TempDir(), "plan.json")
			assert.NoError(t, os.WriteFile(plan, []byte(publishPlan), 0o600), "WriteFile of the plan")
			first := strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))
			vcstest.Git(t, dir, "tag", "v1.0.0")
			second := vcstest.Commit(t, dir, "second")
			status, stdout, stderr := runRelease(t, dir, nil, "release", "publish", "--from-publish-plan", plan)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: release: tag at another commit: v1.0.0 is at "+first+", and the release "+
				"of "+modulePath+" is at "+second+"\n", "the standard error")
		})

		t.Run("publish returns 1 for a go.sum that records another content of a module", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runRelease(t, staleModules(t), nil, "release", "publish")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: release: stale lockfiles: "+staleSum+", which ergon release version "+
				"rewrites\n", "the standard error")
		})

		t.Run("publish returns 1 when an output cannot be written", func(t *testing.T) {
			t.Parallel()
			output := filepath.Join(t.TempDir(), "absent", "output")
			env := map[string]string{"GITHUB_OUTPUT": output}
			status, stdout, stderr := runRelease(
				t,
				goModule(t, gitConfig, nil),
				env,
				"release",
				"publish",
				"--no-git-tag",
			)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stdout, "released "+modulePath+"@1.0.0\n", "the standard output")
			assert.HasPrefix(t, stderr, "ergon: cli: write the output published: ", "the standard error")
		})

		t.Run("git-tag tags the packages whose tags are missing and pushes the tags", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			remote := withOrigin(t, dir)
			status, stdout, stderr := runRelease(t, dir, nil, "release", "git-tag")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "released "+modulePath+"@1.0.0\n", "the standard output")
			assert.Contains(t, stderr, newTag, "the standard error, which shows the output of git push")
			assert.Equal(t, vcstest.Git(t, remote, "tag", "--list"), "v1.0.0\n", "the tags of origin")
		})

		t.Run("git-tag returns 1 outside a working tree of git", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{configPath: files.Text(gitConfig)})
			status, _, stderr := runRelease(t, dir, nil, "release", "git-tag")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
		})

		t.Run("select-mode writes version for a repository with changesets", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{changesetPath: files.Text(changesetText)})
			output := filepath.Join(t.TempDir(), "output")
			env := map[string]string{"GITHUB_OUTPUT": output}
			status, stdout, stderr := runRelease(t, dir, env, "release", "ci", "select-mode")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "mode version\n", "the standard output")
			files.HasContent(t, output, "mode=version\n", "the outputs")
		})

		t.Run("select-mode writes publish and the publish plan for a missing tag", func(t *testing.T) {
			t.Parallel()
			plan := filepath.Join(t.TempDir(), "plan.json")
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "ci", "select-mode",
				"--output", plan)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "mode publish\n", "the standard output")
			files.HasContent(t, plan, publishPlan, "the publish plan")
		})

		t.Run("select-mode writes none for a repository whose tags are at their versions", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			vcstest.Git(t, dir, "tag", "v1.0.0")
			status, stdout, stderr := runRelease(t, dir, nil, "release", "ci", "select-mode")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "mode none\n", "the standard output")
		})

		t.Run("select-mode writes a stale go.sum and version", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runRelease(t, staleModules(t), nil, "release", "ci", "select-mode")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "stale "+staleSum+"\nmode version\n", "the standard output")
		})

		t.Run("select-mode returns 1 for a go.sum that it cannot read", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := runRelease(t, unreadableSum(t), nil, "release", "ci", "select-mode")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: release: the lockfiles of the toolchain go: release: read "+staleSum+
				": ", "the standard error")
		})

		t.Run("select-mode returns 1 for a changeset that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{".changeset/broken.md": files.Text("no front matter\n")})
			status, _, stderr := runRelease(t, dir, nil, "release", "ci", "select-mode")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: release: changeset: invalid changeset: broken.md:1: ",
				"the standard error")
		})

		t.Run("select-mode returns 1 for an --output that cannot be written", func(t *testing.T) {
			t.Parallel()
			plan := filepath.Join(t.TempDir(), "absent", "plan.json")
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "ci", "select-mode",
				"--output", plan)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: write "+plan+": ", "the standard error")
		})

		t.Run("select-mode returns 1 when the output cannot be written", func(t *testing.T) {
			t.Parallel()
			env := map[string]string{"GITHUB_OUTPUT": filepath.Join(t.TempDir(), "absent", "output")}
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), env, "release", "ci", "select-mode")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: write the output mode: ", "the standard error")
		})

		t.Run("select-mode returns 1 outside a working tree of git", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{configPath: files.Text(gitConfig)})
			status, _, stderr := runRelease(t, dir, nil, "release", "ci", "select-mode")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
		})

		t.Run("ci version commits the release and opens the version pull request", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, changesetPath, changesetText)
			head := vcstest.Commit(t, dir, "add a changeset")
			h := &hub{head: head}
			env := h.start(t)
			output := filepath.Join(t.TempDir(), "output")
			env["GITHUB_OUTPUT"] = output
			status, stdout, stderr := runRelease(t, dir, env, "release", "ci", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+changelogPath+"\nremoved "+changesetPath+"\npull request 7\nmarked "+
				hubCommit+" as the version of "+head+"\n", "the standard output")
			files.HasContent(t, output, "pull-request-number=7\n", "the outputs")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/git/ref/heads/main",
				"GET /repos/" + hubRepo + "/git/ref/heads/ergon-staging/ergon-release/main",
				"POST /repos/" + hubRepo + "/git/refs",
				"POST " + graphqlPath,
				"GET /repos/" + hubRepo + "/git/ref/heads/ergon-release/main",
				"POST /repos/" + hubRepo + "/git/refs",
				"DELETE /repos/" + hubRepo + "/git/refs/heads/ergon-staging/ergon-release/main",
				"GET /repos/" + hubRepo + "/pulls?base=main&head=o%3Aergon-release%2Fmain&state=open",
				"POST /repos/" + hubRepo + "/pulls",
				"GET /repos/" + hubRepo + "/git/ref/heads/ergon-release/main",
				"POST /repos/" + hubRepo + "/statuses/" + hubCommit,
			}, "the requests to GitHub")
			assert.Equal(t, h.marks(), []string{`{"state":"success","context":"` + release.VersionStatus +
				`","description":"The version of ` + head + ` that ergon release ci version wrote."}`},
				"the status of the version commit, without the page of a run outside GitHub Actions")
			commits := h.commits()
			assert.Length(t, commits, 1, "the commits")
			var request struct {
				Variables struct {
					Input struct {
						Branch struct {
							Name string `json:"branchName"`
						} `json:"branch"`
						Message struct {
							Headline string `json:"headline"`
						} `json:"message"`
						ExpectedHead string `json:"expectedHeadOid"`
						FileChanges  struct {
							Additions []struct {
								Path string `json:"path"`
							} `json:"additions"`
							Deletions []struct {
								Path string `json:"path"`
							} `json:"deletions"`
						} `json:"fileChanges"`
					} `json:"input"`
				} `json:"variables"`
			}
			assert.NoError(t, json.Unmarshal([]byte(commits[0]), &request), "Unmarshal of the commit")
			input := request.Variables.Input
			expect.That(t, input.Branch.Name).Equal("ergon-staging/ergon-release/main", "the branch of the commit")
			expect.That(t, input.Message.Headline).Equal("chore: version packages", "the message of the commit")
			expect.That(t, input.ExpectedHead).Equal(head, "the head of the branch")
			assert.Length(t, input.FileChanges.Additions, 1, "the files of the commit")
			expect.That(t, input.FileChanges.Additions[0].Path).Equal(changelogPath, "the file of the commit")
			assert.Length(t, input.FileChanges.Deletions, 1, "the deletions of the commit")
			expect.That(t, input.FileChanges.Deletions[0].Path).Equal(changesetPath, "the deletion of the commit")
		})

		t.Run("ci version changes nothing without changesets", func(t *testing.T) {
			t.Parallel()
			h := &hub{}
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), h.start(t), "release", "ci", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "No unreleased changesets found.\n", "the standard output")
			assert.Empty(t, h.calls(), "the requests to GitHub")
		})

		noRelease := []struct {
			name          string
			giveChangeset string
			wantStdout    string
			giveArgs      []string
		}{
			{
				name:          "ci version keeps an empty changeset without a pull request",
				giveChangeset: emptyChangeset,
				giveArgs:      []string{"release", "ci", "version"},
				wantStdout:    "No unreleased changesets found.\n",
			},
			{
				name:          "ci version keeps a changeset at none without a pull request",
				giveChangeset: noneChangeset,
				giveArgs:      []string{"release", "ci", "version"},
				wantStdout:    "No unreleased changesets found.\n",
			},
			{
				name:          "select-mode writes publish for an empty changeset",
				giveChangeset: emptyChangeset,
				giveArgs:      []string{"release", "ci", "select-mode"},
				wantStdout:    "mode publish\n",
			},
			{
				name:          "select-mode writes publish for a changeset at none",
				giveChangeset: noneChangeset,
				giveArgs:      []string{"release", "ci", "select-mode"},
				wantStdout:    "mode publish\n",
			},
		}
		for _, tt := range noRelease {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := goModule(t, gitConfig, nil)
				write(t, dir, changesetPath, tt.giveChangeset)
				vcstest.Commit(t, dir, "add a changeset")
				h := &hub{}
				status, stdout, stderr := runRelease(t, dir, h.start(t), tt.giveArgs...)
				assert.Equal(t, status, statusOK, "the exit status: "+stderr)
				assert.Equal(t, stdout, tt.wantStdout, "the standard output")
				assert.Empty(t, h.calls(), "the requests to GitHub")
				files.HasContent(t, filepath.Join(dir, filepath.FromSlash(changesetPath)), tt.giveChangeset,
					"the changeset")
			})
		}

		t.Run("ci version proposes a stale go.sum without changesets", func(t *testing.T) {
			t.Parallel()
			dir := staleModules(t)
			h := &hub{head: strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))}
			status, stdout, stderr := runRelease(t, dir, h.start(t), "release", "ci", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+staleSum+"\npull request 7\nmarked "+hubCommit+" as the version of "+
				h.head+"\n", "the standard output")
			commits := h.commits()
			assert.Length(t, commits, 1, "the commits")
			assert.Contains(t, commits[0], `"path":"`+staleSum+`"`, "the lockfile of the commit")
		})

		t.Run("ci version skips a commit that is no longer the head of the base branch", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, changesetPath, changesetText)
			head := vcstest.Commit(t, dir, "add a changeset")
			h := &hub{head: laterCommit}
			env := h.start(t)
			output := filepath.Join(t.TempDir(), "output")
			env["GITHUB_OUTPUT"] = output
			status, stdout, stderr := runRelease(t, dir, env, "release", "ci", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+changelogPath+"\nremoved "+changesetPath+"\nskipped "+head+
				", which is no longer the head of main\n", "the standard output")
			files.Absent(t, output, "the outputs, which name no pull request")
			assert.Equal(t, h.calls(), []string{"GET /repos/" + hubRepo + "/git/ref/heads/main"},
				"the requests to GitHub, which change nothing")
		})

		t.Run("ci version marks the version commit with the page of its run", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, changesetPath, changesetText)
			head := vcstest.Commit(t, dir, "add a changeset")
			h := &hub{head: head}
			env := h.start(t)
			env["GITHUB_RUN_ID"] = "42"
			status, _, stderr := runRelease(t, dir, env, "release", "ci", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			marks := h.marks()
			assert.Length(t, marks, 1, "the statuses")
			assert.Contains(t, marks[0], `"target_url":"`+env["GITHUB_SERVER_URL"]+"/"+hubRepo+"/actions/runs/42"+`"`,
				"the page of the status")
		})

		t.Run("ci version returns the error of the status of the version commit", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, changesetPath, changesetText)
			head := vcstest.Commit(t, dir, "add a changeset")
			h := &hub{head: head, fail: "POST /repos/" + hubRepo + "/statuses/"}
			status, _, stderr := runRelease(t, dir, h.start(t), "release", "ci", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: forge: GitHub failed: POST ", "the standard error")
		})

		t.Run("ci version returns 1 without GITHUB_REPOSITORY", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "ci", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: cli: set GITHUB_REPOSITORY to the repository on GitHub, as owner/name\n",
				"the standard error")
		})

		t.Run("ci version returns 1 without GITHUB_TOKEN", func(t *testing.T) {
			t.Parallel()
			env := map[string]string{"GITHUB_REPOSITORY": hubRepo}
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), env, "release", "ci", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: forge: no token: set GITHUB_TOKEN\n", "the standard error")
		})

		t.Run(
			"ci version returns 1 for a changeset of a package that the repository does not have",
			func(t *testing.T) {
				t.Parallel()
				dir := goModule(t, gitConfig, files.Tree{
					changesetPath: files.Text("---\n\"example.com/other\": minor\n---\n\nAdd the Unit type.\n"),
				})
				h := &hub{}
				status, _, stderr := runRelease(t, dir, h.start(t), "release", "ci", "version")
				assert.Equal(t, status, statusFailure, "the exit status")
				assert.HasPrefix(t, stderr, "ergon: release: invalid changeset: ", "the standard error")
				assert.Empty(t, h.calls(), "the requests to GitHub")
			},
		)

		t.Run("ci version returns 1 for a repository without a commit", func(t *testing.T) {
			t.Parallel()
			dir := vcstest.Repository(t, files.Tree{
				"go.mod":      files.Text("module " + modulePath + "\n\ngo 1.27\n"),
				changelogPath: files.Text(firstChangelog),
				configPath:    files.Text(gitConfig),
				changesetPath: files.Text(changesetText),
			})
			h := &hub{}
			status, _, stderr := runRelease(t, dir, h.start(t), "release", "ci", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
		})

		t.Run("ci version returns 1 for links of the changelog that GitHub refuses", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, githubConfig, nil)
			write(t, dir, changesetPath, changesetText)
			vcstest.Commit(t, dir, "add a changeset")
			h := &hub{fail: "POST " + graphqlPath}
			status, _, stderr := runRelease(t, dir, h.start(t), "release", "ci", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: release: the changelog of "+modulePath+": forge: GitHub failed: ",
				"the standard error")
			assert.Equal(t, h.calls(), []string{"POST " + graphqlPath}, "the requests to GitHub")
		})

		t.Run("ci version returns 1 for a changed file that it cannot read", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			write(t, dir, changesetPath, changesetText)
			vcstest.Commit(t, dir, "add a changeset")
			assert.NoError(t, os.Remove(filepath.Join(dir, moduleFile)), "Remove of the source")
			assert.NoError(t, os.MkdirAll(filepath.Join(dir, moduleFile, "inner"), 0o755), "the directory in its place")
			h := &hub{}
			status, _, stderr := runRelease(t, dir, h.start(t), "release", "ci", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: release: read "+moduleFile+": ", "the standard error")
			assert.Empty(t, h.calls(), "the requests to GitHub")
		})

		t.Run("ci version returns the error of GitHub", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, files.Tree{changesetPath: files.Text(changesetText)})
			h := &hub{fail: "GET /repos/" + hubRepo + "/git/ref/heads/"}
			status, _, stderr := runRelease(t, dir, h.start(t), "release", "ci", "version")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: forge: GitHub failed: GET ", "the standard error")
		})

		t.Run("ci skip writes true for a push whose content passed", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			head := strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))
			h := &hub{runs: passedRun}
			env := h.start(t)
			output := filepath.Join(t.TempDir(), "output")
			env["GITHUB_OUTPUT"] = output
			status, stdout, stderr := runRelease(t, dir, env, "release", "ci", "skip")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "skip: "+runPage+" covers the content of "+head+"\n", "the standard output")
			files.HasContent(t, output, "skip=true\n", "the outputs")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/actions/workflows/ci.yml/runs?head_sha=" + head + "&per_page=1&status=success",
			}, "the requests to GitHub")
		})

		t.Run("ci skip writes false for a push without a passed run", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			head := strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))
			h := &hub{runs: noRun}
			env := h.start(t)
			output := filepath.Join(t.TempDir(), "output")
			env["GITHUB_OUTPUT"] = output
			status, stdout, stderr := runRelease(t, dir, env, "release", "ci", "skip")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "test: no passed run of ci.yml covers the content of "+head+"\n",
				"the standard output")
			files.HasContent(t, output, "skip=false\n", "the outputs")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/actions/workflows/ci.yml/runs?head_sha=" + head + "&per_page=1&status=success",
				"GET /repos/" + hubRepo + "/git/commits/" + head,
				"GET /repos/" + hubRepo + "/commits/" + head + "/pulls",
				"GET /repos/" + hubRepo + "/commits/" + head + "/pulls",
				"GET /repos/" + hubRepo + "/commits/" + head + "/status?per_page=100",
			}, "the requests to GitHub")
		})

		t.Run("ci skip writes true for the version commit of a pull request of --commit", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			tree := strings.TrimSpace(vcstest.Git(t, dir, "log", "-1", "--format=%T", "HEAD"))
			h := &hub{runs: passedRun, marked: laterCommit, tree: tree, parent: hubCommit}
			env := h.start(t)
			output := filepath.Join(t.TempDir(), "output")
			env["GITHUB_OUTPUT"] = output
			status, stdout, stderr := runRelease(t, dir, env, "release", "ci", "skip", "--commit", laterCommit)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "skip: "+runPage+" covers the content of "+laterCommit+"\n", "the standard output")
			files.HasContent(t, output, "skip=true\n", "the outputs")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/commits/" + laterCommit + "/status?per_page=100",
				"GET /repos/" + hubRepo + "/git/commits/" + laterCommit,
				"GET /repos/" + hubRepo + "/git/commits/" + laterCommit,
				"GET /repos/" + hubRepo + "/actions/workflows/ci.yml/runs?head_sha=" + hubCommit +
					"&per_page=1&status=success",
			}, "the requests to GitHub, which read no run of the merge that the pull request checks out")
		})

		t.Run("ci skip writes false and a warning without GITHUB_REPOSITORY", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			head := strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))
			status, stdout, stderr := runRelease(t, dir, nil, "release", "ci", "skip")
			assert.Equal(t, status, statusOK, "the exit status")
			assert.Equal(t, stdout, "test: no passed run of ci.yml covers the content of "+head+"\n",
				"the standard output")
			assert.Equal(t, stderr, "ergon: warning: cli: set GITHUB_REPOSITORY to the repository on GitHub, as "+
				"owner/name, so the run tests "+head+"\n", "the standard error")
		})

		t.Run("ci skip writes false and a warning for an error of GitHub", func(t *testing.T) {
			t.Parallel()
			h := &hub{fail: "GET /repos/" + hubRepo + "/actions/"}
			status, stdout, stderr := runRelease(t, goModule(t, gitConfig, nil), h.start(t), "release", "ci", "skip")
			assert.Equal(t, status, statusOK, "the exit status")
			assert.HasPrefix(t, stdout, "test: ", "the standard output")
			assert.HasPrefix(t, stderr, "ergon: warning: forge: GitHub failed: GET ", "the standard error")
		})

		t.Run("ci skip returns 1 outside a working tree of git", func(t *testing.T) {
			t.Parallel()
			h := &hub{runs: passedRun}
			status, _, stderr := runRelease(t, t.TempDir(), h.start(t), "release", "ci", "skip")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
			assert.Empty(t, h.calls(), "the requests to GitHub")
		})

		t.Run("ci verify writes the page of a run of ci.yml that passed on the commit", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			head := strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))
			h := &hub{runs: passedRun}
			status, stdout, stderr := runRelease(t, dir, h.start(t), "release", "ci", "verify")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "ci.yml passed: "+runPage+"\n", "the standard output")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/actions/workflows/ci.yml/runs?head_sha=" + head + "&per_page=1&status=success",
			}, "the requests to GitHub")
		})

		t.Run("ci verify reads the runs of the workflow of --workflow", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			head := strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))
			h := &hub{runs: passedRun}
			args := []string{"release", "ci", "verify", "--workflow", "gate.yml"}
			status, stdout, stderr := runRelease(t, dir, h.start(t), args...)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "gate.yml passed: "+runPage+"\n", "the standard output")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/actions/workflows/gate.yml/runs?head_sha=" + head + "&per_page=1&status=success",
			}, "the requests to GitHub")
		})

		t.Run("ci verify returns 1 for a commit without a run that passed", func(t *testing.T) {
			t.Parallel()
			dir := goModule(t, gitConfig, nil)
			head := strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))
			h := &hub{runs: noRun}
			status, stdout, stderr := runRelease(t, dir, h.start(t), "release", "ci", "verify")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: release: the gate did not pass: no run of ci.yml passed on the content of "+
				head+", so rerun this job after the run of ci.yml for its push passes\n", "the standard error")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/actions/workflows/ci.yml/runs?head_sha=" + head + "&per_page=1&status=success",
				"GET /repos/" + hubRepo + "/git/commits/" + head,
				"GET /repos/" + hubRepo + "/commits/" + head + "/pulls",
			}, "the requests to GitHub")
		})

		t.Run("ci verify returns 1 without GITHUB_REPOSITORY", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), nil, "release", "ci", "verify")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: cli: set GITHUB_REPOSITORY to the repository on GitHub, as owner/name\n",
				"the standard error")
		})

		t.Run("ci verify returns 1 outside a working tree of git", func(t *testing.T) {
			t.Parallel()
			h := &hub{runs: passedRun}
			status, _, stderr := runRelease(t, t.TempDir(), h.start(t), "release", "ci", "verify")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: vcs: git failed: ", "the standard error")
			assert.Empty(t, h.calls(), "the requests to GitHub")
		})

		t.Run("ci verify returns the error of GitHub", func(t *testing.T) {
			t.Parallel()
			h := &hub{fail: "GET /repos/" + hubRepo + "/actions/"}
			status, _, stderr := runRelease(t, goModule(t, gitConfig, nil), h.start(t), "release", "ci", "verify")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: forge: GitHub failed: GET ", "the standard error")
		})
	})
}

// goModule returns a working tree of git on the branch main with one commit: a Go module at the
// root at the version 1.0.0 of its changelog, the configuration of changesets config, and the files
// of extra.
func goModule(t *testing.T, config string, extra files.Tree) string {
	t.Helper()
	tree := files.Tree{
		"go.mod":      files.Text("module " + modulePath + "\n\ngo 1.27\n"),
		moduleFile:    files.Text("// Package demo is the module of the release cases.\npackage demo\n"),
		changelogPath: files.Text(firstChangelog),
		configPath:    files.Text(config),
	}
	maps.Copy(tree, extra)
	dir := vcstest.Repository(t, tree)
	vcstest.Git(t, dir, "symbolic-ref", "HEAD", "refs/heads/main")
	vcstest.Commit(t, dir, "first")
	return dir
}

// staleModules returns the working tree of goModule with the module b in go.work, which imports the
// module of the cases at its version 1.0.0, and whose go.sum records other hashes of it, as a change
// to the module after its version commit leaves them.
func staleModules(t *testing.T) string {
	t.Helper()
	return goModule(t, gitConfig, files.Tree{
		"go.work":  files.Text("go 1.27\n\nuse (\n\t.\n\t./b\n)\n"),
		"b/go.mod": files.Text("module " + modulePath + "/b\n\ngo 1.27\n\nrequire " + modulePath + " v1.0.0\n"),
		"b/b.go":   files.Text("// Package b imports the module.\npackage b\n\nimport _ \"" + modulePath + "\"\n"),
		staleSum:   files.Text(modulePath + " v1.0.0 h1:other=\n" + modulePath + " v1.0.0/go.mod h1:other=\n"),
	})
}

// unreadableSum returns the working tree of goModule with the module b in go.work, whose go.sum is a
// directory.
func unreadableSum(t *testing.T) string {
	t.Helper()
	return goModule(t, gitConfig, files.Tree{
		"go.work":              files.Text("go 1.27\n\nuse (\n\t.\n\t./b\n)\n"),
		"b/go.mod":             files.Text("module " + modulePath + "/b\n\ngo 1.27\n"),
		staleSum + "/inner.md": files.Text("inner\n"),
	})
}

// withOrigin adds a bare repository as the remote origin of the working tree dir, and returns the
// bare repository.
func withOrigin(t *testing.T, dir string) string {
	t.Helper()
	remote := t.TempDir()
	vcstest.Git(t, remote, "init", "--bare", "--quiet")
	vcstest.Git(t, dir, "remote", "add", "origin", remote)
	return remote
}

// runRelease runs the command line args with the registration of ergon in the working directory
// dir, with the variables of env, and returns the exit status, the standard output and the
// standard error.
func runRelease(t *testing.T, dir string, env map[string]string, args ...string) (status int, stdout, stderr string) {
	t.Helper()
	res := clitest.Main(
		t.Context(),
		program(dir, app.Register).Main,
		&clitest.Input{Args: args, Vars: releaseVars(env)},
	)
	return res.Status, res.Stdout, res.Stderr
}

// releaseVars returns the environment of a release case: the environment of environ with each
// variable of releaseVariables set to an empty value and then the variables of env, so the
// environment of the test sets none of the variables that the release commands read.
func releaseVars(env map[string]string) map[string]string {
	vars := environ()
	for _, name := range releaseVariables {
		vars[name] = ""
	}
	maps.Copy(vars, env)
	return vars
}
