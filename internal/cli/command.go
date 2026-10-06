// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.dokimi.dev/ergon/core/language"
)

// The exit statuses of [Run].
const (
	// statusOK is the status of a command that succeeds.
	statusOK = 0

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

	// Stdout receives the output of a command: the help, the version, and the files it wrote.
	Stdout io.Writer

	// Stderr receives every error, after the name of the program.
	Stderr io.Writer

	// Args are the arguments that follow the name of the program.
	Args []string
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

	// dir is the absolute path of the working directory. It is empty until [session.resolve]
	// sets it.
	dir string

	// release is the version of the release of ergon, which the lock of ergon init records.
	release string
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
// languages, and v is the version of the running ergon. Every command receives ctx. Run writes
// every error to p.Stderr, after the name of the program.
//
// The exit status is:
//
//   - 0 when the command succeeds
//   - 1 when register returns an error, and when the command fails, such as for a configuration
//     file that does not parse or a managed file that was edited by hand
//   - 2 for a command line that ergon refuses, such as one with an unknown command or flag, after
//     which Run points at the help of the command
func Run(ctx context.Context, p Process, register func(*language.Catalog) error, v Version) int {
	root, err := command(p, register, v)
	ran := root
	if err == nil {
		// cobra reads the arguments of the process for nil arguments, so the slice is never nil.
		root.SetArgs(append([]string{}, p.Args...))
		root.SetOut(p.Stdout)
		root.SetErr(p.Stderr)
		ran, err = root.ExecuteContextC(ctx)
	}
	if err == nil {
		return statusOK
	}
	fmt.Fprintf(p.Stderr, "%s: %v\n", program, err)
	if _, ok := errors.AsType[usageError](err); ok {
		fmt.Fprintf(p.Stderr, "Run '%s --%s' for usage.\n", ran.CommandPath(), helpFlag)
		return statusUsage
	}
	return statusFailure
}

// command returns the root command of ergon. It registers the languages with register into a
// catalog, which the help lists and the commands use, and returns the error of register. v is the
// version of the running ergon.
//
// Before a command runs, the root command resolves the working directory with p.Getwd and reads
// the configuration with [load]. ergon init and its subcommands resolve the directory alone,
// because ergon init writes the configuration and never reads it. Without a subcommand, the root
// command writes the help. cobra adds the command help, and the command completion, which writes
// the completion script of a shell.
func command(p Process, register func(*language.Catalog) error, v Version) (*cobra.Command, error) {
	s := &session{getwd: p.Getwd, catalog: new(language.Catalog), release: v.Release, now: p.Now}
	if err := register(s.catalog); err != nil {
		return nil, err
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
		Long:          fmt.Sprintf(long, strings.Join(names, ", ")),
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
	root.AddCommand(initCommand(s, names))
	return root, nil
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
