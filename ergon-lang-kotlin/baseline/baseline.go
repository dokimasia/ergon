// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"embed"
	"io/fs"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// Name is the name of the producer of Kotlin in the lock, and of its section of .ergon.yaml, which
// is the name of the language.
const Name = "kotlin"

// templates are the templates of Kotlin: the fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// tools are the tools of the section kotlin at the baseline: ktlint 1.8.0. [Tools.Validate]
// requires its package.
var tools = Tools{Ktlint: "com.pinterest.ktlint:ktlint-cli@1.8.0"}

// Producer renders the files of Kotlin: the fragments of .editorconfig, with the configuration of
// ktlint, .gitattributes, .gitignore and the Makefile. The jvm toolchain renders what Kotlin shares
// with Java. Its zero value is ready to use, and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of Kotlin: the fragments of the shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section kotlin at the baseline: ktlint 1.8.0, the gate of lint, test and
// audit, and ./gradlew test without arguments.
func (Producer) Options() language.Options {
	return &Options{
		Tools: tools,
		Check: option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:  option.Run{Args: []string{}},
	}
}

// Contribution returns the part of Kotlin of the workflows for o, as [Options.Contribution] states
// it, and for the options at the baseline when o is not the section kotlin.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
