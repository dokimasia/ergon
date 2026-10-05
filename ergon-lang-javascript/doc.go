// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package javascript declares JavaScript and the js toolchain: npm, pnpm and bun, which build
// JavaScript and TypeScript packages.
//
// [Register] adds the toolchain and the language to a catalog. The TypeScript module names
// [Toolchain] as the toolchain of its language, so a package with TypeScript and JavaScript
// sources is one package of one toolchain.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language] and [go.dokimi.dev/ergon/core/workspace]. The root
// module of ergon and go.dokimi.dev/ergon/lang/typescript import it.
package javascript
