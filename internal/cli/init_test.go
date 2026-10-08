// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/internal/app"
	"go.dokimi.dev/ergon/internal/cli"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.yaml.in/yaml/v3"
)

// The answers of init new in the cases, other than the name and the year.
const (
	owner      = "Example B.V."
	spdxID     = "MIT"
	repository = "example/demo"
	contact    = "security@example.com"
)

// The paths that the cases read and write.
const (
	lockPath    = ".ergon/init.lock"
	ignorePath  = ".gitignore"
	licensePath = "LICENSE"
)

// conflict is the start of the error of a command that leaves a managed file that was edited by
// hand or exists with other content.
const conflict = "ergon: baseline: managed files edited by hand or existing with other content: "

// initHelp is the help of ergon init, pinned because a person reads it.
const initHelp = `ergon init sets up a repository with the baseline of ergon. The baseline
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
    from the lock. ergon init keeps every other key.

Usage:
  ergon init [flags]
  ergon init [command]

Available Commands:
  add         Add languages to the repository
  check       Report the managed files that differ from the baseline
  new         Write the baseline into a repository without a lock
  remove      Remove languages from the repository
  sync        Bring the managed files to the baseline of this ergon

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command

Use "ergon init [command] --help" for more information about a command.
`

// newHelp is the help of ergon init new for a catalog of alpha and beta, pinned because a person
// reads it.
const newHelp = `ergon init new writes the baseline into a repository that has no lock: the
managed files, the seeded files that are absent, the keys and the options of
.ergon.yaml, and .ergon/init.lock. It prints the path of each file that it
writes.

When a managed file exists with other content, the command fails and does not
write a file. With --force, it overwrites such a file.

--name defaults to the name of the working directory, and --year to the
current year. The languages are:

  alpha, beta

Usage:
  ergon init new [flags]

Examples:
  ergon init new --language go --owner "Example B.V." --license MIT \
    --repository example/demo --security-contact security@example.com

Flags:
      --force                      overwrite the managed files that exist with other content
      --language language          a language of the repository, once for each language
      --license identifier         the license, as an SPDX identifier that ergon license --help lists
      --name name                  the name of the repository
      --owner holder               the copyright holder, such as "Example B.V."
      --repository owner/name      the repository on GitHub, as owner/name
      --security-contact address   the address that receives reports of vulnerabilities
      --year year                  the year of the copyright notice (default 2026)

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// addHelp is the help of ergon init add, pinned because a person reads it.
const addHelp = `ergon init add adds languages to the answers of .ergon/init.lock. It writes
the files of each language, its fragments of the shared files and its section
of .ergon.yaml. It prints the path of each file that it writes.

When a file that the command changes was edited by hand, the command fails and
does not write a file. With --force, it overwrites such a file.

Usage:
  ergon init add <language>... [flags]

Examples:
  ergon init add typescript

Flags:
      --force   overwrite the managed files that were edited by hand

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// removeHelp is the help of ergon init remove, pinned because a person reads it.
const removeHelp = `ergon init remove removes languages from the answers of .ergon/init.lock. It
removes the files of each language and its section of .ergon.yaml, and
rewrites the shared files. It prints the path of each file that it writes or
removes.

When a file that the command changes or removes was edited by hand, the
command fails and does not change a file. Move the edit into the local file of
the path, and run ergon init sync --force first.

Usage:
  ergon init remove <language>... [flags]

Examples:
  ergon init remove typescript

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// checkHelp is the help of ergon init check, pinned because a person reads it.
const checkHelp = `ergon init check compares the managed files with the rendering of this ergon
for the answers of .ergon/init.lock and the options of .ergon.yaml. It does
not write a file. It prints the problem and the path of each managed file that
is missing, edited by hand or outdated. The exit status is 1 when it prints a
file, and when .ergon.yaml has an option that its section does not accept.

Usage:
  ergon init check [flags]

Examples:
  ergon init check --json

