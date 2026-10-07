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

	"github.com/spf13/cobra"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/license"
	licensefiles "go.dokimi.dev/ergon/service/license/baseline"
)

// The help texts of the license commands.
const (
	licenseShort = "Add and check the license header of every file"
	licenseLong  = `ergon license adds and checks the license header of each file of the
repository that has a comment syntax: the copyright of the owner and the SPDX
identifier of the license, as the section license of .ergon.yaml states them.
It reads the files that git tracks or would track, in the repository of the
working directory or of the nearest of its parents with .ergon/init.lock. It
skips the files that the section excludes, the files that a tool generated or
that ergon init manages, and the files without a comment syntax.

A header states the owner and the license, such as:

  // Copyright Example B.V. 2026
  // SPDX-License-Identifier: MIT

The licenses are:

%s`

	licenseCheckShort = "Report the files whose license header differs from the configuration"
	licenseCheckLong  = `ergon license check reports each file whose license header is missing, states
another owner or license, or has a line that is neither a copyright notice nor
an SPDX tag nor empty. It does not write a file. It prints the problem and the
path of each such file, and then the number of files that it checked and
skipped. A header accepts any year, list of years or range of years. The exit
status is 1 when it reports a missing, outdated or conflicting header.`
	licenseCheckExample = "  ergon license check --json"

	licenseFixShort = "Add the missing license headers and replace the outdated ones"
	licenseFixLong  = `ergon license fix adds the license header to each file without one, with the
current year, and replaces each header that states another owner or license,
with its years. It keeps a preamble such as a shebang line above the header. It
prints the path of each file that it wrote, then the problem and the path of
each file that it leaves, and then the number of files that it checked and
skipped. The exit status is 1 when a file has a header with a line that is
neither a copyright notice nor an SPDX tag nor empty, which the command leaves.`
)

// licenseCommand returns ergon license with its subcommands, which work on the repository of the
// working directory of s under ctx.
func licenseCommand(ctx context.Context, s *session) *cobra.Command {
	ids := slices.Collect(spdx.IDs())
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, string(id))
	}
	return group(s, "license", licenseShort, fmt.Sprintf(licenseLong, list(names)),
		licenseCheckCommand(ctx, s), licenseFixCommand(ctx, s))
}

// licenseCheckCommand returns ergon license check, which writes the report of [license.Check]
// under ctx for the repository of s: a line of the kind and the path of each finding, and a line
// of the counts, or the report as JSON with --json. It returns an error for a finding that is not
// of the kind unsupported, and when the JSON cannot be written.
func licenseCheckCommand(ctx context.Context, s *session) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "check",
		Short:   licenseCheckShort,
		Long:    licenseCheckLong,
		Example: licenseCheckExample,
		Args:    usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return s.license(func(dir string, c *license.Config) error {
				report, err := license.Check(ctx, dir, c)
				if err != nil {
					return err
				}
				out := cmd.OutOrStdout()
				if asJSON {
					// A repository whose headers match writes an empty array, not null.
					report.Findings = append([]license.Finding{}, report.Findings...)
					if err := json.NewEncoder(out).Encode(report); err != nil {
						return fmt.Errorf("cli: write the report: %w", err)
					}
				} else {
					write(out, &report)
				}
				failing := func(f license.Finding) bool { return f.Kind != license.Unsupported }
				if slices.ContainsFunc(report.Findings, failing) {
					return errors.New("cli: files have no license header of the section license")
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&asJSON, jsonFlag, false, "write the report as JSON")
	return cmd
}

// licenseFixCommand returns ergon license fix, which fixes the headers of the repository of s with
// [license.Fix] under ctx, with the year of s.now, and writes its report: a line of each file that
// it wrote, a line of the kind and the path of each file that it leaves, and a line of the counts.
// It returns an error for a file with a conflicting header.
func licenseFixCommand(ctx context.Context, s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "fix",
		Short: licenseFixShort,
		Long:  licenseFixLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return s.license(func(dir string, c *license.Config) error {
				report, err := license.Fix(ctx, dir, c, s.now().Year())
				write(cmd.OutOrStdout(), &report)
				if err != nil {
					return err
				}
				conflicting := func(f license.Finding) bool { return f.Kind == license.Conflict }
				if slices.ContainsFunc(report.Findings, conflicting) {
					return errors.New("cli: files have a conflicting license header, which ergon license fix leaves")
				}
				return nil
			})
		},
	}
}

// license runs run with the directory of the repository of s and the options of its section
// license, as every command of ergon init resolves them.
func (s *session) license(run func(dir string, c *license.Config) error) error {
	dir := s.root()
	return s.open(dir, func(r *baseline.Repository) error {
		o, err := r.Options(licensefiles.Name)
		if err != nil {
			return err
		}
		// The producer of the license files has the options of the type *license.Config.
		c, _ := o.(*license.Config)
		return run(dir, c)
	})
}

// write writes the report r to w: a line fixed and the path of each file that ergon license fix
// wrote, a line of the kind and the path of each finding, with the line of a conflict, and a line
// of the counts of the files.
func write(w io.Writer, r *license.Report) {
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
