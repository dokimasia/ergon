// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package pin finds the pins of the options of a producer of ergon init, and resolves the newest
// release of each in its registry.
//
// A pin is a field of the options whose value refers to a release that a registry publishes: a tool
// of a kind of [go.dokimi.dev/ergon/core/option], a release binary, an action, or a version whose
// source tag names its registry. [Find] returns the pins of a section as [Pin] values, each with
// its key in .ergon.yaml and the Go fields that lead to it. A baseline update resolves them, and an
// upgrade compares the pins of .ergon.yaml with the baseline.
//
// # Resolution
//
// [Resolver.Resolve] reads the registry of a pin, by its [Kind]:
//
//   - a Go module from a module proxy of the GOPROXY protocol
//   - a PyPI package from the JSON API of PyPI
//   - an npm package from the npm registry
//   - a crate from the sparse index of crates.io
//   - a Maven artifact from maven-metadata.xml and the POMs of a Maven repository
//   - a Composer package from the metadata of Packagist
//   - a package of Chocolatey from its OData API
//   - a release binary, an action and a version of GitHub from the releases of GitHub, through
//     [GitHub]
//
// It takes the newest release that is stable, newer than the pin, published at least
// [Resolver.MinAge] before the run and of the major version of the pin, and reports the newest
// release of a later major version beside it, as [Resolution] states. A stable version is one to
// four numbers of at most 19 digits separated by dots, with an optional leading v. Versions compare
// by their numbers, not by their dates, and a release ranks above a pre-release or a pseudo-version
// of Go of the same numbers. The release of a release binary has the SHA-256 of the asset of each platform of the pin,
// and the release of an action has the commit of its tag.
//
// # Errors
//
// Find returns an error that wraps [go.dokimi.dev/ergon/service/baseline/options.ErrDefect] for
// options that a producer declares wrong, and [ErrSource] for an invalid source tag. Resolve returns
// an error that wraps [ErrRegistry] for a registry that fails, and [ErrAsset] for a release binary
// whose new release lacks the asset of a platform of the pin. Each error of Resolve names the key of
// the pin.
//
// # Concurrency
//
// Find is safe for concurrent use. A [Resolver] is safe for concurrent use when its client, its
// GitHub and its clock are.
//
// # Dependency position
//
// Imports the standard library, golang.org/x/mod/module, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option], [go.dokimi.dev/ergon/core/workflow],
// [go.dokimi.dev/ergon/service/baseline/options] and [go.dokimi.dev/ergon/service/forge].
// [go.dokimi.dev/ergon/service/baseline] and the root module import it.
package pin
