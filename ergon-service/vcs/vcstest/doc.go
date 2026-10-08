// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package vcstest runs the tests of a package that calls git in working trees of the test's own.
//
// [Isolate] removes the variables of git from the environment of the test binary, such as the
// GIT_DIR and the GIT_INDEX_FILE that a hook of git exports for its own repository, and points git
// at no configuration of the user or of the system. It also turns off the automatic maintenance of
// git, so no process of git runs in a working tree of a test after the command that started it
// returns. A TestMain calls it before the tests run, so a gate that a hook of git runs tests the
// working trees of the tests alone. [Repository] creates a
// working tree with files, and [Git] runs a command of git in one, without the hooks of the
// repository.
//
// # Dependency position
//
// Imports the standard library and go.dokimi.dev/assert. The tests of
// [go.dokimi.dev/ergon/service/vcs], [go.dokimi.dev/ergon/service/licenses],
// [go.dokimi.dev/ergon/service/release], the release of the module of Go and the command line of
// ergon import it.
package vcstest
