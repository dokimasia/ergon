// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/cli"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The exit statuses that Run returns, pinned because scripts and CI read them.
const (
	statusOK      = 0
	statusFailure = 1
	statusUsage   = 2
)

// The fixtures of the cases: the toolchain and the languages that register adds, and the file
// that each language renders.
const (
	tool      workspace.Toolchain = "tool"
	alpha     workspace.Language  = "alpha"
	beta      workspace.Language  = "beta"
	alphaFile                     = "alpha.txt"
	betaFile                      = "beta.txt"
)

// help is the help of ergon for a catalog of alpha and beta, pinned because a person reads it.
const help = `ergon works on repositories with code in one or more of these languages:

  alpha, beta

ergon reads its configuration from .ergon.yaml in the working directory, or
from the file that --config names. The configuration is YAML, whatever the
extension of its file.

Usage:
  ergon [flags]
  ergon [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  init        Set up a repository with the baseline of ergon
  license     Add and check the license header of every file
  release     Plan, write and publish the releases of the packages of the repository
  tool        Run the tools that the sections of .ergon.yaml name

Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
  -v, --version       show the version of ergon

Use "ergon [command] --help" for more information about a command.
`

// hint is the line that follows every error of the command line of ergon itself.
const hint = "Run 'ergon --help' for usage.\n"

// version is the version of ergon in the cases.
var version = cli.Version{Release: "1.2.3", Full: "1.2.3 (abc123, built 2026-10-05)"}

// now is the current time in the cases.
var now = time.Date(2026, time.October, 5, 12, 0, 0, 0, time.UTC)

// errRegister is the error of a registration that fails.
var errRegister = errors.New("register: failed")

// errGetwd is the error of a working directory that no longer exists.
var errGetwd = errors.New("getwd: no such file or directory")

// fixed is a producer of the cases, whose templates are the files of the map.
type fixed fstest.MapFS

// Templates returns the templates of f.
func (f fixed) Templates() fs.FS {
	return fstest.MapFS(f)
}

// TestMain acts as a fake program when fakeEnv is set, as [fake] states. Otherwise it isolates git
// from the configuration of the user, puts the fake npx first on the PATH of the process, from
// which exec resolves a program, and runs the tests.
func TestMain(m *testing.M) {
	if os.Getenv(fakeEnv) != "" {
		os.Exit(fake(os.Args, os.Stdin, os.Stdout))
	}
	vcstest.Isolate()
	self, err := os.Executable()
	var content []byte
	if err == nil {
		content, err = os.ReadFile(self)
	}
	dir, err2 := os.MkdirTemp("", "ergon-cli-fakes-")
	npx := filepath.Join(dir, "npx")
	if runtime.GOOS == "windows" {
		npx += ".exe"
	}
	err3 := os.WriteFile(npx, content, 0o755)
	if joined := errors.Join(err, err2, err3); joined != nil {
		fmt.Fprintln(os.Stderr, "cli_test: the fake npx does not install:", joined)
		os.Exit(2)
	}
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

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
				status: statusOK, stdout: "ergon " + version.Full + "\n",
			},
			{
				name: "writes the name and the version for -v", args: []string{"-v"},
				status: statusOK, stdout: "ergon " + version.Full + "\n",
			},
			{
				name: "returns 2 for an unknown command", args: []string{"bogus"},
				status: statusUsage, stderr: "ergon: unknown command \"bogus\" for \"ergon\"\n" + hint,
			},
			{
				name: "returns 2 for an unknown flag", args: []string{"--bogus"},
				status: statusUsage, stderr: "ergon: unknown flag: --bogus\n" + hint,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				status, stdout, stderr := run(t, t.TempDir(), tt.args...)
				assert.Equal(t, status, tt.status, "the exit status")
				assert.Equal(t, stdout, tt.stdout, "the standard output")
				assert.Equal(t, stderr, tt.stderr, "the standard error")
			})
		}

		t.Run("returns 1 when the registration fails", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			failing := func(*language.Catalog) error { return errRegister }
			p := process(t.TempDir(), &stdout, &stderr, "--help")
			assert.Equal(t, cli.Run(t.Context(), p, failing, version), statusFailure, "the exit status")
			assert.Empty(t, stdout.String(), "the standard output")
			assert.Equal(t, stderr.String(), "ergon: "+errRegister.Error()+"\n", "the standard error")
		})

		t.Run("returns 1 when the working directory cannot be found", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			p := process(t.TempDir(), &stdout, &stderr)
			p.Getwd = func() (string, error) { return "", errGetwd }
			assert.Equal(t, cli.Run(t.Context(), p, register, version), statusFailure, "the exit status")
			assert.Empty(t, stdout.String(), "the standard output")
			assert.Equal(t, stderr.String(), "ergon: cli: find the working directory: "+errGetwd.Error()+"\n",
				"the standard error")
		})

		t.Run("writes the completion script of bash", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, t.TempDir(), "completion", "bash")
			assert.Equal(t, status, statusOK, "the exit status")
			assert.HasPrefix(t, stdout, "# bash completion V2 for ergon", "the standard output")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("reads a --config file that parses", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := run(t, t.TempDir(), "--config", file(t, "ergon.yaml", "license:\n  spdx: MIT\n"))
			assert.Equal(t, status, statusOK, "the exit status")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("reads a --config file of another extension as YAML", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := run(t, t.TempDir(), "--config", file(t, "ergon.json", "license:\n  spdx: MIT\n"))
			assert.Equal(t, status, statusOK, "the exit status")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("returns 1 for a --config file that does not parse", func(t *testing.T) {
			t.Parallel()
			path := file(t, "ergon.yaml", "license: [\n")
			status, _, stderr := run(t, t.TempDir(), "--config", path)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: read "+path+": ", "the standard error")
		})

		t.Run("returns 1 for a --config file that does not exist", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "absent.yaml")
			status, _, stderr := run(t, t.TempDir(), "--config", path)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: read "+path+": ", "the standard error")
		})

		t.Run("returns 1 for a .ergon.yaml of the working directory that does not parse", func(t *testing.T) {
			t.Parallel()
			path := file(t, ".ergon.yaml", "license: [\n")
			status, _, stderr := run(t, filepath.Dir(path))
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: read "+path+": ", "the standard error")
		})

		t.Run("writes the languages of the help in lines of at most 80 columns", func(t *testing.T) {
			t.Parallel()
			ten := func(c *language.Catalog) error {
				if err := language.RegisterToolchain(c, language.Toolchain{Name: tool}); err != nil {
					return err
				}
				for letter := range strings.SplitSeq("abcdefghij", "") {
					d := language.Declaration{Name: workspace.Language("language" + letter), Toolchain: tool}
					if err := language.Register(c, d); err != nil {
						return err
					}
				}
				return nil
			}
			status, stdout, stderr := runWith(t, ten, t.TempDir(), "--help")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.HasPrefix(t, stdout, "ergon works on repositories with code in one or more of these languages:\n\n"+
				"  languagea, languageb, languagec, languaged, languagee, languagef, languageg,\n"+
				"  languageh, languagei, languagej\n\n", "the help")
		})
	})
}

