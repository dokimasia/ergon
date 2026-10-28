// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package terraform

import (
	"context"
	"fmt"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/lang/terraform/baseline"
	"go.dokimi.dev/ergon/lang/terraform/release"
	tfworkspace "go.dokimi.dev/ergon/lang/terraform/workspace"
)

// Language is the name of Terraform in configuration and in reports.
const Language = tfworkspace.Language

// Toolchain is the name of the toolchain of Terraform in configuration and in reports.
const Toolchain = tfworkspace.Toolchain

// Repository is the access to a repository that the toolchain of Terraform needs and that a
// composition root provides, such as the tags of the package vcs of ergon-service and the answers
// of the lock of ergon init.
type Repository struct {
	// Tags returns the tags of the repository at a directory, each with the commit that it names.
	// The discovery reads the version of the package from its tags.
	Tags func(ctx context.Context, dir string) (map[string]string, error)

	// Answers returns the answers of the lock of ergon init of the repository at a directory, and
	// the zero Answers for a repository without a lock. The discovery reads the name and the
	// languages of the repository from them.
	Answers func(ctx context.Context, dir string) (language.Answers, error)
}

// Register adds the toolchain of Terraform and then Terraform to c. The toolchain discovers a
// repository whose lock lists Terraform as one package with [tfworkspace.Discoverer], records the
// version of the package in its CHANGELOG.md and its tags, and releases it with [release.Versioner]
// and [release.Tagger].
//
// It returns an error that wraps [ErrRepository] for a repo without one of its functions, and the
// first error of [language.RegisterToolchain] and [language.Register], which wraps
// [language.ErrRegistered] when c already has either name. When only the language fails, c keeps
// the toolchain.
func Register(c *language.Catalog, repo Repository) error {
	if repo.Tags == nil || repo.Answers == nil {
		return fmt.Errorf("%w: the toolchain of Terraform needs Tags and Answers", ErrRepository)
	}
	toolchain := language.Toolchain{
		Name:             Toolchain,
		Discover:         tfworkspace.Discoverer{Tags: repo.Tags, Answers: repo.Answers}.Discover,
		ChangelogVersion: true,
	}
	if err := language.RegisterToolchain(c, toolchain, release.Versioner{}, release.Tagger{}); err != nil {
		return err
	}
	return language.Register(c, language.Declaration{Name: Language, Toolchain: Toolchain}, baseline.Producer{})
}
