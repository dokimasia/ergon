// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package golang declares Go and its toolchain, the go command.
//
// [Register] adds the toolchain and the language to a catalog. The package is named golang
// because go is a keyword, and its import path is go.dokimi.dev/ergon/lang/go.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language], [go.dokimi.dev/ergon/core/workspace] and its
// package baseline. Only the root module of ergon imports it.
package golang
