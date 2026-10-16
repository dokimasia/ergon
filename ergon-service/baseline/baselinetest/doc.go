// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package baselinetest renders the producers of ergon init in a test, as ergon init new renders
// them, and checks the files that they render.
//
// [New] runs [go.dokimi.dev/ergon/service/baseline.Repository.New] in a directory of the test's own
// for a catalog, answers and base producers, so every option is at its baseline and every
// contribution is collected. [Hygiene] checks every file of the directory: no line ends in a
// space or a tab, the file ends in one newline, and a YAML, JSON or TOML file parses. A producer's
// test compares the directory with a golden tree of go.dokimi.dev/assert/golden.
//
// [Keys] checks that go.stealthscale.io/cli decodes each key of the section of a producer, as
// ergon init writes it. The test over the catalog of ergon calls it for every section.
//
// Every check stops the test at its first failure.
//
// # Dependency position
//
// Imports the standard library, go.dokimi.dev/assert, go.stealthscale.io/cli, go.yaml.in/yaml/v3,
// github.com/pelletier/go-toml/v2, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/spdx], [go.dokimi.dev/ergon/core/workspace],
// [go.dokimi.dev/ergon/service/baseline] and [go.dokimi.dev/ergon/service/baseline/options]. The
// tests of the producers import it.
package baselinetest
