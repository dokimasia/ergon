// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golang

import (
	"context"
	"errors"
	"fmt"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/go/baseline"
	"go.dokimi.dev/ergon/lang/go/release"
	goworkspace "go.dokimi.dev/ergon/lang/go/workspace"
)

// Language is the name of Go in configuration and in reports.
const Language workspace.Language = "go"

// Toolchain is the name of the toolchain of Go in configuration and in reports.
const Toolchain = goworkspace.Toolchain

// ErrGit is the error of [Register] for a [Git] or [Tools] without one of its functions.
var ErrGit = errors.New("golang: incomplete git")

// Git is the access to the repository that the toolchain of Go needs and that a composition root
// provides, such as the functions of the package vcs of ergon-service.
type Git struct {
	// Tags returns the tags of the repository at a directory, each with the commit that it names.
	// The discovery of the modules reads the version of a module from its tags.
	Tags func(ctx context.Context, dir string) (map[string]string, error)

	// Snapshot returns the tree of the working tree of a directory as git would commit it. A
	// release writes the zip of each released module from it.
	Snapshot func(ctx context.Context, dir string) (string, error)

	// Head returns the commit of HEAD of the repository at a directory. A pack tags it with the tag
	// of a module with commands.
	Head func(ctx context.Context, dir string) (string, error)

	// LightTag creates the lightweight tag name at commit in the repository at a directory.
	LightTag func(ctx context.Context, dir, name, commit string) error
}

// Tools runs and finds the tools of the section go of .ergon.yaml, as ergon tool run and ergon tool
// path do, and a composition root provides it. Each function is required.
type Tools struct {
	// Run runs the tool with args in dir, with env added to the environment of the process, such as
	// GoReleaser in a pack. It returns an error for a tool that fails.
	Run func(ctx context.Context, dir, tool string, args, env []string) error

	// Program installs the tool for dir unless the cache has it, and returns its program, such as
	// zig, which the builds with cgo of a pack run as their C compiler. It returns an error for a
	// tool that does not install.
	Program func(ctx context.Context, dir, tool string) (string, error)
}

// Register adds the toolchain of Go and then Go to c. The toolchain discovers the modules of
// go.work with [goworkspace.Discoverer], records the version of a module in its CHANGELOG.md and
// its tags, and releases the modules with [release.Versioner], [release.Tagger] and
// [release.Locker], which read the repository through git, and with [release.Packer], which builds
// the binaries of the commands of a module with GoReleaser and zig through tools.
//
// It returns an error that wraps [ErrGit] for a git or tools without one of its functions, and the
// first error of [language.RegisterToolchain] and [language.Register], which wraps
// [language.ErrRegistered] when c already has either name. When only the language fails, c keeps
// the toolchain.
func Register(c *language.Catalog, git Git, tools Tools) error {
	complete := git.Tags != nil && git.Snapshot != nil && git.Head != nil && git.LightTag != nil &&
		tools.Run != nil && tools.Program != nil
	if !complete {
		return fmt.Errorf("%w: the toolchain of Go needs Tags, Snapshot, Head, LightTag, Run and Program", ErrGit)
	}
	toolchain := language.Toolchain{
		Name:             Toolchain,
		Discover:         goworkspace.Discoverer{Tags: git.Tags}.Discover,
		ChangelogVersion: true,
	}
	versioner := release.Versioner{Snapshot: git.Snapshot}
	locker := release.Locker{Snapshot: git.Snapshot}
	packer := release.Packer{
		Snapshot: git.Snapshot, Tags: git.Tags, Head: git.Head, Tag: git.LightTag, Run: tools.Run,
		Program: tools.Program,
	}
	if err := language.RegisterToolchain(c, toolchain, versioner, release.Tagger{}, locker, packer); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain}, baseline.Producer{})
}
