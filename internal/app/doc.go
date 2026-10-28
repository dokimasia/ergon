// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package app composes ergon: the language modules and the toolchains they declare.
//
// # Languages
//
// [Register] adds every language module to a catalog by an explicit call, so the set of
// languages is a value of this package and not of the imports of the binary. No other package of
// ergon imports a language module. Removing a language deletes its directory, its use line in
// go.work, and its import and its entry in this package.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language], the eleven language modules,
// [go.dokimi.dev/ergon/service/vcs], whose git the toolchains of Go and of Terraform read the
// repository through, and [go.dokimi.dev/ergon/service/baseline/lock], whose lock of ergon init the
// toolchain of Terraform reads the answers of the repository from. cmd/ergon imports it.
package app
