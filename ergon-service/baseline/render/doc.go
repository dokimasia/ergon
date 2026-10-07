// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package render renders the templates of the producers of ergon init.
//
// A producer's templates are a file tree that mirrors the repository, and the directory of a
// template states the [Class] of its file:
//
//   - managed/<path>.tmpl renders a managed file
//   - seeded/<path>.tmpl renders a seeded file
//   - shared/<path>.tmpl renders the producer's fragment of a shared file, such as .gitignore
//
// [Render] executes every template of every [Unit] with text/template, under the delimiters {{% and
// %}}, so the ${{ }} of a workflow and the {{.Dir}} of go list -f remain text. A template reads
// .Answers, .Options, which are the options of its producer, .Data, which are the values that its
// producer computes, and .Contributions, which [Collect] gathers from every producer. A key that
// the data lacks is an error. Render skips a template that renders no byte, and joins the fragments
// of a shared file in the order of the units.
//
// # Functions
//
//   - words writes a list as words of the shell, escaped for make.
//   - make escapes a value for make.
//   - yaml writes a scalar of YAML, or a list of scalars in flow style, that reads back as the
//     value.
//
// # Errors
//
// Templates that a producer declares wrong return an error that wraps [ErrInvalidTemplate], and a
// contribution that a producer declares wrong an error that wraps [ErrInvalidContribution]. Both
// are defects of the producer. A template that does not execute returns its error, which wraps the
// error of a function that it calls.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language] and
// [go.dokimi.dev/ergon/core/workflow]. The package [go.dokimi.dev/ergon/service/baseline] and the
// tests of the producers import it.
package render
