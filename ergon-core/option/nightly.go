// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"slices"
)

// Nightly is the key nightly of a section: the steps that the workflow nightly.yml runs on a
// schedule, each in a job of its own, with the limit of that job in minutes. A producer has the key
// when it has a step that is too long for each push, such as fuzz or mutate. An empty Nightly runs
// no step.
type Nightly map[Step]int

// Validate returns an error that wraps [ErrInvalid] for a step that is not valid, as
// [Step.Validate] states, and for a limit below a minute. It checks the steps in the order of their
// targets, so it returns the same error for the same value.
func (n Nightly) Validate() error {
	for _, s := range n.Steps() {
		if err := s.Validate(); err != nil {
			return err
		}
		if n[s] < 1 {
			return fmt.Errorf("%w: the step %s has the limit %d, which is less than a minute", ErrInvalid, s, n[s])
		}
	}
	return nil
}

// Only returns an error that wraps [ErrInvalid] for the first step of n, in the order of
// [Nightly.Steps], that is not one of allowed: the steps of the producer that run on a schedule. A
// producer's Validate calls it.
func (n Nightly) Only(allowed ...Step) error {
	return only("nightly", n.Steps(), allowed)
}

// Steps returns the steps of n in the order of their targets, and then each step that is not a step
// of ergon in the order of its name.
func (n Nightly) Steps() []Step {
	out := make([]Step, 0, len(n))
	for _, s := range steps {
		if _, ok := n[s]; ok {
			out = append(out, s)
		}
	}
	var other []Step
	for s := range n {
		if !slices.Contains(steps, s) {
			other = append(other, s)
		}
	}
	slices.Sort(other)
	return append(out, other...)
}