Flags:
      --json   write the findings as a JSON array

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// syncHelp is the help of ergon init sync, pinned because a person reads it.
const syncHelp = `ergon init sync brings the managed files to the rendering of this ergon. It
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

Usage:
  ergon init sync [flags]

Examples:
  ergon init sync --owner "Example B.V."

Flags:
      --force                      overwrite the managed files that were edited by hand
      --license identifier         the license, as an SPDX identifier that ergon license --help lists
      --name name                  the name of the repository
      --owner holder               the copyright holder, such as "Example B.V."
      --repository owner/name      the repository on GitHub, as owner/name
      --security-contact address   the address that receives reports of vulnerabilities
      --year year                  the year of the copyright notice

Global Flags:
      --config file   read the configuration from file (default ".ergon.yaml")
  -h, --help          show the help of the command
`

// required are the flags of init new without a default, with the values of the cases.
var required = []string{
	"--owner", owner,
	"--license", spdxID,
	"--repository", repository,
	"--security-contact", contact,
}

// errWrite is the error of a writer that fails.
var errWrite = errors.New("write: failed")

// uses matches a line that runs an action, and captures the reference of the action with its
// comment.
var uses = regexp.MustCompile(`(?m)^\s*(?:- )?uses: (.+)$`)

// pinned matches the reference of an action pinned to the commit of a release, with the release
// in a comment.
var pinned = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._/-]+@[0-9a-f]{40} # v?\d+\.\d+\.\d+$`)

// lock is the part of .ergon/init.lock that the cases read.
type lock struct {
	// Ergon is the release of ergon that wrote the lock.
	Ergon string `json:"ergon"`

	// Answers are the answers that the lock records.
	Answers language.Answers `json:"answers"`
}

