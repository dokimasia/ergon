// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package python

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/python/baseline"
	"go.dokimi.dev/ergon/lang/python/release"
	pyworkspace "go.dokimi.dev/ergon/lang/python/workspace"
)

// Language is the name of Python in configuration and in reports.
const Language workspace.Language = "python"

// Toolchain is the name of the toolchain of Python in configuration and in reports.
const Toolchain = pyworkspace.Toolchain

// ErrTools is the error of [Register] for nil tools.
var ErrTools = errors.New("python: no tools")

// ErrClient is the error of [Register] for a nil client.
var ErrClient = errors.New("python: no client")

// Tools runs the tool of the section python of .ergon.yaml with args in dir, with env added to the
// environment of the process, as ergon tool run does, such as uv in a release. It returns an error
// for a tool that fails.
type Tools func(ctx context.Context, dir, tool string, args, env []string) error

// Register adds the toolchain of Python and then Python to c. The toolchain discovers the projects
// of a uv workspace with [pyworkspace.Discover], and releases them with three roles:
//
//   - [release.Versioner] refreshes uv.lock with uv through tools.
//   - [release.Packer] builds the distributions of each project with uv build through tools.
//   - [release.Publisher] reads the releases of PyPI through client, and uploads the distributions
//     with uv publish through tools.
//
// It returns an error that wraps [ErrTools] for nil tools, an error that wraps [ErrClient] for a nil
// client, and the first error of [language.RegisterToolchain] and [language.Register], which wraps
// [language.ErrRegistered] when c already has either name. When only the language fails, c keeps the
// toolchain.
func Register(c *language.Catalog, tools Tools, client *http.Client) error {
	if tools == nil {
		return fmt.Errorf("%w: the toolchain of Python needs its tools", ErrTools)
	}
	if client == nil {
		return fmt.Errorf("%w: the toolchain of Python needs a client of PyPI", ErrClient)
	}
	toolchain := language.Toolchain{Name: Toolchain, Discover: pyworkspace.Discover}
	if err := language.RegisterToolchain(c, toolchain,
		release.Versioner{Tools: tools},
		release.Packer{Tools: tools},
		release.Publisher{Client: client, Tools: tools},
	); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain}, baseline.Producer{})
}
