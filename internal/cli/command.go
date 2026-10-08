// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
)

// The exit statuses of [Run] other than 0, the status of a command that succeeds.
const (
	// statusFailure is the status of a command that fails, and of a registration of the languages
	// that fails.
	statusFailure = 1

	// statusUsage is the status of a command line that ergon refuses, such as one with an unknown
	// command or flag.
	statusUsage = 2
)

const (
	// program is the name of the program in the help and before every error.
	program = "ergon"

	// configFlag is the flag that names the configuration file.
	configFlag = "config"

	// configFile is the configuration file that a command reads when the command line names none.
	configFile = ".ergon.yaml"

	// configType is the format of every configuration file, whatever its extension.
	configType = "yaml"

	// helpFlag and versionFlag are the flags that ergon defines in place of cobra's, for their
	// help texts.
	helpFlag    = "help"
	versionFlag = "version"

	// versionTemplate makes --version write the name of the program and its version.
	versionTemplate = "{{.Name}} {{.Version}}\n"

	// helpWidth is the number of columns of a line of a list in a help text.
	helpWidth = 80

	// short is the description of ergon in a list of commands.
	short = "Tooling for repositories in one or more languages"

	// long is the description of ergon in its help, with a verb for its languages.
	long = `ergon works on repositories with code in one or more of these languages:

%s

ergon reads its configuration from .ergon.yaml in the working directory, or
from the file that --config names. The configuration is YAML, whatever the
extension of its file.`
)

// Process is what [Run] needs of the process that runs ergon. Every field is required.
type Process struct {
	// Getwd returns the absolute path of the working directory, as [os.Getwd] does. Run calls it
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

	// Stdin is the standard input of a tool that ergon tool run runs, of the editor of ergon release
	// add --open, and of the git tag and the git push of ergon release publish and git-tag, from which
	// ssh-keygen reads the PIN of a signing key.
	Stdin io.Reader

	// Stdout receives the output of a command: the help, the version, and the files it wrote.
	Stdout io.Writer

	// Stderr receives every error, after the name of the program, and the standard error of the git
	// tag and the git push of ergon release publish and git-tag.
	Stderr io.Writer

	// Transport sends the requests of ergon to GitHub, to the module proxy of Go and to the
	// registries of the tools, such as [net/http.DefaultTransport].
	Transport http.RoundTripper

	// Platform is the system and the architecture that ergon runs on, such as linux/amd64, whose
	// release binaries ergon tool run and ergon init upgrade install.
	Platform option.Platform

	// Args are the arguments that follow the name of the program.
	Args []string

	// Env is the environment of the process, as [os.Environ] returns it, which ergon tool run passes
	// to a tool and to the toolchain that installs it.
	Env []string
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

// usageError is an error of the command line, such as an unknown command or flag. [Run] returns
// statusUsage for it and points at the help.
type usageError struct {
	// err is the error that cobra, the flag parser or a command returned.
	err error
}

// Error returns the text of the error that cobra, the flag parser or a command returned.
func (e usageError) Error() string {
	return e.err.Error()
}

// Unwrap returns the error that cobra, the flag parser or a command returned.
func (e usageError) Unwrap() error {
	return e.err
}

// session is the state that the commands of one run of ergon share. The command that runs
// resolves the working directory into it before its own function runs.
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

	// transport sends the requests of ergon.
	transport http.RoundTripper

	// dir is the absolute path of the working directory. It is empty until [session.resolve]
	// sets it.
	dir string

	// release is the version of the release of ergon, which the lock of ergon init records.
	release string

	// platform is the platform that ergon runs on.
	platform option.Platform

	// args are the arguments of the process, which ergon init upgrade passes to the release that it
	// starts.
	args []string

	// env is the environment of the process.
	env []string

	// status is the exit status of the tool that ergon tool run ran, which [Run] returns when the
	// command returns no error, and 0 for every other command.
	status int
}

// resolve sets the working directory of s with its getwd function. It returns an error that wraps
// the error of getwd.
func (s *session) resolve() error {
	dir, err := s.getwd()
	if err != nil {
		return fmt.Errorf("cli: find the working directory: %w", err)
	}
	s.dir = dir
	return nil
}