// workflow is the part of a workflow that the cases check.
type workflow struct {
	// Permissions are the permissions at the top level of the workflow.
	Permissions map[string]string `yaml:"permissions"`

	// Defaults are the defaults of the steps of the workflow, with the shell of its commands.
	Defaults struct {
		// Run are the defaults of the commands.
		Run struct {
			// Shell is the shell of every command.
			Shell string `yaml:"shell"`
		} `yaml:"run"`
	} `yaml:"defaults"`

	// Jobs are the jobs of the workflow, by their identifier.
	Jobs map[string]struct {
		// Uses is the reusable workflow that the job calls, or empty.
		Uses string `yaml:"uses"`

		// RunsOn is the runner of the job.
		RunsOn string `yaml:"runs-on"`

		// Strategy is the strategy of the job, with the systems of its matrix.
		Strategy struct {
			// Matrix is the matrix of the job.
			Matrix struct {
				// OS are the runners of the matrix.
				OS []string `yaml:"os"`
			} `yaml:"matrix"`
		} `yaml:"strategy"`

		// TimeoutMinutes is the timeout of the job: a number of minutes, or the expression of an input
		// of a reusable workflow.
		TimeoutMinutes string `yaml:"timeout-minutes"`

		// Permissions are the permissions of the job.
		Permissions map[string]string `yaml:"permissions"`

		// Steps are the steps of the job.
		Steps []struct {
			// Uses is the action of the step.
			Uses string `yaml:"uses"`

			// Run is the command of the step.
			Run string `yaml:"run"`

			// With are the inputs of the action.
			With map[string]any `yaml:"with"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// failing is a writer whose every write returns errWrite.
type failing struct{}

// Write returns errWrite and writes nothing.
func (failing) Write([]byte) (int, error) {
	return 0, errWrite
}

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		help := []struct {
			name   string
			args   []string
			stdout string
		}{
			{name: "writes the help of init", args: []string{"init", "--help"}, stdout: initHelp},
			{name: "writes the help of init new", args: []string{"init", "new", "--help"}, stdout: newHelp},
			{name: "writes the help of init add", args: []string{"init", "add", "--help"}, stdout: addHelp},
			{name: "writes the help of init remove", args: []string{"init", "remove", "--help"}, stdout: removeHelp},
			{name: "writes the help of init check", args: []string{"init", "check", "--help"}, stdout: checkHelp},
			{name: "writes the help of init sync", args: []string{"init", "sync", "--help"}, stdout: syncHelp},
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

		usage := []struct {
			name   string
			args   []string
			stderr string
		}{
			{
				name: "returns 2 for init without a subcommand", args: []string{"init"},
				stderr: "ergon: cli: init needs a subcommand: add, check, new, remove, sync\n" +
					"Run 'ergon init --help' for usage.\n",
			},
			{
				name: "returns 2 for an unknown subcommand of init", args: []string{"init", "bogus"},
				stderr: "ergon: unknown command \"bogus\" for \"ergon init\"\nRun 'ergon init --help' for usage.\n",
			},
			{
				name: "returns 2 for init new with a --year that is not a number",
				args: slices.Concat([]string{"init", "new", "--year", "next"}, required),
				stderr: "ergon: invalid argument \"next\" for \"--year\" flag: strconv.ParseInt: parsing \"next\": " +
					"invalid syntax\nRun 'ergon init new --help' for usage.\n",
			},
			{
				name: "returns 2 for init add without a language", args: []string{"init", "add"},
				stderr: "ergon: requires at least 1 arg(s), only received 0\nRun 'ergon init add --help' for usage.\n",
			},
			{
				name: "returns 2 for init remove without a language", args: []string{"init", "remove"},
				stderr: "ergon: requires at least 1 arg(s), only received 0\n" +
					"Run 'ergon init remove --help' for usage.\n",
			},
			{
				name: "returns 2 for init remove with --force", args: []string{"init", "remove", "--force", "alpha"},
				stderr: "ergon: unknown flag: --force\nRun 'ergon init remove --help' for usage.\n",
			},
		}
		for _, tt := range usage {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				status, stdout, stderr := run(t, dir, tt.args...)
				assert.Equal(t, status, statusUsage, "the exit status")
				assert.Empty(t, stdout, "the standard output")
				assert.Equal(t, stderr, tt.stderr, "the standard error")
				assert.Empty(t, paths(t, dir), "the files of the repository")
			})
		}

		for i := range len(required) / 2 {
			flag := required[2*i]
			t.Run("returns 2 for init new without "+flag, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				args := slices.Concat([]string{"init", "new"}, required[:2*i], required[2*i+2:])
				status, stdout, stderr := run(t, dir, args...)
				assert.Equal(t, status, statusUsage, "the exit status")
				assert.Empty(t, stdout, "the standard output")
				assert.Equal(t, stderr, "ergon: cli: the flag "+flag+" is required\n"+
					"Run 'ergon init new --help' for usage.\n", "the standard error")
				assert.Empty(t, paths(t, dir), "the files of the repository")
			})
		}

		t.Run("init new writes the files that it reports", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			args := slices.Concat([]string{"init", "new", "--language", "alpha"}, required)
			status, stdout, stderr := run(t, dir, args...)
			assert.Equal(t, status, statusOK, "the exit status")
			assert.Empty(t, stderr, "the standard error")
			lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
			assert.Equal(t, lines[len(lines)-1], "wrote "+lockPath, "the last line of the standard output")
			reported := make([]string, 0, len(lines))
			for _, line := range lines {
				path, ok := strings.CutPrefix(line, "wrote ")
				assert.True(t, ok, "the line "+line)
				reported = append(reported, path)
			}
			slices.Sort(reported)
			assert.Equal(t, reported, paths(t, dir), "the files that init new reports")
			assert.Equal(t, read(t, dir, alphaFile), "alpha\n", "the file of alpha")
		})

		t.Run("init new records the answers in the lock", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, beta, alpha)
			assert.Equal(t, readLock(t, dir), lock{Ergon: version.Release, Answers: language.Answers{
				Name:            filepath.Base(dir),
				Languages:       []workspace.Language{alpha, beta},
				Owner:           owner,
				License:         spdxID,
				Year:            now.Year(),
				Repository:      repository,
				SecurityContact: contact,
			}}, "the lock")
		})

		t.Run("init new records the answers of --name and --year", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			status, _, stderr := run(t, dir, slices.Concat([]string{"init", "new", "--name", "demo", "--year", "2020"},
				required)...)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			answers := readLock(t, dir).Answers
			assert.Equal(t, answers.Name, "demo", "the name in the lock")
			assert.Equal(t, answers.Year, 2020, "the year in the lock")
		})

		t.Run("init new returns 1 for a repository with a lock", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t)
			status, stdout, stderr := run(t, dir, slices.Concat([]string{"init", "new"}, required)...)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, "ergon: baseline: the repository has a lock, so use add or sync\n",
				"the standard error")
		})

		t.Run("init new returns 1 for a managed file with other content", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, licensePath, "Copyright someone else\n")
			status, stdout, stderr := run(t, dir, slices.Concat([]string{"init", "new"}, required)...)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, conflict+licensePath+"\n", "the standard error")
			assert.Equal(t, paths(t, dir), []string{licensePath}, "the files of the repository")
		})

		t.Run("init new overwrites a managed file with other content with --force", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, licensePath, "Copyright someone else\n")
			status, stdout, stderr := run(t, dir, slices.Concat([]string{"init", "new", "--force"}, required)...)
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Contains(t, stdout, "wrote "+licensePath+"\n", "the standard output")
			assert.Contains(t, read(t, dir, licensePath), owner, "the LICENSE")
		})

		t.Run("init new returns 1 for an unknown language", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			status, _, stderr := run(t, dir, slices.Concat([]string{"init", "new", "--language", "cobol"}, required)...)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: baseline: unknown language: \"cobol\", which is none of alpha, beta\n",
				"the standard error")
			assert.Empty(t, paths(t, dir), "the files of the repository")
		})

		t.Run("init new returns 1 for a license that ergon does not have", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			args := slices.Concat([]string{"init", "new"}, required, []string{"--license", "GPL-3.0"})
			status, _, stderr := run(t, dir, args...)
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: language: invalid answer: license \"GPL-3.0\", which is not the SPDX "+
				"identifier of a license of ergon\n", "the standard error")
			assert.Empty(t, paths(t, dir), "the files of the repository")
		})

		t.Run("init add writes the files of the language and the lock", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			status, stdout, stderr := run(t, dir, "init", "add", "beta")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+ignorePath+"\nwrote "+betaFile+"\nwrote "+lockPath+"\n",
				"the standard output")
			assert.Equal(t, readLock(t, dir).Answers.Languages, []workspace.Language{alpha, beta},
				"the languages in the lock")
		})

		t.Run("init add returns 1 for an edited file that it changes", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			write(t, dir, ignorePath, "# edited by hand\n")
			status, stdout, stderr := run(t, dir, "init", "add", "beta")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, conflict+ignorePath+"\n", "the standard error")
			assert.NotContains(t, paths(t, dir), betaFile, "the files of the repository")
		})

		t.Run("init add overwrites an edited file with --force", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			write(t, dir, ignorePath, "# edited by hand\n")
			status, _, stderr := run(t, dir, "init", "add", "--force", "beta")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.HasSuffix(t, read(t, dir, ignorePath), "alpha/\nbeta/\n", "the .gitignore")
		})

		t.Run("init remove removes the files of the language", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha, beta)
			status, stdout, stderr := run(t, dir, "init", "remove", "beta")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+ignorePath+"\nremoved "+betaFile+"\nwrote "+lockPath+"\n",
				"the standard output")
			assert.NotContains(t, paths(t, dir), betaFile, "the files of the repository")
		})

		t.Run("init remove returns 1 for an edited file of the language", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha, beta)
			write(t, dir, betaFile, "beta, edited by hand\n")
			status, stdout, stderr := run(t, dir, "init", "remove", "beta")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the standard output")
			assert.Equal(t, stderr, conflict+betaFile+"\n", "the standard error")
		})

		t.Run("init check writes nothing for a repository at the baseline", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, initialized(t, alpha), "init", "check")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Empty(t, stdout, "the standard output")
		})

		t.Run("init check writes each finding and returns 1", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			write(t, dir, ignorePath, "# edited by hand\n")
			assert.NoError(t, os.Remove(filepath.Join(dir, licensePath)), "Remove of the LICENSE")
			status, stdout, stderr := run(t, dir, "init", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stdout, "edited "+ignorePath+"\nmissing "+licensePath+"\n", "the standard output")
			assert.Equal(t, stderr, "ergon: cli: the repository differs from the baseline\n", "the standard error")
		})

		t.Run("init check writes the findings as a JSON array with --json", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			write(t, dir, ignorePath, "# edited by hand\n")
			status, stdout, _ := run(t, dir, "init", "check", "--json")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stdout, `[{"path":".gitignore","problem":"edited"}]`+"\n", "the standard output")
		})

		t.Run("init check writes an empty JSON array for a repository at the baseline", func(t *testing.T) {
			t.Parallel()
			status, stdout, stderr := run(t, initialized(t, alpha), "init", "check", "--json")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "[]\n", "the standard output")
		})

		t.Run("init check returns 1 when the JSON array cannot be written", func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			p := process(initialized(t, alpha), failing{}, &stderr, "init", "check", "--json")
			assert.Equal(t, cli.Run(t.Context(), p, register, version), statusFailure, "the exit status")
			assert.Equal(t, stderr.String(), "ergon: cli: write the findings: "+errWrite.Error()+"\n",
				"the standard error")
		})

		t.Run("init check returns 1 for a repository without a lock", func(t *testing.T) {
			t.Parallel()
			status, _, stderr := run(t, t.TempDir(), "init", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: baseline: the repository has no lock, so run new first\n",
				"the standard error")
		})

		t.Run("init sync rewrites the files of a changed answer", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			before := readLock(t, dir).Answers
			status, stdout, stderr := run(t, dir, "init", "sync", "--owner", "Other B.V.")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote .ergon.yaml\nwrote "+licensePath+"\nwrote "+lockPath+"\n",
				"the standard output")
			before.Owner = "Other B.V."
			assert.Equal(t, readLock(t, dir).Answers, before, "the answers in the lock")
		})

		t.Run("init sync records the answer of each flag", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			status, _, stderr := run(t, dir, "init", "sync", "--name", "other", "--owner", "Other B.V.",
				"--license", "Apache-2.0", "--year", "2030", "--repository", "other/demo",
				"--security-contact", "security@other.example")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, readLock(t, dir).Answers, language.Answers{
				Name:            "other",
				Languages:       []workspace.Language{alpha},
				Owner:           "Other B.V.",
				License:         "Apache-2.0",
				Year:            2030,
				Repository:      "other/demo",
				SecurityContact: "security@other.example",
			}, "the answers in the lock")
		})

		t.Run("init sync writes the other files and returns 1 for an edited file", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			write(t, dir, ignorePath, "# edited by hand\n")
			status, stdout, stderr := run(t, dir, "init", "sync", "--owner", "Other B.V.")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stdout, "wrote .ergon.yaml\nwrote "+licensePath+"\nwrote "+lockPath+"\n",
				"the standard output")
			assert.Equal(t, stderr, conflict+ignorePath+"\n", "the standard error")
			assert.Equal(t, read(t, dir, ignorePath), "# edited by hand\n", "the .gitignore")
		})

		t.Run("init sync overwrites an edited file with --force", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			write(t, dir, ignorePath, "# edited by hand\n")
			status, stdout, stderr := run(t, dir, "init", "sync", "--force")
			assert.Equal(t, status, statusOK, "the exit status: "+stderr)
			assert.Equal(t, stdout, "wrote "+ignorePath+"\n", "the standard output")
		})

		t.Run("init check returns 1 for a .ergon.yaml that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			write(t, dir, ".ergon.yaml", "name: [\n")
			status, stdout, stderr := run(t, dir, "init", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Empty(t, stdout, "the findings")
			assert.HasPrefix(t, stderr, "ergon: options: invalid .ergon.yaml: ", "the standard error")
		})

		t.Run("init check returns 1 for an option that its section does not accept", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			args := slices.Concat([]string{"init", "new", "--language", "go"}, required)
			status, _, stderr := runWith(t, app.Register, dir, args...)
			assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
			config := strings.Replace(read(t, dir, ".ergon.yaml"), "time: 30s", "time: soon", 1)
			write(t, dir, ".ergon.yaml", config)
			status, _, stderr = runWith(t, app.Register, dir, "init", "check")
			assert.Equal(t, status, statusFailure, "the exit status of init check")
			assert.Equal(t, stderr, "ergon: options: invalid .ergon.yaml: go.fuzz: option: invalid value: time "+
				"\"soon\", which is neither a positive duration nor a positive count such as 100x\n", "the standard error")
		})

		t.Run("init sync renders an option that the repository changed", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			args := slices.Concat([]string{"init", "new", "--language", "go"}, required)
			status, _, stderr := runWith(t, app.Register, dir, args...)
			assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
			write(t, dir, ".ergon.yaml", strings.Replace(read(t, dir, ".ergon.yaml"), "time: 30s", "time: 2m", 1))
			status, stdout, stderr := runWith(t, app.Register, dir, "init", "sync")
			assert.Equal(t, status, statusOK, "the exit status of init sync: "+stderr)
			assert.Equal(t, stdout, "wrote Makefile\nwrote "+lockPath+"\n", "the standard output")
			assert.Contains(t, read(t, dir, "Makefile"), "\nGO_FUZZ_TIME ?= 2m\n", "the Makefile")
		})

		t.Run("init sync returns 1 for a .ergon.yaml that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := initialized(t, alpha)
			write(t, dir, ".ergon.yaml", "name: [\n")
			status, _, stderr := run(t, dir, "init", "sync")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: options: invalid .ergon.yaml: ", "the standard error")
		})

		t.Run("returns 1 for a working directory that does not open", func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "absent")
			status, _, stderr := run(t, dir, "init", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.HasPrefix(t, stderr, "ergon: cli: open the repository: ", "the standard error")
		})

		t.Run("returns 1 for a catalog with a language named as the common files", func(t *testing.T) {
			t.Parallel()
			named := func(c *language.Catalog) error {
				if err := language.RegisterToolchain(c, language.Toolchain{Name: tool}); err != nil {
					return err
				}
				return language.Register(c, language.Declaration{Name: "common", Toolchain: tool})
			}
			status, _, stderr := runWith(t, named, t.TempDir(), "init", "check")
			assert.Equal(t, status, statusFailure, "the exit status")
			assert.Equal(t, stderr, "ergon: baseline: invalid arguments to Open: producer \"common\"\n",
				"the standard error")
		})

		var catalog language.Catalog
		assert.NoError(t, app.Register(&catalog), "Register of the languages of ergon")
		var names []string
		for d := range catalog.Languages() {
			names = append(names, string(d.Name))
		}
		t.Run("init new renders the files of every language by the rules of their formats", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			args := slices.Concat([]string{"init", "new"}, required)
			for _, name := range names {
				args = append(args, "--language", name)
			}
			status, _, stderr := runWith(t, app.Register, dir, args...)
			assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
			baselinetest.Hygiene(t, dir)

			pins := map[string]string{}
			for _, path := range paths(t, dir) {
				if !strings.HasPrefix(path, ".github/") {
					continue
				}
				for _, match := range uses.FindAllStringSubmatch(read(t, dir, path), -1) {
					reference := match[1]
					if strings.HasPrefix(reference, "./.github/") {
						continue
					}
					assert.True(t, pinned.MatchString(reference), "the pin of "+reference+" in "+path)
					action, _, _ := strings.Cut(reference, "@")
					if previous, ok := pins[action]; ok {
						assert.Equal(t, reference, previous, "the pin of "+action+" in "+path)
					}
					pins[action] = reference
				}
			}
			platform, _ := github.Producer{}.Options().(*github.Options)
			checkout := platform.CI.Actions.Checkout
			assert.Equal(t, pins[checkout.Uses], checkout.Uses+"@"+checkout.Commit+" # "+checkout.Release,
				"the pin of "+checkout.Uses)

			workflows, err := filepath.Glob(filepath.Join(dir, ".github", "workflows", "*.yml"))
			assert.NoError(t, err, "Glob of the workflows")
			jobs := map[string]bool{}
			for _, path := range workflows {
				data, err := os.ReadFile(path)
				assert.NoError(t, err, "ReadFile of "+path)
				var w workflow
				assert.NoError(t, yaml.Unmarshal(data, &w), "Unmarshal of "+path)
				assert.True(t, w.Permissions != nil && len(w.Permissions) == 0, "the permissions of "+path)
				gate := filepath.Base(path) == "ci.yml"
				if gate {
					assert.Equal(t, w.Defaults.Run.Shell, "bash", "the shell of the commands of "+path)
				}
				for id, job := range w.Jobs {
					jobs[id] = true
					assert.NotEmpty(t, job.Permissions, "the permissions of "+id)
					if job.Uses != "" {
						continue
					}
					assert.NotEmpty(t, job.TimeoutMinutes, "the timeout of "+id)
					assert.True(t, strings.HasPrefix(job.Steps[0].Uses, checkout.Uses+"@"), "the first step of "+id)
					assert.Equal(t, job.Steps[0].With["persist-credentials"], any(false), "persist-credentials of "+id)
					matrix := gate && strings.HasPrefix(id, "check-")
					if !matrix {
						assert.Equal(t, job.RunsOn, platform.Linux, "the runner of "+id)
						continue
					}
					assert.Equal(t, job.RunsOn, "${{ matrix.os }}", "the runner of "+id)
					assert.Equal(t, job.Strategy.Matrix.OS, []string(platform.Runners), "the systems of "+id)
					setup := -1
					for i, step := range job.Steps {
						if step.Uses == "./.github/actions/setup-make" {
							setup = i
						}
						if strings.HasPrefix(step.Run, "make ") {
							assert.True(t, setup >= 0 && setup < i, "setup-make before make in "+id)
						}
					}
				}
			}
			for _, name := range names {
				assert.True(t, jobs["check-"+name], "the job check-"+name)
			}

			var dependabot struct {
				Updates []struct {
					Ecosystem   string   `yaml:"package-ecosystem"`
					Directory   string   `yaml:"directory"`
					Directories []string `yaml:"directories"`
				} `yaml:"updates"`
			}
			assert.NoError(t, yaml.Unmarshal([]byte(read(t, dir, ".github/dependabot.yml")), &dependabot),
				"Unmarshal of dependabot.yml")
			ecosystems := make([]string, 0, len(dependabot.Updates))
			for _, u := range dependabot.Updates {
				assert.NotContains(t, ecosystems, u.Ecosystem, "the ecosystems of dependabot.yml")
				ecosystems = append(ecosystems, u.Ecosystem)
			}
			assert.Equal(t, ecosystems, []string{
				"nuget", "gradle", "composer", "npm", "gomod", "uv", "cargo", "terraform",
			}, "the ecosystems of dependabot.yml")
		})

		for i, first := range names {
			for _, second := range names[i+1:] {
				t.Run("init new renders "+first+" with "+second+" at the baseline", func(t *testing.T) {
					t.Parallel()
					dir := t.TempDir()
					args := slices.Concat([]string{"init", "new", "--language", first, "--language", second}, required)
					status, _, stderr := runWith(t, app.Register, dir, args...)
					assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
					status, stdout, stderr := runWith(t, app.Register, dir, "init", "check")
					assert.Equal(t, status, statusOK, "the exit status of init check: "+stderr)
					assert.Empty(t, stdout, "the findings of init check")
				})
			}
		}
	})
}

// initialized returns a new repository after init new with the answers of the cases and
// languages.
func initialized(t *testing.T, languages ...workspace.Language) string {
	t.Helper()
	dir := t.TempDir()
	args := slices.Concat([]string{"init", "new"}, required)
	for _, l := range languages {
		args = append(args, "--language", string(l))
	}
	status, _, stderr := run(t, dir, args...)
	assert.Equal(t, status, statusOK, "the exit status of init new: "+stderr)
	return dir
}

// readLock returns the lock of the repository in dir.
func readLock(t *testing.T, dir string) lock {
	t.Helper()
	var l lock
	assert.NoError(t, json.Unmarshal([]byte(read(t, dir, lockPath)), &l), "Unmarshal of the lock")
	return l
}

// paths returns the paths of the files in dir, slash-separated and sorted by their bytes.
func paths(t *testing.T, dir string) []string {
	t.Helper()
	var found []string
	err := fs.WalkDir(os.DirFS(dir), ".", func(name string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			found = append(found, name)
		}
		return err
	})
	assert.NoError(t, err, "WalkDir of the repository")
	slices.Sort(found)
	return found
}

// read returns the content of the file name in dir.
func read(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	assert.NoError(t, err, "ReadFile of "+name)
	return string(data)
}

// write writes content to the file name in dir.
func write(t *testing.T, dir, name, content string) {
	t.Helper()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644), "WriteFile of "+name)
}
