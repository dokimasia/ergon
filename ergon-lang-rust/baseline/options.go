// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/workflow"
)

// toolchain is the file that pins the toolchain of Rust of a repository.
const toolchain = "rust-toolchain.toml"

// steps are the steps of the gate of Rust, which the key check of the section names.
var steps = []option.Step{option.StepLint, option.StepTest, option.StepAudit}

// Options are the options of the section rust of .ergon.yaml.
type Options struct {
	// Tools are the tools of the targets of Rust.
	Tools Tools `yaml:"tools" doc:"The tools of the targets of Rust, which ergon tool run installs with cargo install --locked, as <crate>@<version>."`

	// Check are the steps of check-rust.
	Check option.Check `yaml:"check" doc:"The steps that check-rust runs, in order: lint, test or audit."`

	// Test are the options of test-rust.
	Test option.Run `yaml:"test" doc:"test-rust runs cargo test with args."`

	// Audit are the options of audit-rust.
	Audit option.Audit `yaml:"audit" doc:"audit-rust scans Cargo.lock with cargo-audit, and accepts each advisory of ignore, by its identifier, such as RUSTSEC-2024-0001."`

	// CI are the pin of the setup of Rust, the runners, the toolchains and the limit of the job
	// check-rust.
	CI option.MatrixCI[Actions] `yaml:"ci" doc:"The pin of setup-rust-toolchain, the runners and the toolchains of Rust of the job check-rust, and its limit in minutes. Empty runners select every runner of the section github, and empty versions the toolchain of rust-toolchain.toml."`
}

// Tools are the tools of the targets of Rust.
type Tools struct {
	// CargoAudit scans the dependencies for known vulnerabilities.
	CargoAudit option.Crate `yaml:"cargo-audit" doc:"The vulnerability scan of audit-rust, over Cargo.lock."`
}

// Actions are the pins of the actions of the job check-rust.
type Actions struct {
	// SetupRustToolchain installs Rust.
	SetupRustToolchain workflow.Action `yaml:"setup-rust-toolchain" doc:"The action that installs the toolchain of Rust of rust-toolchain.toml or of the matrix, with rustfmt and clippy."`
}

// Validate returns an error that wraps [option.ErrInvalid] for a step of check that Rust does not
// have. The command checks each option by the Validate method of its type before it calls Validate.
func (o *Options) Validate() error {
	return o.Check.Only(steps...)
}

// Contribution returns the part of Rust of the workflows for o:
//
//   - The job check-rust runs make check-rust on the runners of o, once the repository has
//     rust-toolchain.toml. setup-rust-toolchain installs the toolchain of rust-toolchain.toml, or
//     the toolchain of the matrix where o lists versions, with rustfmt and clippy. The job keeps
//     cargo-audit, which ergon tool run installs, in the cache of GitHub Actions.
//   - The CodeQL analysis of rust reads the sources without a build.
//   - Dependabot updates Cargo.lock.
func (o *Options) Contribution() workflow.Contribution {
	with := map[string]string{"components": "rustfmt, clippy"}
	if len(o.CI.Versions) > 0 {
		with["toolchain"] = "${{ matrix.version }}"
	}
	return workflow.Contribution{
		Jobs: []workflow.Job{{
			ID:          "check-rust",
			Name:        "Rust",
			Permissions: map[string]string{"contents": "read"},
			Setup: &workflow.Setup{
				Files:    toolchain,
				Runners:  o.CI.Runners,
				Versions: o.CI.Versions,
				Steps:    []workflow.Step{{Name: "Set up Rust", Uses: o.CI.Actions.SetupRustToolchain, With: with}},
				Timeout:  o.CI.Timeout,
			},
			Tools: true,
			Steps: []workflow.Step{{Name: "Check Rust", Run: []string{"make check-rust"}}},
		}},
		CodeQL: []workflow.CodeQL{
			{Language: "rust", Name: "Rust", BuildMode: "none", Files: toolchain, Timeout: o.CI.Timeout},
		},
		Updates: []workflow.Update{{Ecosystem: "cargo", Directories: []string{"/"}}},
	}
}
