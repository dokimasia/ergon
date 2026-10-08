// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"testing/iotest"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/rewrite"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/pin"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The toolchains and the languages of the cases. The toolchain tool and the languages alpha, gamma
// and zeta have producers, and the toolchain plain and the languages delta and epsilon have none.
const (
	toolName    workspace.Toolchain = "tool"
	plainName   workspace.Toolchain = "plain"
	alphaName   workspace.Language  = "alpha"
	gammaName   workspace.Language  = "gamma"
	zetaName    workspace.Language  = "zeta"
	deltaName   workspace.Language  = "delta"
	epsilonName workspace.Language  = "epsilon"
)

// The names of the base producers of the cases.
const (
	baseSection  = "base"
	plainSection = "plain"
)

// The modules of the pins of the cases, and the versions of their releases.
const (
	lintModule   = "example.com/lint"
	formatModule = "example.com/format"
	brokenModule = "example.com/broken"
	earlier      = "v1.0.0"
	newer        = "v1.1.0"
	newest       = "v1.2.0"
	later        = "v2.0.0"
)

// The files of the repository of the cases, relative to its root and slash-separated.
const (
	toolFile   = "producers/tool.go"
	alphaFile  = "producers/alpha.go"
	goldenFile = "beta/testdata/golden/out.txt"
	lockFile   = ".ergon/init.lock"
)

// The packages of the release of the cases, which the toolchain tool discovers in the directories
// of their names.
const (
	producersName = "producers"
	betaName      = "beta"
)

// betaPackage is the import path of the package with golden files that go list states in the cases.
const betaPackage = "example.com/beta"

// The command line of the go command that a run calls: the program, the format and the pattern of
// go list, and the package of ergon.
const (
	goCommand   = "go"
	listFormat  = "{{.ImportPath}} {{.Dir}}"
	ergonModule = "go.dokimi.dev/ergon/..."
	ergonCmd    = "./cmd/ergon"
)

// The lines of output of the go command of the cases.
const (
	testedLine = "ok " + betaPackage
	syncedLine = "wrote " + lockFile
)

// config is the configuration of the release of the cases.
const config = `{"changelog": "@changesets/cli/changelog", "commit": false}` + "\n"

// The repository on GitHub of the cases, the path of its API of GraphQL, and the commit that each
// commit of the fake API makes.
const (
	hubRepo     = "o/r"
	graphqlPath = "/graphql"
	hubCommit   = "c0ffeec0ffeec0ffeec0ffeec0ffeec0ffeec0ff"
)

// updated is the summary of the changeset that moves the linters of the toolchain tool and of the
// language alpha to newer.
const updated = updateSummary + "\n\n" +
	"- `tool.tools.lint` from v1.0.0 to v1.1.0\n" +
	"- `alpha.tools.lint` from v1.0.0 to v1.1.0"

// proposed is the body of the pull request of that change, with the release later of a later major
// version.
const proposed = updateIntro + "\n\n" +
	"# Updates\n\n" +
	"- `tool.tools.lint` from v1.0.0 to v1.1.0\n" +
	"- `alpha.tools.lint` from v1.0.0 to v1.1.0\n\n" +
	"# Later major versions\n\n" +
	majorsIntro + "\n\n" +
	"- `tool.tools.lint`: v2.0.0\n" +
	"- `alpha.tools.lint`: v2.0.0"

// The errors of the fakes of the cases.
var (
	errWorkingDirectory = errors.New("getwd: failed")
	errRegister         = errors.New("register: failed")
	errDiscover         = errors.New("discover: failed")
	errProgram          = errors.New("program: failed")
	errRandom           = errors.New("random: failed")
)

// now is the time of the runs of the cases, and published is the time at which the proxy published
// the releases of the cases, eight days before now.
var (
	now       = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	published = now.Add(-8 * 24 * time.Hour)
)

// digits are the random digits of the changesets of the cases.
var digits = []byte{0xab, 0xcd}

// producersPackage is the import path of the package of the producers of the cases in the test
// binary, which go list states with the directory of their sources.
var producersPackage = reflect.TypeFor[alphaProducer]().PkgPath()

