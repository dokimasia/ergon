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
					option.LinuxAMD64:   "f29c3e94ffabd7e6f3ba521dba169a9b7ec12100bf3257087f0f24f6123f3645",
					option.DarwinARM64:  "60fbd5c8a1ed681845eea28d9d622fcde03c51e91a43465b64a1e17db1ed3472",
					option.WindowsAMD64: "a2a8820d5eea32f08ef467ef3b7f66883a9b076d5737a2fc2fbb5f458d1bb50f",
				},
				Version: "0.13.6",
			}},
			Ruff:     "ruff@0.17.3",
			Mypy:     "mypy@2.4.1",
			Pytest:   "pytest@9.2.0",
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
