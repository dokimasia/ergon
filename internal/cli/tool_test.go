// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/internal/cli"
)

// fakeEnv is the variable of the environment that makes the test binary act as the program that the
// base name of its path states, as [fake] states.
const fakeEnv = "ERGON_CLI_FAKE"

// The variables of the environment that the fake program reads: the exit status, which exitFlag
// overrides, and the release that an upgrade of ergon started, which the fake writes.
const (
	fakeExitEnv = "ERGON_CLI_FAKE_EXIT"
	upgradeEnv  = "ERGON_UPGRADE"
)

// The flags of the fake program.
const (
	// exitFlag makes the fake exit with the status after the equals sign.
	exitFlag = "--exit="

	// pwdFlag makes the fake write its working directory.
	pwdFlag = "--pwd"

	// stdinFlag makes the fake copy its standard input to its standard output.
	stdinFlag = "--stdin"
)

// biome is the line that the fake npx writes for the Biome of the section js at the baseline,
// before the arguments of the tool.
const biome = "npx --yes --package=@biomejs/biome@2.5.15 -- biome"

// toolHelp is the help of ergon tool, pinned because a person reads it.
const toolHelp = `ergon tool runs the tools that the sections of .ergon.yaml name, at the
versions that they name. Every target of the Makefile and every hook of the
baseline of ergon init runs its tools this way.

Usage:
  ergon tool [flags]
  ergon tool [command]

Available Commands:
  run         Install a tool of a section and run it

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command

Use "ergon tool [command] --help" for more information about a command.
`

