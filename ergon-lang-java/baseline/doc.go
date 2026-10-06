// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline renders what Java and the jvm toolchain contribute to a repository that ergon
// init sets up: the Gradle init script that lints the Java sources, and their fragments of the
// shared files.
//
// [Initializer] renders the init script, [InitScript], and the fragments of Java, among them its
// job check-java of the workflow of the gate. [Toolchain] renders the fragments that Java and
// Kotlin share, once for a repository with either or both: the target audit-jvm of the Makefile,
// the CodeQL analysis of java-kotlin and the Gradle updates of Dependabot.
//
// The templates under templates/ mirror the paths of the files, and end in .tmpl. The templates
// of the toolchain are under templates/toolchain/.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language]. The root package of the module imports it.
package baseline
