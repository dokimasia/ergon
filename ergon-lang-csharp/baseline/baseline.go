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

// Name is the name of the producer of C# in the lock, and of its section of .ergon.yaml, which is
// the name of the language.
const Name = "csharp"

// templates are the templates of C#: the configuration of the analyzers under managed/, and the
// fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// Producer renders the files of C#: the configuration of the analyzers of .NET, and the fragments
// of .editorconfig, .gitattributes, .gitignore and the Makefile. Its zero value is ready to use,
// and it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of C#: .globalconfig under managed/, and the fragments of the
// shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section csharp at the baseline: the gate of lint, test and audit, no
// generators, a scan that fails on every known vulnerability, the release of setup-dotnet, and a
// limit of 30 minutes for the job check-csharp on every runner and the SDK of global.json.
func (Producer) Options() language.Options {
	return &Options{
		Check:    option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:     option.Run{Args: []string{}},
		Generate: option.Generate{Command: []string{}, Args: []string{}},
		Audit:    option.Threshold{Severity: option.SeverityLow},
		CI: option.MatrixCI[Actions]{
			Actions: Actions{SetupDotnet: workflow.Action{
				Uses:    "actions/setup-dotnet",
				Commit:  "a98b56852c35b8e3190ac28c8c2271da59106c68",
				Release: "v6.0.0",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
			Timeout:  30,
		},
	}
}

// Contribution returns the part of C# of the workflows for o, as [Options.Contribution] states it,
// and for the options at the baseline when o is not the section csharp.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
