// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package kotlin

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/kotlin/baseline"
)

// Language is the name of Kotlin in configuration and in reports.
const Language workspace.Language = "kotlin"

// Register adds Kotlin to c, with the jvm toolchain. It returns the error of [language.Register],
// which wraps [language.ErrUnknownToolchain] when c does not have the jvm toolchain, and
// [language.ErrRegistered] when c already has Kotlin.
func Register(c *language.Catalog) error {
	return language.Register(c, language.Declaration{Name: Language, Toolchain: java.Toolchain}, baseline.Producer{})
}
