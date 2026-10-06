// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/common"
	"go.dokimi.dev/ergon/service/baseline/github"
)

// The flags of the init commands.
const (
	nameFlag       = "name"
	languageFlag   = "language"
	ownerFlag      = "owner"
	licenseFlag    = "license"
	yearFlag       = "year"
	repositoryFlag = "repository"
	contactFlag    = "security-contact"
	forceFlag      = "force"
	jsonFlag       = "json"
)

// The help texts of the init commands.
const (
	initShort = "Set up a repository with the baseline of ergon"
	initLong  = `ergon init sets up a repository with the baseline of ergon. The baseline
consists of the common files and the GitHub files of every repository, and the
files of each language of the repository. ergon init records its answers and
the digest of each managed file in .ergon/init.lock. Commit the lock with the
files.

ergon init treats a file by its class:

  - A managed file is the rendering of ergon. ergon init check reports a
    managed file that differs from the rendering. To add a setting of the
    repository to the managed file at a path, write the setting to the same
    path under .ergon/local. ergon merges that local file into the rendering.
  - A seeded file, such as README.md, is written when it is absent. The
    repository maintains it from then on.
  - ergon init writes the keys of its answers into .ergon.yaml and keeps every
    other key.`

	newShort = "Write the baseline into a repository without a lock"
	newLong  = `ergon init new writes the baseline into a repository that has no lock: the
managed files, the seeded files that are absent, the keys of .ergon.yaml, and
.ergon/init.lock. It prints the path of each file that it writes.

When a managed file exists with other content, the command fails and does not
write a file. With --force, it overwrites such a file.

--name defaults to the name of the working directory, and --year to the
current year. The languages are:

  %s`
	newExample = `  ergon init new --language go --owner "Example B.V." --license MIT \
    --repository example/demo --security-contact security@example.com`

	addShort = "Add languages to the repository"
	addLong  = `ergon init add adds languages to the answers of .ergon/init.lock. It writes
the files of each language and its fragments of the shared files. It prints
the path of each file that it writes.

When a file that the command changes was edited by hand, the command fails and
does not write a file. With --force, it overwrites such a file.`
	addExample = "  ergon init add typescript"

	removeShort = "Remove languages from the repository"
	removeLong  = `ergon init remove removes languages from the answers of .ergon/init.lock. It
removes the files of each language and rewrites the shared files. It prints
the path of each file that it writes or removes.

When a file that the command changes or removes was edited by hand, the
command fails and does not change a file. Move the edit into the local file of
the path, and run ergon init sync --force first.`
	removeExample = "  ergon init remove typescript"

	checkShort = "Report the managed files that differ from the baseline"
	checkLong  = `ergon init check compares the managed files with the rendering of this ergon
for the answers of .ergon/init.lock. It does not write a file. It prints the
problem and the path of each managed file that is missing, edited by hand or
outdated. The exit status is 1 when it prints a file.`
	checkExample = "  ergon init check --json"

	syncShort = "Bring the managed files to the baseline of this ergon"
	syncLong  = `ergon init sync brings the managed files to the rendering of this ergon. It
writes each missing and outdated file and .ergon/init.lock, and removes each
file that the rendering no longer contains. It prints the path of each file
that it writes or removes.

A flag of ergon init sync changes its answer in the lock. An answer without a
flag keeps its value.

The command does not change a file that was edited by hand. It writes every
other file, and then exits with the status 1. With --force, it overwrites such
a file.`
	syncExample = `  ergon init sync --owner "Example B.V."`
)

// answers are the values of the flags of the answers, which init new and init sync define.
type answers struct {
	// name, owner, license, repository and contact are the values of --name, --owner, --license,
	// --repository and --security-contact.
	name, owner, license, repository, contact string

	// year is the value of --year.
	year int
}

