// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	_ "embed"

	"go.dokimi.dev/ergon/core/language"
)

// TFLint is the path of the configuration of tflint, which Terraform renders as a managed file of
// its own.
const TFLint = ".tflint.hcl"

// The template of the configuration of tflint.
//
//go:embed templates/.tflint.hcl.tmpl
var tflint string

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

	//go:embed templates/.github/dependabot.yml.tmpl
	dependabot string
)

// Initializer returns the init role of Terraform: the configuration of tflint, and the fragments
// that Terraform contributes to the shared files of a repository. They do not depend on the
// answers. CodeQL does not analyze Terraform.
func Initializer() language.Initializer {
	return language.Fixed{
		{Path: TFLint, Class: language.Managed, Content: []byte(tflint)},
		{Path: language.EditorConfig, Class: language.Managed, Fragment: []byte(editorconfig)},
		{Path: language.GitAttributes, Class: language.Managed, Fragment: []byte(gitattributes)},
		{Path: language.GitIgnore, Class: language.Managed, Fragment: []byte(gitignore)},
		{Path: language.Makefile, Class: language.Managed, Fragment: []byte(makefile)},
		{Path: language.CI, Class: language.Managed, Fragment: []byte(language.Job(ci))},
		{Path: language.Dependabot, Class: language.Managed, Fragment: []byte(dependabot)},
	}
}
