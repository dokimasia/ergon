// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/baseline"
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
)

// toolCommand returns ergon tool with its subcommand run, which runs a tool under ctx.
func toolCommand(ctx context.Context, s *session) *cobra.Command {
	return group(s, "tool", toolShort, toolLong, runCommand(ctx, s))
}

// runCommand returns ergon tool run, which runs the tool <section>.<tool> of its first argument
// with the arguments that follow it, through a [tool.Runner] under ctx in the working directory of
// s, and records the exit status of the tool in s. It returns a [usageError] for a first argument
// without a dot, for a section of no producer of the repository and for a tool that the section
// does not name, and the errors of the options and of the runner.
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
			cache, err := s.cacheDir()
			if err != nil {
				return fmt.Errorf("cli: find the cache directory: %w", err)
			}
			return s.open(s.root(), func(r *baseline.Repository) error {
				o, err := r.Options(section)
				if errors.Is(err, baseline.ErrUnknownSection) {
					return usageError{err: err}
				}
				if err != nil {
					return err
				}
				runner := tool.Runner{
					Client:   &http.Client{Timeout: downloadTimeout},
					Stdin:    cmd.InOrStdin(),
					Stdout:   cmd.OutOrStdout(),
					Stderr:   cmd.ErrOrStderr(),
					Cache:    filepath.Join(cache, program, "tools"),
					Dir:      s.dir,
					Platform: option.Platform(runtime.GOOS + "/" + runtime.GOARCH),
					Env:      s.env,
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
