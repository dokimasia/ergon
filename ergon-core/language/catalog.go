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

// ErrUnknownRole is the error for a role that implements none of the role interfaces of this
// package, such as a value where a pointer implements the role.
var ErrUnknownRole = errors.New("language: unknown role")

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

// toolchainEntry is a registered toolchain and the roles it implements.
type toolchainEntry struct {
	// toolchain is the toolchain.
	toolchain Toolchain

	// roles are the role implementations of the toolchain, in the order of registration.
	roles []any
}

// entry is a registered language and the roles it implements.
type entry struct {
	// declaration is the language.
	declaration Declaration

	// roles are the role implementations of the language, in the order of registration.
	roles []any
}

// Catalog is the set of toolchains and languages that a composition root registers. The zero
// value is an empty catalog, ready for registration.
//
// A catalog keeps the entries of each kind in the order of registration. A registration and a
// lookup compare a name with every entry of its kind.
//
// # Concurrency
//
// [RegisterToolchain] and [Register] write the catalog and are not safe for concurrent use. The
// methods of a Catalog, [Role] and [ToolchainRole] only read it, so they are safe for concurrent
// use after the last registration has returned.
type Catalog struct {
	// toolchains are the registered toolchains, in the order of registration.
	toolchains []toolchainEntry

	// languages are the registered languages, in the order of registration.
	languages []entry
}

// Languages returns an iterator over the languages registered before the call, in the order of
// registration. The iterator yields nothing for the zero value.
func (c *Catalog) Languages() iter.Seq[Declaration] {
	languages := c.languages
	return func(yield func(Declaration) bool) {
		for _, e := range languages {
			if !yield(e.declaration) {
				return
			}
		}
	}
}

// Language returns the declaration of the language named name, and reports whether the catalog
// has it.
func (c *Catalog) Language(name workspace.Language) (Declaration, bool) {
	i := slices.IndexFunc(c.languages, func(e entry) bool { return e.declaration.Name == name })
	if i < 0 {
		return Declaration{}, false
	}
	return c.languages[i].declaration, true
}

// Toolchain returns the toolchain named name, and reports whether the catalog has it.
func (c *Catalog) Toolchain(name workspace.Toolchain) (Toolchain, bool) {
	i := slices.IndexFunc(c.toolchains, func(e toolchainEntry) bool { return e.toolchain.Name == name })
	if i < 0 {
		return Toolchain{}, false
	}
	return c.toolchains[i].toolchain, true
}

// RegisterToolchain adds t to c, with the roles that the toolchain implements, such as the
// [Initializer] of the files that the languages of a shared toolchain share. It returns an error
// that wraps [ErrInvalidName] for an invalid t.Name, [ErrRegistered] when c has a toolchain of
// that name, and [ErrUnknownRole] for a role that implements no role interface. c is unchanged
// when RegisterToolchain returns an error. RegisterToolchain panics for a nil c.
func RegisterToolchain(c *Catalog, t Toolchain, roles ...any) error {
	if !t.Name.Valid() {
		return fmt.Errorf("%w: toolchain %q", ErrInvalidName, t.Name)
	}
	if _, ok := c.Toolchain(t.Name); ok {
		return fmt.Errorf("%w: toolchain %q", ErrRegistered, t.Name)
	}
	if err := checkRoles(roles, "toolchain", string(t.Name)); err != nil {
		return err
	}
	c.toolchains = append(c.toolchains, toolchainEntry{toolchain: t, roles: slices.Clone(roles)})
	return nil
}

// Register adds d to c, with the roles that the language implements, such as an [Initializer].
// It returns an error that wraps [ErrInvalidName] for an invalid d.Name, [ErrRegistered] when c
// has a language of that name, [ErrUnknownToolchain] when c has no toolchain named d.Toolchain,
// and [ErrUnknownRole] for a role that implements no role interface. c is unchanged when Register
// returns an error. Register panics for a nil c.
func Register(c *Catalog, d Declaration, roles ...any) error {
	if !d.Name.Valid() {
		return fmt.Errorf("%w: language %q", ErrInvalidName, d.Name)
	}
	if _, ok := c.Language(d.Name); ok {
		return fmt.Errorf("%w: language %q", ErrRegistered, d.Name)
	}
	if _, ok := c.Toolchain(d.Toolchain); !ok {
		return fmt.Errorf("%w: %q, which language %q names", ErrUnknownToolchain, d.Toolchain, d.Name)
	}
	if err := checkRoles(roles, "language", string(d.Name)); err != nil {
		return err
	}
	c.languages = append(c.languages, entry{declaration: d, roles: slices.Clone(roles)})
	return nil
}

// Role returns the first role of the language named name that implements R, in the order of
// registration, and reports whether one does. It reports false for a language that c does not
// have.
func Role[R any](c *Catalog, name workspace.Language) (R, bool) {
	i := slices.IndexFunc(c.languages, func(e entry) bool { return e.declaration.Name == name })
	if i < 0 {
		var none R
		return none, false
	}
	return first[R](c.languages[i].roles)
}

// ToolchainRole returns the first role of the toolchain named name that implements R, in the
// order of registration, and reports whether one does. It reports false for a toolchain that c
// does not have.
func ToolchainRole[R any](c *Catalog, name workspace.Toolchain) (R, bool) {
	i := slices.IndexFunc(c.toolchains, func(e toolchainEntry) bool { return e.toolchain.Name == name })
	if i < 0 {
		var none R
		return none, false
	}
	return first[R](c.toolchains[i].roles)
}

// checkRoles returns an error that wraps [ErrUnknownRole] for the first of roles that implements
// no role interface, naming the kind and the name of its owner, and nil when every role
// implements one.
func checkRoles(roles []any, kind, name string) error {
	for _, role := range roles {
		if _, ok := role.(Initializer); !ok {
			return fmt.Errorf("%w: %T of %s %q", ErrUnknownRole, role, kind, name)
		}
	}
	return nil
}

// first returns the first of roles that implements R, and reports whether one does.
func first[R any](roles []any) (R, bool) {
	for _, role := range roles {
		if r, ok := role.(R); ok {
			return r, true
		}
	}
	var none R
	return none, false
}
