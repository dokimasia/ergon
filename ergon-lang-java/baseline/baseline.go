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

// Name is the name of the producer of Java in the lock, and of its section of .ergon.yaml, which
// is the name of the language.
const Name = "java"

// templates are the templates of Java under java/, and of the jvm toolchain under jvm/. Each tree
// has the managed files under managed/ and the fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// tools are the tools of the section java at the baseline: a release of PMD. [Tools.Validate]
// requires its package.
var tools = Tools{PMD: "net.sourceforge.pmd:pmd-java@7.28.0"}

// Producer renders the files of Java: the Gradle init script that lints the Java sources, and the
// fragments of .editorconfig, .gitattributes, .gitignore and the Makefile. The jvm toolchain
// renders what Java shares with Kotlin. Its zero value is ready to use, and it is safe for
// concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of Java: gradle/ergon-java.init.gradle.kts under managed/, and
// the fragments of the shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates/java, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates/"+Name)
	return sub
}

// Options returns the section java at the baseline: the release of PMD, the gate of lint, test and
// audit, ./gradlew test without arguments, and no generators.
func (Producer) Options() language.Options {
	return &Options{
		Tools:    tools,
		Check:    option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:     option.Run{Args: []string{}},
		Generate: option.Generate{Command: []string{}, Args: []string{}},
	}
}

// Contribution returns the part of Java of the workflows for o, as [Options.Contribution] states
// it, and for the options at the baseline when o is not the section java.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
