// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
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
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/internal/cli"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
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

// releaseHelp is the help of ergon release, pinned because a person reads it.
const releaseHelp = `ergon release releases the packages of the repository with the changesets of
.changeset and the configuration .changeset/config.json, in the format of
changesets: a pull request adds a changeset that names the packages it changes
with their bumps, the version pull request writes the new versions and the
changelogs, and its merge publishes and tags the packages. The packages are the
packages that each toolchain of ergon discovers, such as the modules of go.work.

Usage:
  ergon release [flags]
  ergon release [command]

Available Commands:
  add          Add a changeset
  ci           Run the jobs of the release workflow
  git-tag      Tag the packages whose tags are missing
  pack         Build the artifacts of the packages of a publish plan
  publish      Publish and tag the packages of a publish plan
  publish-plan Write the packages that a publish uploads or tags
  status       Report the changes of the branch that no changeset releases
  version      Write the new versions, the requirements and the changelogs

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command

Use "ergon release [command] --help" for more information about a command.
`

// addChangesetHelp is the help of ergon release add, pinned because a person reads it.
const addChangesetHelp = `ergon release add writes a changeset into .changeset: the packages that --bump
names, each with its bump, and the summary of --message as the entry of their
changelogs. --empty writes a changeset without packages, which satisfies ergon
release status for a change that releases nothing. --open opens the changeset
in the editor of VISUAL or EDITOR after it writes it.

Usage:
  ergon release add [flags]

Examples:
  ergon release add --bump go.dokimi.dev/ergon/core=minor -m "Add the Unit type."

Flags:
      --bump name=level   release name=level, where level is major, minor, patch or none
      --empty             write a changeset without packages
  -m, --message summary   the summary of the changeset
      --open              open the changeset in the editor

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// statusHelp is the help of ergon release status, pinned because a person reads it.
const statusHelp = `ergon release status writes the release plan of the changesets that the branch
added since --since, the base branch by default. It reports each package that
the branch changed without a changeset that names it, and each requirement of a
package on another package of the repository that excludes the current version
of the other. The exit status is 1 when it reports either. --output writes the
plan as JSON into a file.

Usage:
  ergon release status [flags]

Flags:
      --output file   write the release plan as JSON into file
      --since ref     compare against ref, the base branch by default
      --verbose       list the changesets of each release

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// versionHelp is the help of ergon release version, pinned because a person reads it.
const versionHelp = `ergon release version writes the release plan of the changesets into the
repository: the new version of each package, the rewritten requirements of its
dependents, the entries of the changelogs and the refreshed lockfiles. It
removes the changesets that it consumed, and restores every file when a step
fails. Without changesets it rewrites the lockfiles that record the earlier
content of a package whose release waits for its publish. --dry-run writes the
plan and changes nothing.

Usage:
  ergon release version [flags]

Flags:
      --dry-run   write the release plan and change nothing

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// publishPlanHelp is the help of ergon release publish-plan, pinned because a person reads it.
const publishPlanHelp = `ergon release publish-plan writes the publish plan as JSON: each package whose
registry lacks its version, and each package whose tag is its release and is
missing, in chunks of dependency order. --output writes the plan into a file.

Usage:
  ergon release publish-plan [flags]

Flags:
      --output file   write the publish plan into file

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// packHelp is the help of ergon release pack, pinned because a person reads it.
const packHelp = `ergon release pack builds the artifacts of each package of the publish plan
that its registry receives as an artifact, into --out-dir.

Usage:
  ergon release pack [flags]

Flags:
      --from-publish-plan file   read the publish plan from file
      --out-dir dir              build the artifacts into dir (default "dist")

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// publishHelp is the help of ergon release publish, pinned because a person reads it.
const publishHelp = `ergon release publish uploads each package of the publish plan whose registry
lacks its version, and then tags each package at HEAD with the section of its
changelog as the notes. On a workstation it creates annotated tags and pushes
them to origin in one push. In GitHub Actions it creates the tags and the
GitHub Releases through the API of GitHub. It refuses a plan whose lockfiles
record other content of its packages than the working tree. --no-git-tag
creates no tag.

