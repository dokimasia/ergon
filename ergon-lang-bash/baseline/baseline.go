// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	_ "embed"

	"go.dokimi.dev/ergon/core/language"
)

// ShellCheckRC is the path of the configuration of shellcheck, which Bash renders as a managed
// file of its own.
const ShellCheckRC = ".shellcheckrc"

// The template of the configuration of shellcheck.
//
//go:embed templates/.shellcheckrc.tmpl
var shellcheckrc string

// The templates of the fragments, which mirror the paths of the shared files.
var (
	//go:embed templates/.editorconfig.tmpl
	editorconfig string

	//go:embed templates/.gitattributes.tmpl
	gitattributes string

	//go:embed templates/Makefile.tmpl
	makefile string

	//go:embed templates/.github/workflows/ci.yml.tmpl
	ci string
)

// Initializer returns the init role of Bash: the configuration of shellcheck, and the fragments
// that Bash contributes to the shared files of a repository. They do not depend on the answers.
// Bash has no build output, no package manager and no CodeQL analysis.
func Initializer() language.Initializer {
	return language.Fixed{
		{Path: ShellCheckRC, Class: language.Managed, Content: []byte(shellcheckrc)},
		{Path: language.EditorConfig, Class: language.Managed, Fragment: []byte(editorconfig)},
		{Path: language.GitAttributes, Class: language.Managed, Fragment: []byte(gitattributes)},
		{Path: language.Makefile, Class: language.Managed, Fragment: []byte(makefile)},
		{Path: language.CI, Class: language.Managed, Fragment: []byte(language.Job(ci))},
	}
}
