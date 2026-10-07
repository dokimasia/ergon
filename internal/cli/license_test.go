// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/internal/cli"
	"go.dokimi.dev/ergon/service/license"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The files of the cases of ergon license, one per kind of finding.
const (
	missingFile  = "missing.go"
	outdatedFile = "outdated.go"
	conflictFile = "conflict.go"
	binaryFile   = "binary.go"
)

// The contents of the files of the cases.
const (
	// headless is a file without a header.
	headless = "package main\n"

	// outdated is a file whose header states another owner and another license.
	outdated = "// Copyright Other B.V. 2020\n// SPDX-License-Identifier: Apache-2.0\n\npackage main\n"

	// conflicting is a file whose header has a line that is no copyright notice, no tag of SPDX and
	// not empty.
	conflicting = "// Copyright Other Inc. 2020\n// All rights reserved.\n\npackage main\n"

	// binary is a file whose content is not text.
	binary = "\x00\x01\x02\x03"
)

// counts matches the last line of a report of ergon license, which ends in a newline.
const counts = "checked [1-9][0-9]* files and skipped [1-9][0-9]*\n$"

// licenseHelp is the help of ergon license, pinned because a person reads it.
const licenseHelp = `ergon license adds and checks the license header of each file of the
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

  0BSD, AFL-3.0, AGPL-3.0-only, AGPL-3.0-or-later, Apache-2.0, Artistic-2.0,
  BlueOak-1.0.0, BSD-2-Clause, BSD-2-Clause-Patent, BSD-3-Clause,
  BSD-3-Clause-Clear, BSD-4-Clause, BSL-1.0, BUSL-1.1, CC0-1.0, CECILL-2.1,
  ECL-2.0, EPL-1.0, EPL-2.0, EUPL-1.1, EUPL-1.2, GPL-2.0-only, GPL-2.0-or-later,
  GPL-3.0-only, GPL-3.0-or-later, ISC, LGPL-2.1-only, LGPL-2.1-or-later,
  LGPL-3.0-only, LGPL-3.0-or-later, MIT, MIT-0, MPL-2.0, MS-PL, MS-RL,
  MulanPSL-2.0, NCSA, OSL-3.0, PostgreSQL, Unlicense, UPL-1.0, Vim, WTFPL, Zlib

Usage:
  ergon license [flags]
  ergon license [command]

Available Commands:
  check       Report the files whose license header differs from the configuration
  fix         Add the missing license headers and replace the outdated ones

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command

Use "ergon license [command] --help" for more information about a command.
`

// licenseCheckHelp is the help of ergon license check, pinned because a person reads it.
const licenseCheckHelp = `ergon license check reports each file whose license header is missing, states
another owner or license, or has a line that is neither a copyright notice nor
an SPDX tag nor empty. It does not write a file. It prints the problem and the
path of each such file, and then the number of files that it checked and
skipped. A header accepts any year, list of years or range of years. The exit
status is 1 when it reports a missing, outdated or conflicting header.

Usage:
  ergon license check [flags]

Examples:
  ergon license check --json

Flags:
      --json   write the report as JSON

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// licenseFixHelp is the help of ergon license fix, pinned because a person reads it.
const licenseFixHelp = `ergon license fix adds the license header to each file without one, with the
current year, and replaces each header that states another owner or license,
with its years. It keeps a preamble such as a shebang line above the header. It
prints the path of each file that it wrote, then the problem and the path of
each file that it leaves, and then the number of files that it checked and
skipped. The exit status is 1 when a file has a header with a line that is
neither a copyright notice nor an SPDX tag nor empty, which the command leaves.

Usage:
  ergon license fix [flags]

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

