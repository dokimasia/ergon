// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/rewrite"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/pin"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
)

// The variables of the environment that the command reads.
const (
	// authEnv is the token that authorizes the requests to GitHub.
	authEnv = "GITHUB_TOKEN"

	// repositoryEnv is the repository of the pull request, as owner/name.
	repositoryEnv = "GITHUB_REPOSITORY"

	// apiEnv, graphqlEnv and serverEnv are the addresses of the APIs and of the web pages of GitHub,
	// which GitHub Actions sets.
	apiEnv     = "GITHUB_API_URL"
	graphqlEnv = "GITHUB_GRAPHQL_URL"
	serverEnv  = "GITHUB_SERVER_URL"
)

// The limits of the requests.
const (
	// downloadTimeout is the limit of a request to a registry, such as the download of an asset whose
	// digest GitHub does not state.
	downloadTimeout = 10 * time.Minute

	// apiTimeout is the limit of a request to the API of GitHub.
	apiTimeout = time.Minute
)

// The modules of ergon.
const (
	// exempt is the prefix of the paths of ergon's modules, whose releases an update takes at once.
	exempt = "go.dokimi.dev/ergon/"

	// modules is the pattern of the go command that matches the packages of ergon's modules.
	modules = exempt + "..."
)

// headRef is the ref of the commit of the working tree, against which the command finds changes.
const headRef = "HEAD"

// The programs and the files of the regeneration.
const (
	// goProgram is the go command.
	goProgram = "go"

	// testdata and golden are the directories of the golden files of a package, testdata/golden below
	// the directory of the package.
	testdata = "testdata"
	golden   = "golden"

	// ergonPackage is the package of the command ergon, relative to the root of the repository.
	ergonPackage = "./cmd/ergon"
)

// The pull request of -propose.
const (
	// updateBranch is the prefix of its branch, before the name of the base branch.
	updateBranch = "ergon-update/"

	// updateTitle is its title and the message of its commits.
	updateTitle = "build: update the pins of the baseline"

	// updateSummary opens the summary of the changeset, before the list of the updates.
	updateSummary = "Move the pins of the baseline to their newest releases."

	// updateIntro opens the body of the pull request.
	updateIntro = "This pull request was opened by `update-baseline`. It rewrites the baseline of each " +
		"producer whose pins have a newer release, the golden files, and the managed files of this repository."

	// majorsIntro opens the list of the releases of a later major version.
	majorsIntro = "These pins have a release of a later major version, which `update-baseline -major` takes."

	// failuresIntro opens the list of the pins that did not resolve.
	failuresIntro = "These pins did not resolve, and keep their versions. The log of the run states each error."
)

// target is a producer with the options of a section, the pins of its options at the baseline, and
// the updates of the pins.
type target struct {
	// producer is the producer, whose package and type locate the source of its options.
	producer language.Configurable

	// pins are the pins of the options at the baseline.
	pins []pin.Pin

	// updates are the pins that have a newer release, with the release.
	updates []rewrite.Update
}

// major is a pin with a release of a later major version, which the update leaves to a person.
type major struct {
	// key is the key of the pin.
	key string

	// version is the version of the release.
	version string
}

// updater is one run of the command in the repository at root.
type updater struct {
	// p is the process of the run.
	p *process

	// o are the options of the run.
	o *options

	// client reads the releases of GitHub, and proposes the changes.
	client *forge.Client

	// catalog has the languages of p.
	catalog *language.Catalog

	// root is the root of ergon's repository.
	root string

	// head is the commit of HEAD, on which the run started.
	head string

	// targets are the producers with the options of a section.
	targets []target

	// majors are the pins with a release of a later major version that the run leaves out.
	majors []major

	// failures are the keys of the pins that did not resolve.
	failures []string
}