// define defines the flags of the answers on cmd. year is the default of --year, and 0 for a flag
// without a default.
func (a *answers) define(cmd *cobra.Command, year int) {
	flags := cmd.Flags()
	flags.StringVar(&a.name, nameFlag, "", "the `name` of the repository")
	flags.StringVar(&a.owner, ownerFlag, "", "the copyright `holder`, such as \"Example B.V.\"")
	flags.StringVar(&a.license, licenseFlag, "",
		"the license, as the SPDX `identifier` "+common.MIT+" or "+common.Apache)
	flags.IntVar(&a.year, yearFlag, year, "the `year` of the copyright notice")
	flags.StringVar(&a.repository, repositoryFlag, "", "the repository on GitHub, as `owner/name`")
	flags.StringVar(&a.contact, contactFlag, "", "the `address` that receives reports of vulnerabilities")
}

// apply sets each answer of to whose flag the command line of cmd sets, to the value of the flag.
// It leaves every other answer of to.
func (a *answers) apply(cmd *cobra.Command, to *language.Answers) {
	flags := cmd.Flags()
	if flags.Changed(nameFlag) {
		to.Name = a.name
	}
	if flags.Changed(ownerFlag) {
		to.Owner = a.owner
	}
	if flags.Changed(licenseFlag) {
		to.License = a.license
	}
	if flags.Changed(yearFlag) {
		to.Year = a.year
	}
	if flags.Changed(repositoryFlag) {
		to.Repository = language.Repository(a.repository)
	}
	if flags.Changed(contactFlag) {
		to.SecurityContact = a.contact
	}
}

// initCommand returns ergon init with its subcommands, which work on the repository in the working
// directory of s. names are the names of the languages of the catalog of s.
//
// ergon init and its subcommands resolve the working directory and read no configuration, because
// they write .ergon.yaml. In a repository whose .ergon.yaml does not parse, check runs as usual,
// and every command that writes returns the parse error of the file. ergon init without a
// subcommand returns a [usageError] that lists the subcommands.
func initCommand(s *session, names []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: initShort,
		Long:  initLong,
		Args:  usage(cobra.NoArgs),
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return s.resolve()
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			commands := cmd.Commands()
			subcommands := make([]string, 0, len(commands))
			for _, sub := range commands {
				subcommands = append(subcommands, sub.Name())
			}
			err := fmt.Errorf("cli: %s needs a subcommand: %s", cmd.Name(), strings.Join(subcommands, ", "))
			return usageError{err: err}
		},
	}
	cmd.AddCommand(newCommand(s, names), addCommand(s), removeCommand(s), checkCommand(s), syncCommand(s))
	return cmd
}

// newCommand returns ergon init new, which writes the baseline into the repository of s with
// [baseline.Repository.New]. names are the names of the languages that --language takes.
//
// --owner, --license, --repository and --security-contact are required, and a command line
// without one of them returns a [usageError]. --name defaults to the name of the working
// directory, and --year to the year of s.now.
func newCommand(s *session, names []string) *cobra.Command {
	var given answers
	var languages []string
	var force bool
	cmd := &cobra.Command{
		Use:     "new",
		Short:   newShort,
		Long:    fmt.Sprintf(newLong, strings.Join(names, ", ")),
		Example: newExample,
		Args:    usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, name := range []string{ownerFlag, licenseFlag, repositoryFlag, contactFlag} {
				if !cmd.Flags().Changed(name) {
					return usageError{err: fmt.Errorf("cli: the flag --%s is required", name)}
				}
			}
			a := language.Answers{Name: filepath.Base(s.dir), Languages: asLanguages(languages), Year: given.year}
			given.apply(cmd, &a)
			return s.change(cmd.OutOrStdout(), func(r *baseline.Repository) ([]baseline.Change, error) {
				return r.New(&a, baseline.Options{Force: force})
			})
		},
	}
	given.define(cmd, s.now().Year())
	cmd.Flags().StringSliceVar(&languages, languageFlag, nil,
		"a `language` of the repository, once for each language")
	cmd.Flags().BoolVar(&force, forceFlag, false, "overwrite the managed files that exist with other content")
	return cmd
}

// addCommand returns ergon init add, which adds the languages of its arguments to the repository
// of s with [baseline.Repository.Add].
func addCommand(s *session) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "add <language>...",
		Short:   addShort,
		Long:    addLong,
		Example: addExample,
		Args:    usage(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.change(cmd.OutOrStdout(), func(r *baseline.Repository) ([]baseline.Change, error) {
				return r.Add(asLanguages(args), baseline.Options{Force: force})
			})
		},
	}
	cmd.Flags().BoolVar(&force, forceFlag, false, "overwrite the managed files that were edited by hand")
	return cmd
}

