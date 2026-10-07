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

// Name is the name of the producer of Python in the lock, and of its section of .ergon.yaml, which
// is the name of the language.
const Name = "python"

// templates are the templates of Python: the configuration of ruff under managed/, and the
// fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// Producer renders the files of Python: the configuration of ruff, and the fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile. Its zero value is ready to use, and
// it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of Python: ruff.toml under managed/, and the fragments of the
// shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section python at the baseline: uv 0.12.23 with the digests that its release
// states, ruff 0.16.10, mypy 2.4.0, pytest 9.1.1 and pip-audit 2.10.1, the root of the repository,
// the gate of lint, test and audit, and a limit of 30 minutes for the job check-python on every
// runner and the Python of pyproject.toml.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{
			UV: option.UV{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "9167d72b3319674b6303c4cbe071854bba13ebdf3d76b1a7cbdc175471fb66d6",
					option.DarwinARM64:  "50487ae565ccd96e499056b4674d438f4c53170202617b4c759defe0c6a1b544",
					option.WindowsAMD64: "75d05de6762778c31ee183398de7dd15093fad0ed90b1f236d8205ea5ec00c90",
				},
				Version: "0.12.23",
			}},
			Ruff:     "ruff@0.16.10",
			Mypy:     "mypy@2.4.0",
			Pytest:   "pytest@9.1.1",
			PipAudit: "pip-audit@2.10.1",
		},
		Paths: option.Paths{"."},
		Check: option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:  option.Run{Args: []string{}},
		Audit: option.Audit{Ignore: []string{}},
		CI:    option.MatrixCI[struct{}]{Runners: option.Runners{}, Versions: []string{}, Timeout: 30},
	}
}

// Contribution returns the part of Python of the workflows for o, as [Options.Contribution] states
// it, and for the options at the baseline when o is not the section python.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
