// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/workflow"
)

// runner matches the label of a runner image, such as ubuntu-26.04.
var runner = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// Runners are runner images of GitHub Actions, each pinned to a version of its system, such as
// ubuntu-26.04 or windows-2025.
type Runners []string

// Validate returns an error that wraps [ErrInvalid] for a runner that is not a label of letters,
// digits, '.', '_' and '-', and for a runner that r names twice.
func (r Runners) Validate() error {
	for i, label := range r {
		if !runner.MatchString(label) || slices.Contains(r[:i], label) {
			return fmt.Errorf("%w: runner %q, which is not the label of a runner or is named twice", ErrInvalid, label)
		}
	}
	return nil
}

// CI is the key ci of a section whose jobs run on the runners that the job states: the Linux
// runner for a check of text, or every runner of the section github. A is the struct of the pins
// of the actions of the producer's jobs, one [go.dokimi.dev/ergon/core/workflow.Action] per field,
// and struct{} for a producer whose jobs run no action of their own.
type CI[A any] struct {
	// Actions are the pins of the actions of the producer's jobs.
	Actions A `yaml:"actions"`

	// Timeout is the limit of each job of the producer in minutes, at least 1.
	Timeout int `yaml:"timeout"`
}

// Validate returns an error that wraps [ErrInvalid] for a Timeout below 1. The caller validates
// each action of Actions.
func (c CI[A]) Validate() error {
	return timeout(c.Timeout)
}

// RunnerCI is the key ci of the section of a toolchain without a runtime version, such as Bash:
// the pins of the actions of its jobs, the runners of their matrix, the steps that follow the
// setup of the toolchain, and their timeout.
type RunnerCI[A any] struct {
	// Actions are the pins of the actions of the producer's jobs.
	Actions A `yaml:"actions"`

	// Runners are the runner images of the matrix of the producer's jobs. An empty list selects
	// every runner of the section github.
	Runners Runners `yaml:"runners"`

	// Steps are the steps of the repository that each job of the producer runs after the setup of
	// the toolchain and before ergon.
	Steps []workflow.Step `yaml:"steps" doc:"The steps that each job of the toolchain runs after the setup of the toolchain and before ergon, such as the installation of a compiler that the tests run. A step has the keys name, id, if and env, and either uses with its inputs under with, or run. uses is an action as uses, commit and release, and each input is a string. run is the list of the lines of a command of bash."`

	// Timeout is the limit of each job of the producer in minutes, at least 1.
	Timeout int `yaml:"timeout"`
}

// Validate returns an error that wraps [ErrInvalid] for runners that are not valid, as
// [Runners.Validate] states, for a step that is not valid, as [workflow.Step.Validate] states, and
// for a Timeout below 1. The producer of the GitHub files rejects a runner that its own section does
// not list.
func (c RunnerCI[A]) Validate() error {
	if err := c.Runners.Validate(); err != nil {
		return err
	}
	if err := validateSteps(c.Steps); err != nil {
		return err
	}
	return timeout(c.Timeout)
}

// MatrixCI is the key ci of the section of a toolchain with runtime versions, such as Go: the pins
// of the actions of its jobs, the runners and the runtime versions of their matrix, the steps that
// follow the setup of the toolchain, and their timeout.
type MatrixCI[A any] struct {
	// Actions are the pins of the actions of the producer's jobs, such as the setup of its
	// toolchain.
	Actions A `yaml:"actions"`

	// Runners are the runner images of the matrix of the producer's jobs. An empty list selects
	// every runner of the section github.
	Runners Runners `yaml:"runners"`

	// Versions are the runtime versions of the matrix of the producer's jobs. An empty list runs the
	// version that the toolchain's pin file states.
	Versions []string `yaml:"versions"`

	// Steps are the steps of the repository that each job of the producer runs after the setup of
	// the toolchain and before ergon.
	Steps []workflow.Step `yaml:"steps" doc:"The steps that each job of the toolchain runs after the setup of the toolchain and before ergon, such as the installation of a compiler that the tests run. A step has the keys name, id, if and env, and either uses with its inputs under with, or run. uses is an action as uses, commit and release, and each input is a string. run is the list of the lines of a command of bash."`

	// Timeout is the limit of each job of the producer in minutes, at least 1.
	Timeout int `yaml:"timeout"`
}

// Validate returns an error that wraps [ErrInvalid] for runners that are not valid, as
// [Runners.Validate] states, a version that is empty, spans lines or that Versions names twice, a
// step that is not valid, as [workflow.Step.Validate] states, and a Timeout below 1.
func (c MatrixCI[A]) Validate() error {
	if err := c.Runners.Validate(); err != nil {
		return err
	}
	for i, v := range c.Versions {
		if v == "" || strings.ContainsAny(v, "\r\n") || slices.Contains(c.Versions[:i], v) {
			return fmt.Errorf("%w: version %q, which is empty, spans lines or is named twice", ErrInvalid, v)
		}
	}
	if err := validateSteps(c.Steps); err != nil {
		return err
	}
	return timeout(c.Timeout)
}

// validateSteps returns an error that wraps [ErrInvalid] and [workflow.ErrInvalidStep], with the
// number of the step, for the first step of steps that is not valid, as [workflow.Step.Validate]
// states.
func validateSteps(steps []workflow.Step) error {
	for i := range steps {
		if err := steps[i].Validate(); err != nil {
			return fmt.Errorf("%w: step %d: %w", ErrInvalid, i+1, err)
		}
	}
	return nil
}

// timeout returns an error that wraps [ErrInvalid] for minutes below 1.
func timeout(minutes int) error {
	if minutes < 1 {
		return fmt.Errorf("%w: timeout %d, which is less than a minute", ErrInvalid, minutes)
	}
	return nil
}