// run runs the command line args in the working directory dir, with the registration of alpha
// and beta, and returns the exit status, the standard output and the standard error.
func run(t *testing.T, dir string, args ...string) (status int, stdout, stderr string) {
	t.Helper()
	return runWith(t, register, dir, args...)
}

// runWith runs the command line args in the working directory dir, with the registration
// register, and returns the exit status, the standard output and the standard error.
func runWith(
	t *testing.T, register func(*language.Catalog) error, dir string, args ...string,
) (status int, stdout, stderr string) {
	t.Helper()
	var out, errs bytes.Buffer
	status = cli.Run(t.Context(), process(dir, &out, &errs, args...), register, version)
	return status, out.String(), errs.String()
}

// process returns the process of a case: the command line args, the working directory dir, the
// time now, the temporary directory of the system as the cache directory, the random digits ab and
// cd, no standard input, the environment of the test with fakeEnv set, and the writers stdout and
// stderr.
func process(dir string, stdout, stderr io.Writer, args ...string) *cli.Process {
	return &cli.Process{
		Getwd:    func() (string, error) { return dir, nil },
		Now:      func() time.Time { return now },
		CacheDir: func() (string, error) { return os.TempDir(), nil },
		Random:   bytes.NewReader([]byte{0xab, 0xcd}),
		Stdin:    strings.NewReader(""),
		Stdout:   stdout,
		Stderr:   stderr,
		Args:     args,
		Env:      append(os.Environ(), fakeEnv+"=1"),
	}
}

// register adds the toolchain tool and the languages alpha and beta to c. Each language renders
// its own file and a fragment of .gitignore.
func register(c *language.Catalog) error {
	if err := language.RegisterToolchain(c, language.Toolchain{Name: tool}); err != nil {
		return err
	}
	err := language.Register(c, language.Declaration{Name: alpha, Toolchain: tool}, producer(alpha, alphaFile))
	if err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: beta, Toolchain: tool}, producer(beta, betaFile))
}

// producer returns the producer of the language name: the managed file path, whose content is the
// name, and the fragment name/ of .gitignore.
func producer(name workspace.Language, path string) fixed {
	return fixed{
		"shared/.gitignore.tmpl":    {Data: []byte(string(name) + "/\n")},
		"managed/" + path + ".tmpl": {Data: []byte(string(name) + "\n")},
	}
}

// file writes content to a file named name in a temporary directory of t, and returns its path.
func file(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o600), "WriteFile of "+name)
	return path
}
