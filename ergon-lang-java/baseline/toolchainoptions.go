// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// pin is the file that pins the Java of a repository.
const pin = ".java-version"

// windows is the system of Windows, whose programs end in .exe.
const windows = "windows"

// ToolchainOptions are the options of the section jvm of .ergon.yaml: the scan that Java and Kotlin
// share, and the setup of their jobs.
type ToolchainOptions struct {
	// Tools are the tools of the shared targets.
	Tools ToolchainTools `yaml:"tools" doc:"The tools of the targets that Java and Kotlin share."`

	// CI are the pin of the setup of Java, the runners, the versions and the limit of the jobs of
	// Java and Kotlin.
	CI option.MatrixCI[ToolchainActions] `yaml:"ci" doc:"The pin of setup-java, the runners and the versions of Java of the jobs check-java and check-kotlin, and their limit in minutes. Empty runners select every runner of the section github, and empty versions the Java of .java-version."`
}

// ToolchainTools are the tools of the targets that Java and Kotlin share.
type ToolchainTools struct {
	// OSVScanner scans the Gradle lockfiles.
	OSVScanner OSVScanner `yaml:"osv-scanner" doc:"The release of osv-scanner, the vulnerability scan of audit-jvm, over the Gradle lockfiles."`
}

// OSVScanner is the release binary of osv-scanner, google/osv-scanner on GitHub, whose release
// publishes the program itself for every platform.
type OSVScanner struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = OSVScanner{}

// osvScannerRepository is the repository on GitHub whose releases publish osv-scanner.
const osvScannerRepository = "google/osv-scanner"

// ToolchainActions are the pins of the actions of the setup of the jvm toolchain.
type ToolchainActions struct {
	// SetupJava installs Java.
	SetupJava workflow.Action `yaml:"setup-java" doc:"The action that installs the Temurin build of the Java of .java-version or of the matrix."`
}

// Validate returns nil. The section jvm has no rule beyond the Validate methods of the types of its
// options, which the command calls before Validate.
func (*ToolchainOptions) Validate() error {
	return nil
}

// Contribution returns the part of the jvm toolchain of the workflows for o:
//
//   - The setup that the jobs check-java and check-kotlin run on the runners of o, once the
//     repository has .java-version: setup-java installs the Temurin build of the Java of
//     .java-version, or of the version of the matrix where o lists versions. The steps of
//     ci.steps run after setup-java.
//   - The CodeQL analysis of java-kotlin builds the sources with the build of the repository, which
//     the analysis of Kotlin requires.
//   - Dependabot updates the Gradle dependencies.
func (o *ToolchainOptions) Contribution() workflow.Contribution {
	with := map[string]string{"distribution": "temurin", "java-version-file": pin}
	if len(o.CI.Versions) > 0 {
		with = map[string]string{"distribution": "temurin", "java-version": "${{ matrix.version }}"}
	}
	return workflow.Contribution{
		Setup: &workflow.Setup{
			Files:    pin,
			Runners:  o.CI.Runners,
			Versions: o.CI.Versions,
			Steps: slices.Concat([]workflow.Step{{Name: "Set up Java", Uses: o.CI.Actions.SetupJava, With: with}},
				o.CI.Steps),
			Timeout: o.CI.Timeout,
		},
		CodeQL: []workflow.CodeQL{{
			Language:  "java-kotlin",
			Name:      "Java and Kotlin",
			BuildMode: "autobuild",
			Files:     pin,
			Timeout:   o.CI.Timeout,
		}},
		Updates: []workflow.Update{{Ecosystem: "gradle", Directories: []string{"/"}}},
	}
}

// Asset returns the program of osv-scanner for p from the release of the version of s on GitHub:
// osv-scanner_<os>_<arch>, and osv-scanner_<os>_<arch>.exe on Windows. It returns an error that
// wraps [option.ErrNoAsset] for a platform that is not valid.
func (s OSVScanner) Asset(p option.Platform) (option.Asset, error) {
	if p.Validate() != nil {
		return option.Asset{}, fmt.Errorf("%w: osv-scanner has no asset for %q", option.ErrNoAsset, p)
	}
	program := "osv-scanner_" + p.OS() + "_" + p.Arch()
	if p.OS() == windows {
		program += ".exe"
	}
	return option.Asset{
		URL: "https://github.com/" + osvScannerRepository + "/releases/download/v" + s.Version + "/" + program,
	}, nil
}

// Repository returns google/osv-scanner, the repository on GitHub whose releases publish
// osv-scanner.
func (OSVScanner) Repository() string {
	return osvScannerRepository
}
