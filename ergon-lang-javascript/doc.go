// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package javascript declares JavaScript and the js toolchain: npm and pnpm, which build
// JavaScript and TypeScript packages.
//
// [Register] adds the toolchain and the language to a catalog. The TypeScript module names
// [Toolchain] as the toolchain of its language, so a package with TypeScript and JavaScript
// sources is one package of one toolchain. The toolchain renders the files of ergon init that
// JavaScript and TypeScript share, discovers the packages of an npm or a pnpm workspace, and
// releases them to the npm registry with the package manager of the repository and npm publish of
// the section js, which [Tools] runs.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/workspace] and its packages baseline, release and workspace. The root
// module of ergon and go.dokimi.dev/ergon/lang/typescript import it.
package javascript
