// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/lang/javascript"
)

// steps are the steps of the gate of TypeScript, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepGenerate, option.StepAudit}

// Options are the options of the section typescript of .ergon.yaml. The section js sets the tools,
// the paths and the scan that TypeScript shares with JavaScript, and the setup of their jobs.
type Options struct {
	// Tools are the tools of the targets of TypeScript.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of TypeScript, as <package>@<version> of npm, which ergon tool run runs with npx."`

	// Check are the steps of check-typescript.
	Check option.Check `yaml:"check" doc:"The steps that check-typescript runs, in order: lint, test, generate or audit. lint runs lint-js of the section js before tsc, and audit runs audit-js."`

	// Test are the options of test-typescript.
	Test option.Run `yaml:"test" doc:"test-typescript runs npm test with args."`

	// Generate are the options of generate-typescript and verify-generate-typescript.
	Generate option.Generate `yaml:"generate" doc:"generate-typescript runs command with args at the root of the repository, such as npm run generate. verify-generate-typescript, the step generate of check, runs the same and fails when the run changes a file of the repository. Without a command TypeScript has neither target."`
}

// Tools are the tools of the targets of TypeScript.
type Tools struct {
	// TypeScript checks the types of the sources.
	TypeScript option.NPM `yaml:"typescript" program:"tsc" doc:"The type checker of lint-typescript, tsc, with its type-safety options."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that TypeScript does
// not have, and for the step generate without a command. The command checks each option by the
// Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	if err := o.Check.Only(steps...); err != nil {
		return err
	}
	return o.Generate.CheckStep(o.Check)
}

// Contribution returns the part of TypeScript of the workflows: the job check-typescript, which
// runs make check-typescript with the setup of the js toolchain, on its runners and versions. The
// js toolchain contributes the CodeQL analysis and the updates that TypeScript shares with
// JavaScript.
func (*Options) Contribution() workflow.Contribution {
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-typescript",
			Name:        "TypeScript",
			Toolchain:   string(javascript.Toolchain),
			Permissions: map[string]string{"contents": "read"},
			Steps:       []workflow.Step{{Name: "Check TypeScript", Run: []string{"make check-typescript"}}},
		}},
	}
}
