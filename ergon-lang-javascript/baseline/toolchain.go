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

// Toolchain renders the files that JavaScript and TypeScript share, once for a repository with
// either or both: the configuration of Biome, and the fragments of .gitattributes, .gitignore and
// the Makefile with the targets fmt-js, lint-js and audit-js. Its zero value is ready to use, and
// it is safe for concurrent use.
type Toolchain struct{}

var (
	_ language.Producer     = Toolchain{}
	_ language.Configurable = Toolchain{}
	_ language.Contributor  = Toolchain{}
)

// Templates returns the templates of the js toolchain: biome.json under managed/, and the
// fragments of the shared files under shared/.
func (Toolchain) Templates() fs.FS {
	// templates has the directory templates/js, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates/"+ToolchainName)
	return sub
}

// Options returns the section js at the baseline: the release of Biome, the root of the repository,
// a scan that fails on every known vulnerability, the release of setup-node, and a limit of 30
// minutes for the jobs of JavaScript and TypeScript on every runner and the Node.js of package.json.
func (Toolchain) Options() language.Options {
	return &ToolchainOptions{
		Tools: ToolchainTools{Biome: "@biomejs/biome@2.5.15"},
		Paths: option.Paths{"."},
		Audit: option.Threshold{Severity: option.SeverityLow},
		CI: option.MatrixCI[ToolchainActions]{
			Actions: ToolchainActions{SetupNode: workflow.Action{
				Uses:    "actions/setup-node",
				Commit:  "820762786026740c76f36085b0efc47a31fe5020",
				Release: "v7.0.0",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
			Steps:    []workflow.Step{},
			Timeout:  30,
		},
	}
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