// runHelp is the help of ergon tool run, pinned because a person reads it.
const runHelp = `ergon tool run installs the tool that a section of .ergon.yaml names, unless
the cache has it, and runs it in the working directory with the arguments that
follow the tool. It installs the tool under ergon/tools in the cache directory
of the user, and checks a release binary against the digest that the section
states for the platform. It reads the options of the repository of the working
directory or of the nearest of its parents with .ergon/init.lock.

The exit status is the exit status of the tool. It is 1 when the tool does not
install, and 2 when the section or the tool does not exist.

Usage:
  ergon tool run <section>.<tool> [-- <argument>...] [flags]

Examples:
  ergon tool run go.golangci-lint -- run ./...

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// errCache is the error of a cache directory that cannot be found.
var errCache = errors.New("cache: $HOME is not defined")

func TestTool(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		help := []struct {
			name   string
			args   []string
			stdout string
		}{
			{name: "writes the help of tool", args: []string{"tool", "--help"}, stdout: toolHelp},
			{name: "writes the help of tool run", args: []string{"tool", "run", "--help"}, stdout: runHelp},
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
				name: "returns 2 for tool without a subcommand", args: []string{"tool"},
				stderr: "ergon: cli: tool needs a subcommand: run\nRun 'ergon tool --help' for usage.\n",
			},
			{
				name: "returns 2 for tool run without a tool", args: []string{"tool", "run"},
				stderr: "ergon: requires at least 1 arg(s), only received 0\nRun 'ergon tool run --help' for usage.\n",
			},
			{
				name:   "returns 2 for a tool without a section",
				args:   []string{"tool", "run", "biome"},
				stderr: "ergon: cli: \"biome\", which is not <section>.<tool>\nRun 'ergon tool run --help' for usage.\n",
			},
			{
				name: "returns 2 for a tool that the section does not name",
				args: []string{"tool", "run", "js.prettier"},
				stderr: "ergon: tool: the section names no such tool: js.prettier, which is none of js.biome\n" +
					"Run 'ergon tool run --help' for usage.\n",
			},
			{
				name: "returns 2 for a section of no producer of the repository",
				args: []string{"tool", "run", "go.vet"},
				stderr: "ergon: baseline: no producer of the repository has the section: \"go\"\n" +
					"Run 'ergon tool run --help' for usage.\n",
			},
		}
		for _, tt := range usage {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				status, stdout, stderr := runWith(t, app.Register, javascriptRepository(t), tt.args...)
				assert.Equal(t, status, statusUsage, "the exit status")
				assert.Empty(t, stdout, "the standard output")
				assert.Equal(t, stderr, tt.stderr, "the standard error")
			})
		}

		t.Run("runs the tool of the section with the arguments after the tool", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runWith(t, app.Register, javascriptRepository(t),
				"tool", "run", "js.biome", "--", "check", "--write")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, biome+" check --write\n", "the output of the tool")
		})

		t.Run("returns the exit status of the tool without an error", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := runWith(t, app.Register, javascriptRepository(t),
				"tool", "run", "js.biome", "--", exitFlag+"3")
			assert.Equal(t, status, 3, "the exit status")
			assert.Equal(t, stdout, biome+" "+exitFlag+"3\n", "the output of the tool")
			assert.Empty(t, stderr, "the standard error")
		})

		t.Run("runs the tool in the working directory", func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(javascriptRepository(t), "packages", "web")
			assert.NoError(t, os.MkdirAll(dir, 0o755), "MkdirAll of the package")
			resolved, err := filepath.EvalSymlinks(dir)
			assert.NoError(t, err, "EvalSymlinks of the package")
			status, stdout, stderr := runWith(t, app.Register, resolved, "tool", "run", "js.biome", "--", pwdFlag)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, biome+" "+pwdFlag+"\n"+resolved+"\n", "the output of the tool")
		})

		t.Run("passes the standard input of the process to the tool", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			p := process(javascriptRepository(t), &stdout, &stderr, "tool", "run", "js.biome", "--", stdinFlag)
			p.Stdin = strings.NewReader("src/index.js\n")
			assert.Equal(t, cli.Run(t.Context(), p, app.Register, version), statusOK, "the exit status")
			assert.Equal(t, stdout.String(), biome+" "+stdinFlag+"\nsrc/index.js\n", "the output of the tool")
		})

		t.Run("returns 1 for a tool without a digest for the platform", func(t *testing.T) {
			t.Parallel()
			dir := javascriptRepository(t)
			host := option.Platform(runtime.GOOS + "/" + runtime.GOARCH)
			other := option.LinuxARM64
			if host == other {
				other = option.LinuxAMD64
			}
			config := read(t, dir, ".ergon.yaml")
			start := strings.Index(config, "      sha256:\n")
			end := strings.Index(config, "      version: 0.12.0\n")
			assert.True(t, start > 0, "the digests of commitlint in .ergon.yaml")
			assert.True(t, end > start, "the version of commitlint after its digests")
			digests := "      sha256:\n        " + string(other) + ": " + strings.Repeat("ab", 32) + "\n"
			write(t, dir, ".ergon.yaml", config[:start]+digests+config[end:])
			status, stdout, stderr := runWith(t, app.Register, dir, "tool", "run", "common.commitlint")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: tool: the tool does not install: commitlint 0.12.0 has no digest for "+
				string(host)+"\n", "the standard error")
		})

		t.Run("returns 1 for a cache directory that cannot be found", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			p := process(javascriptRepository(t), &stdout, &stderr, "tool", "run", "js.biome")
			// The error comes with a directory of the test, which a run that went on would install into.
			cache := t.TempDir()
			p.CacheDir = func() (string, error) { return cache, errCache }
			assert.Equal(t, cli.Run(t.Context(), p, app.Register, version), statusFailure, "the exit status")
			assert.Equal(t, stderr.String(), "ergon: cli: find the cache directory: "+errCache.Error()+"\n",
				"the standard error")
		})

		t.Run("returns 1 for a .ergon.yaml that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := javascriptRepository(t)
			write(t, dir, ".ergon.yaml", "js: [\n")
			status, _, stderr := runWith(t, app.Register, dir, "tool", "run", "js.biome")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: options: invalid .ergon.yaml: ", "the standard error")
		})
	})
}

// javascriptRepository returns a new repository after init new with JavaScript and the answers of
// the cases.
func javascriptRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	args := slices.Concat([]string{"init", "new", "--language", "javascript"}, required)
	status, _, stderr := runWith(t, app.Register, dir, args...)
	assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
	return dir
}

// fake acts as the program of the base name of args[0], without the suffix .exe. It writes the
// name and the other arguments to stdout, on one line, and the line upgradeEnv=<release> when the
// environment sets upgradeEnv. Then it writes its working directory for pwdFlag, copies stdin to
// stdout for stdinFlag, and exits with the status of exitFlag, or of fakeExitEnv, or 0.
func fake(args []string, stdin io.Reader, stdout io.Writer) int {
	name := strings.TrimSuffix(filepath.Base(args[0]), ".exe")
	fmt.Fprintln(stdout, strings.Join(append([]string{name}, args[1:]...), " "))
	if release := os.Getenv(upgradeEnv); release != "" {
		fmt.Fprintln(stdout, upgradeEnv+"="+release)
	}
	status, _ := strconv.Atoi(os.Getenv(fakeExitEnv))
	for _, arg := range args[1:] {
		switch {
		case arg == pwdFlag:
			dir, _ := os.Getwd()
			fmt.Fprintln(stdout, dir)
		case arg == stdinFlag:
			_, _ = io.Copy(stdout, stdin)
		case strings.HasPrefix(arg, exitFlag):
			status, _ = strconv.Atoi(strings.TrimPrefix(arg, exitFlag))
		}
	}
	return status
}
