// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"regexp"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// windows is the system of Windows, whose programs end in .exe.
const windows = "windows"

// word matches a word of a command that the shell, make and pre-commit read as it is: letters,
// digits and the characters _ @ % + = : , . / ^ -.
var word = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./^-]+$`)

// Options are the options of the section common of .ergon.yaml.
type Options struct {
	// Ergon is the command that runs ergon in the targets of the Makefile and in the hooks.
	Ergon Command `yaml:"ergon" doc:"The command that runs ergon in the targets of the Makefile and in the hooks of .pre-commit-config.yaml, as its words, such as [ergon] for the ergon on the PATH. The repository of ergon runs ergon from its own source with [go, run, go.dokimi.dev/ergon/cmd/ergon]."`

	// Tools are the tools of the common files.
	Tools Tools `yaml:"tools" doc:"The tools of the common files."`

	// PreCommitHooks is the release of github.com/pre-commit/pre-commit-hooks.
	PreCommitHooks option.Version `yaml:"pre-commit-hooks" source:"github:pre-commit/pre-commit-hooks" doc:"The release of github.com/pre-commit/pre-commit-hooks, whose hooks check the hygiene of the files before each commit."`

	// Hooks are the targets of the Makefile that the hooks of .pre-commit-config.yaml run.
	Hooks Hooks `yaml:"hooks" doc:"The targets of the Makefile that the hooks of .pre-commit-config.yaml run, at each stage of git. An empty list turns the hooks of its stage off."`

	// CI are the pins of the actions of the jobs docs and commits, and their limit.
	CI option.CI[Actions] `yaml:"ci" doc:"The pins of the actions of the jobs docs and commits of ci.yml, and the limit of each job in minutes."`
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
}

// Actions are the pins of the actions of the jobs of the common files.
type Actions struct {
	// Markdownlint lints the Markdown files.
	Markdownlint workflow.Action `yaml:"markdownlint" doc:"The action that lints the Markdown files in the job docs, with the rules of .markdownlint.yml."`
}

// Commitlint is the release binary of commitlint, conventionalcommit/commitlint on GitHub, whose
// release publishes a .tar.gz for every platform.
type Commitlint struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = Commitlint{}

// commitlintRepository is the repository on GitHub whose releases publish commitlint.
const commitlintRepository = "conventionalcommit/commitlint"

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