// update moves the pins of the baselines of the producers of p to their newest releases under ctx,
// in the repository at the working directory, as the documentation of the command states. It
// returns an error for a working tree with changes, for a step that fails, and for pins that did not
// resolve, after every other step.
func update(ctx context.Context, p *process, o *options) error {
	root, err := p.getwd()
	if err != nil {
		return fmt.Errorf("find the working directory: %w", err)
	}
	u := updater{p: p, o: o, root: root, catalog: new(language.Catalog)}
	//dokimi:mutate-skip ror-false: vcs.Changed fails with vcs.ErrGit on every working tree on which vcs.Head fails
	if u.head, err = vcs.Head(ctx, root); err != nil {
		return err
	}
	changed, err := vcs.Changed(ctx, root, headRef)
	if err != nil {
		return err
	}
	if len(changed) > 0 {
		return fmt.Errorf("the working tree has changes, so commit them first: %s", strings.Join(changed, ", "))
	}
	err = p.register(u.catalog)
	if err != nil {
		return err
	}
	err = u.find()
	if err != nil {
		return err
	}
	u.client, err = forge.New(&http.Client{Transport: p.transport, Timeout: apiTimeout}, forge.Config{
		Token: p.getenv(authEnv), API: p.getenv(apiEnv), GraphQL: p.getenv(graphqlEnv), Server: p.getenv(serverEnv),
	})
	if err != nil {
		return err
	}
	u.resolve(ctx)
	if slices.ContainsFunc(u.targets, func(t target) bool { return len(t.updates) > 0 }) {
		err = u.change(ctx)
	} else {
		fmt.Fprintln(p.stdout, "no pin has a newer release")
	}
	if err != nil {
		return err
	}
	if len(u.failures) > 0 {
		return fmt.Errorf("the pins %s did not resolve", strings.Join(u.failures, ", "))
	}
	return nil
}

// find sets the targets of u: each producer of p.base with the options of a section, and then the
// producer of the toolchain of each language of the catalog, once for each toolchain, and the
// producer of the language, each with the options of a section. It returns the error of
// [pin.Find].
func (u *updater) find() error {
	type section struct {
		producer language.Configurable
		name     string
	}
	var sections []section
	for _, b := range u.p.base {
		if c, ok := b.Producer.(language.Configurable); ok {
			sections = append(sections, section{name: b.Name, producer: c})
		}
	}
	var toolchains []workspace.Toolchain
	for d := range u.catalog.Languages() {
		c, ok := language.ToolchainRole[language.Configurable](u.catalog, d.Toolchain)
		if ok && !slices.Contains(toolchains, d.Toolchain) {
			toolchains = append(toolchains, d.Toolchain)
			sections = append(sections, section{name: string(d.Toolchain), producer: c})
		}
		if c, ok := language.Role[language.Configurable](u.catalog, d.Name); ok {
			sections = append(sections, section{name: string(d.Name), producer: c})
		}
	}
	for _, s := range sections {
		pins, err := pin.Find(s.name, s.producer.Options())
		if err != nil {
			return err
		}
		u.targets = append(u.targets, target{producer: s.producer, pins: pins})
	}
	return nil
}

// resolve resolves each pin of the targets of u, and writes a line of each pin with a newer release
// to the standard output, and the error of each resolution that fails to the standard error. It
// records the updates in the targets, and the releases of a later major version and the keys of the
// pins that did not resolve in u. A pin with the kind, the name and the version of a pin that
// resolved before takes its resolution, so the pins of a tool that two sections share move to the
// same release.
func (u *updater) resolve(ctx context.Context) {
	resolver := pin.Resolver{
		Client:     &http.Client{Transport: u.p.transport, Timeout: downloadTimeout},
		GitHub:     u.client,
		Now:        u.p.now,
		Registries: u.p.registries,
		Exempt:     []string{exempt},
		MinAge:     u.o.minAge,
		Major:      u.o.major,
	}
	type identity struct {
		name, version string
		kind          pin.Kind
	}
	type result struct {
		err error
		res pin.Resolution
	}
	resolved := map[identity]result{}
	for i := range u.targets {
		t := &u.targets[i]
		for k := range t.pins {
			p := &t.pins[k]
			id := identity{name: p.Name, version: p.Version, kind: p.Kind}
			r, ok := resolved[id]
			if !ok {
				r.res, r.err = resolver.Resolve(ctx, p)
				resolved[id] = r
				if r.err != nil {
					fmt.Fprintf(u.p.stderr, "%s: %v\n", name, r.err)
				}
			}
			if r.err != nil {
				u.failures = append(u.failures, p.Key)
				continue
			}
			if r.res.Major != nil {
				u.majors = append(u.majors, major{key: p.Key, version: r.res.Major.Version})
			}
			if r.res.Next != nil {
				t.updates = append(t.updates, rewrite.Update{Pin: *p, Release: *r.res.Next})
				fmt.Fprintf(u.p.stdout, "update %s from %s to %s\n", p.Key, p.Version, r.res.Next.Version)
			}
		}
	}
}

