// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package java

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
)

// Language is the name of Java in configuration and in reports.
const Language workspace.Language = "java"

// Toolchain is the name of the jvm toolchain in configuration and in reports. Java and Kotlin
// name it.
const Toolchain workspace.Toolchain = "jvm"

// Register adds the jvm toolchain and then Java to c. It returns the first error of
// [language.RegisterToolchain] and [language.Register], which wraps [language.ErrRegistered] when
// c already has either name. When only the language fails, c keeps the toolchain.
func Register(c *language.Catalog) error {
	if err := language.RegisterToolchain(c, language.Toolchain{Name: Toolchain}); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain})
}
