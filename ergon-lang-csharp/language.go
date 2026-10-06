// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package csharp

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/csharp/baseline"
)

// Language is the name of C# in configuration and in reports.
const Language workspace.Language = "csharp"

// Toolchain is the name of the toolchain of C# in configuration and in reports.
const Toolchain workspace.Toolchain = "csharp"

// Register adds the toolchain of C# and then C# to c. It returns the first error of
// [language.RegisterToolchain] and [language.Register], which wraps [language.ErrRegistered] when
// c already has either name. When only the language fails, c keeps the toolchain.
func Register(c *language.Catalog) error {
	if err := language.RegisterToolchain(c, language.Toolchain{Name: Toolchain}); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain},
		baseline.Initializer())
}
