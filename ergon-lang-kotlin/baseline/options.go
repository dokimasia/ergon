// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"fmt"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/java"
)

// steps are the steps of the gate of Kotlin, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepGenerate, option.StepAudit}

// Options are the options of the section kotlin of .ergon.yaml. The section jvm sets the scan that
// Kotlin shares with Java, and the setup of their jobs.
type Options struct {
	// Tools are the tools of the targets of Kotlin.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Kotlin, as <group>:<artifact>@<version> of Maven Central."`

	// Check are the steps of check-kotlin.
	Check option.Check `yaml:"check" doc:"The steps that check-kotlin runs, in order: lint, test, generate or audit. audit runs audit-jvm of the section jvm."`

	// Test are the options of test-kotlin.
	Test option.Run `yaml:"test" doc:"test-kotlin runs ./gradlew test with args."`

	// Generate are the options of generate-kotlin and verify-generate-kotlin.
	Generate option.Generate `yaml:"generate" doc:"generate-kotlin runs command with args at the root of the repository, such as ./gradlew with a task of a generator. verify-generate-kotlin, the step generate of check, runs the same and fails when the run changes a file of the repository. Without a command Kotlin has neither target."`
}

// Tools are the tools of the targets of Kotlin.
type Tools struct {
	// Ktlint formats and lints the sources.
	Ktlint option.Maven `yaml:"ktlint" classifier:"all" doc:"The ktlint of fmt-kotlin and lint-kotlin, with the rules of the section of Kotlin of .editorconfig. ergon tool run runs the jar of com.pinterest.ktlint:ktlint-cli with its dependencies, so the tool accepts another version of it and no other artifact."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a ktlint that is not a version of
// com.pinterest.ktlint:ktlint-cli, whose jar with the classifier all contains ktlint and its
// dependencies. The command checks the format of the tool by the Validate method of its type before
// it calls Validate.
func (t *Tools) Validate() error {
	if t.Ktlint.Package() != tools.Ktlint.Package() {
		return fmt.Errorf("%w: ktlint %q, which is not a version of %s", option.ErrInvalid, t.Ktlint,
			tools.Ktlint.Package())
	}
	return nil
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that Kotlin does not
// have, and for the step generate without a command. The command checks each option by the
// Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	if err := o.Check.Only(steps...); err != nil {
		return err
	}
	return o.Generate.CheckStep(o.Check)
}

// Contribution returns the part of Kotlin of the workflows: the job check-kotlin, which runs make
// check-kotlin with the setup of the jvm toolchain, on its runners and versions. The job keeps
// ktlint and osv-scanner, which ergon tool run installs, in the cache of GitHub Actions. The jvm
// toolchain contributes the CodeQL analysis and the updates that Kotlin shares with Java.
func (*Options) Contribution() workflow.Contribution {
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-kotlin",
			Name:        "Kotlin",
			Toolchain:   string(java.Toolchain),
			Permissions: map[string]string{"contents": "read"},
			Tools:       true,
			Steps:       []workflow.Step{{Name: "Check Kotlin", Run: []string{"make check-kotlin"}}},
		}},
	}
}