// templated is the part of the producers of the cases that renders no file.
type templated struct{}

// Templates returns no template.
func (templated) Templates() fs.FS {
	return fstest.MapFS{}
}

// tools are the tools of a section of the cases.
type tools struct {
	// Lint is the linter, a Go module.
	Lint option.Module `yaml:"lint"`
}

// settings are the options of a section of the cases.
type settings struct {
	// Tools are the tools of the section.
	Tools tools `yaml:"tools"`
}

// Validate returns nil.
func (*settings) Validate() error {
	return nil
}

// sourced are options with a source tag of no registry.
type sourced struct {
	// Hooks is a version with the source tag.
	Hooks option.Version `yaml:"hooks" source:"npm:hooks"`
}

// Validate returns nil.
func (*sourced) Validate() error {
	return nil
}

// toolProducer is the producer of the toolchain tool, which the cases register as a pointer. Its
// linter is the module lintModule at earlier.
type toolProducer struct{ templated }

// Options returns the baseline of the toolchain tool.
func (*toolProducer) Options() language.Options {
	return &settings{Tools: tools{Lint: lintModule + "@" + earlier}}
}

// alphaProducer is the producer of the language alpha. Its linter is the module lintModule at
// earlier, which the toolchain tool also pins.
type alphaProducer struct{ templated }

// Options returns the baseline of the language alpha.
func (alphaProducer) Options() language.Options {
	return &settings{Tools: tools{Lint: lintModule + "@" + earlier}}
}

// baseProducer is a base producer. Its linter is the module formatModule at earlier, whose newest
// release it is.
type baseProducer struct{ templated }

// Options returns the baseline of the base producer.
func (baseProducer) Options() language.Options {
	return &settings{Tools: tools{Lint: formatModule + "@" + earlier}}
}

// brokenProducer is the producer of the languages gamma and zeta. Its linter is the module
// brokenModule, which the proxy does not have.
type brokenProducer struct{ templated }

// Options returns the baseline of the languages gamma and zeta.
func (brokenProducer) Options() language.Options {
	return &settings{Tools: tools{Lint: brokenModule + "@" + earlier}}
}

// sourcedProducer is a base producer whose options have a source tag of no registry.
type sourcedProducer struct{ templated }

// Options returns the options with the source tag.
func (sourcedProducer) Options() language.Options {
	return &sourced{}
}

// plainProducer is a base producer without options.
type plainProducer struct{ templated }

// versioner is the versioner of the toolchain tool, which the graph of a release requires. The
// packages of the cases require none of each other, and no changeset of the cases releases one, so
// the graph calls none of its methods.
type versioner struct{}

// Resolve returns ResolutionSelected.
func (versioner) Resolve(string, version.Version) (language.Resolution, error) {
	return language.ResolutionSelected, nil
}

// Rewrite returns req.
func (versioner) Rewrite(req string, _ version.Version) (string, error) {
	return req, nil
}

// Validate returns nil.
func (versioner) Validate(*workspace.Package, version.Version) error {
	return nil
}

// Apply changes no file.
func (versioner) Apply(context.Context, string, []language.Edit) ([]string, error) {
	return nil, nil
}

// proxy is a module proxy of the cases. It responds to a request for a path of routes with the body
// of the route, and to every other request with the status 404. It records the path of each
// request, and is safe for concurrent use.
type proxy struct {
	// routes are the bodies of the responses, by the paths of the requests.
	routes map[string]string

	// requests are the paths of the requests, in their order.
	requests []string

	// mu guards routes and requests.
	mu sync.Mutex
}

// ServeHTTP records the request and responds to it.
func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	p.requests = append(p.requests, r.URL.Path)
	body, ok := p.routes[r.URL.Path]
	p.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = io.WriteString(w, body)
}

// paths returns the paths of the requests that p received, in their order.
func (p *proxy) paths() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.requests)
}

