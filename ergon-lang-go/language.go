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

// ErrGit is the error of [Register] for a [Git] without one of its functions.
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
}

// Register adds the toolchain of Go and then Go to c. The toolchain discovers the modules of
// go.work with [goworkspace.Discoverer], records the version of a module in its CHANGELOG.md and
// its tags, and releases the modules with [release.Versioner], [release.Tagger] and
// [release.Locker], which read the repository through git.
//
// It returns an error that wraps [ErrGit] for a git without Tags or Snapshot, and the first error
// of [language.RegisterToolchain] and [language.Register], which wraps [language.ErrRegistered]
// when c already has either name. When only the language fails, c keeps the toolchain.
func Register(c *language.Catalog, git Git) error {
	if git.Tags == nil || git.Snapshot == nil {
		return fmt.Errorf("%w: the toolchain of Go needs Tags and Snapshot", ErrGit)
	}
	toolchain := language.Toolchain{
		Name:             Toolchain,
		Discover:         goworkspace.Discoverer{Tags: git.Tags}.Discover,
		ChangelogVersion: true,
	}
	versioner := release.Versioner{Snapshot: git.Snapshot}
	locker := release.Locker{Snapshot: git.Snapshot}
	if err := language.RegisterToolchain(c, toolchain, versioner, release.Tagger{}, locker); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain}, baseline.Producer{})
}
