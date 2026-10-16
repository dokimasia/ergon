// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	gocli "go.stealthscale.io/cli"
)

const (
	// program is the name of the program in the help and before every error.
	program = "ergon"

	// helpWidth is the number of columns of a line of a list in a help text.
	helpWidth = 80

	// help is the help text of ergon, with a verb for its languages.
	help = `ergon works on repositories with code in one or more of these languages:

%s`
)

// Program is the program ergon: what it needs of the process that runs it, the registration of its
// languages and its version. Every field is required. [Program.Main] runs a command line.
type Program struct {
	// Getwd returns the absolute path of the working directory, as [os.Getwd] does. Main calls it
	// once, before the command runs, and the command works on that directory.
	Getwd func() (string, error)

	// Now returns the current time, for the defaults that depend on it, such as the year of a
	// copyright notice.
	Now func() time.Time

	// CacheDir returns the cache directory of the user, as [os.UserCacheDir] does. ergon tool run
	// installs the tools under ergon/tools in it.
	CacheDir func() (string, error)

	// Random is the source of the random digits of the name of a new changeset, such as
	// [crypto/rand.Reader].
	Random io.Reader

	// Transport sends the requests of ergon to GitHub, to the module proxy of Go and to the
	// registries of the tools, such as [net/http.DefaultTransport].
	Transport http.RoundTripper

	// Platform is the system and the architecture that ergon runs on, such as linux/amd64, whose
	// release binaries ergon tool run and ergon init upgrade install.
	Platform option.Platform

	// Register fills the catalog of the languages.
	Register func(*language.Catalog) error

	// Version is the version of the running ergon.
	Version Version
}

// Version is the version of the running ergon.
type Version struct {
	// Release is the version of the release, such as 1.2.3, and dev for a build without one.
	// ergon init records it in the lock.
	Release string

	// Full is what --version writes after the name of the program, such as
	// "1.2.3 (abc123, built 2026-10-05)".
	Full string
}

// usageError is an error of the command line that a command finds, such as a tool of ergon tool run
// that its section does not name. [Program.Main] returns the status of a usage error for it.
type usageError struct {
	// err is the error that the command returned.
	err error
}

// Error returns the text of the error that the command returned.
func (e usageError) Error() string {
	return e.err.Error()
}

// Unwrap returns the error that the command returned.
func (e usageError) Unwrap() error {
	return e.err
}

// noOptions are the global options of ergon, and the options of a command without options of its
// own. go-cli adds the global flags --flag-debug, --help and --version.
type noOptions struct{}

// session is the state that the commands of one run of ergon share. The hook of the app resolves
// the working directory into it before a command runs.
type session struct {
	// getwd returns the absolute path of the working directory.
	getwd func() (string, error)

	// catalog has the languages and their roles.
	catalog *language.Catalog

	// now returns the current time.
	now func() time.Time

	// cacheDir returns the cache directory of the user.
	cacheDir func() (string, error)

	// random is the source of the random digits of the name of a new changeset.
	random io.Reader

	// stdin, stdout and stderr are the standard streams of the run.
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	// transport sends the requests of ergon.
	transport http.RoundTripper

	// dir is the absolute path of the working directory. It is empty until the hook of the app sets
	// it.
	dir string

	// release is the version of the release of ergon, which the lock of ergon init records.
	release string

	// platform is the platform that ergon runs on.
	platform option.Platform

	// args are the arguments of the run, which ergon init upgrade passes to the release that it
	// starts.
	args []string

	// env is the environment of the run, as KEY=value pairs.
	env []string

	// status is the exit status of the tool that ergon tool run ran, which [Program.Main] returns
	// when the command returns no error, and 0 for every other command.
	status int
}