func TestLicense(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		help := []struct {
			name   string
			args   []string
			stdout string
		}{
			{name: "writes the help of license", args: []string{"license", "--help"}, stdout: licenseHelp},
			{
				name:   "writes the help of license check",
				args:   []string{"license", "check", "--help"},
				stdout: licenseCheckHelp,
			},
			{
				name:   "writes the help of license fix",
				args:   []string{"license", "fix", "--help"},
				stdout: licenseFixHelp,
			},
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

		t.Run("returns 2 for license without a subcommand", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, t.TempDir(), "license")
			assert.Equal(t, status, statusUsage, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: cli: license needs a subcommand: check, fix\n"+
				"Run 'ergon license --help' for usage.\n", "the standard error")
		})

		t.Run("license check writes the counts for a repository whose headers match", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, licensed(t, nil), "license", "check")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Matches(t, stdout, "^"+counts, "the standard output")
		})

		t.Run("license check writes each finding and returns 1", func(t *testing.T) {
			t.Parallel()
			dir := licensed(t, map[string]string{
				missingFile:  headless,
				outdatedFile: outdated,
				conflictFile: conflicting,
			})
			status, stdout, stderr := run(t, dir, "license", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Matches(t, stdout, "^conflict "+conflictFile+":2\nmissing "+missingFile+"\noutdated "+
				outdatedFile+"\n"+counts, "the standard output")
			assert.Equal(t, stderr, "ergon: cli: files have no license header of the section license\n",
				"the standard error")
		})

		t.Run("license check returns 0 for a file whose content is not text", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, licensed(t, map[string]string{binaryFile: binary}), "license", "check")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Matches(t, stdout, "^unsupported "+binaryFile+"\n"+counts, "the standard output")
		})

		t.Run("license check writes the report as JSON with --json", func(t *testing.T) {
			t.Parallel()
			dir := licensed(t, map[string]string{conflictFile: conflicting})
			status, stdout, _ := run(t, dir, "license", "check", "--json")
			assert.Equal(t, status, statusFailure, "the exit status")
			var report license.Report
			assert.NoError(t, json.Unmarshal([]byte(stdout), &report), "Unmarshal of the report")
			assert.Equal(t, report.Findings, []license.Finding{{Path: conflictFile, Kind: license.Conflict, Line: 2}},
				"the findings of the report")
		})

		t.Run("license check writes an empty array of findings with --json", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, licensed(t, nil), "license", "check", "--json")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Matches(t, stdout, `^\{"findings":\[\],"checked":[1-9][0-9]*,"skipped":[1-9][0-9]*\}`+"\n$",
				"the standard output")
		})

		t.Run("license check returns 1 when the JSON cannot be written", func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			p := process(licensed(t, nil), failing{}, &stderr, "license", "check", "--json")
			assert.Equal(t, cli.Run(t.Context(), p, register, version), statusFailure, "the exit status")
			assert.Equal(t, stderr.String(), "ergon: cli: write the report: "+errWrite.Error()+"\n",
				"the standard error")
		})

		t.Run("license check returns 1 outside a working tree of git", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, initialized(t), "license", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.HasPrefix(t, stderr, "ergon: vcs: ", "the standard error")
		})

		t.Run("license check returns 1 for a .ergon.yaml that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := licensed(t, nil)
			write(t, dir, ".ergon.yaml", "license: [\n")
			status, _, stderr := run(t, dir, "license", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: options: invalid .ergon.yaml: ", "the standard error")
		})

		t.Run("license check returns 1 for a license.spdx that differs from the lock", func(t *testing.T) {
			t.Parallel()
			dir := licensed(t, nil)
			write(t, dir, ".ergon.yaml", "license:\n  spdx: Apache-2.0\n")
			status, stdout, stderr := run(t, dir, "license", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, `ergon: options: invalid .ergon.yaml: license.spdx "Apache-2.0", which ergon init `+
				`sync --license sets, differs from the answer "MIT" of ergon init`+"\n", "the standard error")
		})

		t.Run("license fix writes the header of each missing and outdated file", func(t *testing.T) {
			t.Parallel()
			dir := licensed(t, map[string]string{missingFile: headless, outdatedFile: outdated})
			status, stdout, stderr := run(t, dir, "license", "fix")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Matches(t, stdout, "^fixed "+missingFile+"\nfixed "+outdatedFile+"\n"+counts, "the standard output")
			files.HasContent(t, filepath.Join(dir, missingFile),
				"// Copyright "+owner+" 2026\n// SPDX-License-Identifier: MIT\n\npackage main\n", "the missing file")
			files.HasContent(t, filepath.Join(dir, outdatedFile),
				"// Copyright "+owner+" 2020\n// SPDX-License-Identifier: MIT\n\npackage main\n", "the outdated file")
		})

		t.Run("license fix leaves a conflicting file and returns 1", func(t *testing.T) {
			t.Parallel()
			dir := licensed(t, map[string]string{conflictFile: conflicting})
			status, stdout, stderr := run(t, dir, "license", "fix")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Matches(t, stdout, "^conflict "+conflictFile+":2\n"+counts, "the standard output")
			assert.Equal(t, stderr, "ergon: cli: files have a conflicting license header, which ergon license fix "+
				"leaves\n", "the standard error")
			files.HasContent(t, filepath.Join(dir, conflictFile), conflicting, "the conflicting file")
		})

		t.Run("license fix returns 1 for a file that does not write", func(t *testing.T) {
			t.Parallel()
			dir := licensed(t, map[string]string{missingFile: headless})
			locked := filepath.Join(dir, missingFile)
			assert.NoError(t, os.Chmod(locked, 0o444), "Chmod of the file")
			t.Cleanup(func() { _ = os.Chmod(locked, 0o644) })
			status, _, stderr := run(t, dir, "license", "fix")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: license: write "+missingFile+": ", "the standard error")
		})
	})
}

// licensed returns a new working tree of git after init new with the answers of the cases and
// license fix, with the files of extra, by their slash-separated paths, which license fix has not
// seen.
func licensed(t *testing.T, extra map[string]string) string {
	t.Helper()
	dir := vcstest.Repository(t, files.Tree{})
	args := slices.Concat([]string{"init", "new", "--language", string(alpha)}, required)
	status, _, stderr := run(t, dir, args...)
	assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
	status, _, stderr = run(t, dir, "license", "fix")
	assert.Equal(t, status, statusOK, "the exit status of license fix: "+stderr)
	for path, content := range extra {
		assert.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0o755), "MkdirAll of "+path)
		write(t, dir, path, content)
	}
	return dir
}