Usage:
  ergon release publish [flags]

Flags:
      --from-pack-dir dir        upload the artifacts of dir (default "dist")
      --from-publish-plan file   read the publish plan from file
      --no-git-tag               create no tag
      --output file              write the released packages as JSON into file

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// gitTagHelp is the help of ergon release git-tag, pinned because a person reads it.
const gitTagHelp = `ergon release git-tag creates the annotated tag of each package whose tag at
its version is missing, at HEAD, and pushes the tags to origin in one push. It
refuses a plan whose lockfiles record other content of its packages than the
working tree.

Usage:
  ergon release git-tag [flags]

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// ciHelp is the help of ergon release ci, pinned because a person reads it.
const ciHelp = `ergon release ci runs the steps of the jobs of the release workflow in GitHub
Actions, and writes their outputs into the file of GITHUB_OUTPUT.

Usage:
  ergon release ci [flags]
  ergon release ci [command]

Available Commands:
  select-mode Choose the job that the release workflow runs
  verify      Verify that a CI run passed on the content of the commit
  version     Write the release and open the version pull request

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command

Use "ergon release ci [command] --help" for more information about a command.
`

// selectModeHelp is the help of ergon release ci select-mode, pinned because a person reads it.
const selectModeHelp = `ergon release ci select-mode writes the mode of the release workflow: version
for a repository with changesets or with lockfiles that record the earlier
content of a package of the publish plan, publish for a publish plan with a
package, and none otherwise. It writes each such lockfile. --output writes the
publish plan into a file for the job publish.

Usage:
  ergon release ci select-mode [flags]

Flags:
      --output file   write the publish plan into file

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// ciVersionHelp is the help of ergon release ci version, pinned because a person reads it.
const ciVersionHelp = `ergon release ci version writes the release plan of the changesets as ergon
release version does. It then commits the changes on the branch
ergon-release/<base> through the API of GitHub, which signs the commits, and
opens or updates the version pull request into the base branch. Without
changesets it proposes the lockfiles that ergon release version rewrites, and
otherwise changes nothing. It skips a commit of HEAD that is no longer the head
of the base branch, and leaves the pull request to the run of the newer head.

Usage:
  ergon release ci version [flags]

Flags:
      --title title   the title of the pull request and its commits (default "chore: version packages")

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// ciVerifyHelp is the help of ergon release ci verify, pinned because a person reads it.
const ciVerifyHelp = `ergon release ci verify finds a run of the workflow --workflow that passed on the
content of the commit of HEAD, through the API of GitHub: a run of the commit
itself, such as the run of its push or of its merge group, or a run of the head
of a pull request that merged the commit with the same tree. It reads each run
once and waits for none. The exit status is 1 without such a run.

Usage:
  ergon release ci verify [flags]

Flags:
      --workflow file   find a passed run of the workflow file (default "ci.yml")

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// releaseVariables are the variables of GitHub Actions and of the editors that the release
// commands read, which every release case sets to an empty value before its own.
var releaseVariables = []string{
	"GITHUB_TOKEN", "GITHUB_REPOSITORY", "GITHUB_API_URL", "GITHUB_GRAPHQL_URL", "GITHUB_SERVER_URL",
	"GITHUB_OUTPUT", "GITHUB_ACTIONS", "VISUAL", "EDITOR",
}

// errRandom is the error of a source of random digits that fails.
var errRandom = errors.New("random: failed")

