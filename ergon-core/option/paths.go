// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"slices"
)

// Paths is the key paths of a section: what the targets of its producer work on, such as the
// package patterns of each Go module, the paths of the sources, or the pathspecs of git of the
// scripts. The Makefile writes each path as a word of the shell.
type Paths []string

// Validate returns an error that wraps [ErrInvalid] for a path that is empty, spans lines, or that
// p names twice.
func (p Paths) Validate() error {
	for i, path := range p {
		if path == "" || slices.Contains(p[:i], path) {
			return fmt.Errorf("%w: paths %q, which is empty or named twice", ErrInvalid, path)
		}
	}
	return lines("paths", p)
}
