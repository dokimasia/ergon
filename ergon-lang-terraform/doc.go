// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package terraform declares Terraform and its toolchain, the terraform command.
//
// [Register] adds the toolchain and the language to a catalog. The toolchain releases a repository
// whose lock of ergon init lists Terraform as one package, through the [Repository] that a
// composition root provides.
//
// # Dependency position
//
// Imports [go.dokimi.dev/ergon/core/language] and its packages baseline, release and workspace.
// Only the root module of ergon imports it.
package terraform
