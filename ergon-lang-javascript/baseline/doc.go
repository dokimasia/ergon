// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baseline is the producer of JavaScript of ergon init, and the producer of the js
// toolchain, which renders what JavaScript shares with TypeScript.
//
// [Producer] renders the fragments of JavaScript of .editorconfig and the Makefile from
// templates/javascript/. [Toolchain] renders its managed file and its fragments of .gitattributes,
// .gitignore and the Makefile from templates/js/, once for a repository with JavaScript,
// TypeScript or both. The managed file is biome.json for the toolkit biome, and .prettierignore
// for the toolkit vite-plus. biome.json is JSON, so it opens with no managed comment.
// .prettierignore lists .ergon/, .ergon.yaml and every managed file of the rendering, so vp fmt
// leaves them as ergon init renders them.
//
// # Exclusions
//
// The key exclude of the section js contains the globs of [Exclude]. fmt-js and lint-js skip the
// paths that they match. For the toolkit biome, files.includes of biome.json starts with the glob of
// every path and has each exclusion after it with the prefix !. ergon init lays the list out as Biome
// formats it. For the toolkit vite-plus, ergon init writes each exclusion from the root into
// .prettierignore and .eslintignore. vp fmt reads .prettierignore, and the lint of vp check reads
// .eslintignore.
//
// # Package manager and toolkit
//
// The section js sets the package manager, npm or pnpm, and the toolkit, biome or vite-plus.
// [Toolchain.Data] returns them as [Choices], which the templates of the js toolchain read as .Data
// and the templates of JavaScript and TypeScript as .Toolchain. [ToolchainOptions] implements
// [go.dokimi.dev/ergon/core/option.PackageManaged], so ergon tool run starts the npm packages of
// every section with npx for npm and with pnpm dlx for pnpm.
//
// # Makefile
//
// The fragment of the js toolchain runs fmt-js and lint-js, and sets JS_TEST, the command of the
// tests of JavaScript and TypeScript:
//
//   - For biome, fmt-js and lint-js run Biome through ergon tool run, and JS_TEST runs the test
//     script of package.json with the package manager.
//   - For vite-plus, fmt-js runs vp fmt, lint-js runs vp check, and JS_TEST runs vp test --run. The
//     package manager runs the vp of the packages of the repository.
//
// audit-js runs npm audit or pnpm audit. The fragment of JavaScript runs test-javascript with
// JS_TEST, and check-javascript, which requires the targets of the steps that the key check of the
// section names. lint-javascript and audit-javascript require lint-js and audit-js. With a command
// in the key generate, the fragment of JavaScript also renders generate-javascript, which runs it,
// and verify-generate-javascript, which fails when it changes a file. In make help, the line
// ##@ JavaScript and TypeScript starts the group of the js toolchain, and the line ##@ JavaScript
// starts the group of JavaScript.
//
// # Options
//
// [ToolchainOptions] is the section js. It sets these options:
//
//   - the package manager and the toolkit
//   - the versions of Biome and of the npm of a publish
//   - the paths that Biome checks
//   - the paths that fmt-js and lint-js skip
//   - the lowest severity that fails the scan
//   - the key ci of the setup of Node.js
//
// [ToolchainOptions.Contribution] returns these parts of the workflows:
//
//   - the setup of Node.js and the packages in the jobs of the gate
//   - the same setup in the jobs of a release
//   - the CodeQL analysis of javascript-typescript
//   - the updates of the npm packages
//   - the uploads to the npm registry
//
// [Options] is the section javascript. It sets the steps of the gate, and the options of
// test-javascript and of the generators. [Options.Contribution] returns the job check-javascript,
// which runs the setup of the js toolchain.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option] and [go.dokimi.dev/ergon/core/workflow]. The root package of the
// module imports it.
package baseline
