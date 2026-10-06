// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	_ "embed"

	"go.dokimi.dev/ergon/core/language"
)

// Clippy is the path of the configuration of clippy, which Rust renders as a managed file of its
// own.
const Clippy = "clippy.toml"

// The template of the configuration of clippy.
//
//go:embed templates/clippy.toml.tmpl
var clippy string

// The templates of the fragments, which mirror the paths of the shared files.
var (
	//go:embed templates/.editorconfig.tmpl
	editorconfig string

	//go:embed templates/.gitattributes.tmpl
	gitattributes string

	//go:embed templates/.gitignore.tmpl
	gitignore string

	//go:embed templates/Makefile.tmpl
	makefile string

	//go:embed templates/.github/workflows/ci.yml.tmpl
	ci string

	//go:embed templates/.github/workflows/security.yml.tmpl
	security string

	//go:embed templates/.github/dependabot.yml.tmpl
	dependabot string
)

// Initializer returns the init role of Rust: the configuration of clippy, and the fragments that
// Rust contributes to the shared files of a repository. They do not depend on the answers.
func Initializer() language.Initializer {
	return language.Fixed{
		{Path: Clippy, Class: language.Managed, Content: []byte(clippy)},
		{Path: language.EditorConfig, Class: language.Managed, Fragment: []byte(editorconfig)},
		{Path: language.GitAttributes, Class: language.Managed, Fragment: []byte(gitattributes)},
		{Path: language.GitIgnore, Class: language.Managed, Fragment: []byte(gitignore)},
		{Path: language.Makefile, Class: language.Managed, Fragment: []byte(makefile)},
		{Path: language.CI, Class: language.Managed, Fragment: []byte(language.Job(ci))},
		{Path: language.Security, Class: language.Managed, Fragment: []byte(security)},
		{Path: language.Dependabot, Class: language.Managed, Fragment: []byte(dependabot)},
	}
}
