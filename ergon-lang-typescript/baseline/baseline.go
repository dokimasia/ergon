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

// Name is the name of the producer of TypeScript in the lock, and of its section of .ergon.yaml,
// which is the name of the language.
const Name = "typescript"

// templates are the templates of TypeScript: the fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// Producer renders the files of TypeScript: the fragments of .editorconfig, .gitignore and the
// Makefile. The js toolchain renders what TypeScript shares with JavaScript. Its zero value is
// ready to use, and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of TypeScript: the fragments of the shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section typescript at the baseline: tsc 7.0.2, the gate of lint, test and
// audit, and npm test without arguments.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{TypeScript: "typescript@7.0.2"},
		Check: option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:  option.Run{Args: []string{}},
	}
}

// Contribution returns the part of TypeScript of the workflows for o, as [Options.Contribution]
// states it, and for the options at the baseline when o is not the section typescript.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
