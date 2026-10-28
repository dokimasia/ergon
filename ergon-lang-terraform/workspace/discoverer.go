// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// Language is the name of Terraform in configuration and in reports, which the lock of ergon init
// lists among the languages of a repository of Terraform.
const Language workspace.Language = "terraform"

// Toolchain is the name of the toolchain of Terraform in configuration and in reports.
const Toolchain workspace.Toolchain = "terraform"

// ChangelogFile is the changelog of the package, at the root of the repository. Its headings record
// the versions of the package.
const ChangelogFile = changeset.Changelog

// tagPrefix starts the tag of each version of the package, as in v1.2.0.
const tagPrefix = "v"

// Discoverer discovers a repository of Terraform as one package of [Toolchain].
//
// # Concurrency
//
// A Discoverer is safe for concurrent use when its Tags and its Answers are.
type Discoverer struct {
	// Tags returns the tags of the repository at a directory, each with the commit that it names.
	// It must not be nil.
	Tags func(ctx context.Context, dir string) (map[string]string, error)

	// Answers returns the answers of the lock of ergon init of the repository at a directory, and
	// the zero Answers for a repository without a lock. It must not be nil.
	Answers func(ctx context.Context, dir string) (language.Answers, error)
}

// Discover returns the package of the repository at root when the answers of its lock list
// [Language]. The package has the name of the repository as its Name and the root as its
// directory. Its version is the higher of the highest version of a heading of its CHANGELOG.md and
// the highest version of a tag v<version>, as the package documentation states.
//
// It returns no package and no error for a repository whose answers do not list [Language], such
// as a repository without a lock. It returns the error of d.Answers, the error of opening root, the
// error of reading the changelog, with its file, and the error of d.Tags.
func (d Discoverer) Discover(ctx context.Context, root string) ([]workspace.Package, error) {
	answers, err := d.Answers(ctx, root)
	if err != nil || !slices.Contains(answers.Languages, Language) {
		return nil, err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("workspace: open the repository: %w", err)
	}
	defer func() { _ = r.Close() }()
	v, err := changelogVersion(r)
	if err != nil {
		return nil, err
	}
	tags, err := d.Tags(ctx, root)
	if err != nil {
		return nil, err
	}
	if tagged := tagVersion(tags); tagged.Compare(v) > 0 {
		v = tagged
	}
	return []workspace.Package{{Name: answers.Name, Toolchain: Toolchain, Dir: ".", Version: v}}, nil
}

// changelogVersion returns the highest version that a heading of the changelog at the root of the
// repository of r names, as [changeset.Latest] reads it, and the zero version for a repository
// without a changelog. It returns the error of reading the changelog.
func changelogVersion(r *os.Root) (version.Version, error) {
	data, err := r.ReadFile(ChangelogFile)
	if errors.Is(err, fs.ErrNotExist) {
		return version.Version{}, nil
	}
	if err != nil {
		return version.Version{}, fmt.Errorf("workspace: read %s: %w", ChangelogFile, err)
	}
	return changeset.Latest(data), nil
}

// tagVersion returns the highest version that a tag v<version> of tags names, and the zero version
// for none.
func tagVersion(tags map[string]string) version.Version {
	var highest version.Version
	for name := range tags {
		rest, ok := strings.CutPrefix(name, tagPrefix)
		if !ok {
			continue
		}
		if v, err := version.Parse(rest); err == nil && v.Compare(highest) > 0 {
			highest = v
		}
	}
	return highest
}
