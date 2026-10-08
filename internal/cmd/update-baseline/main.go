// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/internal/cli"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/pin"
)

// name is the name of the command before every error.
const name = "update-baseline"

// The exit statuses of [run] other than 0, the status of a run that succeeds.
const (
	// statusFailure is the status of a run that fails, and of a run with a pin that does not resolve.
	statusFailure = 1

	// statusUsage is the status of a command line that the command refuses.
	statusUsage = 2
)

// defaultMinAge is the age of a release before an update takes it, unless -min-age sets another.
const defaultMinAge = 168 * time.Hour

// process is what [run] needs of the process that runs the command. Every field is required.
type process struct {
	// getwd returns the absolute path of the working directory, the root of ergon's repository.
	getwd func() (string, error)

	// getenv returns the value of a variable of the environment, and the empty string for a variable
	// that is not set.
	getenv func(string) string

	// now returns the current time, against which the minimum age counts.
	now func() time.Time

	// execute runs a program, as [execute] states.
	execute func(ctx context.Context, dir string, stdout, stderr io.Writer, program string, args ...string) error

	// register fills the catalog of the languages whose producers the command updates.
	register func(*language.Catalog) error

	// random is the source of the random digits of the name of the changeset.
	random io.Reader

	// stdout receives the updates, the files that the command writes, and the pull request.
	stdout io.Writer

	// stderr receives every error, after the name of the command, and the standard error of the
	// programs that the command runs.
	stderr io.Writer

	// transport sends the requests to the registries and to GitHub.
	transport http.RoundTripper

	// registries are the addresses of the registries of the pins.
	registries pin.Registries

	// base are the producers of the files that every repository has, before the languages.
	base []baseline.Producer

	// args are the arguments that follow the name of the command.
	args []string
}

// options are the flags of a run.
type options struct {
	// minAge is the age of a release before an update takes it.
	minAge time.Duration

	// major lets an update take a release of a later major version.
	major bool

	// propose commits the changes on the branch ergon-update/<base> and opens its pull request.
	propose bool
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	p := process{
		getwd:      os.Getwd,
		getenv:     os.Getenv,
		now:        time.Now,
		execute:    execute,
		register:   app.Register,
		random:     rand.Reader,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		transport:  http.DefaultTransport,
		registries: pin.PublicRegistries(),
		base:       cli.Producers(),
		args:       os.Args[1:],
	}
	status := run(ctx, &p)
	stop() //dokimi:mutate-skip sbr-delete: os.Exit ends the process next, with the handlers of the signals
	os.Exit(status)
}

// run runs the command line of p under ctx, and returns the exit status. It writes every error to
// the standard error of p, after the name of the command. The exit status is:
//
//   - 0 when every pin resolves and every step succeeds, and for -h
//   - 1 when a step fails, and when a pin does not resolve, after every other step
//   - 2 for a command line that the command refuses, after the usage
func run(ctx context.Context, p *process) int {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(p.stderr)
	var o options
	flags.DurationVar(&o.minAge, "min-age", defaultMinAge, "take a release once it is `age` old")
	flags.BoolVar(&o.major, "major", false, "take a release of a later major version")
	flags.BoolVar(
		&o.propose,
		"propose",
		false,
		"open the pull request of the changes on the branch ergon-update/<base>",
	)
	if err := flags.Parse(p.args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return statusUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(p.stderr, "%s: the command takes no argument, and has %q\n", name, flags.Args())
		flags.Usage()
		return statusUsage
	}
	if err := update(ctx, p, &o); err != nil {
		fmt.Fprintf(p.stderr, "%s: %v\n", name, err)
		return statusFailure
	}
	return 0
}

// execute runs program with args in dir under ctx, with stdout and stderr as its standard output and
// standard error and no standard input. It returns an error for a program that does not start and
// for an exit status other than 0, with the command line.
func execute(ctx context.Context, dir string, stdout, stderr io.Writer, program string, args ...string) error {
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", program, strings.Join(args, " "), err)
	}
	return nil
}
