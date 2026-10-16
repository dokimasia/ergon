// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/tool"
	gocli "go.stealthscale.io/cli"
)

// downloadTimeout is the limit of a request of ergon tool run, such as the download of a release
// binary or of a jar, which a slow connection reaches for an asset of tens of megabytes.
const downloadTimeout = 10 * time.Minute

// The help texts of the tool commands. The first line of each is its summary in the list of the
// commands of its parent.
const (
	toolHelp = `Run the tools that the sections of .ergon.yaml name.

ergon tool runs the tools that the sections of .ergon.yaml name, at the
versions that they name. Every target of the Makefile and every hook of the
baseline of ergon init runs its tools this way.`

	runHelp = `Install a tool of a section and run it.

ergon tool run installs the tool that a section of .ergon.yaml names, unless
the cache has it, and runs it in the working directory with the arguments that
follow the tool. It installs the tool under ergon/tools in the cache directory
of the user, and checks a release binary against the digest that the section
states for the platform. It reads the options of the repository of the working
directory or of the nearest of its parents with .ergon/init.lock.

The exit status is the exit status of the tool. It is 1 when the tool does not
install, and 2 when the section or the tool does not exist.

Example:
  ergon tool run go.golangci-lint -- run ./...`

	pruneHelp = `Remove the tools that no section of .ergon.yaml names.

ergon tool prune removes each file from ergon/tools in the cache directory of
the user that installs no tool of the sections of .ergon.yaml at the version
that they name: the earlier versions of the tools, the programs of a Go module
for another version of the go command, the programs of golangci-lint for other
plugins, and the files of an install that did not finish. It also removes the
tools of other repositories that share the directory. Each job of the managed
workflows that keeps the tools of ergon in the cache of GitHub Actions runs it
before the cache saves the directory.`

	toolCIHelp = `Run the jobs that keep the tools of ergon in GitHub Actions.

ergon tool ci runs the steps of the jobs of the managed workflows that keep the
tool directory of ergon in the cache of GitHub Actions.`

	ciPruneHelp = `Delete the tool caches that newer caches replaced.

ergon tool ci prune deletes each cache of the tools of ergon in GitHub Actions
that a newer cache of the same job replaced, through the API of GitHub with the
token of GITHUB_TOKEN, which needs the permission actions: write. A job saves a
cache under a new key for each change of the files of its key, and restores
only the newest cache of its own when its key misses. The caches of a branch and
of a pull request are separate.`
)

// runOptions are the options of ergon tool run: the tool and its arguments.
type runOptions struct {
	Tool      string   `help:"Tool to run, as <section>.<tool>." arg:""`
	Arguments []string `help:"Arguments of the tool. Pass them after --." arg:"" default:""`
}

// toolCommands adds ergon tool to app, with its subcommands run, prune and ci prune.
func (s *session) toolCommands(app *gocli.App[noOptions]) {
	parent := app.Command[noOptions]("tool", toolHelp, nil)
	parent.Command("run", runHelp, s.toolRun)
	parent.Command("prune", pruneHelp, s.toolPrune)
	ci := parent.Command[noOptions]("ci", toolCIHelp, nil)
	ci.Command("prune", ciPruneHelp, s.toolCIPrune)
}

// toolRun runs the tool <section>.<tool> of its first argument with the arguments that follow it,
// through the runner of s under ctx in the working directory of s, and records the exit status of
// the tool in s. The runner starts an npm package with the package manager that the section of
// [option.PackageManaged] sets. It returns a [usageError] for a first argument without a dot, for a
// section of no producer of the repository and for a tool that the section does not name, and the
// errors of the runner, of the sections and of [tool.Runner.Run].
func (s *session) toolRun(ctx context.Context, run *gocli.Run[noOptions, runOptions]) error {
	o := run.Config()
	section, name, ok := strings.Cut(o.Tool, ".")
	if !ok {
		return usageError{err: fmt.Errorf("cli: %q, which is not <section>.<tool>", o.Tool)}
	}
	runner, err := s.runner()
	if err != nil {
		return err
	}
	return s.open(s.root(), func(r *baseline.Repository) error {
		sections, err := r.Sections()
		if err != nil {
			return err
		}
		var options language.Options
		for _, sec := range sections {
			if sec.Name == section {
				options = sec.Options
			}
			if m, ok := sec.Options.(option.PackageManaged); ok {
				runner.PackageManager = m.Manager()
			}
		}
		if options == nil {
			return usageError{err: fmt.Errorf("%w: %q", baseline.ErrUnknownSection, section)}
		}
		s.status, err = runner.Run(ctx, section, options, name, o.Arguments)
		if errors.Is(err, tool.ErrUnknown) {
			return usageError{err: err}
		}
		return err
	})
}

// toolPrune removes with [tool.Runner.Prune] under ctx each file from the cache of the runner of s
// that is not the install of a tool of the sections of the repository of the working directory of
// s. It writes a line for each removed path. It returns the errors of the runner, of the sections
// and of Prune.
func (s *session) toolPrune(ctx context.Context, _ *gocli.Run[noOptions, noOptions]) error {
	runner, err := s.runner()
	if err != nil {
		return err
	}
	return s.open(s.root(), func(r *baseline.Repository) error {
		sections, err := r.Sections()
		if err != nil {
			return err
		}
		options := make(map[string]language.Options, len(sections))
		for _, section := range sections {
			options[section.Name] = section.Options
		}
		removed, err := runner.Prune(ctx, options)
		for _, p := range removed {
			fmt.Fprintf(s.stdout, "removed %s\n", p)
		}
		return err
	})
}

// toolCIPrune deletes with [tool.PruneCaches] the caches of the tools of ergon of the repository of
// GITHUB_REPOSITORY that newer caches replaced, through the client of GitHub of s under ctx, and
// writes a line for each deleted cache. It returns the error of [session.forge] and of PruneCaches.
func (s *session) toolCIPrune(ctx context.Context, _ *gocli.Run[noOptions, noOptions]) error {
	client, repo, err := s.forge()
	if err != nil {
		return err
	}
	deleted, err := tool.PruneCaches(ctx, client, repo, github.ToolsCache)
	for _, c := range deleted {
		fmt.Fprintf(s.stdout, "deleted %s of %s\n", c.Key, c.Ref)
	}
	return err
}

// runner returns the runner of the tools of s: the tool directory ergon/tools in the cache directory
// of the user, the working directory, the platform, the environment and the standard streams of s,
// and a client of the transport of s that ends each request after downloadTimeout. It returns an
// error for a cache directory that cannot be found.
func (s *session) runner() (*tool.Runner, error) {
	cache, err := s.cacheDir()
	if err != nil {
		return nil, fmt.Errorf("cli: find the cache directory: %w", err)
	}
	return &tool.Runner{
		Client:   &http.Client{Transport: s.transport, Timeout: downloadTimeout},
		Stdin:    s.stdin,
		Stdout:   s.stdout,
		Stderr:   s.stderr,
		Cache:    filepath.Join(cache, program, "tools"),
		Dir:      s.dir,
		Platform: s.platform,
		Env:      s.env,
	}, nil
}