// hub is a fake API of GitHub. It responds as GitHub does for a repository without the branch of the
// pull request and without pull requests: it creates the branch, makes each commit as hubCommit and
// opens the pull request 7. It responds to a request whose method and path start with fail with the
// status 500. It records the requests, the bodies of the requests of GraphQL and the bodies of the
// pull requests that it opens, and is safe for concurrent use.
type hub struct {
	// fail is the start of the method and the path of the requests that fail, or empty for none.
	fail string

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
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/git/ref/"):
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
	case r.URL.Path == graphqlPath:
		_, _ = io.WriteString(w, `{"data":{"createCommitOnBranch":{"commit":{"oid":"`+hubCommit+`"}}}}`)
	case r.Method == http.MethodGet:
		_, _ = io.WriteString(w, "[]")
	default:
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"number":7}`)
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

// fixture is the environment of a case: a working tree of git with the files of the cases, a module
// proxy, GitHub, and the go command, whose calls it records. A case changes its fields before the
// run, and a run reads each field when it needs it.
type fixture struct {
	// t is the test of the case.
	t *testing.T

	// register adds the toolchains and the languages of the case to a catalog.
	register func(*language.Catalog) error

	// discover is the discovery of the toolchain tool.
	discover func(context.Context, string) ([]workspace.Package, error)

	// tested and synced change the working tree as go test -update and ergon init sync do.
	tested, synced func()

	// proxy is the module proxy.
	proxy *proxy

	// hub is GitHub.
	hub *hub

	// env are the variables of the environment.
	env map[string]string

	// root is the working tree.
	root string

	// head is the commit of HEAD of the working tree.
	head string

	// proxyURL is the address of the proxy.
	proxyURL string

	// failing is the first argument of the call of the go command that fails, or empty for none.
	failing string

	// listing are the lines of go list: an import path and a directory each.
	listing []string

	// calls are the calls of the go command, each as its directory, its program and its arguments.
	calls [][]string
}

// standard adds the toolchains and the languages of the cases to c: the toolchain tool with the
// discovery of fx, versioner and toolProducer, its language alpha with alphaProducer and its
// language delta without a producer, and the toolchain plain and its language epsilon, both without
// a producer.
func (fx *fixture) standard(c *language.Catalog) error {
	return errors.Join(
		language.RegisterToolchain(
			c,
			language.Toolchain{Name: toolName, Discover: fx.discover},
			versioner{},
			&toolProducer{},
		),
		language.Register(c, language.Declaration{Name: alphaName, Toolchain: toolName}, alphaProducer{}),
		language.Register(c, language.Declaration{Name: deltaName, Toolchain: toolName}),
		language.RegisterToolchain(c, language.Toolchain{Name: plainName}),
		language.Register(c, language.Declaration{Name: epsilonName, Toolchain: plainName}),
	)
}

// broken adds the languages of standard to c, and the languages gamma and zeta of the toolchain
// tool with brokenProducer.
func (fx *fixture) broken(c *language.Catalog) error {
	return errors.Join(
		fx.standard(c),
		language.Register(c, language.Declaration{Name: gammaName, Toolchain: toolName}, brokenProducer{}),
		language.Register(c, language.Declaration{Name: zetaName, Toolchain: toolName}, brokenProducer{}),
	)
}

// execute is the go command of fx. It records the call, returns errProgram for the call whose first
// argument is fx.failing, and otherwise writes fx.listing for go list, and changes the working tree
// with fx.tested for go test and with fx.synced for go run, each after a line of output.
func (fx *fixture) execute(_ context.Context, dir string, stdout, _ io.Writer, program string, args ...string) error {
	fx.calls = append(fx.calls, slices.Concat([]string{dir, program}, args))
	if args[0] == fx.failing {
		return errProgram
	}
	switch args[0] {
	case "list":
		for _, line := range fx.listing {
			_, _ = fmt.Fprintln(stdout, line)
		}
	case "test":
		_, _ = fmt.Fprintln(stdout, testedLine)
		fx.tested()
	default:
		_, _ = fmt.Fprintln(stdout, syncedLine)
		fx.synced()
	}
	return nil
}

// process returns the process of a run in fx with args, which writes to stdout and stderr. It reads
// the random digits of digits.
func (fx *fixture) process(stdout, stderr io.Writer, args ...string) *process {
	return &process{
		getwd:      func() (string, error) { return fx.root, nil },
		getenv:     func(name string) string { return fx.env[name] },
		now:        func() time.Time { return now },
		execute:    fx.execute,
		register:   func(c *language.Catalog) error { return fx.register(c) },
		random:     bytes.NewReader(digits),
		stdout:     stdout,
		stderr:     stderr,
		transport:  http.DefaultTransport,
		registries: pin.Registries{GoProxy: fx.proxyURL},
		base: []baseline.Producer{
			{Name: plainSection, Producer: plainProducer{}},
			{Name: baseSection, Producer: baseProducer{}},
		},
		args: args,
	}
}

// updateWith runs update in fx with o, and returns its standard output, its standard error and its
// error.
func (fx *fixture) updateWith(o *options) (stdout, stderr string, err error) {
	var out, errs bytes.Buffer
	err = update(fx.t.Context(), fx.process(&out, &errs), o)
	return out.String(), errs.String(), err
}

// wantCalls returns the first n calls of the go command of a run in fx that moves a pin: go list of
// the packages of ergon's modules, go test of beta with -update, and ergon init sync.
func (fx *fixture) wantCalls(n int) [][]string {
	calls := [][]string{
		{fx.root, goCommand, "list", "-f", listFormat, ergonModule},
		{fx.root, goCommand, "test", "-count=1", filepath.Join(fx.root, betaName), "-update"},
		{fx.root, goCommand, "run", ergonCmd, "init", "sync"},
	}
	return calls[:n]
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		id, err := release.ChangesetID(updated, bytes.NewReader(digits))
		assert.NoError(t, err, "ChangesetID of the summary of the cases")
		changesetFile := ".changeset/" + id + ".md"

		t.Run("moves each pin to its newest release of the same major version", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			stdout, stderr, err := fx.updateWith(&options{minAge: defaultMinAge})
			assert.NoError(t, err, "update")
			expect.Equal(t, stdout, "update tool.tools.lint from v1.0.0 to v1.1.0\n"+
				"update alpha.tools.lint from v1.0.0 to v1.1.0\n"+
				"wrote "+toolFile+"\nwrote "+alphaFile+"\n"+
				testedLine+"\n"+syncedLine+"\n"+
				"wrote "+changesetFile+"\n", "the standard output")
			expect.Empty(t, stderr, "the standard error")
			expect.Equal(t, files.Read(t, filepath.Join(fx.root, filepath.FromSlash(toolFile))),
				source("*toolProducer", newer), "the source of the toolchain tool")
			expect.Equal(t, files.Read(t, filepath.Join(fx.root, filepath.FromSlash(alphaFile))),
				source("alphaProducer", newer), "the source of the language alpha")
			expect.Equal(t, fx.calls, fx.wantCalls(3), "the calls of the go command")
			expect.Empty(t, fx.hub.calls(), "the requests to GitHub")
		})

		t.Run("resolves a pin that two sections share once", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			_, _, err := fx.updateWith(&options{minAge: defaultMinAge})
			assert.NoError(t, err, "update")
			assert.Equal(t, fx.proxy.paths(), []string{
				"/" + formatModule + "/@latest",
				"/" + formatModule + "/@v/list",
				"/" + lintModule + "/@latest",
				"/" + lintModule + "/@v/list",
				"/" + lintModule + "/@v/" + newer + ".info",
				"/" + lintModule + "/@v/" + later + ".info",
			}, "the requests to the proxy")
		})

		t.Run("writes a changeset of a patch of each package whose producer moved", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			_, _, err := fx.updateWith(&options{minAge: defaultMinAge})
			assert.NoError(t, err, "update")
			want := changeset.Changeset{ID: id, Summary: updated, Releases: []changeset.Release{
				{Name: producersName, Bump: version.BumpPatch},
				{Name: betaName, Bump: version.BumpNone},
			}}
			assert.Equal(t, files.Read(t, filepath.Join(fx.root, filepath.FromSlash(changesetFile))),
				string(changeset.Format(&want)), "the changeset")
		})

		t.Run("proposes the changes on the branch ergon-update/main with propose", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			stdout, stderr, err := fx.updateWith(&options{minAge: defaultMinAge, propose: true})
			assert.NoError(t, err, "update")
			expect.HasSuffix(t, stdout, "wrote "+changesetFile+"\npull request 7\n", "the standard output")
			expect.Empty(t, stderr, "the standard error")
			assert.Equal(t, fx.hub.calls(), []string{
				"GET /repos/" + hubRepo + "/git/ref/heads/ergon-update/main",
				"POST /repos/" + hubRepo + "/git/refs",
				"POST " + graphqlPath,
				"GET /repos/" + hubRepo + "/pulls?base=main&head=o%3Aergon-update%2Fmain&state=open",
				"POST /repos/" + hubRepo + "/pulls",
			}, "the requests to GitHub")
			commits := fx.hub.commits()
			assert.Length(t, commits, 1, "the commits")
			expect.That(t, commits[0]).
				Contains(`"headline":"build: update the pins of the baseline"`, "the message of the commit").
				Contains(`"expectedHeadOid":"`+fx.head+`"`, "the commit on which the run started").
				Contains(`"path":"`+toolFile+`"`, "the source of the toolchain tool in the commit").
				Contains(`"path":"`+alphaFile+`"`, "the source of the language alpha in the commit").
				Contains(`"path":"`+goldenFile+`"`, "the golden file in the commit").
				Contains(`"path":"`+lockFile+`"`, "the lock in the commit").
				Contains(`"path":"`+changesetFile+`"`, "the changeset in the commit")
			expect.Equal(t, fx.hub.opened(), []string{proposed}, "the bodies of the pull requests")
		})

		t.Run("takes the release of a later major version with major", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			stdout, _, err := fx.updateWith(&options{minAge: defaultMinAge, major: true, propose: true})
			assert.NoError(t, err, "update")
			expect.HasPrefix(t, stdout, "update tool.tools.lint from v1.0.0 to v2.0.0\n"+
				"update alpha.tools.lint from v1.0.0 to v2.0.0\n", "the standard output")
			expect.Equal(t, files.Read(t, filepath.Join(fx.root, filepath.FromSlash(alphaFile))),
				source("alphaProducer", later), "the source of the language alpha")
			expect.Equal(t, fx.hub.opened(), []string{updateIntro + "\n\n# Updates\n\n" +
				"- `tool.tools.lint` from v1.0.0 to v2.0.0\n- `alpha.tools.lint` from v1.0.0 to v2.0.0"},
				"the bodies of the pull requests, without later major versions")
		})

		t.Run("takes no release that the registry published less than the minimum age ago", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			stdout, _, err := fx.updateWith(&options{minAge: 9 * 24 * time.Hour})
			assert.NoError(t, err, "update")
			expect.Equal(t, stdout, "no pin has a newer release\n", "the standard output")
			expect.Empty(t, fx.calls, "the calls of the go command")
		})

		t.Run("writes that no pin has a newer release, and writes no file", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.proxy.routes["/"+lintModule+"/@v/list"] = earlier + "\n"
			var stdout string
			var err error
			files.Unchanged(t, os.DirFS(filepath.Join(fx.root, producersName)), func() {
				stdout, _, err = fx.updateWith(&options{minAge: defaultMinAge, propose: true})
			}, "the sources of the producers")
			assert.NoError(t, err, "update")
			expect.Equal(t, stdout, "no pin has a newer release\n", "the standard output")
			expect.Empty(t, fx.calls, "the calls of the go command")
			expect.Empty(t, fx.hub.calls(), "the requests to GitHub")
		})

		t.Run("runs no test without a package with golden files", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.listing = fx.listing[:1]
			_, _, err := fx.updateWith(&options{minAge: defaultMinAge})
			assert.NoError(t, err, "update")
			expect.Equal(t, fx.calls, [][]string{fx.wantCalls(1)[0], fx.wantCalls(3)[2]},
				"the calls of the go command")
		})

		t.Run("returns the pins that did not resolve after it proposed the other pins", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.register = fx.broken
			stdout, stderr, err := fx.updateWith(&options{minAge: defaultMinAge, propose: true})
			assert.Equal(t, err.Error(), "the pins gamma.tools.lint, zeta.tools.lint did not resolve", "the error")
			expect.HasSuffix(t, stdout, "pull request 7\n", "the standard output")
			expect.Equal(t, stderr, "update-baseline: pin: resolve gamma.tools.lint: pin: the registry failed: "+
				"no module of the proxy has the package "+brokenModule+"\n", "the standard error")
			expect.Equal(t, fx.hub.opened(), []string{proposed + "\n\n# Failures\n\n" + failuresIntro + "\n\n" +
				"- `gamma.tools.lint`\n- `zeta.tools.lint`"}, "the bodies of the pull requests")
		})

		t.Run("returns an error for a working tree with changes", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			files.Write(t, fx.root, files.Tree{"notes.txt": files.Text("notes\n")})
			_, _, err := fx.updateWith(&options{minAge: defaultMinAge})
			assert.Equal(t, err.Error(), "the working tree has changes, so commit them first: notes.txt", "the error")
		})

		t.Run("returns an error without GITHUB_REPOSITORY", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			delete(fx.env, repositoryEnv)
			_, _, err := fx.updateWith(&options{minAge: defaultMinAge, propose: true})
			assert.Equal(t, err.Error(), "set GITHUB_REPOSITORY to the repository on GitHub, as owner/name",
				"the error")
		})

		t.Run("returns the error of a changed file that does not read", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.synced = func() {
				files.Write(t, fx.root, files.Tree{"nested/notes.txt": files.Text("notes\n")})
				vcstest.Git(t, filepath.Join(fx.root, "nested"), "init", "--quiet")
			}
			_, _, err := fx.updateWith(&options{minAge: defaultMinAge, propose: true})
			assert.HasPrefix(t, err.Error(), "release: read nested/: ", "the error")
			expect.Empty(t, fx.hub.calls(), "the requests to GitHub")
		})

		t.Run("writes no file before it read the configuration of the release", func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			files.Write(t, fx.root, files.Tree{release.ConfigPath: files.Text("[]\n")})
			vcstest.Commit(t, fx.root, "break the configuration")
			var err error
			files.Unchanged(t, os.DirFS(filepath.Join(fx.root, producersName)), func() {
				_, _, err = fx.updateWith(&options{minAge: defaultMinAge})
			}, "the sources of the producers")
			assert.ErrorIs(t, err, release.ErrConfig, "the error")
		})

		failures := []struct {
			name   string
			change func(t *testing.T, fx *fixture, p *process, o *options)
			want   error
			calls  int
		}{
			{
				name: "returns the error of the working directory",
				change: func(_ *testing.T, _ *fixture, p *process, _ *options) {
					p.getwd = func() (string, error) { return "", errWorkingDirectory }
				},
				want: errWorkingDirectory,
			},
			{
				name: "returns ErrGit outside a working tree of git",
				change: func(t *testing.T, _ *fixture, p *process, _ *options) {
					t.Helper()
					dir := t.TempDir()
					p.getwd = func() (string, error) { return dir, nil }
				},
				want: vcs.ErrGit,
			},
			{
				name: "returns ErrGit for an index that git does not read",
				change: func(t *testing.T, fx *fixture, _ *process, _ *options) {
					t.Helper()
					files.Write(t, fx.root, files.Tree{".git/index": files.Text("corrupt\n")})
				},
				want: vcs.ErrGit,
			},
			{
				name: "returns the error of the registration",
				change: func(_ *testing.T, fx *fixture, _ *process, _ *options) {
					fx.register = func(*language.Catalog) error { return errRegister }
				},
				want: errRegister,
			},
			{
				name: "returns ErrSource for a source tag of no registry",
				change: func(_ *testing.T, _ *fixture, p *process, _ *options) {
					p.base = append(p.base, baseline.Producer{Name: baseSection, Producer: sourcedProducer{}})
				},
				want: pin.ErrSource,
			},
			{
				name: "returns ErrToken without GITHUB_TOKEN",
				change: func(_ *testing.T, fx *fixture, _ *process, _ *options) {
					delete(fx.env, authEnv)
				},
				want: forge.ErrToken,
			},
			{
				name: "returns the error of go list",
				change: func(_ *testing.T, fx *fixture, _ *process, _ *options) {
					fx.failing = "list"
				},
				want:  errProgram,
				calls: 1,
			},
			{
				name: "returns the error of the discovery of the packages",
				change: func(_ *testing.T, fx *fixture, _ *process, _ *options) {
					fx.discover = func(context.Context, string) ([]workspace.Package, error) { return nil, errDiscover }
				},
				want:  errDiscover,
				calls: 1,
			},
			{
				name: "returns ErrNotExist for a repository without the configuration of the release",
				change: func(t *testing.T, fx *fixture, _ *process, _ *options) {
					t.Helper()
					assert.NoError(t, os.Remove(filepath.Join(fx.root, filepath.FromSlash(release.ConfigPath))),
						"Remove of the configuration")
					vcstest.Commit(t, fx.root, "remove the configuration")
				},
				want:  fs.ErrNotExist,
				calls: 1,
			},
			{
				name: "returns ErrMismatch for a source whose literal differs from the pin",
				change: func(t *testing.T, fx *fixture, _ *process, _ *options) {
					t.Helper()
					files.Write(t, fx.root, files.Tree{toolFile: files.Text(source("*toolProducer", "v0.9.0"))})
					vcstest.Commit(t, fx.root, "change the source")
				},
				want:  rewrite.ErrMismatch,
				calls: 1,
			},
			{
				name: "returns ErrInvalid for a changeset that does not parse",
				change: func(t *testing.T, fx *fixture, _ *process, _ *options) {
					t.Helper()
					files.Write(t, fx.root, files.Tree{".changeset/broken.md": files.Text("---\n")})
					vcstest.Commit(t, fx.root, "add a changeset")
				},
				want:  changeset.ErrInvalid,
				calls: 1,
			},
			{
				name: "returns the error of go test",
				change: func(_ *testing.T, fx *fixture, _ *process, _ *options) {
					fx.failing = "test"
				},
				want:  errProgram,
				calls: 2,
			},
			{
				name: "returns the error of go run",
				change: func(_ *testing.T, fx *fixture, _ *process, _ *options) {
					fx.failing = "run"
				},
				want:  errProgram,
				calls: 3,
			},
			{
				name: "returns ErrChangeset for a changeset of a package that the regeneration adds",
				change: func(t *testing.T, fx *fixture, _ *process, _ *options) {
					t.Helper()
					fx.tested = func() {
						files.Write(t, fx.root, files.Tree{".changeset/missing.md": files.Text(
							"---\n\"missing\": patch\n---\n\nRelease a package that the repository does not have.\n",
						)})
					}
				},
				want:  release.ErrChangeset,
				calls: 3,
			},
			{
				name: "returns the error of the random digits",
				change: func(_ *testing.T, _ *fixture, p *process, _ *options) {
					p.random = iotest.ErrReader(errRandom)
				},
				want:  errRandom,
				calls: 3,
			},
			{
				name: "returns ErrExist for a changeset whose file the repository has",
				change: func(t *testing.T, fx *fixture, _ *process, _ *options) {
					t.Helper()
					files.Write(t, fx.root, files.Tree{changesetFile: files.Text(
						"---\n\"" + producersName + "\": patch\n---\n\nRelease the producers.\n",
					)})
					vcstest.Commit(t, fx.root, "add a changeset")
				},
				want:  fs.ErrExist,
				calls: 3,
			},
			{
				name: "returns ErrGitHub for a pull request that GitHub refuses",
				change: func(_ *testing.T, fx *fixture, _ *process, o *options) {
					fx.hub.fail = "POST /repos/" + hubRepo + "/git/refs"
					o.propose = true
				},
				want:  forge.ErrGitHub,
				calls: 3,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				fx := newFixture(t)
				var stdout, stderr bytes.Buffer
				p := fx.process(&stdout, &stderr)
				o := options{minAge: defaultMinAge}
				tt.change(t, fx, p, &o)
				err := update(t.Context(), p, &o)
				assert.ErrorIs(t, err, tt.want, "the error")
				expect.Equal(t, fx.calls, fx.wantCalls(tt.calls), "the calls of the go command", assert.EquateEmpty())
			})
		}
	})
}

// newFixture returns the fixture of a case: a working tree with one commit of the configuration of
// the release, the sources of toolProducer and alphaProducer at earlier and the golden file of
// beta, the proxy with the releases of the cases, GitHub, the toolchains and the languages of
// standard, the packages producers and beta, and a go command that lists them, rewrites the golden
// file of beta and writes the lock.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := vcstest.Repository(t, files.Tree{
		release.ConfigPath: files.Text(config),
		toolFile:           files.Text(source("*toolProducer", earlier)),
		alphaFile:          files.Text(source("alphaProducer", earlier)),
		goldenFile:         files.Text("earlier\n"),
	})
	head := vcstest.Commit(t, root, "first")
	first, err := version.Parse("1.0.0")
	assert.NoError(t, err, "Parse of the version of the packages")
	packages := []workspace.Package{
		{Name: producersName, Toolchain: toolName, Dir: producersName, Version: first},
		{Name: betaName, Toolchain: toolName, Dir: betaName, Version: first},
	}
	fx := &fixture{
		t:     t,
		proxy: &proxy{routes: releases()},
		hub:   &hub{},
		root:  root,
		head:  head,
		listing: []string{
			producersPackage + " " + filepath.Join(root, producersName),
			betaPackage + " " + filepath.Join(root, betaName),
		},
	}
	fx.register = fx.standard
	fx.discover = func(context.Context, string) ([]workspace.Package, error) { return packages, nil }
	fx.tested = func() { files.Write(t, root, files.Tree{goldenFile: files.Text("regenerated\n")}) }
	fx.synced = func() { files.Write(t, root, files.Tree{lockFile: files.Text("{}\n")}) }
	proxyServer := httptest.NewServer(fx.proxy)
	t.Cleanup(proxyServer.Close)
	hubServer := httptest.NewServer(fx.hub)
	t.Cleanup(hubServer.Close)
	fx.proxyURL = proxyServer.URL
	fx.env = map[string]string{
		authEnv: "token", repositoryEnv: hubRepo, apiEnv: hubServer.URL, graphqlEnv: hubServer.URL + graphqlPath,
		serverEnv: hubServer.URL,
	}
	return fx
}

// releases returns the routes of the proxy of the cases: the module lintModule at earlier with the
// release newer and the release later of a later major version, and the module formatModule at
// earlier without a later release, each release published at published.
func releases() map[string]string {
	info := func(v string) string {
		return `{"Version":"` + v + `","Time":"` + published.Format(time.RFC3339) + `"}`
	}
	return map[string]string{
		"/" + lintModule + "/@latest":               info(newer),
		"/" + lintModule + "/@v/list":               earlier + "\n" + newer + "\n" + later + "\n",
		"/" + lintModule + "/@v/" + newer + ".info": info(newer),
		"/" + lintModule + "/@v/" + later + ".info": info(later),
		"/" + formatModule + "/@latest":             info(earlier),
		"/" + formatModule + "/@v/list":             earlier + "\n",
	}
}

// source returns a Go file of the package of the producers of the cases with the Options method of
// receiver, whose linter is the module lintModule at the version v, as gofmt formats it.
func source(receiver, v string) string {
	return "package producers\n\n" +
		"// Options returns the baseline of the producer.\n" +
		"func (" + receiver + ") Options() language.Options {\n" +
		"\treturn &settings{Tools: tools{Lint: \"" + lintModule + "@" + v + "\"}}\n" +
		"}\n"
}
