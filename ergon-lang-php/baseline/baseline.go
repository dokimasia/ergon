// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	_ "embed"

	"go.dokimi.dev/ergon/core/language"
)

// PHPStan is the path of the configuration of PHPStan, which PHP renders as a managed file of its
// own. lint-php passes it to PHPStan with --configuration, so a phpstan.neon of the repository
// does not replace it.
const PHPStan = "phpstan.dist.neon"

// The template of the configuration of PHPStan.
//
//go:embed templates/phpstan.dist.neon.tmpl
var phpstan string

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

// Initializer returns the init role of PHP: the configuration of PHPStan, and the fragments that
// PHP contributes to the shared files of a repository. They do not depend on the answers. CodeQL
// does not analyze PHP.
func Initializer() language.Initializer {
	return language.Fixed{
		{Path: PHPStan, Class: language.Managed, Content: []byte(phpstan)},
		{Path: language.EditorConfig, Class: language.Managed, Fragment: []byte(editorconfig)},
		{Path: language.GitAttributes, Class: language.Managed, Fragment: []byte(gitattributes)},
		{Path: language.GitIgnore, Class: language.Managed, Fragment: []byte(gitignore)},
		{Path: language.Makefile, Class: language.Managed, Fragment: []byte(makefile)},
		{Path: language.CI, Class: language.Managed, Fragment: []byte(language.Job(ci))},
		{Path: language.Dependabot, Class: language.Managed, Fragment: []byte(dependabot)},
	}
}
