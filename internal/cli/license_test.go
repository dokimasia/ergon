// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/service/licenses"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
	"go.stealthscale.io/cli/clitest"
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

// licenseRuns are the command lines of the transcript of ergon license.
var licenseRuns = []clitest.Input{
	{Args: []string{"license", "--help"}},
	{Args: []string{"license"}},
	{Args: []string{"license", "check", "--help"}},
	{Args: []string{"license", "fix", "--help"}},
}

func TestLicense(t *testing.T) {
	t.Parallel()

	t.Run("Main", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "ergon", program(t.TempDir(), register).Main, licenseRuns...)
			golden.Match(t, "license.txt", got, golden.ShouldUpdate())
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
			var report licenses.Report
			assert.NoError(t, json.Unmarshal([]byte(stdout), &report), "Unmarshal of the report")
			assert.Equal(t, report.Findings, []licenses.Finding{{Path: conflictFile, Kind: licenses.Conflict, Line: 2}},
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
			res := clitest.Main(t.Context(), program(licensed(t, nil), register).Main, &clitest.Input{
				Args: []string{"license", "check", "--json"}, Vars: environ(), FailStdout: true,
			})
			assert.Equal(t, res.Status, statusFailure, "the exit status")
			assert.Equal(t, res.Stderr, "ergon: cli: write the report: "+clitest.ErrStdout.Error()+"\n",
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
			assert.HasPrefix(t, stderr, "ergon: licenses: write "+missingFile+": ", "the standard error")
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
