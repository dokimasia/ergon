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

// Name is the name of the producer of Rust in the lock, and of its section of .ergon.yaml, which is
// the name of the language.
const Name = "rust"

// templates are the templates of Rust: the configuration of clippy under managed/, and the
// fragments of the shared files under shared/.
//
//go:embed all:templates
var templates embed.FS

// Producer renders the files of Rust: the configuration of clippy, and the fragments of
// .editorconfig, .gitattributes, .gitignore and the Makefile. Its zero value is ready to use, and
// it is safe for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Contributor  = Producer{}
)

// Templates returns the templates of Rust: clippy.toml under managed/, and the fragments of the
// shared files under shared/.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section rust at the baseline: cargo-audit 0.22.2, the gate of lint, test and
// audit, the tests of every target, setup-rust-toolchain v2.0.0, and a limit of 30 minutes for the
// job check-rust on every runner and the toolchain of rust-toolchain.toml.
func (Producer) Options() language.Options {
	return &Options{
		Tools: Tools{CargoAudit: "cargo-audit@0.22.2"},
		Check: option.Check{option.StepLint, option.StepTest, option.StepAudit},
		Test:  option.Run{Args: []string{"--all-targets"}},
		Audit: option.Audit{Ignore: []string{}},
		CI: option.MatrixCI[Actions]{
			Actions: Actions{SetupRustToolchain: workflow.Action{
				Uses:    "actions-rust-lang/setup-rust-toolchain",
				Commit:  "ecabd13d1c56bd1345c230e542e9144811ad706f",
				Release: "v2.0.0",
			}},
			Runners:  option.Runners{},
			Versions: []string{},
			Timeout:  30,
		},
	}
}

// Contribution returns the part of Rust of the workflows for o, as [Options.Contribution] states
// it, and for the options at the baseline when o is not the section rust.
func (p Producer) Contribution(o language.Options) workflow.Contribution {
	opts, ok := o.(*Options)
	if !ok {
		opts, _ = p.Options().(*Options)
	}
	return opts.Contribution()
}
