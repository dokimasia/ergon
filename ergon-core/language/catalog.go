// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package language

import (
	"errors"
	"fmt"
	"iter"
	"slices"

	"go.dokimi.dev/ergon/core/workspace"
)

// ErrInvalidName is the error for a toolchain or a language whose name is not valid, as
// [workspace.Toolchain.Valid] and [workspace.Language.Valid] define it.
var ErrInvalidName = errors.New("language: invalid name")

// ErrRegistered is the error for a toolchain or a language whose name the catalog already has.
var ErrRegistered = errors.New("language: already registered")

// ErrUnknownToolchain is the error for a language whose toolchain the catalog does not have.
var ErrUnknownToolchain = errors.New("language: unknown toolchain")

// Toolchain states the facts about a build toolchain that the commands working on packages read.
// A toolchain package constructs exactly one.
type Toolchain struct {
	// Name identifies the toolchain in configuration and in reports.
	Name workspace.Toolchain
}

// Declaration states the facts about a language that every command reads. A language module
// constructs exactly one.
type Declaration struct {
	// Name identifies the language in configuration and in reports.
	Name workspace.Language

	// Toolchain names the toolchain that builds the packages of the language.
	Toolchain workspace.Toolchain
}

// Catalog is the set of toolchains and languages that a composition root registers. The zero
// value is an empty catalog, ready for registration.
//
// A catalog keeps the entries of each kind in the order of registration. A registration compares
// its name with every entry of its kind.
//
// # Concurrency
//
// [RegisterToolchain] and [Register] write the catalog and are not safe for concurrent use. The
// methods of a Catalog only read it, so they are safe for concurrent use after the last
// registration has returned.
type Catalog struct {
	// toolchains are the registered toolchains, in the order of registration.
	toolchains []Toolchain

	// languages are the registered languages, in the order of registration.
	languages []Declaration
}

// Languages returns an iterator over the languages registered before the call, in the order of
// registration. The iterator yields nothing for the zero value.
func (c *Catalog) Languages() iter.Seq[Declaration] {
	return slices.Values(c.languages)
}

// RegisterToolchain adds t to c. It returns an error that wraps [ErrInvalidName] for an invalid
// t.Name, and [ErrRegistered] when c has a toolchain of that name. c is unchanged when
// RegisterToolchain returns an error. RegisterToolchain panics for a nil c.
func RegisterToolchain(c *Catalog, t Toolchain) error {
	if !t.Name.Valid() {
		return fmt.Errorf("%w: toolchain %q", ErrInvalidName, t.Name)
	}
	if slices.ContainsFunc(c.toolchains, func(r Toolchain) bool { return r.Name == t.Name }) {
		return fmt.Errorf("%w: toolchain %q", ErrRegistered, t.Name)
	}
	c.toolchains = append(c.toolchains, t)
	return nil
}

// Register adds d to c. It returns an error that wraps [ErrInvalidName] for an invalid d.Name,
// [ErrRegistered] when c has a language of that name, and [ErrUnknownToolchain] when c has no
// toolchain named d.Toolchain. c is unchanged when Register returns an error. Register panics for
// a nil c.
func Register(c *Catalog, d Declaration) error {
	if !d.Name.Valid() {
		return fmt.Errorf("%w: language %q", ErrInvalidName, d.Name)
	}
	if slices.ContainsFunc(c.languages, func(r Declaration) bool { return r.Name == d.Name }) {
		return fmt.Errorf("%w: language %q", ErrRegistered, d.Name)
	}
	if !slices.ContainsFunc(c.toolchains, func(r Toolchain) bool { return r.Name == d.Toolchain }) {
		return fmt.Errorf("%w: %q, which language %q names", ErrUnknownToolchain, d.Toolchain, d.Name)
	}
	c.languages = append(c.languages, d)
	return nil
}