// change writes the updates into the sources of their producers, regenerates the golden files and
// the managed files of the repository, writes the changeset, and proposes the change with -propose.
// It reads the packages, the graph of the release and its configuration before it writes a file. It
// returns the error of each step.
func (u *updater) change(ctx context.Context) error {
	dirs, err := u.packages(ctx)
	if err != nil {
		return err
	}
	g, err := release.Discover(ctx, u.catalog, u.root)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(u.root, filepath.FromSlash(release.ConfigPath)))
	if err != nil {
		return fmt.Errorf("read %s: %w", release.ConfigPath, err)
	}
	config, err := release.ParseConfig(data, g)
	if err != nil {
		return err
	}
	err = u.rewrite(dirs)
	if err != nil {
		return err
	}
	patched, err := u.changed(ctx, g, &config)
	if err != nil {
		return err
	}
	err = u.regenerate(ctx, dirs)
	if err != nil {
		return err
	}
	all, err := u.changed(ctx, g, &config)
	if err != nil {
		return err
	}
	c := changeset.Changeset{Summary: strings.Join(slices.Concat([]string{updateSummary, ""}, u.lines()), "\n")}
	for _, pkg := range all {
		bump := version.BumpNone
		if slices.Contains(patched, pkg) {
			bump = version.BumpPatch
		}
		c.Releases = append(c.Releases, changeset.Release{Name: pkg, Bump: bump})
	}
	if c.ID, err = release.ChangesetID(c.Summary, u.p.random); err != nil {
		return err
	}
	file, err := release.AddChangeset(u.root, &c)
	if err != nil {
		return err
	}
	fmt.Fprintf(u.p.stdout, "wrote %s\n", file)
	if u.o.propose {
		return u.propose(ctx, config.BaseBranch)
	}
	return nil
}

// packages returns the directory of each package of ergon's modules, by its import path, as go
// list states them. It returns the error of go list.
func (u *updater) packages(ctx context.Context) (map[string]string, error) {
	var listed bytes.Buffer
	err := u.p.execute(ctx, u.root, &listed, u.p.stderr, goProgram, "list", "-f", "{{.ImportPath}} {{.Dir}}", modules)
	if err != nil {
		return nil, err
	}
	dirs := map[string]string{}
	for line := range strings.Lines(listed.String()) {
		pkg, dir, _ := strings.Cut(strings.TrimSuffix(line, "\n"), " ")
		dirs[pkg] = dir
	}
	return dirs, nil
}

// rewrite writes the updates of each target with updates into the source of its producer, in the
// directory that dirs states for the package of the producer's type, and writes a line of each file
// that it writes. It returns the error of [rewrite.Apply].
func (u *updater) rewrite(dirs map[string]string) error {
	for _, t := range u.targets {
		if len(t.updates) == 0 {
			continue
		}
		producer := typeOf(t.producer)
		written, err := rewrite.Apply(dirs[producer.PkgPath()], producer.Name(), t.updates)
		if err != nil {
			return err
		}
		for _, file := range written {
			rel, _ := filepath.Rel(u.root, file)
			fmt.Fprintf(u.p.stdout, "wrote %s\n", filepath.ToSlash(rel))
		}
	}
	return nil
}

