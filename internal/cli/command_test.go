// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/cli"
)

// The exit statuses that Run returns, pinned because scripts and CI read them.
const (
	statusOK      = 0
	statusFailure = 1
	statusUsage   = 2
)

// The fixtures of the cases: the version that --version writes, and the toolchain and the
// languages that register adds.
const (
	version                     = "1.2.3"
	tool    workspace.Toolchain = "tool"
	alpha   workspace.Language  = "alpha"
	beta    workspace.Language  = "beta"
)

// help is the help of ergon for a catalog of alpha and beta, pinned because a person reads it.
const help = `ergon works on repositories with code in one or more of these languages:

  alpha, beta

ergon reads its configuration from .ergon.yaml in the working directory, or
from the file that --config names. The configuration is YAML, whatever the
extension of its file.

Run 'ergon completion --help' for the completion scripts of bash, fish,
PowerShell and zsh.

Usage:
  ergon [flags]

Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
  -v, --version       show the version of ergon
`

// hint is the line that follows every error of the command line.
const hint = "Run 'ergon --help' for usage.\n"

// errRegister is the error of a registration that fails.
var errRegister = errors.New("register: failed")

func TestCommand(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			args   []string
			status int
			stdout string
			stderr string
		}{
			{name: "writes the help for --help", args: []string{"--help"}, status: statusOK, stdout: help},
			{name: "writes the help for -h", args: []string{"-h"}, status: statusOK, stdout: help},
			{name: "writes the help without an argument", args: nil, status: statusOK, stdout: help},
			{
				name: "writes the name and the version for --version", args: []string{"--version"},
				status: statusOK, stdout: "ergon " + version + "\n",
			},
			{
				name: "writes the name and the version for -v", args: []string{"-v"},
				status: statusOK, stdout: "ergon " + version + "\n",
			},
			{
				name: "returns 2 for an unknown command", args: []string{"release"},
				status: statusUsage, stderr: "ergon: unknown command \"release\" for \"ergon\"\n" + hint,
			},
			{
				name: "returns 2 for an unknown flag", args: []string{"--bogus"},
				status: statusUsage, stderr: "ergon: unknown flag: --bogus\n" + hint,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				status, stdout, stderr := run(t, tt.args...)
				assert.Equal(t, status, tt.status, "the exit status")
				assert.Equal(t, stdout, tt.stdout, "the standard output")
				assert.Equal(t, stderr, tt.stderr, "the standard error")
			})
		}

		t.Run("returns 1 when the registration fails", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			failing := func(*language.Catalog) error { return errRegister }
			assert.Equal(t, cli.Run(t.Context(), []string{"--help"}, failing, version, &stdout, &stderr),
				statusFailure, "the exit status")
			assert.Empty(t, stdout.String(), "the standard output")
			assert.Equal(t, stderr.String(), "ergon: "+errRegister.Error()+"\n", "the standard error")
		})

		t.Run("writes the completion script of bash", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, "completion", "bash")
			assert.Equal(t, status, statusOK, "the exit status")
			assert.HasPrefix(t, stdout, "# bash completion V2 for ergon", "the standard output")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("reads a --config file that parses", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := run(t, "--config", file(t, "ergon.yaml", "license:\n  spdx: MIT\n"))
			assert.Equal(t, status, statusOK, "the exit status")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("reads a --config file of another extension as YAML", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := run(t, "--config", file(t, "ergon.json", "license:\n  spdx: MIT\n"))
			assert.Equal(t, status, statusOK, "the exit status")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("returns 1 for a --config file that does not parse", func(t *testing.T) {
			t.Parallel()
			path := file(t, "ergon.yaml", "license: [\n")
			status, _, stderr := run(t, "--config", path)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: read "+path+": ", "the standard error")
		})

		t.Run("returns 1 for a --config file that does not exist", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "absent.yaml")
			status, _, stderr := run(t, "--config", path)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: read "+path+": ", "the standard error")
		})
	})
}

// run runs the command line args with the registration of alpha and beta, and returns the exit
// status, the standard output and the standard error.
func run(t *testing.T, args ...string) (status int, stdout, stderr string) {
	t.Helper()
	var out, errs bytes.Buffer
	status = cli.Run(t.Context(), args, register, version, &out, &errs)
	return status, out.String(), errs.String()
}

// register adds the toolchain tool and the languages alpha and beta to c.
func register(c *language.Catalog) error {
	if err := language.RegisterToolchain(c, language.Toolchain{Name: tool}); err != nil {
		return err
	}
	if err := language.Register(c, language.Declaration{Name: alpha, Toolchain: tool}); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: beta, Toolchain: tool})
}

// file writes content to a file named name in a temporary directory of t, and returns its path.
func file(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o600), "WriteFile of "+name)
	return path
}
