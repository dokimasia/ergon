// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package spdx declares the SPDX identifiers of the licenses that ergon supports.
//
// An [ID] names a license, such as [MIT] or [Apache20]. ergon supports 44 licenses: the licenses
// for software of GitHub's license list, under their current SPDX identifiers, and [BUSL11]. Each
// GNU license has two identifiers, -only and -or-later, which name the same text. [IDs] returns
// the identifiers in the order of the SPDX License List.
//
// # Names
//
// Each constant is its identifier without hyphens and full stops: [Apache20] is Apache-2.0, and
// [CC010] is CC0-1.0. [ZeroBSD] is 0BSD, because a name in Go cannot start with a digit.
//
// # Spelling
//
// [ID.Valid] accepts each identifier in its canonical case only. The SPDX specification matches
// identifiers without regard to case. ergon writes an identifier into the files of a
// repository, so one license has one spelling in every repository.
//
// # Dependency position
//
// Position 0 of ergon-core. Imports the standard library.
package spdx
