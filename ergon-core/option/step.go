// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"slices"
)

// verifyPrefix starts the target that checks the generated files of a section, as in
// verify-generate-go.
const verifyPrefix = "verify-"

// Step is a step of the gate of a section, and a target of the Makefile: the step test of the
// section python is the target test-python. The gate runs the target that [Step.Target] returns,
// which is verify-generate-python for the step generate. A section has the key of a step's options
// only for a step that its producer has.
type Step string

// The steps of the gate of a section, in the order of their targets.
const (
	// StepFmt formats the sources.
	StepFmt Step = "fmt"

	// StepLint checks the format and the lint of the sources.
	StepLint Step = "lint"

	// StepTest runs the tests.
	StepTest Step = "test"

	// StepRace runs the tests under a race detector.
	StepRace Step = "race"

	// StepFuzz fuzzes each fuzz target for a time.
	StepFuzz Step = "fuzz"

	// StepBench runs the benchmarks.
	StepBench Step = "bench"

	// StepMutate runs the tests against the mutants of the code.
	StepMutate Step = "mutate"

	// StepGenerate runs the generators of the sources. Its target in the gate fails on generated code
	// that is out of date.
	StepGenerate Step = "generate"

	// StepAudit scans the dependencies, or the configuration, for known vulnerabilities.
	StepAudit Step = "audit"
)

// steps are the steps, in the order of their targets.
var steps = []Step{StepFmt, StepLint, StepTest, StepRace, StepFuzz, StepBench, StepMutate, StepGenerate, StepAudit}

// Validate returns an error that wraps [ErrInvalid] for an s that is none of the steps.
func (s Step) Validate() error {
	if !slices.Contains(steps, s) {
		return fmt.Errorf("%w: step %q, which is none of fmt, lint, test, race, fuzz, bench, mutate, generate and "+
			"audit", ErrInvalid, s)
	}
	return nil
}

// Target returns the target of the Makefile that the gate check-<section> runs for s:
// verify-generate-<section> for the step generate, which fails when the generators change a file,
// and <step>-<section> for every other step.
func (s Step) Target(section string) string {
	target := string(s) + "-" + section
	if s == StepGenerate {
		return verifyPrefix + target
	}
	return target
}

// Check is the key check of a section: the steps that its target check-<section> runs, in order.
type Check []Step

// Validate returns an error that wraps [ErrInvalid] for a step of c that is not valid, as
// [Step.Validate] states, and for a step that c names twice. An empty Check is valid: its target
// runs no step.
func (c Check) Validate() error {
	for i, s := range c {
		if err := s.Validate(); err != nil {
			return err
		}
		if slices.Contains(c[:i], s) {
			return fmt.Errorf("%w: step %s, which check names twice", ErrInvalid, s)
		}
	}
	return nil
}

// Only returns an error that wraps [ErrInvalid] for the first step of c that is not one of
// allowed: the steps that the producer of the section has. A producer's Validate calls it for the
// key check.
func (c Check) Only(allowed ...Step) error {
	return only("check", c, allowed)
}

// only returns an error that wraps [ErrInvalid] for the first step of steps, the steps of key, that
// is not one of allowed.
func only(key string, steps, allowed []Step) error {
	for _, s := range steps {
		if !slices.Contains(allowed, s) {
			return fmt.Errorf("%w: %s names %s, which is no step of the section", ErrInvalid, key, s)
		}
	}
	return nil
}
