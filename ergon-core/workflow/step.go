// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workflow

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// ErrInvalidStep is the error for a step that a workflow cannot contain.
var ErrInvalidStep = errors.New("workflow: invalid step")

// The rules of the names in a step.
var (
	// identifier matches the identifier of a step or of a job: a letter or '_', then letters,
	// digits, '_' and '-'.
	identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

	// input matches the name of an input of an action.
	input = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

	// variable matches the name of an environment variable.
	variable = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// Step is a step of a job: an action, which Uses names, or a command of bash, which Run states.
// The renderer writes the command as a literal block of YAML, one line of Run per line.
type Step struct {
	// Name is the name of the step in the log of the job, or empty for the name that GitHub
	// derives from the action or the command.
	Name string

	// ID identifies the step to the expressions of later steps, such as steps.pin.outputs.version,
	// or is empty.
	ID string

	// If is the condition of the step, or empty for a step that always runs.
	If string

	// Uses is the action of the step, and the zero value for a command.
	Uses Action

	// With are the inputs of the action, by name. A value may span lines.
	With map[string]string

	// Env are the environment variables of the step, by name.
	Env map[string]string

	// Run are the lines of the command, which bash runs, or nil for an action.
	Run []string
}

// Validate returns an error that wraps [ErrInvalidStep] for the first value of s that a workflow
// cannot contain:
//
//   - a Name or an If that spans lines
//   - an ID that is not a letter or '_' followed by letters, digits, '_' and '-'
//   - both or neither of an action and a command
//   - an action that is not valid, as [Action.Validate] states
//   - inputs of a command, or an input whose name is not letters, digits, '_' and '-'
//   - an environment variable whose name is not a letter or '_' followed by letters, digits and '_'
//   - a line of the command that is empty, spans lines, or ends in a space or a tab, and a first
//     line that starts with one, which would change the indentation of the block
//
// It reads the maps in the order of their keys, so it returns the same error for the same step.
func (s *Step) Validate() error {
	if strings.ContainsAny(s.Name, "\r\n") || strings.ContainsAny(s.If, "\r\n") {
		return fmt.Errorf("%w: the name %q or the condition %q spans lines", ErrInvalidStep, s.Name, s.If)
	}
	if s.ID != "" && !identifier.MatchString(s.ID) {
		return fmt.Errorf("%w: id %q, which is not an identifier", ErrInvalidStep, s.ID)
	}
	action := s.Uses != Action{}
	if action == (len(s.Run) > 0) {
		return fmt.Errorf("%w: %q has both or neither of an action and a command", ErrInvalidStep, s.Name)
	}
	if action {
		if err := s.Uses.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidStep, err)
		}
	}
	if !action && len(s.With) > 0 {
		return fmt.Errorf("%w: the command %q has inputs, which only an action takes", ErrInvalidStep, s.Run[0])
	}
	for _, name := range slices.Sorted(maps.Keys(s.With)) {
		if !input.MatchString(name) {
			return fmt.Errorf("%w: input %q of %s, which is not a name of an input", ErrInvalidStep, name, s.Uses.Uses)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(s.Env)) {
		if !variable.MatchString(name) {
			return fmt.Errorf("%w: environment variable %q, which is not a name of a variable", ErrInvalidStep, name)
		}
	}
	for i, line := range s.Run {
		if line == "" || strings.ContainsAny(line, "\r\n") || strings.TrimRight(line, " \t") != line {
			return fmt.Errorf("%w: line %d of the command, %q, is empty, spans lines or ends in a blank",
				ErrInvalidStep, i+1, line)
		}
	}
	if len(s.Run) > 0 && strings.TrimLeft(s.Run[0], " \t") != s.Run[0] {
		return fmt.Errorf("%w: the command opens with a blank, %q, which changes the indentation of its block",
			ErrInvalidStep, s.Run[0])
	}
	return nil
}
