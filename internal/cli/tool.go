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

	"github.com/spf13/cobra"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/tool"
)

// downloadTimeout is the limit of a request of ergon tool run, such as the download of a release
// binary or of a jar, which a slow connection reaches for an asset of tens of megabytes.
const downloadTimeout = 10 * time.Minute

// The help texts of the tool commands.
const (
	toolShort = "Run the tools that the sections of .ergon.yaml name"
	toolLong  = `ergon tool runs the tools that the sections of .ergon.yaml name, at the
versions that they name. Every target of the Makefile and every hook of the
baseline of ergon init runs its tools this way.`

	runShort = "Install a tool of a section and run it"
	runLong  = `ergon tool run installs the tool that a section of .ergon.yaml names, unless
the cache has it, and runs it in the working directory with the arguments that
follow the tool. It installs the tool under ergon/tools in the cache directory
of the user, and checks a release binary against the digest that the section
states for the platform. It reads the options of the repository of the working
directory or of the nearest of its parents with .ergon/init.lock.

The exit status is the exit status of the tool. It is 1 when the tool does not
install, and 2 when the section or the tool does not exist.`
	runExample = "  ergon tool run go.golangci-lint -- run ./..."

	pruneShort = "Remove the tools that no section of .ergon.yaml names"
	pruneLong  = `ergon tool prune removes each file from ergon/tools in the cache directory of
the user that installs no tool of the sections of .ergon.yaml at the version
that they name: the earlier versions of the tools, the programs of a Go module
for another version of the go command, the programs of golangci-lint for other
plugins, and the files of an install that did not finish. It also removes the
tools of other repositories that share the directory. Each job of the managed
workflows that keeps the tools of ergon in the cache of GitHub Actions runs it
before the cache saves the directory.`

	toolCIShort = "Run the jobs that keep the tools of ergon in GitHub Actions"
	toolCILong  = `ergon tool ci runs the steps of the jobs of the managed workflows that keep the
tool directory of ergon in the cache of GitHub Actions.`

	ciPruneShort = "Delete the tool caches that newer caches replaced"
	ciPruneLong  = `ergon tool ci prune deletes each cache of the tools of ergon in GitHub Actions
that a newer cache of the same job replaced, through the API of GitHub with the
token of GITHUB_TOKEN, which needs the permission actions: write. A job saves a
cache under a new key for each change of the files of its key, and restores
only the newest cache of its own when its key misses. The caches of a branch and
of a pull request are separate.`
)

// toolCommand returns ergon tool with its subcommands run, which runs a tool under ctx, prune and
// ci.
func toolCommand(ctx context.Context, s *session) *cobra.Command {
	ci := group(s, "ci", toolCIShort, toolCILong, ciPruneCommand(ctx, s))
	return group(s, "tool", toolShort, toolLong, runCommand(ctx, s), pruneCommand(ctx, s), ci)
}

// runCommand returns ergon tool run, which runs the tool <section>.<tool> of its first argument
// with the arguments that follow it, through the runner of s under ctx in the working directory of
// s, and records the exit status of the tool in s. It returns a [usageError] for a first argument
// without a dot, for a section of no producer of the repository and for a tool that the section
// does not name, and the errors of the runner, of the options and of [tool.Runner.Run].
func runCommand(ctx context.Context, s *session) *cobra.Command {
	return &cobra.Command{
		Use:     "run <section>.<tool> [-- <argument>...]",
		Short:   runShort,
		Long:    runLong,
		Example: runExample,
		Args:    usage(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			section, name, ok := strings.Cut(args[0], ".")
			if !ok {
				return usageError{err: fmt.Errorf("cli: %q, which is not <section>.<tool>", args[0])}
			}
			runner, err := s.runner(cmd)
			if err != nil {
				return err
			}
			return s.open(s.root(), func(r *baseline.Repository) error {
				o, err := r.Options(section)
				if errors.Is(err, baseline.ErrUnknownSection) {
					return usageError{err: err}
				}
				if err != nil {
					return err
				}
				s.status, err = runner.Run(ctx, section, o, name, args[1:])
				if errors.Is(err, tool.ErrUnknown) {
					return usageError{err: err}
				}
				return err
			})
		},
	}
}

// pruneCommand returns ergon tool prune, which removes with [tool.Runner.Prune] under ctx each file
// from the cache of the runner of s that is not the install of a tool of the sections of the
// repository of the working directory of s. It writes a line for each removed path. It returns the
// errors of the runner, of the sections and of Prune.
func pruneCommand(ctx context.Context, s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: pruneShort,
		Long:  pruneLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			runner, err := s.runner(cmd)
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
					fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", p)
				}
				return err
			})
		},
	}
}

// ciPruneCommand returns ergon tool ci prune, which deletes with [tool.PruneCaches] the caches of
// the tools of ergon of the repository of GITHUB_REPOSITORY that newer caches replaced, through the
// client of GitHub of s under ctx, and writes a line for each deleted cache. It returns the error of
// [session.forge] and of PruneCaches.
func ciPruneCommand(ctx context.Context, s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: ciPruneShort,
		Long:  ciPruneLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, repo, err := s.forge()
			if err != nil {
				return err
			}
			deleted, err := tool.PruneCaches(ctx, client, repo, github.ToolsCache)
			for _, c := range deleted {
				fmt.Fprintf(cmd.OutOrStdout(), "deleted %s of %s\n", c.Key, c.Ref)
			}
			return err
		},
	}
}

// runner returns the runner of the tools of s for cmd: the tool directory ergon/tools in the cache
// directory of the user, the working directory, the platform and the environment of s, the
// standard streams of cmd, and a client of the transport of s that ends each request after
// downloadTimeout. It returns an error for a cache directory that cannot be found.
func (s *session) runner(cmd *cobra.Command) (*tool.Runner, error) {
	cache, err := s.cacheDir()
	if err != nil {
		return nil, fmt.Errorf("cli: find the cache directory: %w", err)
	}
	return &tool.Runner{
		Client:   &http.Client{Transport: s.transport, Timeout: downloadTimeout},
		Stdin:    cmd.InOrStdin(),
		Stdout:   cmd.OutOrStdout(),
		Stderr:   cmd.ErrOrStderr(),
		Cache:    filepath.Join(cache, program, "tools"),
		Dir:      s.dir,
		Platform: s.platform,
		Env:      s.env,
	}, nil
}
