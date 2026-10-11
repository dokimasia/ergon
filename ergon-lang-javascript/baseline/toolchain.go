// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"io/fs"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// ToolchainName is the name of the producer of the js toolchain in the lock, and of its section of
// .ergon.yaml, which is the name of the toolchain.
const ToolchainName = "js"

// Choices are the package manager and the toolkit of the section js, which the templates of the js
// toolchain read as .Data, and the templates of JavaScript and TypeScript as .Toolchain.
type Choices struct {
	// PNPM reports the package manager pnpm, and npm when false.
	PNPM bool

	// VitePlus reports the toolkit vite-plus, and biome when false.
	VitePlus bool
}

// Toolchain renders the files that JavaScript and TypeScript share, once for a repository with
// either or both: the configuration of Biome, or the .prettierignore of vite-plus, and the
// fragments of .gitattributes, .gitignore and the Makefile with the targets fmt-js, lint-js and
// audit-js. Its zero value is ready to use, and it is safe for concurrent use.
type Toolchain struct{}

var (
	_ language.Producer     = Toolchain{}
	_ language.Configurable = Toolchain{}
	_ language.Calculator   = Toolchain{}
	_ language.Contributor  = Toolchain{}
)

// Templates returns the templates of the js toolchain: biome.json and .prettierignore under
// managed/, and the fragments of the shared files under shared/.
func (Toolchain) Templates() fs.FS {
	// templates has the directory templates/js, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates/"+ToolchainName)
	return sub
}

// Options returns the section js at the baseline: npm and the toolkit biome, the release of Biome,
// the root of the repository, a scan that fails on every known vulnerability, the releases of
// setup-node and pnpm/setup, and a limit of 30 minutes for the jobs of JavaScript and TypeScript
// on every runner and the Node.js of package.json.
func (Toolchain) Options() language.Options {
	return &ToolchainOptions{
		PackageManager: option.PackageManagerNPM,
		Toolkit:        ToolkitBiome,
		Tools:          ToolchainTools{Biome: "@biomejs/biome@2.5.15"},
		Paths:          option.Paths{"."},
		Audit:          option.Threshold{Severity: option.SeverityLow},
		CI: option.MatrixCI[ToolchainActions]{
			Actions: ToolchainActions{
				SetupNode: workflow.Action{
					Uses:    "actions/setup-node",
					Commit:  "820762786026740c76f36085b0efc47a31fe5020",
					Release: "v7.0.0",
				},
				SetupPNPM: workflow.Action{
					Uses:    "pnpm/setup",
					Commit:  "a0881977cd8ea73b1ed212236c99ca10dcc80cf4",
					Release: "v3.1.1",
				},
			},
			Runners:  option.Runners{},
			Versions: []string{},
			Steps:    []workflow.Step{},
			Timeout:  30,
		},
	}
}

// Data returns the [Choices] of o, and those of the options at the baseline when o is not the
// section js. It returns no error.
func (t Toolchain) Data(_ *language.Answers, o language.Options, _ *workflow.Contribution) (any, error) {
	opts, ok := o.(*ToolchainOptions)
	if !ok {
		opts, _ = t.Options().(*ToolchainOptions)
	}
	return Choices{
		PNPM:     opts.PackageManager == option.PackageManagerPNPM,
		VitePlus: opts.Toolkit == ToolkitVitePlus,
	}, nil
}

// Contribution returns the part of the js toolchain of the workflows for o, as
// [ToolchainOptions.Contribution] states it, and for the options at the baseline when o is not the
// section js.
func (t Toolchain) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*ToolchainOptions)
	if !ok {
		opts, _ = t.Options().(*ToolchainOptions)
	}
	return opts.Contribution()
}
