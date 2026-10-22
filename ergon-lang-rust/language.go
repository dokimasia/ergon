// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package rust

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/rust/baseline"
	"go.dokimi.dev/ergon/lang/rust/release"
	rustworkspace "go.dokimi.dev/ergon/lang/rust/workspace"
)

// Language is the name of Rust in configuration and in reports.
const Language workspace.Language = "rust"

// Toolchain is the name of the toolchain of Rust in configuration and in reports.
const Toolchain = rustworkspace.Toolchain

// ErrTools is the error of [Register] for tools without one of their functions.
var ErrTools = errors.New("rust: no tools")

// ErrClient is the error of [Register] for a nil client.
var ErrClient = errors.New("rust: no client")

// Tools runs the programs of a release of Rust and reads the environment of the process, and a
// composition root provides it. Each function is required.
type Tools struct {
	// Command runs the program name of the PATH of the process with args in dir, with env added to
	// the environment of the process, such as cargo in a release. It returns an error for a program
	// that fails.
	Command func(ctx context.Context, dir, name string, args, env []string) error

	// Getenv returns the variable key of the environment of the process, as os.Getenv does, such as
	// a variable of the OIDC token of a job of GitHub Actions.
	Getenv func(key string) string
}

// Register adds the toolchain of Rust and then Rust to c. The toolchain discovers the crates of a
// Cargo workspace with [rustworkspace.Discover], and releases them with three roles:
//
//   - [release.Versioner] refreshes Cargo.lock with cargo update through tools.Command.
//   - [release.Packer] packages and verifies the crates with cargo package through tools.Command.
//   - [release.Publisher] reads the index of crates.io and exchanges the tokens of trusted
//     publishing through client, and uploads the crates with cargo publish through tools.Command.
//
// It returns an error that wraps [ErrTools] for tools without one of their functions, an error
// that wraps [ErrClient] for a nil client, and the first error of [language.RegisterToolchain] and
// [language.Register], which wraps [language.ErrRegistered] when c already has either name. When
// only the language fails, c keeps the toolchain.
func Register(c *language.Catalog, tools Tools, client *http.Client) error {
	if tools.Command == nil || tools.Getenv == nil {
		return fmt.Errorf("%w: the toolchain of Rust needs Command and Getenv", ErrTools)
	}
	if client == nil {
		return fmt.Errorf("%w: the toolchain of Rust needs a client of crates.io", ErrClient)
	}
	toolchain := language.Toolchain{Name: Toolchain, Discover: rustworkspace.Discover}
	if err := language.RegisterToolchain(c, toolchain,
		release.Versioner{Command: tools.Command},
		release.Packer{Command: tools.Command},
		release.Publisher{Client: client, Command: tools.Command, Getenv: tools.Getenv},
	); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain}, baseline.Producer{})
}
