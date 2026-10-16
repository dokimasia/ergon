// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
	gocli "go.stealthscale.io/cli"
)

// The help texts of the init commands. The first line of each is its summary in the list of the
// commands of its parent.
const (
	initHelp = `Set up a repository with the baseline of ergon.

ergon init sets up a repository with the baseline of ergon. The baseline
consists of the common files, the GitHub files and the license files of every
repository, and the files of each language of the repository. ergon init
records its answers, the baseline value of each option and the digest of each
managed file in .ergon/init.lock. Commit the lock with the files.

ergon init treats a file by its class:

  - A managed file is the rendering of ergon. ergon init check reports a
    managed file that differs from the rendering. To add a setting of the
    repository to the managed file at a path, write the setting to the same
    path under .ergon/local. ergon merges that local file into the rendering.
  - A seeded file, such as README.md, is written when it is absent. The
    repository maintains it from then on.
  - ergon init writes the keys of its answers into .ergon.yaml, and a section
    for each language with options, such as the versions of its tools. The
    repository changes an option there, and ergon init renders the managed
    files from it. The flags of ergon init sync change the key of an answer,
    such as license.spdx, and every command fails while such a key differs
    from the lock. ergon init keeps every other key.`

	newHelp = `Write the baseline into a repository without a lock.

ergon init new writes the baseline into a repository that has no lock: the
managed files, the seeded files that are absent, the keys and the options of
.ergon.yaml, and .ergon/init.lock. It prints the path of each file that it
writes.

When a managed file exists with other content, the command fails and does not
write a file. With --force, it overwrites such a file.

--name defaults to the name of the working directory, and --year to the
current year. The languages are:

%s

Example:
  ergon init new --language go --owner "Example B.V." --license MIT \
    --repository example/demo --security-contact security@example.com`

	addHelp = `Add languages to the repository.

ergon init add adds languages to the answers of .ergon/init.lock. It writes
the files of each language, its fragments of the shared files and its section
of .ergon.yaml. It prints the path of each file that it writes.

When a file that the command changes was edited by hand, the command fails and
does not write a file. With --force, it overwrites such a file.

Example:
  ergon init add typescript`

	removeHelp = `Remove languages from the repository.

ergon init remove removes languages from the answers of .ergon/init.lock. It
removes the files of each language and its section of .ergon.yaml, and
rewrites the shared files. It prints the path of each file that it writes or
removes.

When a file that the command changes or removes was edited by hand, the
command fails and does not change a file. Move the edit into the local file of
the path, and run ergon init sync --force first.

Example:
  ergon init remove typescript`

	checkHelp = `Report the managed files that differ from the baseline.

ergon init check compares the managed files with the rendering of this ergon
for the answers of .ergon/init.lock and the options of .ergon.yaml. It does
not write a file. It prints the problem and the path of each managed file that
is missing, edited by hand or outdated. The exit status is 1 when it prints a
file, and when .ergon.yaml has an option that its section does not accept.

Example:
  ergon init check --json`

	syncHelp = `Bring the managed files to the baseline of this ergon.

ergon init sync brings the managed files to the rendering of this ergon. It
writes each missing and outdated file and .ergon/init.lock, and removes each
file that the rendering no longer contains. It prints the path of each file
that it writes or removes.

A flag of ergon init sync changes its answer in the lock. An answer without a
flag keeps its value. An option of .ergon.yaml that still has the baseline
value of the lock moves to the baseline value of this ergon, and an option
that the repository changed keeps its value.

The command does not change a file that was edited by hand. It writes every
other file, and then exits with the status 1. With --force, it overwrites such
a file.

Example:
  ergon init sync --owner "Example B.V."`
)

// newOptions are the options of ergon init new: the answers, the languages and --force.
type newOptions struct {
	Name            string               `help:"Name of the repository."`
	Owner           string               `help:"Copyright holder, such as \"Example B.V.\"."                     required:""`
	License         spdx.ID              `help:"License, as an SPDX identifier that ergon license --help lists." required:""`
	Repository      language.Repository  `help:"Repository on GitHub, as owner/name."                            required:""`
	SecurityContact string               `help:"Address that receives reports of vulnerabilities."               required:""`
	Languages       []workspace.Language `help:"Language of the repository, once for each language."                         name:"language"`
	Year            int                  `help:"Year of the copyright notice."`
	Force           bool                 `help:"Overwrite the managed files that exist with other content."`
}

// languagesOptions are the options of ergon init remove: the languages of its arguments.
type languagesOptions struct {
	Languages []workspace.Language `help:"Language of the repository." arg:"" name:"language"`
}

// addOptions are the options of ergon init add: the languages of its arguments and --force.
type addOptions struct {
	languagesOptions

	Force bool `help:"Overwrite the managed files that were edited by hand."`
}

// checkOptions are the options of ergon init check.
type checkOptions struct {
	JSON bool `help:"Write the findings as a JSON array."`
}

// syncOptions are the options of ergon init sync: the answers and --force. An answer is nil when
// no source sets it, and the lock keeps its value then.
type syncOptions struct {
	Name            *string              `help:"Name of the repository."`
	Owner           *string              `help:"Copyright holder, such as \"Example B.V.\"."`
	License         *spdx.ID             `help:"License, as an SPDX identifier that ergon license --help lists."`
	Year            *int                 `help:"Year of the copyright notice."`
	Repository      *language.Repository `help:"Repository on GitHub, as owner/name."`
	SecurityContact *string              `help:"Address that receives reports of vulnerabilities."`
	Force           bool                 `help:"Overwrite the managed files that were edited by hand."`
}

