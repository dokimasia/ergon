// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"regexp"

	"go.dokimi.dev/ergon/core/option"
)

// windows is the system of Windows, whose programs end in .exe.
const windows = "windows"

// The repositories on GitHub whose releases publish the release binaries of the section.
const (
	commitlintRepository = "conventionalcommit/commitlint"
	rumdlRepository      = "rvben/rumdl"
)

// word matches a word of a command that the shell, make and pre-commit read as it is: letters,
// digits and the characters _ @ % + = : , . / ^ -.
var word = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./^-]+$`)

// rumdlTargets are the target triples of the assets of rumdl, by platform. On Linux the assets are
// the static builds of musl, and the release does not publish an asset for Windows on ARM.
var rumdlTargets = map[option.Platform]string{
	option.LinuxAMD64:   "x86_64-unknown-linux-musl",
	option.LinuxARM64:   "aarch64-unknown-linux-musl",
	option.DarwinAMD64:  "x86_64-apple-darwin",
	option.DarwinARM64:  "aarch64-apple-darwin",
	option.WindowsAMD64: "x86_64-pc-windows-msvc",
}

// Options are the options of the section common of .ergon.yaml.
type Options struct {
	// Ergon is the command that runs ergon in the targets of the Makefile and in the hooks.
	Ergon Command `yaml:"ergon" doc:"The command that runs ergon in the targets of the Makefile and in the hooks of .pre-commit-config.yaml, as its words, such as [ergon] for the ergon on the PATH. The repository of ergon runs ergon from its own source with [go, run, go.dokimi.dev/ergon/cmd/ergon]."`

	// Tools are the tools of the common files.
	Tools Tools `yaml:"tools" doc:"The tools of the common files."`

	// PreCommitHooks is the release of github.com/pre-commit/pre-commit-hooks.
	PreCommitHooks option.Version `yaml:"pre-commit-hooks" source:"github:pre-commit/pre-commit-hooks" doc:"The release of github.com/pre-commit/pre-commit-hooks, whose hooks check the hygiene of the files before each commit."`

	// Hooks are the hooks of .pre-commit-config.yaml.
	Hooks Hooks `yaml:"hooks" doc:"The hooks of .pre-commit-config.yaml: the paths that the hooks of file hygiene keep as they are, and the targets of the Makefile that the hooks run at each stage of git. An empty list of targets turns the hooks of its stage off."`

	// CI is the limit of the jobs docs, commits and changeset.
	CI option.CI[struct{}] `yaml:"ci" doc:"The limit in minutes of each job of the common files in ci.yml: docs, commits and changeset. The jobs run ergon, and they do not run an action of their own."`
}

// Command is a command as its words, such as go, run and the package of a program.
type Command []string

// Validate returns an error that wraps [option.ErrInvalid] for a command without a word, and for a
// word with a character other than a letter, a digit and _ @ % + = : , . / ^ -. The Makefile and
// .pre-commit-config.yaml then write each word as it is.
func (c Command) Validate() error {
	if len(c) == 0 {
		return fmt.Errorf("%w: a command without a word", option.ErrInvalid)
	}
	for _, w := range c {
		if !word.MatchString(w) {
			return fmt.Errorf("%w: the word %q, which has a character other than a letter, a digit and _ @ %% + = "+
				": , . / ^ -", option.ErrInvalid, w)
		}
	}
	return nil
}

// Tools are the tools of the common files.
type Tools struct {
	// Commitlint checks the commit messages.
	Commitlint Commitlint `yaml:"commitlint" doc:"The release of commitlint, which checks each commit message in the commit-msg hook of .pre-commit-config.yaml and in the job commits of ci.yml."`

	// Rumdl lints the Markdown files.
	Rumdl Rumdl `yaml:"rumdl" doc:"The release of rumdl, which lint-common runs on the Markdown files with the rules of .markdownlint.yml. It skips the files of .markdownlintignore."`
}

// Commitlint is the release binary of commitlint, conventionalcommit/commitlint on GitHub, whose
// release publishes a .tar.gz for every platform.
type Commitlint struct {
	option.Binary `yaml:",inline"`
}

// Rumdl is the release binary of rumdl, rvben/rumdl on GitHub. rumdl lints Markdown files with the
// rules of markdownlint, and its release publishes an asset for every platform but Windows on ARM.
type Rumdl struct {
	option.Binary `yaml:",inline"`
}

var (
	_ option.Release = Commitlint{}
	_ option.Release = Rumdl{}
)

// Validate returns nil. The options of the section common have no rule between them, and the
// command checks each option by the Validate method of its type before it calls Validate.
func (*Options) Validate() error {
	return nil
}

// Asset returns the archive of commitlint for p from the release of the version of c on GitHub:
// commitlint_v<version>_<os>_<arch>.tar.gz, with the program at its root, and commitlint.exe on
// Windows. It returns an error that wraps [option.ErrNoAsset] for a platform that is not valid.
func (c Commitlint) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil {
		return option.Asset{}, fmt.Errorf("%w: commitlint has no asset for %q", option.ErrNoAsset, p)
	}
	program := "commitlint"
	if p.OS() == windows {
		program += ".exe"
	}
	download := "https://github.com/" + commitlintRepository + "/releases/download/v" + c.Version
	return option.Asset{
		URL:     download + "/commitlint_v" + c.Version + "_" + p.OS() + "_" + p.Arch() + ".tar.gz",
		Program: program,
	}, nil
}

// Repository returns conventionalcommit/commitlint, the repository on GitHub whose releases
// publish commitlint.
func (Commitlint) Repository() string {
	return commitlintRepository
}

// Asset returns the archive of rumdl for p from the release of the version of r on GitHub:
// rumdl-v<version>-<target>.tar.gz with the program at its root, and a .zip with rumdl.exe on
// Windows. It returns an error that wraps [option.ErrNoAsset] for Windows on ARM, whose asset the
// release does not publish, and for a platform that is not valid.
func (r Rumdl) Asset(p option.Platform) (option.Asset, error) {
	target, ok := rumdlTargets[p]
	if !ok {
		return option.Asset{}, fmt.Errorf("%w: rumdl has no asset for %q", option.ErrNoAsset, p)
	}
	download := "https://github.com/" + rumdlRepository + "/releases/download/v" + r.Version + "/rumdl-v" +
		r.Version + "-" + target
	if p.OS() == windows {
		return option.Asset{URL: download + ".zip", Program: "rumdl.exe"}, nil
	}
	return option.Asset{URL: download + ".tar.gz", Program: "rumdl"}, nil
}

// Repository returns rvben/rumdl, the repository on GitHub whose releases publish rumdl.
func (Rumdl) Repository() string {
	return rumdlRepository
}
