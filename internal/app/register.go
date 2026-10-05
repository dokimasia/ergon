// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package app

import (
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/bash"
	"go.dokimi.dev/ergon/lang/csharp"
	golang "go.dokimi.dev/ergon/lang/go"
	"go.dokimi.dev/ergon/lang/java"
	"go.dokimi.dev/ergon/lang/javascript"
	"go.dokimi.dev/ergon/lang/kotlin"
	"go.dokimi.dev/ergon/lang/php"
	"go.dokimi.dev/ergon/lang/python"
	"go.dokimi.dev/ergon/lang/rust"
	"go.dokimi.dev/ergon/lang/terraform"
	"go.dokimi.dev/ergon/lang/typescript"
)

// registrations are the Register functions of the language modules, in the order that [Register]
// calls them. Java precedes Kotlin and JavaScript precedes TypeScript, because Kotlin and
// TypeScript name the toolchains that Java and JavaScript register.
var registrations = []func(*language.Catalog) error{
	csharp.Register,
	java.Register,
	kotlin.Register,
	php.Register,
	javascript.Register,
	typescript.Register,
	golang.Register,
	python.Register,
	rust.Register,
	terraform.Register,
	bash.Register,
}

// Register adds the toolchains and the languages of ergon to c, one language module at a time.
// It stops at the first error of a module's Register and returns it, such as an error that wraps
// [language.ErrRegistered] for a catalog that already has one of the names.
func Register(c *language.Catalog) error {
	for _, register := range registrations {
		if err := register(c); err != nil {
			return err
		}
	}
	return nil
}
