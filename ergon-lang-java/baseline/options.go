// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// steps are the steps of the gate of Java, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepGenerate, option.StepAudit}

// pmdPackage is the artifact of PMD that the PMD plugin of Gradle downloads and runs.
const pmdPackage = "net.sourceforge.pmd:pmd-java"

// formatRepository is the repository on GitHub whose releases publish google-java-format.
const formatRepository = "google/google-java-format"

// formatAssets are the suffixes of the names of the programs of google-java-format, by platform.
// The release does not publish a program for macOS on x86-64 or for Windows on 64-bit ARM.
var formatAssets = map[option.Platform]string{
	option.LinuxAMD64:   "linux-x86-64",
	option.LinuxARM64:   "linux-arm64",
	option.DarwinARM64:  "darwin-arm64",
	option.WindowsAMD64: "windows-x86-64.exe",
}

// Options are the options of the section java of .ergon.yaml. The section jvm sets the scan that
// Java shares with Kotlin, and the setup of their jobs.
type Options struct {
	// Tools are the tools of the targets of Java.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Java: PMD as <group>:<artifact>@<version> of Maven Central, and the release binary of google-java-format by its version and the SHA-256 of the asset of each platform."`

	// Rulesets are the rule sets of PMD of lint-java.
	Rulesets Rulesets `yaml:"rulesets" doc:"The rule sets of PMD that lint-java checks the Java sources with. A rule set is the path of a rule set file of the repository, relative to its root, such as config/pmd.xml, or a rule set of PMD, such as its base ruleset rulesets/java/quickstart.xml or the category category/java/bestpractices.xml. A rule set file can refer to a rule set of PMD and exclude some of its rules."`

	// Check are the steps of check-java.
	Check option.Check `yaml:"check" doc:"The steps that check-java runs, in order: lint, test, generate or audit. audit runs audit-jvm of the section jvm."`

	// Test are the options of test-java.
	Test option.Run `yaml:"test" doc:"test-java runs ./gradlew test with args."`

	// Generate are the options of generate-java and verify-generate-java.
	Generate option.Generate `yaml:"generate" doc:"generate-java runs each command line of commands in order at the root of the repository, such as ./gradlew with a task of a generator. verify-generate-java, the step generate of check, runs the same and fails when the run changes a file of the repository. Without a command line Java has neither target."`
}

// Tools are the tools of the targets of Java.
type Tools struct {
	// PMD checks the sources.
	PMD option.Maven `yaml:"pmd" doc:"The PMD of lint-java, which the PMD plugin of Gradle runs with the rule sets of rulesets. The plugin runs net.sourceforge.pmd:pmd-java, so the tool accepts another version of it and no other artifact."`

	// GoogleJavaFormat formats the sources.
	GoogleJavaFormat GoogleJavaFormat `yaml:"google-java-format" doc:"The release of google-java-format, the formatter of fmt-java and lint-java, in Google Java Style."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a PMD that is not a version of
// net.sourceforge.pmd:pmd-java, the artifact that the PMD plugin of Gradle runs. The command checks
// the format of each tool by the Validate method of its type before it calls Validate.
func (t *Tools) Validate() error {
	if t.PMD.Package() != pmdPackage {
		return fmt.Errorf("%w: pmd %q, which is not a version of %s", option.ErrInvalid, t.PMD, pmdPackage)
	}
	return nil
}

// GoogleJavaFormat is the release binary of google-java-format, google/google-java-format on
// GitHub, whose release publishes the program itself for Linux, for macOS on Apple silicon and for
// Windows on x86-64.
type GoogleJavaFormat struct {
	option.Binary `yaml:",inline"`
}

var _ option.Release = GoogleJavaFormat{}

// Asset returns the program of google-java-format for p from the release of the version of f on
// GitHub: google-java-format_<os>-<arch>, such as google-java-format_linux-x86-64, with the suffix
// .exe on Windows. It returns an error that wraps [option.ErrNoAsset] for macOS on x86-64, for
// Windows on 64-bit ARM and for a platform that is not valid.
func (f GoogleJavaFormat) Asset(p option.Platform) (option.Asset, error) {
	suffix, ok := formatAssets[p]
	if !ok {
		return option.Asset{}, fmt.Errorf("%w: google-java-format has no asset for %q", option.ErrNoAsset, p)
	}
	return option.Asset{
		URL: "https://github.com/" + formatRepository + "/releases/download/v" + f.Version +
			"/google-java-format_" + suffix,
	}, nil
}

// Repository returns google/google-java-format, the repository on GitHub whose releases publish
// google-java-format.
func (GoogleJavaFormat) Repository() string {
	return formatRepository
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that Java does not
// have, and for the step generate without a command. The command checks each option by the
// Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	if err := o.Check.Only(steps...); err != nil {
		return err
	}
	return o.Generate.CheckStep(o.Check)
}

// Contribution returns the part of Java of the workflows: the job check-java, which runs make
// check-java with the setup of the jvm toolchain, on its runners and versions. The job keeps
// google-java-format and osv-scanner, which ergon tool run installs, in the cache of GitHub
// Actions. Gradle downloads PMD. The jvm toolchain contributes the CodeQL analysis and the updates
// that Java shares with Kotlin.
func (*Options) Contribution() workflow.Contribution {
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-java",
			Name:        "Java",
			Toolchain:   ToolchainName,
			Permissions: map[string]string{"contents": "read"},
			Tools:       true,
			Steps:       []workflow.Step{{Name: "Check Java", Run: []string{"make check-java"}}},
		}},
	}
}
