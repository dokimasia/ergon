// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	_ "embed"

	"go.dokimi.dev/ergon/core/language"
)

// The templates of the fragments of JavaScript, which mirror the paths of the shared files.
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
)

// Biome is the path of the configuration of Biome, which the js toolchain renders as a managed
// file of its own for JavaScript and TypeScript. It is JSON, so it opens with no managed comment,
// and a local file does not extend it.
const Biome = "biome.json"

// The template of the configuration of Biome.
//
//go:embed templates/toolchain/biome.json.tmpl
var biome string

// The templates of the fragments of the js toolchain, which mirror the paths of the shared files.
var (
	//go:embed templates/toolchain/Makefile.tmpl
	toolchainMakefile string

	//go:embed templates/toolchain/.github/workflows/security.yml.tmpl
	toolchainSecurity string

	//go:embed templates/toolchain/.github/dependabot.yml.tmpl
	toolchainDependabot string
)

// Initializer returns the init role of JavaScript: the fragments that JavaScript contributes to
// the shared files of a repository. They do not depend on the answers.
func Initializer() language.Initializer {
	return language.Fixed{
		{Path: language.EditorConfig, Class: language.Managed, Fragment: []byte(editorconfig)},
		{Path: language.GitAttributes, Class: language.Managed, Fragment: []byte(gitattributes)},
		{Path: language.GitIgnore, Class: language.Managed, Fragment: []byte(gitignore)},
		{Path: language.Makefile, Class: language.Managed, Fragment: []byte(makefile)},
		{Path: language.CI, Class: language.Managed, Fragment: []byte(language.Job(ci))},
	}
}

// Toolchain returns the init role of the js toolchain: the files that JavaScript and TypeScript
// share, which a repository receives once for either or both. They are the configuration of
// Biome; the targets fmt-js, lint-js and audit-js of the Makefile, which Biome and npm audit run
// and the gates of both languages require; the CodeQL analysis of javascript-typescript; and the
// npm updates of Dependabot. They do not depend on the answers.
func Toolchain() language.Initializer {
	return language.Fixed{
		{Path: Biome, Class: language.Managed, Content: []byte(biome)},
		{Path: language.Makefile, Class: language.Managed, Fragment: []byte(toolchainMakefile)},
		{Path: language.Security, Class: language.Managed, Fragment: []byte(toolchainSecurity)},
		{Path: language.Dependabot, Class: language.Managed, Fragment: []byte(toolchainDependabot)},
	}
}
