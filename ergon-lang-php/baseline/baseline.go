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

// Name is the name of the producer of PHP in the lock, and of its section of .ergon.yaml, which is
// the name of the language.
const Name = "php"

// templates are the templates of PHP: the configuration of PHPStan under managed/, and the
// fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// tools are the tools of the section php at the baseline: a release of PHPStan with releases of
// phpstan-strict-rules and of the extension installer, and a release of PHP-CS-Fixer from its
// package without dependencies. [Tools.Validate] requires the package of each.
var tools = Tools{
	PHPStan:            "phpstan/phpstan@2.2.17",
	StrictRules:        "phpstan/phpstan-strict-rules@2.0.12",
	ExtensionInstaller: "phpstan/extension-installer@1.4.3",
	PHPCSFixer:         "php-cs-fixer/shim@3.95.27",
}

// Producer renders the files of PHP: the configuration of PHPStan, and the fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile. Its zero value is ready to use, and
// it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of PHP: phpstan.dist.neon under managed/, and the fragments of
// the shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section php at the baseline: the releases of PHPStan, phpstan-strict-rules,
// the extension installer and PHP-CS-Fixer, the root of the repository, the gate of lint, test and
// audit, no generators, the release of setup-php, and a limit of 30 minutes for the job check-php on
// every runner and the PHP of .php-version.
func (Producer) Options() language.Options {
	return &Options{
		Tools:    tools,
		Paths:    option.Paths{"."},
		Check:    option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:     option.Run{Args: []string{}},
		Generate: option.Generate{Command: []string{}, Args: []string{}},
		CI: option.MatrixCI[Actions]{
			Actions: Actions{SetupPHP: workflow.Action{
				Uses:    "shivammathur/setup-php",
				Commit:  "f3e473d116dcccaddc5834248c87452386958240",
				Release: "2.37.2",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
			Timeout:  30,
		},
	}
}

// Contribution returns the part of PHP of the workflows for o, as [Options.Contribution] states it,
// and for the options at the baseline when o is not the section php.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
