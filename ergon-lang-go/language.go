// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
)

// Language is the name of Go in configuration and in reports.
const Language workspace.Language = "go"

// Toolchain is the name of the toolchain of Go in configuration and in reports.
const Toolchain workspace.Toolchain = "go"

// Register adds the toolchain of Go and then Go to c. It returns the first error of
// [language.RegisterToolchain] and [language.Register], which wraps [language.ErrRegistered] when
// c already has either name. When only the language fails, c keeps the toolchain.
func Register(c *language.Catalog) error {
	if err := language.RegisterToolchain(c, language.Toolchain{Name: Toolchain}); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain})
}
