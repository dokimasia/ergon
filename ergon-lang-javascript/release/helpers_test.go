// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"errors"

	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// pinnedToolchain is the spelling of the js toolchain in configuration, which the test pins.
const pinnedToolchain workspace.Toolchain = "js"

// errProgram is the error of a run of a program or of a tool that fails.
var errProgram = errors.New("program: the run failed")

// releasedA and releasedB are the packages @probe/a and @probe/b at the versions of their release.
var (
	releasedA = workspace.Package{
		Name: "@probe/a", Toolchain: pinnedToolchain, Dir: "packages/a", Version: version.Version{Minor: 3},
	}
	releasedB = workspace.Package{
		Name: "@probe/b", Toolchain: pinnedToolchain, Dir: "packages/b", Version: version.Version{Minor: 2},
	}
)

// call is a run of a program or of a tool that a fake records.
type call struct {
	dir  string
	name string
	args []string
}
