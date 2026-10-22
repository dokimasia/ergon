// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"go.dokimi.dev/ergon/lang/python/release"
)

// errUV is the error of a run of uv that fails.
var errUV = errors.New("uv: the run failed")

// call is a run of a tool that a fake records.
type call struct {
	dir  string
	tool string
	args []string
}

// tools is a fake of ergon tool run: it records each call, writes lock into uv.lock of dir when
// lock is not empty, removes uv.lock when remove is set, and returns err.
type tools struct {
	err    error
	lock   string
	calls  []call
	remove bool
}

// run records the call and acts as tools states.
func (f *tools) run(_ context.Context, dir, tool string, args, _ []string) error {
	f.calls = append(f.calls, call{dir: dir, tool: tool, args: args})
	if f.remove {
		return os.Remove(filepath.Join(dir, release.LockFile))
	}
	if f.lock != "" {
		if err := os.WriteFile(filepath.Join(dir, release.LockFile), []byte(f.lock), 0o644); err != nil {
			return err
		}
	}
	return f.err
}
