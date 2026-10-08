// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package producer

import "go.dokimi.dev/ergon/core/language"

// Options is a function of the package, and no method of a producer.
func Options() language.Options {
	return &Settings{Tools: Tools{Lint: "example.com/lint/cmd/lint@v0.9.0"}}
}

// Generic is a producer of a generic type.
type Generic[T any] struct{}

// Options returns the baseline of Generic.
func (Generic[T]) Options() language.Options {
	return &Settings{Tools: Tools{Lint: "example.com/lint/cmd/lint@v1.1.0"}}
}

// Other is a producer whose Options method has a pointer receiver.
type Other struct{}

// Options returns the baseline of Other.
func (*Other) Options() language.Options {
	return &Settings{Tools: Tools{Lint: "example.com/lint/cmd/lint@v1.0.0"}}
}

// Local is a producer whose Options method returns a local variable.
type Local struct{}

// Options returns a local variable.
func (Local) Options() language.Options {
	s := &Settings{Tools: Tools{Lint: "example.com/lint/cmd/lint@v1.2.0"}}
	return s
}

// Bodiless is a producer whose Options method has no body.
type Bodiless struct{}

// Options has no body.
func (Bodiless) Options() language.Options

// Empty is a producer whose Options method has no statement.
type Empty struct{}

// Options has no statement.
func (Empty) Options() language.Options {}

// Pair is a producer whose Options method returns two results.
type Pair struct{}

// Options returns the baseline and an error.
func (Pair) Options() (language.Options, error) {
	return &Settings{Tools: Tools{Lint: "example.com/lint/cmd/lint@v1.2.0"}}, nil
}

// Panicking is a producer whose Options method ends in no return.
type Panicking struct{}

// Options panics.
func (Panicking) Options() language.Options {
	panic("no baseline")
}