// hub is a fake API of GitHub for the release cases. It responds as GitHub does for a repository
// whose branch main points at head, without the branch of the version pull request, without tags and
// without pull requests, and makes each commit as hubCommit with the tree hubTree. It responds with
// runs to a request for the runs of a workflow, and with the status 500 to a request whose method
// and path start with fail. It is safe for concurrent use, as the goroutines of a server use it.
type hub struct {
	// fail is the start of the method and the path of the requests that fail, or empty for none.
	fail string

	// head is the commit of the branch main, or empty for a repository without the branch.
	head string

	// runs is the body of the response to a request for the runs of a workflow.
	runs string

	// requests are the method and the path with the query of each request, in their order.
	requests []string

	// graphql are the bodies of the requests of GraphQL, in their order.
	graphql []string

	// pulls are the bodies of the pull requests that the requests open, in their order.
	pulls []string

	// mu guards requests, graphql and pulls.
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
	h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case h.fail != "" && strings.HasPrefix(call, h.fail):
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"message":"failed"}`)
	case r.Method == http.MethodGet && h.head != "" && strings.HasSuffix(r.URL.Path, "/git/ref/heads/main"):
		_, _ = io.WriteString(w, `{"ref":"refs/heads/main","object":{"type":"commit","sha":"`+h.head+`"}}`)
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/ref/"):
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/commits/"):
		_, _ = io.WriteString(w, `{"tree":{"sha":"`+hubTree+`"}}`)
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

func TestRelease(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		help := []struct {
			name   string
			args   []string
			stdout string
		}{
			{name: "writes the help of release", args: []string{"release", "--help"}, stdout: releaseHelp},
			{
				name:   "writes the help of release add",
				args:   []string{"release", "add", "--help"},
				stdout: addChangesetHelp,
			},
			{
				name:   "writes the help of release status",
				args:   []string{"release", "status", "--help"},
				stdout: statusHelp,
			},
			{
				name: "writes the help of release version", args: []string{"release", "version", "--help"},
				stdout: versionHelp,
			},
			{
				name: "writes the help of release publish-plan", args: []string{"release", "publish-plan", "--help"},
				stdout: publishPlanHelp,
			},
			{name: "writes the help of release pack", args: []string{"release", "pack", "--help"}, stdout: packHelp},
			{
				name: "writes the help of release publish", args: []string{"release", "publish", "--help"},
				stdout: publishHelp,
			},
			{
				name: "writes the help of release git-tag", args: []string{"release", "git-tag", "--help"},
				stdout: gitTagHelp,
			},
			{name: "writes the help of release ci", args: []string{"release", "ci", "--help"}, stdout: ciHelp},
			{
				name:   "writes the help of release ci select-mode",
				args:   []string{"release", "ci", "select-mode", "--help"},
				stdout: selectModeHelp,
			},
			{
				name: "writes the help of release ci version", args: []string{"release", "ci", "version", "--help"},
				stdout: ciVersionHelp,
			},
			{
				name: "writes the help of release ci verify", args: []string{"release", "ci", "verify", "--help"},
				stdout: ciVerifyHelp,
			},
		}
		for _, tt := range help {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				status, stdout, stderr := run(t, t.TempDir(), tt.args...)
				assert.Equal(t, status, statusOK, "the exit status")
				assert.Equal(t, stdout, tt.stdout, "the standard output")
				assert.Empty(t, stderr, "the standard error")
			})
		}

		usage := []struct {
			name   string
			args   []string
			stderr string
		}{
			{
				name: "returns 2 for release without a subcommand",
				args: []string{"release"},
				stderr: "ergon: cli: release needs a subcommand: add, ci, git-tag, pack, publish, publish-plan, status, " +
					"version\nRun 'ergon release --help' for usage.\n",
			},
			{
				name: "returns 2 for release ci without a subcommand",
				args: []string{"release", "ci"},
				stderr: "ergon: cli: ci needs a subcommand: select-mode, verify, version\n" +
					"Run 'ergon release ci --help' for usage.\n",
			},
			{
				name: "returns 2 for release add without --bump and without --empty", args: []string{"release", "add"},
				stderr: "ergon: cli: name the packages with --bump, or pass --empty\n" +
					"Run 'ergon release add --help' for usage.\n",
			},
			{
				name: "returns 2 for a --bump without a level",
				args: []string{"release", "add", "--bump", modulePath, "-m", "x"},
				stderr: `ergon: cli: --bump "example.com/demo", which is not <name>=<major|minor|patch|none>` + "\n" +
					"Run 'ergon release add --help' for usage.\n",
			},
			{
				name: "returns 2 for a --bump with a level that does not exist",
				args: []string{"release", "add", "--bump", modulePath + "=huge", "-m", "x"},
				stderr: `ergon: cli: --bump "example.com/demo=huge", which is not <name>=<major|minor|patch|none>` +
					"\nRun 'ergon release add --help' for usage.\n",
			},
		}
		for _, tt := range usage {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				status, stdout, stderr := runRelease(t, t.TempDir(), nil, tt.args...)
				assert.Equal(t, status, statusUsage, "the exit status")
				assert.Empty(t, stdout, "the standard output")
				assert.Equal(t, stderr, tt.stderr, "the standard error")
			})
		}

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
			var stdout, stderr bytes.Buffer
			p := releaseProcess(goModule(t, gitConfig, nil), nil, &stdout, &stderr, "release", "add", "--bump",
				modulePath+"=minor", "-m", "Add the Unit type.")
			p.Random = iotest.ErrReader(errRandom)
			assert.Equal(t, cli.Run(t.Context(), p, app.Register, version), statusFailure, "the exit status")
			assert.Equal(t, stderr.String(), "ergon: release: read the random digits of a changeset: "+
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
			var stderr bytes.Buffer
			p := releaseProcess(goModule(t, gitConfig, nil), nil, failing{}, &stderr, "release", "publish-plan")
			assert.Equal(t, cli.Run(t.Context(), p, app.Register, version), statusFailure, "the exit status")
			assert.Equal(t, stderr.String(), "ergon: cli: write the JSON: "+errWrite.Error()+"\n", "the standard error")
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
				"POST /repos/" + hubRepo + "/git/refs",
				"POST /repos/" + hubRepo + "/releases",
			}, "the requests to GitHub")
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
			assert.Equal(t, stdout, "wrote "+changelogPath+"\nremoved "+changesetPath+"\npull request 7\n",
				"the standard output")
			files.HasContent(t, output, "pull-request-number=7\n", "the outputs")
			assert.Equal(t, h.calls(), []string{
				"GET /repos/" + hubRepo + "/git/ref/heads/main",
				"GET /repos/" + hubRepo + "/git/ref/heads/ergon-release/main",
				"POST /repos/" + hubRepo + "/git/refs",
				"POST " + graphqlPath,
				"GET /repos/" + hubRepo + "/pulls?base=main&head=o%3Aergon-release%2Fmain&state=open",
				"POST /repos/" + hubRepo + "/pulls",
			}, "the requests to GitHub")
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
			expect.That(t, input.Branch.Name).Equal("ergon-release/main", "the branch of the commit")
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

		t.Run("ci version proposes a stale go.sum without changesets", func(t *testing.T) {
			t.Parallel()
			dir := staleModules(t)
			h := &hub{head: strings.TrimSpace(vcstest.Git(t, dir, "rev-parse", "HEAD"))}
			status, stdout, stderr := runRelease(t, dir, h.start(t), "release", "ci", "version")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+staleSum+"\npull request 7\n", "the standard output")
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
	var out, errs bytes.Buffer
	status = cli.Run(t.Context(), releaseProcess(dir, env, &out, &errs, args...), app.Register, version)
	return status, out.String(), errs.String()
}

// releaseProcess returns the process of a release case, as process states it, with each variable
// of releaseVariables set to an empty value and then the variables of env, so the environment of
// the test sets none of the variables that the release commands read.
func releaseProcess(dir string, env map[string]string, stdout, stderr io.Writer, args ...string) *cli.Process {
	p := process(dir, stdout, stderr, args...)
	for _, name := range releaseVariables {
		p.Env = append(p.Env, name+"=")
	}
	for name, value := range env {
		p.Env = append(p.Env, name+"="+value)
	}
	return p
}
