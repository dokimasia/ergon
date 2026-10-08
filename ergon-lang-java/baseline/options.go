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

// Options are the options of the section java of .ergon.yaml. The section jvm sets the scan that
// Java shares with Kotlin, and the setup of their jobs.
type Options struct {
	// Tools are the tools of the targets of Java.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Java, as <group>:<artifact>@<version> of Maven Central."`

	// Check are the steps of check-java.
	Check option.Check `yaml:"check" doc:"The steps that check-java runs, in order: lint, test, generate or audit. audit runs audit-jvm of the section jvm."`

	// Test are the options of test-java.
	Test option.Run `yaml:"test" doc:"test-java runs ./gradlew test with args."`

	// Generate are the options of generate-java and verify-generate-java.
	Generate option.Generate `yaml:"generate" doc:"generate-java runs command with args at the root of the repository, such as ./gradlew with a task of a generator. verify-generate-java, the step generate of check, runs the same and fails when the run changes a file of the repository. Without a command Java has neither target."`
}

// Tools are the tools of the targets of Java.
type Tools struct {
	// PMD checks the sources.
	PMD option.Maven `yaml:"pmd" doc:"The PMD of lint-java, which the PMD plugin of Gradle runs with its base ruleset. The plugin runs net.sourceforge.pmd:pmd-java, so the tool accepts another version of it and no other artifact."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a PMD that is not a version of
// net.sourceforge.pmd:pmd-java, the artifact that the PMD plugin of Gradle runs. The command checks
// the format of the tool by the Validate method of its type before it calls Validate.
func (t *Tools) Validate() error {
	if t.PMD.Package() != tools.PMD.Package() {
		return fmt.Errorf("%w: pmd %q, which is not a version of %s", option.ErrInvalid, t.PMD, tools.PMD.Package())
	}
	return nil
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
// check-java with the setup of the jvm toolchain, on its runners and versions. The job keeps PMD and
// osv-scanner, which ergon tool run installs, in the cache of GitHub Actions. The jvm toolchain
// contributes the CodeQL analysis and the updates that Java shares with Kotlin.
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
