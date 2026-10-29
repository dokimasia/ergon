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

// Options returns the section python at the baseline: the release of uv with the digests that it
// states, the releases of ruff, mypy, pytest and pip-audit, the root of the repository, the gate of
// lint, test and audit, no generators, a scan of every package of uv.lock, and a limit of 30 minutes
// for the job check-python on every runner and the Python of pyproject.toml.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{
			UV: option.UV{Binary: option.Binary{
				SHA256: map[option.Platform]string{
					option.LinuxAMD64:   "f68d291c75783122dad58ab77def0e13f3233f358731db5d6d7e8715603b928f",
					option.DarwinARM64:  "743f85035a1996ecb350659e01cd9faea9709829be09b1b4b03c73eaf3c9cd73",
					option.WindowsAMD64: "a38b8d1902b7cfbc76b423ba1bb456f623f42064cdc72628b6075b89afd1019a",
				},
				Version: "0.13.5",
			}},
			Ruff:     "ruff@0.17.1",
			Mypy:     "mypy@2.4.0",
			Pytest:   "pytest@9.1.1",
			PipAudit: "pip-audit@2.10.1",
		},
		Paths:    option.Paths{"."},
		Check:    option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:     option.Run{Args: []string{}},
		Generate: option.Generate{Commands: [][]string{}},
		Audit:    Audit{Audit: option.Audit{Ignore: []string{}}, Skip: []string{}},
		CI: option.MatrixCI[struct{}]{
			Runners:  option.Runners{},
			Versions: []string{},
			Steps:    []workflow.Step{},
			Timeout:  30,
		},
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
