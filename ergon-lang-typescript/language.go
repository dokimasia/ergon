// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package typescript

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/typescript/baseline"
)

// Language is the name of TypeScript in configuration and in reports.
const Language workspace.Language = "typescript"

// Register adds TypeScript to c, with the js toolchain. It returns the error of
// [language.Register], which wraps [language.ErrUnknownToolchain] when c does not have the js
// toolchain, and [language.ErrRegistered] when c already has TypeScript.
func Register(c *language.Catalog) error {
	return language.Register(c, language.Declaration{Name: Language, Toolchain: javascript.Toolchain},
		baseline.Producer{})
}