// Main runs the command line of env and returns the exit status, as [gocli.App.Main] does for a
// program of go-cli. Every command runs under ctx. Main writes every error to the standard error of
// env, after the name of the program.
//
// env is the environment of the run, such as [gocli.OS]. Its Environ, Stdout and Stderr are
// required:
//
//   - The commands read the variable of each option through Lookup, and ergon release and ergon
//     init upgrade read the variables of GitHub Actions, of the editors and of the module proxy of
//     Go from Environ. ergon tool run passes Environ to a tool and to the toolchain that installs
//     it.
//   - Stdin is the standard input of a tool that ergon tool run runs, of the editor of ergon release
//     add --open, and of the git tag and the git push of ergon release publish and git-tag, from
//     which ssh-keygen reads the PIN of a signing key.
//   - Stdout receives the output of a command: the help, the version, and the files it wrote.
//   - Stderr receives every error, after the name of the program, and the standard error of the git
//     tag and the git push of ergon release publish and git-tag.
//
// The exit status is:
//
//   - 0 when the command succeeds
//   - 1 when p.Register returns an error, and when the command fails, such as for a configuration
//     file that does not parse or a managed file that was edited by hand
//   - 2 for a command line that ergon refuses, such as one with an unknown command or flag, or one
//     that ends at a command without its subcommand, after the help of that command
//   - the exit status of the tool that ergon tool run ran, without an error of ergon
func (p *Program) Main(ctx context.Context, env *gocli.Env) int {
	s := &session{
		getwd:     p.Getwd,
		catalog:   new(language.Catalog),
		now:       p.Now,
		cacheDir:  p.CacheDir,
		random:    p.Random,
		stdin:     env.Stdin,
		stdout:    env.Stdout,
		stderr:    env.Stderr,
		transport: p.Transport,
		release:   p.Version.Release,
		platform:  p.Platform,
		args:      env.Args,
		env:       env.Environ(),
	}
	if err := p.Register(s.catalog); err != nil {
		fmt.Fprintf(env.Stderr, "%s: %v\n", program, err)
		return gocli.ExitFailure
	}
	app := s.app(p.Version)
	renderer, err := app.ExecuteRenderer(ctx, env)
	f := app.Failure(err)
	if f == nil {
		return s.status
	}
	if _, ok := errors.AsType[usageError](err); ok {
		f.Usage = true
	}
	// The status reports the error also when the failure fails to write, as gocli.App.Main does.
	_ = renderer.Error(env.Stderr, s.env, f)
	if f.Usage {
		return gocli.ExitUsage
	}
	return gocli.ExitFailure
}

// app returns the app of go-cli of s: the commands init, license, release and tool with their
// subcommands, the version v.Full after the name of the program, and a hook that resolves the
// working directory with the getwd function of s before a command runs. The help lists the languages
// of the catalog of s. go-cli adds the command completion, which writes the completion script of a
// shell.
func (s *session) app(v Version) *gocli.App[noOptions] {
	var names []string
	for d := range s.catalog.Languages() {
		names = append(names, string(d.Name))
	}
	app := gocli.New[noOptions](program, fmt.Sprintf(help, list(names)))
	app.Version(program + " " + v.Full)
	app.Use(func(ctx context.Context, _ *gocli.Run[noOptions, noOptions], next func(context.Context) error) error {
		dir, err := s.getwd()
		if err != nil {
			return fmt.Errorf("cli: find the working directory: %w", err)
		}
		s.dir = dir
		return next(ctx)
	})
	s.initCommands(app, names)
	s.licenseCommands(app)
	s.releaseCommands(app)
	s.toolCommands(app)
	return app
}

// list returns words as a list of a help text: the words separated by commas, in lines of at most
// helpWidth columns that start with two spaces. A word longer than a line has a line of its own.
func list(words []string) string {
	const indent = "  "
	var b strings.Builder
	line := indent
	for i, w := range words {
		if i < len(words)-1 {
			w += ","
		}
		switch {
		case line == indent:
			line += w
		case len(line)+1+len(w) > helpWidth:
			b.WriteString(line)
			b.WriteString("\n")
			line = indent + w
		default:
			line += " " + w
		}
	}
	b.WriteString(line)
	return b.String()
}
