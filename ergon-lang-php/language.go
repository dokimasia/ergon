// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package php

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
)

// Language is the name of PHP in configuration and in reports.
const Language workspace.Language = "php"

// Toolchain is the name of the toolchain of PHP in configuration and in reports.
const Toolchain workspace.Toolchain = "php"

// Register adds the toolchain of PHP and then PHP to c. It returns the first error of
// [language.RegisterToolchain] and [language.Register], which wraps [language.ErrRegistered] when
// c already has either name. When only the language fails, c keeps the toolchain.
func Register(c *language.Catalog) error {
	if err := language.RegisterToolchain(c, language.Toolchain{Name: Toolchain}); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain})
}
