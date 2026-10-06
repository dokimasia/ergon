// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package kotlin declares Kotlin, which the jvm toolchain of [go.dokimi.dev/ergon/lang/java]
// builds.
//
// [Register] adds the language to a catalog that already has the jvm toolchain.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language], [go.dokimi.dev/ergon/core/workspace], its
// package baseline, and [go.dokimi.dev/ergon/lang/java] for the name of the jvm toolchain. Only
// the root module of ergon imports it.
package kotlin
