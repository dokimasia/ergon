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
// runs these targets:
//
//   - fmt-java formats the Java sources that git lists with google-java-format, which ergon tool
//     run installs.
//   - lint-java checks the format of the same sources. It then applies the init script to the
//     Gradle build of the repository with --init-script, and PMD checks the sources with the rule
//     sets of [Rulesets].
//   - test-java runs ./gradlew test.
//   - check-java requires the targets of the steps that the key check of the section names.
//   - audit-java requires audit-jvm.
//
// With a command in the key generate, the fragment of Java also renders generate-java, which runs
// it, and verify-generate-java, which fails when it changes a file. In make help, the line
// ##@ Java and Kotlin starts the group of the jvm toolchain, and the line ##@ Java starts the group
// of Java.
//
// # Options
//
// [ToolchainOptions] is the section jvm. It has the release of osv-scanner and the key ci of the
// setup of Java. [ToolchainOptions.Contribution] returns that setup, the CodeQL analysis of
// java-kotlin and the Gradle updates. [Options] is the section java, with these keys:
//
//   - tools: the version of PMD, whose artifact [Tools.Validate] requires, and the release of
//     google-java-format
//   - rulesets: the rule sets of PMD, which [Rulesets.Validate] checks
//   - check: the steps of the gate
//   - test and generate: the options of test-java and of the generators
//
// [Options.Contribution] returns the job check-java, which runs the setup of the jvm toolchain.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
