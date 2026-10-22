// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package javascript

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/javascript/baseline"
	"go.dokimi.dev/ergon/lang/javascript/release"
	jsworkspace "go.dokimi.dev/ergon/lang/javascript/workspace"
)

// Language is the name of JavaScript in configuration and in reports.
const Language workspace.Language = "javascript"

// Toolchain is the name of the js toolchain in configuration and in reports. JavaScript and
// TypeScript name it.
const Toolchain = jsworkspace.Toolchain

// ErrTools is the error of [Register] for tools without one of their functions.
var ErrTools = errors.New("javascript: no tools")

// ErrClient is the error of [Register] for a nil client.
var ErrClient = errors.New("javascript: no client")

// Tools runs the programs of a release of the js toolchain, and a composition root provides it.
// Each function is required.
type Tools struct {
	// Run runs the tool of the section js of .ergon.yaml with args in dir, with env added to the
	// environment of the process, as ergon tool run does, such as npm in a publish. It returns an
	// error for a tool that fails.
	Run func(ctx context.Context, dir, tool string, args, env []string) error

	// Command runs the program name of the PATH of the process with args in dir, with env added to
	// the environment of the process, such as the package manager in the refresh of a lockfile and
	// in a pack. It returns an error for a program that fails.
	Command func(ctx context.Context, dir, name string, args, env []string) error
}

// Register adds the js toolchain, with the producer of the files that JavaScript and TypeScript
// share, and then JavaScript to c. The toolchain discovers the packages of an npm or a pnpm
// workspace with [jsworkspace.Discover], and releases them with three roles:
//
//   - [release.Versioner] refreshes the lockfile with the package manager through tools.Command.
//   - [release.Packer] packs the tarball of each package with the package manager through
//     tools.Command.
//   - [release.Publisher] reads the npm registry through client, and uploads the tarballs with npm
//     publish of the section js through tools.Run.
//
// It returns an error that wraps [ErrTools] for tools without one of their functions, an error
// that wraps [ErrClient] for a nil client, and the first error of [language.RegisterToolchain] and
// [language.Register], which wraps [language.ErrRegistered] when c already has either name. When
// only the language fails, c keeps the toolchain.
func Register(c *language.Catalog, tools Tools, client *http.Client) error {
	if tools.Run == nil || tools.Command == nil {
		return fmt.Errorf("%w: the js toolchain needs Run and Command", ErrTools)
	}
	if client == nil {
		return fmt.Errorf("%w: the js toolchain needs a client of the npm registry", ErrClient)
	}
	toolchain := language.Toolchain{Name: Toolchain, Discover: jsworkspace.Discover}
	if err := language.RegisterToolchain(c, toolchain,
		baseline.Toolchain{},
		release.Versioner{Command: tools.Command},
		release.Packer{Command: tools.Command},
		release.Publisher{Client: client, Tools: tools.Run},
	); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain}, baseline.Producer{})
}
