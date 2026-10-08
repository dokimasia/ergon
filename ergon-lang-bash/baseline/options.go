// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// windows is the system of Windows, whose asset of shellcheck is a .zip.
const windows = "windows"

// shellcheckArch are the architectures of the assets of shellcheck, by the architecture of Go.
var shellcheckArch = map[string]string{"amd64": "x86_64", "arm64": "aarch64"}

// Options are the options of the section bash of .ergon.yaml.
type Options struct {
	// Tools are the tools of the targets of Bash.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Bash."`

	// Paths are the pathspecs of git of the scripts.
	Paths option.Paths `yaml:"paths" doc:"The pathspecs of git of the scripts that lint-bash checks."`

	// Check are the steps of check-bash.
	Check option.Check `yaml:"check" doc:"The steps that check-bash runs: lint."`

	// CI are the runners and the limit of the job check-bash.
	CI option.RunnerCI[struct{}] `yaml:"ci" doc:"The runners of the job check-bash, and its limit in minutes. Bash needs no setup, so the job runs no action of its own. Empty runners select every runner of the section github."`
}

// Tools are the tools of the targets of Bash.
type Tools struct {
	// Shellcheck checks the scripts.
	Shellcheck Shellcheck `yaml:"shellcheck" doc:"The release of shellcheck, the linter of lint-bash, with the checks of .shellcheckrc."`
}

// Shellcheck is the release binary of shellcheck, koalaman/shellcheck on GitHub, whose release
// publishes a .tar.gz for Linux and macOS on x86-64 and ARM, and a .zip for Windows on x86-64.
type Shellcheck struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = Shellcheck{}

// shellcheckRepository is the repository on GitHub whose releases publish shellcheck.
const shellcheckRepository = "koalaman/shellcheck"

// Validate returns an error that wraps [option.ErrInvalid] for a step of check other than lint.
// The command checks each option by the Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	return o.Check.Only(option.StepLint)
}

// Contribution returns the part of Bash of the workflows for o: the job check-bash, which runs
// make check-bash on the runners of o in every repository, because Bash has no file that pins a
// toolchain. The job keeps shellcheck, which ergon tool run installs, in the cache of GitHub
// Actions. CodeQL analyzes no Bash, and Dependabot updates no script.
func (o *Options) Contribution() workflow.Contribution {
	return workflow.Contribution{Jobs: []workflow.Job{{
		ID:          "check-bash",
		Name:        "Bash",
		Permissions: map[string]string{"contents": "read"},
		Setup:       &workflow.Setup{Runners: o.CI.Runners, Timeout: o.CI.Timeout},
		Tools:       true,
		Steps:       []workflow.Step{{Name: "Check Bash", Run: []string{"make check-bash"}}},
	}}}
}

// Asset returns the archive of shellcheck for p from the release of the version of s on GitHub:
// shellcheck-v<version>.<os>.<x86_64|aarch64>.tar.gz with the program in the directory
// shellcheck-v<version>, and shellcheck-v<version>.zip with shellcheck.exe at its root on Windows
// on x86-64. It returns an error that wraps [option.ErrNoAsset] for a platform that is not valid,
// and for Windows on ARM, which the release does not publish.
func (s Shellcheck) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil || p == option.WindowsARM64 {
		return option.Asset{}, fmt.Errorf("%w: shellcheck has no asset for %q", option.ErrNoAsset, p)
	}
	name := "shellcheck-v" + s.Version
	release := "https://github.com/" + shellcheckRepository + "/releases/download/v" + s.Version + "/" + name
	if p.OS() == windows {
		return option.Asset{URL: release + ".zip", Program: "shellcheck.exe"}, nil
	}
	return option.Asset{
		URL:     release + "." + p.OS() + "." + shellcheckArch[p.Arch()] + ".tar.gz",
		Program: name + "/shellcheck",
	}, nil
}

// Repository returns koalaman/shellcheck, the repository on GitHub whose releases publish
// shellcheck.
func (Shellcheck) Repository() string {
	return shellcheckRepository
}
