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

// Name is the name of the producer of Bash in the lock, and of its section of .ergon.yaml, which is
// the name of the language.
const Name = "bash"

// templates are the templates of Bash: the configuration of shellcheck under managed/, and the
// fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// Producer renders the files of Bash: the configuration of shellcheck, and the fragments of
// .editorconfig, .gitattributes and the Makefile. Its zero value is ready to use, and it is safe
// for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of Bash: .shellcheckrc under managed/, and the fragments of the
// shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section bash at the baseline: the release of shellcheck with the digests of
// its assets, the scripts *.sh and *.bash, the gate of lint, no generators, and a limit of 30
// minutes for the job check-bash on every runner.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{Shellcheck: Shellcheck{Binary: option.Binary{
			SHA256: map[option.Platform]string{
				option.LinuxAMD64:   "b7af85e41cc99489dcc21d66c6d5f3685138f06d34651e6d34b42ec6d54fe6f6",
				option.DarwinARM64:  "339b930feb1ea764467013cc1f72d09cd6b869ebf1013296ba9055ab2ffbd26f",
				option.WindowsAMD64: "8a4e35ab0b331c85d73567b12f2a444df187f483e5079ceffa6bda1faa2e740e",
			},
			Version: "0.11.0",
		}}},
		Paths:    option.Paths{"*.sh", "*.bash"},
		Check:    option.Check{option.StepLint},
		Generate: option.Generate{Command: []string{}, Args: []string{}},
		CI:       option.RunnerCI[struct{}]{Runners: option.Runners{}, Timeout: 30},
	}
}

// Contribution returns the part of Bash of the workflows for o, as [Options.Contribution] states
// it, and for the options at the baseline when o is not the section bash.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
