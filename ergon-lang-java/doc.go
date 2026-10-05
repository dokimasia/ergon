// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package java declares Java and the jvm toolchain: Gradle, which builds Java and Kotlin
// projects.
//
// [Register] adds the toolchain and the language to a catalog. The Kotlin module names
// [Toolchain] as the toolchain of its language, so a Gradle project with Java and Kotlin sources
// is one package of one toolchain.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language] and [go.dokimi.dev/ergon/core/workspace]. The root
// module of ergon and go.dokimi.dev/ergon/lang/kotlin import it.
package java
