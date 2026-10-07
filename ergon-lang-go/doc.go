// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package golang declares Go and its toolchain, the go command.
//
// [Register] adds the toolchain and the language to a catalog. The toolchain discovers the modules
// of a repository and releases them through the [Git] that the composition root provides. The
// package is named golang because go is a keyword, and its import path is
// go.dokimi.dev/ergon/lang/go.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language], [go.dokimi.dev/ergon/core/workspace] and its
// packages baseline, release and workspace. Only the root module of ergon imports it.
package golang