// apply sets each answer of a that o sets, to the value of o. It leaves every other answer of a.
func (o *syncOptions) apply(a *language.Answers) {
	if o.Name != nil {
		a.Name = *o.Name
	}
	if o.Owner != nil {
		a.Owner = *o.Owner
	}
	if o.License != nil {
		a.License = *o.License
	}
	if o.Year != nil {
		a.Year = *o.Year
	}
	if o.Repository != nil {
		a.Repository = *o.Repository
	}
	if o.SecurityContact != nil {
		a.SecurityContact = *o.SecurityContact
	}
}

// initCommands adds ergon init to app, with its subcommands new, add, remove, check, sync,
// upgrade and ci upgrade, which work on the repository in the working directory of s. names are the
// names of the languages of the catalog of s.
//
// The subcommands read the options of .ergon.yaml through [baseline.Repository], so in a repository
// whose .ergon.yaml does not parse every subcommand returns the parse error of the file.
func (s *session) initCommands(app *gocli.App[noOptions], names []string) {
	parent := app.Command[noOptions]("init", initHelp, nil)
	parent.Command("new", fmt.Sprintf(newHelp, list(names)), s.initNew)
	parent.Command("add", addHelp, s.initAdd)
	parent.Command("remove", removeHelp, s.initRemove)
	parent.Command("check", checkHelp, s.initCheck)
	parent.Command("sync", syncHelp, s.initSync)
	parent.Command("upgrade", upgradeHelp, s.initUpgrade)
	ci := parent.Command[noOptions]("ci", initCIHelp, nil)
	ci.Command("upgrade", ciUpgradeHelp, s.ciUpgrade)
}

// initNew writes the baseline into the repository of s with [baseline.Repository.New]. The name
// defaults to the name of the working directory, and the year to the year of s.now.
func (s *session) initNew(_ context.Context, run *gocli.Run[noOptions, newOptions]) error {
	o := run.Config()
	a := language.Answers{
		Name:            cmp.Or(o.Name, filepath.Base(s.dir)),
		Languages:       o.Languages,
		Owner:           o.Owner,
		License:         o.License,
		Year:            cmp.Or(o.Year, s.now().Year()),
		Repository:      o.Repository,
		SecurityContact: o.SecurityContact,
	}
	return s.change(func(r *baseline.Repository) ([]baseline.Change, error) {
		return r.New(&a, baseline.Options{Force: o.Force})
	})
}

// initAdd adds the languages of its arguments to the repository of s with
// [baseline.Repository.Add].
func (s *session) initAdd(_ context.Context, run *gocli.Run[noOptions, addOptions]) error {
	o := run.Config()
	return s.change(func(r *baseline.Repository) ([]baseline.Change, error) {
		return r.Add(o.Languages, baseline.Options{Force: o.Force})
	})
}

// initRemove removes the languages of its arguments from the repository of s with
// [baseline.Repository.Remove].
func (s *session) initRemove(_ context.Context, run *gocli.Run[noOptions, languagesOptions]) error {
	return s.change(func(r *baseline.Repository) ([]baseline.Change, error) {
		return r.Remove(run.Config().Languages)
	})
}

// initCheck writes the findings of [baseline.Repository.Check] for the repository of s: a line of
// the problem and the path of each finding, or a JSON array with --json. It returns an error when it
// finds a managed file that differs from the baseline, and when the JSON array cannot be written.
func (s *session) initCheck(_ context.Context, run *gocli.Run[noOptions, checkOptions]) error {
	return s.open(s.dir, func(r *baseline.Repository) error {
		findings, err := r.Check()
		if err != nil {
			return err
		}
		if run.Config().JSON {
			// A repository at the baseline writes an empty array, not null.
			if err := json.NewEncoder(s.stdout).Encode(append([]baseline.Finding{}, findings...)); err != nil {
				return fmt.Errorf("cli: write the findings: %w", err)
			}
		} else {
			for _, f := range findings {
				fmt.Fprintf(s.stdout, "%s %s\n", f.Problem, f.Path)
			}
		}
		if len(findings) > 0 {
			return errors.New("cli: the repository differs from the baseline")
		}
		return nil
	})
}

// initSync brings the repository of s to the baseline with [baseline.Repository.Sync]. An answer
// that a source sets changes that answer of the lock.
func (s *session) initSync(_ context.Context, run *gocli.Run[noOptions, syncOptions]) error {
	o := run.Config()
	return s.change(func(r *baseline.Repository) ([]baseline.Change, error) {
		return r.Sync(o.apply, baseline.Options{Force: o.Force})
	})
}

// change opens the repository in the working directory of s, runs apply on it, and writes a line to
// the standard output of s for each file that apply wrote or removed: the action and the path. It
// writes the lines of the files that apply changed before it failed too, and returns the error of
// apply.
func (s *session) change(apply func(*baseline.Repository) ([]baseline.Change, error)) error {
	return s.open(s.dir, func(r *baseline.Repository) error {
		changes, err := apply(r)
		for _, c := range changes {
			fmt.Fprintf(s.stdout, "%s %s\n", c.Action, c.Path)
		}
		return err
	})
}
