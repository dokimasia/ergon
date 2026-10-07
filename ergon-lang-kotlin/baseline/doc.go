// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of Kotlin of ergon init: the fragments of Kotlin of the shared
// files, the section kotlin of .ergon.yaml, and the job of Kotlin in the workflows.
//
// [Producer] renders its fragments of .editorconfig, .gitattributes, .gitignore and the Makefile
// from templates/shared/. The section of Kotlin of .editorconfig is the configuration of ktlint:
// the official code style of Kotlin, with the experimental rules of ktlint. The jvm toolchain of
// the Java module renders the target audit-jvm, the CodeQL analysis of java-kotlin and the Gradle
// updates, once for a repository with Java, Kotlin or both.
//
// # Makefile
//
// The fragment of the Makefile runs fmt-kotlin and lint-kotlin with ktlint through ergon tool run,
// over the Kotlin files that git lists, lint-kotlin also with the checks of the Gradle build,
// test-kotlin with ./gradlew test, and check-kotlin, which requires the targets of the steps that
// the key check of the section names. audit-kotlin requires audit-jvm.
//
// # Options
//
// [Options] is the section kotlin: the version of ktlint, which [Tools.Validate] requires of
// ktlint-cli, the steps of the gate and the options of test-kotlin. [Options.Contribution] returns
// the job check-kotlin, which runs the setup of the jvm toolchain.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option], [go.dokimi.dev/ergon/core/workflow], and the root package of
// the Java module for the name of the jvm toolchain. The root package of the module imports it.
package baseline
