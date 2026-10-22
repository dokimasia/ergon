// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"slices"
	"sync"

	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// pinnedToolchain is the spelling of the toolchain of Rust in configuration, which the test pins.
const pinnedToolchain workspace.Toolchain = "rust"

// errProgram is the error of a run of a program that fails.
var errProgram = errors.New("program: the run failed")

// releasedA and releasedB are the crates probe-a and probe-b at the versions of their release.
var (
	releasedA = workspace.Package{
		Name: "probe-a", Toolchain: pinnedToolchain, Dir: "crates/a", Version: version.Version{Minor: 3},
	}
	releasedB = workspace.Package{
		Name: "probe-b", Toolchain: pinnedToolchain, Dir: "crates/b", Version: version.Version{Minor: 2, Patch: 1},
	}
)

// call is a run of a program that a fake records.
type call struct {
	dir  string
	name string
	args []string
	env  []string
}

// commands is a fake of the programs of a release. It records each run and then calls do, the
// function of the case, which can write the files of the run. do returns the error of the run.
type commands struct {
	do    func(c call) error
	calls []call
	mu    sync.Mutex
}

// run records the run of name with args and env in dir, and returns the error of do, or nil
// without do.
func (f *commands) run(_ context.Context, dir, name string, args, env []string) error {
	c := call{dir: dir, name: name, args: slices.Clone(args), env: slices.Clone(env)}
	f.mu.Lock()
	f.calls = append(f.calls, c)
	f.mu.Unlock()
	if f.do == nil {
		return nil
	}
	return f.do(c)
}