// regenerate runs the tests of the packages of dirs with golden files with -update, in the order
// of their import paths, and then ergon init sync of the ergon that the sources build. It returns
// the error of the go command.
func (u *updater) regenerate(ctx context.Context, dirs map[string]string) error {
	var goldens []string
	for _, pkg := range slices.Sorted(maps.Keys(dirs)) {
		if _, err := os.Stat(filepath.Join(dirs[pkg], testdata, golden)); err == nil {
			goldens = append(goldens, dirs[pkg])
		}
	}
	if len(goldens) > 0 {
		args := slices.Concat([]string{"test", "-count=1"}, goldens, []string{"-update"})
		if err := u.p.execute(ctx, u.root, u.p.stdout, u.p.stderr, goProgram, args...); err != nil {
			return err
		}
	}
	return u.p.execute(ctx, u.root, u.p.stdout, u.p.stderr, goProgram, "run", ergonPackage, "init", "sync")
}

// changed returns the names of the packages of g whose files differ from HEAD, as [release.NewStatus]
// counts them under config. It returns the errors of reading the changesets and of NewStatus.
func (u *updater) changed(ctx context.Context, g *release.Graph, config *release.Config) ([]string, error) {
	sets, err := release.ReadChangesets(u.root)
	if err != nil {
		return nil, err
	}
	status, err := release.NewStatus(ctx, u.root, g, config, sets, headRef)
	if err != nil {
		return nil, err
	}
	return status.Changed, nil
}

// propose commits the changes of the working tree on the branch ergon-update/<base> through the API
// of GitHub, on the commit where the run started, and opens or updates the pull request into base.
// It writes the number of the pull request, or that it skipped the commit when base moved past it,
// which [release.ErrMoved] reports. It returns an error for an environment without the repository,
// and the errors of git, of reading a file and of GitHub.
func (u *updater) propose(ctx context.Context, base string) error {
	repo := u.p.getenv(repositoryEnv)
	if repo == "" {
		return fmt.Errorf("set %s to the repository on GitHub, as owner/name", repositoryEnv)
	}
	files, deleted, err := release.ChangedFiles(ctx, u.root)
	if err != nil {
		return err
	}
	p := release.Proposal{
		Files: files, Repo: repo, Base: base, Branch: updateBranch + base, Head: u.head, Title: updateTitle,
		Body: u.body(), Deleted: deleted,
	}
	number, err := release.Propose(ctx, u.client, &p)
	if errors.Is(err, release.ErrMoved) {
		fmt.Fprintf(u.p.stdout, "skipped %s, which is no longer the head of %s\n", u.head, base)
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(u.p.stdout, "pull request %d\n", number)
	return nil
}

// lines returns a line of each update of u in Markdown: the key of the pin, its version and the
// version of its release.
func (u *updater) lines() []string {
	var lines []string
	for _, t := range u.targets {
		for i := range t.updates {
			up := &t.updates[i]
			lines = append(lines, "- `"+up.Pin.Key+"` from "+up.Pin.Version+" to "+up.Release.Version)
		}
	}
	return lines
}

// body returns the body of the pull request in Markdown: updateIntro, the updates, the releases of
// a later major version and the pins that did not resolve, each list under its heading.
func (u *updater) body() string {
	lines := slices.Concat([]string{updateIntro, "", "# Updates", ""}, u.lines())
	if len(u.majors) > 0 {
		lines = append(lines, "", "# Later major versions", "", majorsIntro, "")
		for _, m := range u.majors {
			lines = append(lines, "- `"+m.key+"`: "+m.version)
		}
	}
	if len(u.failures) > 0 {
		lines = append(lines, "", "# Failures", "", failuresIntro, "")
		for _, key := range u.failures {
			lines = append(lines, "- `"+key+"`")
		}
	}
	return strings.Join(lines, "\n")
}

// typeOf returns the type of producer, and the type that a pointer points to for a pointer.
func typeOf(producer language.Configurable) reflect.Type {
	t := reflect.TypeOf(producer)
	if t.Kind() == reflect.Pointer {
		return t.Elem()
	}
	return t
}
