// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/cli"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
	"go.stealthscale.io/cli/clitest"
)

// The exit statuses that Main returns, pinned because scripts and CI read them.
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

// version is the version of ergon in the cases.
var version = cli.Version{Release: "1.2.3", Full: "1.2.3 (abc123, built 2026-10-05)"}

// now is the current time in the cases.
var now = time.Date(2026, time.October, 5, 12, 0, 0, 0, time.UTC)

// native is the platform of the test, which the program of each case states.
var native = option.Platform(runtime.GOOS + "/" + runtime.GOARCH)

// errRegister is the error of a registration that fails.
var errRegister = errors.New("register: failed")

// errGetwd is the error of a working directory that no longer exists.
var errGetwd = errors.New("getwd: no such file or directory")

// commandRuns are the command lines of the transcript of ergon itself.
var commandRuns = []clitest.Input{
	{Args: []string{"--help"}},
	{Args: []string{"-h"}},
	{Args: nil},
	{Args: []string{"--version"}},
	{Args: []string{"init", "--version"}},
	{Args: []string{"bogus"}},
	{Args: []string{"--bogus"}},
}

// fixed is a producer of the cases, whose templates are the files of the map.
type fixed fstest.MapFS

// Templates returns the templates of f.
func (f fixed) Templates() fs.FS {
	return fstest.MapFS(f)
}

// TestMain acts as a fake program when fakeEnv is set, as [fake] states. Otherwise it isolates git
// from the configuration of the user, puts the fake npx and pnpm first on the PATH of the process,
// from which exec resolves a program, and runs the tests.
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
	var err3 error
	for _, name := range []string{"npx", "pnpm"} {
		binary := filepath.Join(dir, name)
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		err3 = errors.Join(err3, os.WriteFile(binary, content, 0o755))
	}
	if joined := errors.Join(err, err2, err3); joined != nil {
		fmt.Fprintln(os.Stderr, "cli_test: the fakes of npx and pnpm do not install:", joined)
		os.Exit(2)
	}
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestCommand(t *testing.T) {
	t.Parallel()

	t.Run("Main", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "ergon", program(t.TempDir(), register).Main, commandRuns...)
			golden.Match(t, "command.txt", got, golden.ShouldUpdate())
		})

		t.Run("writes the version to the standard output", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, t.TempDir(), "--version")
			assert.Equal(t, status, statusOK, "the exit status")
			assert.Equal(t, stdout, "ergon "+version.Full+"\n", "the standard output")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("writes the help to the standard output", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, t.TempDir(), "--help")
			assert.Equal(t, status, statusOK, "the exit status")
			assert.HasPrefix(t, stdout, "Usage: ergon <command> [options]\n", "the standard output")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("returns 2 without a command after the help on the standard error", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, t.TempDir())
			assert.Equal(t, status, statusUsage, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			expect.That(t, stderr).
				HasPrefix("Usage: ergon <command> [options]\n", "the help on the standard error").
				HasSuffix("\nergon: missing command\n", "the error after the help")
		})

		t.Run("returns 1 when the registration fails", func(t *testing.T) {
			t.Parallel()
			p := program(t.TempDir(), func(*language.Catalog) error { return errRegister })
			res := clitest.Main(t.Context(), p.Main, &clitest.Input{Args: []string{"--help"}})
			assert.Equal(t, res.Status, statusFailure, "the exit status")
			assert.Empty(t, res.Stdout, "the standard output")
			assert.Equal(t, res.Stderr, "ergon: "+errRegister.Error()+"\n", "the standard error")
		})

		t.Run("returns 1 when the working directory cannot be found", func(t *testing.T) {
			t.Parallel()
			p := program(t.TempDir(), register)
			p.Getwd = func() (string, error) { return "", errGetwd }
			res := clitest.Main(t.Context(), p.Main, &clitest.Input{Args: []string{"init", "check"}})
			assert.Equal(t, res.Status, statusFailure, "the exit status")
			assert.Empty(t, res.Stdout, "the standard output")
			assert.Equal(t, res.Stderr, "ergon: cli: find the working directory: "+errGetwd.Error()+"\n",
				"the standard error")
		})

		t.Run("writes the completion script of bash", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, t.TempDir(), "completion", "bash")
			assert.Equal(t, status, statusOK, "the exit status")
			assert.Contains(t, stdout, "\n# bash completion of ergon, for bash 4.1 or later.\n", "the standard output")
			assert.Empty(t, stderr, "the standard error")
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
			assert.Contains(t, stdout, "\nergon works on repositories with code in one or more of these languages:\n\n"+
				"  languagea, languageb, languagec, languaged, languagee, languagef, languageg,\n"+
				"  languageh, languagei, languagej\n\n", "the help")
		})
	})
}

// program returns the program of a case: the working directory dir, the time now, the temporary
// directory of the system as the cache directory, the random digits ab and cd, the transport of
// net/http, the platform of the test, the registration register and the version of the cases.
func program(dir string, register func(*language.Catalog) error) *cli.Program {
	return &cli.Program{
		Getwd:     func() (string, error) { return dir, nil },
		Now:       func() time.Time { return now },
		CacheDir:  func() (string, error) { return os.TempDir(), nil },
		Random:    bytes.NewReader([]byte{0xab, 0xcd}),
		Transport: http.DefaultTransport,
		Platform:  native,
		Register:  register,
		Version:   version,
	}
}

// environ returns the environment of the test with fakeEnv set, as the variables of an input.
func environ() map[string]string {
	vars := make(map[string]string)
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			vars[key] = value
		}
	}
	vars[fakeEnv] = "1"
	return vars
}

// run runs the command line args in the working directory dir, with the registration of alpha
// and beta, and returns the exit status, the standard output and the standard error.
func run(t *testing.T, dir string, args ...string) (status int, stdout, stderr string) {
	t.Helper()
	return runWith(t, register, dir, args...)
}

// runWith runs the command line args in the working directory dir, with the registration
// register and the environment of environ, and returns the exit status, the standard output and
// the standard error.
func runWith(
	t *testing.T, register func(*language.Catalog) error, dir string, args ...string,
) (status int, stdout, stderr string) {
	t.Helper()
	res := clitest.Main(t.Context(), program(dir, register).Main, &clitest.Input{Args: args, Vars: environ()})
	return res.Status, res.Stdout, res.Stderr
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
