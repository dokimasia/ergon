// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

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
extension of its file.

Run 'ergon completion --help' for the completion scripts of bash, fish,
PowerShell and zsh.`
)

// usageError is an error of the command line, such as an unknown command or flag. [Run] returns
// statusUsage for it and points at the help.
type usageError struct {
	// err is the error that cobra or the flag parser returned.
	err error
}

// Error returns the text of the error that cobra or the flag parser returned.
func (e usageError) Error() string {
	return e.err.Error()
}

// Unwrap returns the error that cobra or the flag parser returned.
func (e usageError) Unwrap() error {
	return e.err
}

// Run runs the command line args and returns the exit status. args are the arguments that follow
// the name of the program. register fills the catalog whose languages the help lists. version is
// the text that --version writes after the name of the program. Every command receives ctx, and a
// caller that cancels ctx on a signal stops the running command. Run writes the help and the
// version to stdout. It writes every error to stderr, after the name of the program.
//
// The exit status is:
//
//   - 0 when the command succeeds
//   - 1 when register returns an error, and when the command fails, such as for a configuration
//     file that does not exist or does not parse
//   - 2 for a command line that ergon refuses, such as one with an unknown command or flag, after
//     which Run points at the help
func Run(
	ctx context.Context, args []string, register func(*language.Catalog) error, version string,
	stdout, stderr io.Writer,
) int {
	root, err := command(register, version)
	if err == nil {
		// cobra reads the arguments of the process for nil arguments, so the slice is never nil.
		root.SetArgs(append([]string{}, args...))
		root.SetOut(stdout)
		root.SetErr(stderr)
		err = root.ExecuteContext(ctx)
	}
	if err == nil {
		return statusOK
	}
	fmt.Fprintf(stderr, "%s: %v\n", program, err)
	if _, ok := errors.AsType[usageError](err); ok {
		fmt.Fprintf(stderr, "Run '%s --%s' for usage.\n", program, helpFlag)
		return statusUsage
	}
	return statusFailure
}

// command returns the root command of ergon. It registers the languages with register into a
// catalog, whose languages the help lists, and returns the error of register. version is what
// --version writes.
//
// Before any command runs, the root command reads the configuration with [load]. Without a
// subcommand, it writes the help. cobra adds the command completion, which writes the completion
// script of a shell, when the command line calls it.
func command(register func(*language.Catalog) error, version string) (*cobra.Command, error) {
	var catalog language.Catalog
	if err := register(&catalog); err != nil {
		return nil, err
	}
	var names []string
	for d := range catalog.Languages() {
		names = append(names, string(d.Name))
	}

	config := viper.New()
	var file string
	root := &cobra.Command{
		Use:     program,
		Short:   short,
		Long:    fmt.Sprintf(long, strings.Join(names, ", ")),
		Version: version,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return usageError{err: err}
			}
			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return load(config, file, cmd.Flags().Changed(configFlag))
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
	return root, nil
}

// load reads the configuration file into config as YAML. named reports whether the command line
// named file. A missing file is no error when the command line names none, because a repository
// without .ergon.yaml uses the defaults. load returns an error for a named file that does not
// exist, and for a file that does not parse.
func load(config *viper.Viper, file string, named bool) error {
	config.SetConfigFile(file)
	config.SetConfigType(configType)
	err := config.ReadInConfig()
	if err == nil || (!named && errors.Is(err, fs.ErrNotExist)) {
		return nil
	}
	return fmt.Errorf("cli: read %s: %w", file, err)
}
