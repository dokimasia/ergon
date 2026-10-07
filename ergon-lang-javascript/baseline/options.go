// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// steps are the steps of the gate of JavaScript, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepAudit}

// Options are the options of the section javascript of .ergon.yaml. The section js sets the tools,
// the paths and the scan that JavaScript shares with TypeScript, and the setup of their jobs.
type Options struct {
	// Check are the steps of check-javascript.
	Check option.Check `yaml:"check" doc:"The steps that check-javascript runs, in order: lint, test or audit. lint runs lint-js of the section js, and audit runs audit-js."`

	// Test are the options of test-javascript.
	Test option.Run `yaml:"test" doc:"test-javascript runs npm test with args."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that JavaScript does
// not have. The command checks each option by the Validate method of its type before it calls
// Validate.
func (o *Options) Validate() error {
	return o.Check.Only(steps...)
}

// Contribution returns the part of JavaScript of the workflows: the job check-javascript, which
// runs make check-javascript with the setup of the js toolchain, on its runners and versions. The js
// toolchain contributes the CodeQL analysis and the updates that JavaScript shares with TypeScript.
func (*Options) Contribution() workflow.Contribution {
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-javascript",
			Name:        "JavaScript",
			Toolchain:   ToolchainName,
			Permissions: map[string]string{"contents": "read"},
			Steps:       []workflow.Step{{Name: "Check JavaScript", Run: []string{"make check-javascript"}}},
		}},
	}
}