// removeCommand returns ergon init remove, which removes the languages of its arguments from the
// repository of s with [baseline.Repository.Remove].
func removeCommand(s *session) *cobra.Command {
	return &cobra.Command{
		Use:     "remove <language>...",
		Short:   removeShort,
		Long:    removeLong,
		Example: removeExample,
		Args:    usage(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.change(cmd.OutOrStdout(), func(r *baseline.Repository) ([]baseline.Change, error) {
				return r.Remove(asLanguages(args))
			})
		},
	}
}

// checkCommand returns ergon init check, which writes the findings of [baseline.Repository.Check]
// for the repository of s: a line of the problem and the path for each, or a JSON array with
// --json. It returns an error when it finds a managed file that differs from the baseline, and
// when the JSON array cannot be written.
func checkCommand(s *session) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "check",
		Short:   checkShort,
		Long:    checkLong,
		Example: checkExample,
		Args:    usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return s.open(func(r *baseline.Repository) error {
				findings, err := r.Check()
				if err != nil {
					return err
				}
				out := cmd.OutOrStdout()
				if asJSON {
					// A repository at the baseline writes an empty array, not null.
					if err := json.NewEncoder(out).Encode(append([]baseline.Finding{}, findings...)); err != nil {
						return fmt.Errorf("cli: write the findings: %w", err)
					}
				} else {
					for _, f := range findings {
						fmt.Fprintf(out, "%s %s\n", f.Problem, f.Path)
					}
				}
				if len(findings) > 0 {
					return errors.New("cli: the repository differs from the baseline")
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&asJSON, jsonFlag, false, "write the findings as a JSON array")
	return cmd
}

// syncCommand returns ergon init sync, which brings the repository of s to the baseline with
// [baseline.Repository.Sync]. A flag of an answer changes that answer of the lock.
func syncCommand(s *session) *cobra.Command {
	var given answers
	var force bool
	cmd := &cobra.Command{
		Use:     "sync",
		Short:   syncShort,
		Long:    syncLong,
		Example: syncExample,
		Args:    usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return s.change(cmd.OutOrStdout(), func(r *baseline.Repository) ([]baseline.Change, error) {
				return r.Sync(func(a *language.Answers) { given.apply(cmd, a) }, baseline.Options{Force: force})
			})
		},
	}
	given.define(cmd, 0)
	cmd.Flags().BoolVar(&force, forceFlag, false, "overwrite the managed files that were edited by hand")
	return cmd
}

// asLanguages returns names as the names of languages, in their order.
func asLanguages(names []string) []workspace.Language {
	languages := make([]workspace.Language, 0, len(names))
	for _, name := range names {
		languages = append(languages, workspace.Language(name))
	}
	return languages
}

// open opens the repository in the working directory of s, with the common files and the GitHub
// files before the languages of the catalog of s, runs run on it, and closes it. It returns the
// errors of the open, of run and of the close, joined.
func (s *session) open(run func(*baseline.Repository) error) error {
	root, err := os.OpenRoot(s.dir)
	if err != nil {
		return fmt.Errorf("cli: open the repository: %w", err)
	}
	r, err := baseline.Open(root, s.catalog, s.release,
		baseline.Producer{Name: common.Name, Initializer: common.Initializer{}},
		baseline.Producer{Name: github.Name, Initializer: github.Initializer{}})
	if err == nil {
		err = run(r)
	}
	return errors.Join(err, root.Close())
}

// change opens the repository of s, runs apply on it, and writes a line to w for each file that
// apply wrote or removed: the action and the path. It writes the lines of the files that apply
// changed before it failed too, and returns the error of apply.
func (s *session) change(w io.Writer, apply func(*baseline.Repository) ([]baseline.Change, error)) error {
	return s.open(func(r *baseline.Repository) error {
		changes, err := apply(r)
		for _, c := range changes {
			fmt.Fprintf(w, "%s %s\n", c.Action, c.Path)
		}
		return err
	})
}
