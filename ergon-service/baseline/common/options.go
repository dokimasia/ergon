// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common

import (
	"fmt"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// windows is the system of Windows, whose programs end in .exe.
const windows = "windows"

// Options are the options of the section common of .ergon.yaml.
type Options struct {
	// Tools are the tools of the common files.
	Tools Tools `yaml:"tools" doc:"The tools of the common files."`

	// PreCommitHooks is the release of github.com/pre-commit/pre-commit-hooks.
	PreCommitHooks option.Version `yaml:"pre-commit-hooks" doc:"The release of github.com/pre-commit/pre-commit-hooks, whose hooks check the hygiene of the files before each commit."`

	// CI are the pins of the actions of the jobs docs and commits, and their limit.
	CI option.CI[Actions] `yaml:"ci" doc:"The pins of the actions of the jobs docs and commits of ci.yml, and the limit of each job in minutes."`
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
	download := "https://github.com/conventionalcommit/commitlint/releases/download/v" + c.Version
	return option.Asset{
		URL:     download + "/commitlint_v" + c.Version + "_" + p.OS() + "_" + p.Arch() + ".tar.gz",
		Program: program,
	}, nil
}
