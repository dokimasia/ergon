// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	_ "embed"

	"go.dokimi.dev/ergon/core/language"
)

// InitScript is the path of the Gradle init script that lints the Java sources, which Java renders
// as a managed file of its own. lint-java passes it to Gradle with --init-script.
const InitScript = "gradle/ergon-java.init.gradle.kts"

// The template of the init script.
//
//go:embed templates/gradle/ergon-java.init.gradle.kts.tmpl
var initScript string

// The templates of the fragments of Java, which mirror the paths of the shared files.
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

// The templates of the fragments of the jvm toolchain, which mirror the paths of the shared files.
var (
	//go:embed templates/toolchain/Makefile.tmpl
	toolchainMakefile string

	//go:embed templates/toolchain/.github/workflows/security.yml.tmpl
	toolchainSecurity string

	//go:embed templates/toolchain/.github/dependabot.yml.tmpl
	toolchainDependabot string
)

// Initializer returns the init role of Java: the init script that lints the Java sources, and the
// fragments that Java contributes to the shared files of a repository. They do not depend on the
// answers.
func Initializer() language.Initializer {
	return language.Fixed{
		{Path: InitScript, Class: language.Managed, Content: []byte(initScript)},
		{Path: language.EditorConfig, Class: language.Managed, Fragment: []byte(editorconfig)},
		{Path: language.GitAttributes, Class: language.Managed, Fragment: []byte(gitattributes)},
		{Path: language.GitIgnore, Class: language.Managed, Fragment: []byte(gitignore)},
		{Path: language.Makefile, Class: language.Managed, Fragment: []byte(makefile)},
		{Path: language.CI, Class: language.Managed, Fragment: []byte(language.Job(ci))},
	}
}

// Toolchain returns the init role of the jvm toolchain: the fragments that Java and Kotlin share,
// which a repository receives once for either or both. They are the target audit-jvm of the
// Makefile, which osv-scanner runs over the Gradle lockfiles and the gates of both languages
// require, the CodeQL analysis of java-kotlin and the Gradle updates of Dependabot. They do not
// depend on the answers.
func Toolchain() language.Initializer {
	return language.Fixed{
		{Path: language.Makefile, Class: language.Managed, Fragment: []byte(toolchainMakefile)},
		{Path: language.Security, Class: language.Managed, Fragment: []byte(toolchainSecurity)},
		{Path: language.Dependabot, Class: language.Managed, Fragment: []byte(toolchainDependabot)},
	}
}