// Run runs the command line of p and returns the exit status. register fills the catalog of the
// languages, and v is the version of the running ergon. Every command runs under ctx. Run writes
// every error to p.Stderr, after the name of the program.
//
// The exit status is:
//
//   - 0 when the command succeeds
//   - 1 when register returns an error, and when the command fails, such as for a configuration
//     file that does not parse or a managed file that was edited by hand
//   - 2 for a command line that ergon refuses, such as one with an unknown command or flag, after
//     which Run points at the help of the command
//   - the exit status of the tool that ergon tool run ran, without an error of ergon
func Run(ctx context.Context, p *Process, register func(*language.Catalog) error, v Version) int {
	root, s, err := command(ctx, p, register, v)
	ran := root
	if err == nil {
		// cobra reads the arguments of the process for nil arguments, so the slice is never nil.
		root.SetArgs(append([]string{}, p.Args...))
		root.SetIn(p.Stdin)
		root.SetOut(p.Stdout)
		root.SetErr(p.Stderr)
		ran, err = root.ExecuteContextC(ctx)
	}
	if err == nil {
		return s.status
	}
	fmt.Fprintf(p.Stderr, "%s: %v\n", program, err)
	if _, ok := errors.AsType[usageError](err); ok {
		fmt.Fprintf(p.Stderr, "Run '%s --%s' for usage.\n", ran.CommandPath(), helpFlag)
		return statusUsage
	}
	return statusFailure
}

// command returns the root command of ergon and the session that its commands share. It registers
// the languages with register into a catalog, which the help lists and the commands use, and
// returns the error of register. v is the version of the running ergon, and the commands run under
// ctx.
//
// Before a command runs, the root command resolves the working directory with p.Getwd and reads
// the configuration with [load]. ergon init, ergon license and ergon tool resolve the directory
// alone, because they read the options of .ergon.yaml through the baseline of ergon init. Without a
// subcommand, the root command writes the help. cobra adds the command help, and the command
// completion, which writes the completion script of a shell.
func command(ctx context.Context, p *Process, register func(*language.Catalog) error, v Version) (
	*cobra.Command, *session, error,
) {
	s := &session{
		getwd:     p.Getwd,
		catalog:   new(language.Catalog),
		now:       p.Now,
		cacheDir:  p.CacheDir,
		random:    p.Random,
		transport: p.Transport,
		release:   v.Release,
		args:      p.Args,
		env:       p.Env,
		platform:  p.Platform,
	}
	if err := register(s.catalog); err != nil {
		return nil, nil, err
	}
	var names []string
	for d := range s.catalog.Languages() {
		names = append(names, string(d.Name))
	}

	config := viper.New()
	var file string
	root := &cobra.Command{
		Use:           program,
		Short:         short,
		Long:          fmt.Sprintf(long, list(names)),
		Version:       v.Full,
		Args:          usage(cobra.NoArgs),
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := s.resolve(); err != nil {
				return err
			}
			return load(config, s.dir, file, cmd.Flags().Changed(configFlag))
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return usageError{err: err}
	})
	root.SetVersionTemplate(versionTemplate)

	flags := root.PersistentFlags()
	flags.StringVar(&file, configFlag, configFile, "read the configuration from `file`")
	flags.BoolP(helpFlag, "h", false, "show the help of the command")
	root.Flags().BoolP(versionFlag, "v", false, "show the version of ergon")
	root.AddCommand(initCommand(ctx, s, names), licenseCommand(ctx, s), releaseCommand(ctx, s), toolCommand(ctx, s))
	return root, s, nil
}

// group returns a command that groups subcommands, such as ergon init, which resolves the working
// directory of s before a subcommand runs and reads no configuration through viper. Without a
// subcommand, it returns a [usageError] that lists its subcommands.
func group(s *session, use, short, long string, subcommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  usage(cobra.NoArgs),
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return s.resolve()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			names := make([]string, 0, len(subcommands))
			for _, sub := range cmd.Commands() {
				names = append(names, sub.Name())
			}
			err := fmt.Errorf("cli: %s needs a subcommand: %s", cmd.Name(), strings.Join(names, ", "))
			return usageError{err: err}
		},
	}
	cmd.AddCommand(subcommands...)
	return cmd
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

// usage returns the validation of args as a validation whose error is a [usageError].
func usage(args cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, given []string) error {
		if err := args(cmd, given); err != nil {
			return usageError{err: err}
		}
		return nil
	}
}

// load reads the configuration file into config as YAML. A relative file is relative to dir.
// named reports whether the command line named file. A missing file is no error when the command
// line names none, because a repository without .ergon.yaml uses the defaults. load returns an
// error for a named file that does not exist, and for a file that does not parse.
func load(config *viper.Viper, dir, file string, named bool) error {
	if !filepath.IsAbs(file) {
		file = filepath.Join(dir, file)
	}
	config.SetConfigFile(file)
	config.SetConfigType(configType)
	err := config.ReadInConfig()
	if err == nil || (!named && errors.Is(err, fs.ErrNotExist)) {
		return nil
	}
	return fmt.Errorf("cli: read %s: %w", file, err)
}
