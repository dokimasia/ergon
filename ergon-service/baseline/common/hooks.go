// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"slices"

	"go.dokimi.dev/ergon/core/option"
)

// Target is an aggregate target of the Makefile that a hook of .pre-commit-config.yaml runs. Each
// language adds its own target of the same kind to it, such as lint-go to lint.
type Target string

// The aggregate targets of the Makefile, in the order of the Makefile.
const (
	// TargetFmt formats the sources of every language.
	TargetFmt Target = "fmt"

	// TargetLint lints the sources of every language.
	TargetLint Target = "lint"

	// TargetTest runs the tests of every language.
	TargetTest Target = "test"

	// TargetAudit runs the vulnerability scan of every language.
	TargetAudit Target = "audit"

	// TargetCheck runs the gate of every language.
	TargetCheck Target = "check"
)

// targets are the aggregate targets, in the order of the Makefile.
var targets = []Target{TargetFmt, TargetLint, TargetTest, TargetAudit, TargetCheck}

// Validate returns an error that wraps [option.ErrInvalid] for a t that is none of the aggregate
// targets.
func (t Target) Validate() error {
	if !slices.Contains(targets, t) {
		return fmt.Errorf("%w: target %q, which is none of fmt, lint, test, audit and check", option.ErrInvalid, t)
	}
	return nil
}

// Targets are the targets that the hooks of one stage run, in order.
type Targets []Target

// Validate returns an error that wraps [option.ErrInvalid] for a target of t that is not valid, as
// [Target.Validate] states, and for a target that t names twice. An empty Targets is valid, and
// turns the hooks of its stage off.
func (t Targets) Validate() error {
	for i, target := range t {
		if err := target.Validate(); err != nil {
			return err
		}
		if slices.Contains(t[:i], target) {
			return fmt.Errorf("%w: target %s, which the stage names twice", option.ErrInvalid, target)
		}
	}
	return nil
}

// Hooks are the targets that the hooks of .pre-commit-config.yaml run at each stage of git. A
// target may appear at both stages.
type Hooks struct {
	// PreCommit are the targets that the hooks run before each commit.
	PreCommit Targets `yaml:"pre-commit" doc:"The targets that run before each commit, in order: fmt, lint, test, audit or check."`

	// PrePush are the targets that the hooks run before each push.
	PrePush Targets `yaml:"pre-push" doc:"The targets that run before each push, in order: fmt, lint, test, audit or check."`
}
