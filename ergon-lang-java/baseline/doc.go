// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Java of ergon init, and the producer of the jvm toolchain,
// which renders what Java shares with Kotlin.
//
// [Producer] renders the Gradle init script gradle/ergon-java.init.gradle.kts as a managed file,
// and the fragments of Java of .editorconfig, .gitattributes, .gitignore and the Makefile, from
// templates/java/. [Toolchain] renders its fragments of .gitignore and the Makefile from
// templates/jvm/, once for a repository with Java, Kotlin or both.
//
// # Makefile
//
// The fragment of the jvm toolchain runs audit-jvm, which scans every Gradle lockfile with
// osv-scanner through ergon tool run, and fails in a repository without one. The fragment of Java
// runs lint-java, which applies the init script to the Gradle build of the repository with
// --init-script, test-java with ./gradlew test, and check-java, which requires the targets of the
// steps that the key check of the section names. audit-java requires audit-jvm.
//
// # Options
//
// [ToolchainOptions] is the section jvm: the release of osv-scanner, and the key ci of the setup of
// Java. [ToolchainOptions.Contribution] returns that setup, the CodeQL analysis of java-kotlin and
// the Gradle updates. [Options] is the section java: the version of PMD, which [Tools.Validate]
// requires of the artifact that the PMD plugin of Gradle runs, the steps of the gate and the
// options of test-java. [Options.Contribution] returns the job check-java, which runs the setup of
// the jvm toolchain.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
