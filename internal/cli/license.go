// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/licenses"
	licensefiles "go.dokimi.dev/ergon/service/licenses/baseline"
	gocli "go.stealthscale.io/cli"
)

// The help texts of the license commands. The first line of each is its summary in the list of the
// commands of its parent.
const (
	licenseHelp = `Add and check the license header of every file.

ergon license adds and checks the license header of each file of the
repository that has a comment syntax: the copyright of the owner and the SPDX
identifier of the license, as the section license of .ergon.yaml states them.
It reads the files that git tracks or would track, in the repository of the
working directory or of the nearest of its parents with .ergon/init.lock. It
skips the files that the section excludes, the changesets of .changeset, the
files that a tool generated or that ergon init manages, and the files without
a comment syntax.

A header states the owner and the license, such as:

  // Copyright Example B.V. 2026
  // SPDX-License-Identifier: MIT

The licenses are:

%s`

	licenseCheckHelp = `Report the files whose license header differs from the configuration.

ergon license check reports each file whose license header is missing, states
another owner or license, or has a line that is neither a copyright notice nor
an SPDX tag nor empty. It does not write a file. It prints the problem and the
path of each such file, and then the number of files that it checked and
skipped. A header accepts any year, list of years or range of years. The exit
status is 1 when it reports a missing, outdated or conflicting header.

Example:
  ergon license check --json`

	licenseFixHelp = `Add the missing license headers and replace the outdated ones.

ergon license fix adds the license header to each file without one, with the
current year, and replaces each header that states another owner or license,
with its years. It keeps a preamble such as a shebang line above the header. It
prints the path of each file that it wrote, then the problem and the path of
each file that it leaves, and then the number of files that it checked and
skipped. The exit status is 1 when a file has a header with a line that is
neither a copyright notice nor an SPDX tag nor empty, which the command leaves.`
)

// licenseCheckOptions are the options of ergon license check.
type licenseCheckOptions struct {
	JSON bool `help:"Write the report as JSON."`
}

// licenseCommands adds ergon license to app, with its subcommands check and fix, which work on the
// repository of the working directory of s.
func (s *session) licenseCommands(app *gocli.App[noOptions]) {
	ids := slices.Collect(spdx.IDs())
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, string(id))
	}
	parent := app.Command[noOptions]("license", fmt.Sprintf(licenseHelp, list(names)), nil)
	parent.Command("check", licenseCheckHelp, s.licenseCheck)
	parent.Command("fix", licenseFixHelp, s.licenseFix)
}

// licenseCheck writes the report of [licenses.Check] under ctx for the repository of s: a line of
// the kind and the path of each finding, and a line of the counts, or the report as JSON with
// --json. It returns an error for a finding that is not of the kind unsupported, and when the JSON
// cannot be written.
func (s *session) licenseCheck(ctx context.Context, run *gocli.Run[noOptions, licenseCheckOptions]) error {
	return s.license(func(dir string, c *licenses.Config) error {
		report, err := licenses.Check(ctx, dir, c)
		if err != nil {
			return err
		}
		if run.Config().JSON {
			// A repository whose headers match writes an empty array, not null.
			report.Findings = append([]licenses.Finding{}, report.Findings...)
			if err := json.NewEncoder(s.stdout).Encode(report); err != nil {
				return fmt.Errorf("cli: write the report: %w", err)
			}
		} else {
			write(s.stdout, &report)
		}
		failing := func(f licenses.Finding) bool { return f.Kind != licenses.Unsupported }
		if slices.ContainsFunc(report.Findings, failing) {
			return errors.New("cli: files have no license header of the section license")
		}
		return nil
	})
}

// licenseFix fixes the headers of the repository of s with [licenses.Fix] under ctx, with the year
// of s.now, and writes its report: a line of each file that it wrote, a line of the kind and the path
// of each file that it leaves, and a line of the counts. It returns an error for a file with a
// conflicting header.
func (s *session) licenseFix(ctx context.Context, _ *gocli.Run[noOptions, noOptions]) error {
	return s.license(func(dir string, c *licenses.Config) error {
		report, err := licenses.Fix(ctx, dir, c, s.now().Year())
		write(s.stdout, &report)
		if err != nil {
			return err
		}
		conflicting := func(f licenses.Finding) bool { return f.Kind == licenses.Conflict }
		if slices.ContainsFunc(report.Findings, conflicting) {
			return errors.New("cli: files have a conflicting license header, which ergon license fix leaves")
		}
		return nil
	})
}

// license runs run with the directory of the repository of s and the options of its section
// license, as every command of ergon init resolves them.
func (s *session) license(run func(dir string, c *licenses.Config) error) error {
	dir := s.root()
	return s.open(dir, func(r *baseline.Repository) error {
		o, err := r.Options(licensefiles.Name)
		if err != nil {
			return err
		}
		// The producer of the license files has the options of the type *license.Config.
		c, _ := o.(*licenses.Config)
		return run(dir, c)
	})
}

// write writes the report r to w: a line fixed and the path of each file that ergon license fix
// wrote, a line of the kind and the path of each finding, with the line of a conflict, and a line
// of the counts of the files.
func write(w io.Writer, r *licenses.Report) {
	for _, path := range r.Fixed {
		fmt.Fprintf(w, "fixed %s\n", path)
	}
	for _, f := range r.Findings {
		if f.Line > 0 {
			fmt.Fprintf(w, "%s %s:%d\n", f.Kind, f.Path, f.Line)
		} else {
			fmt.Fprintf(w, "%s %s\n", f.Kind, f.Path)
		}
	}
	fmt.Fprintf(w, "checked %d files and skipped %d\n", r.Checked, r.Skipped)
}
