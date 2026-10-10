// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package tool installs and runs the tools that the sections of .ergon.yaml name, for ergon tool
// run.
//
// A section names each tool of its producer by the tool's own name, in its group tools, and the type
// of the field states the kind of the tool, as [go.dokimi.dev/ergon/core/option] declares the
// kinds. [Runner.Run] installs a tool into its cache once, under the kind, the name, the version and
// the platform of the tool, and reuses it on every later run:
//
//   - A release binary, such as commitlint or shellcheck, downloads the asset of the platform. The
//     runner checks the asset against the SHA-256 that the pin states for the platform before it
//     unpacks the program from a .tar.gz, a .tar.xz or a .zip, or takes the asset as the program.
//   - A Maven artifact downloads its jar, which the runner checks against the .sha256 file beside
//     it in Maven Central, and runs with java -jar.
//   - A Go module installs with go install, and a crate with cargo install --locked. Both check
//     what they download against the checksums of their registry. A Go module installs once for
//     each version of the go command in the working directory, because a program that reads the
//     packages of Go can refuse a go command of another version.
//   - A Go module whose field has the tag plugins, such as golangci-lint, builds again with
//     golangci-lint custom when the option that the tag names lists a module plugin. The cache
//     keeps the program of each set of plugins beside the program of go install.
//   - A PyPI package runs through the release binary of uv of its section, and an npm package with
//     npx, which install it into their own caches.
//   - The Composer packages of a section install together into one project, so a package such as
//     phpstan-strict-rules extends another, and run with php. The project allows the plugins of
//     its packages, such as the extension installer of PHPStan.
//
// A tool runs in the working directory of the command, such as a module of Go, with the arguments
// of the command, and the command exits with its exit status. A toolchain that installs a tool
// writes its output to the standard error, so the standard output of the tool stays its own.
// [Runner.RunRelease] installs and runs a release binary of no section the same way, such as a
// release of ergon itself.
//
// # Pruning
//
// [Runner.Prune] removes each file of the cache that is not the install of a tool of the sections
// at the version that they name, such as an earlier version of a tool or the program of a Go module
// for another version of the go command. ergon tool prune calls it.
//
// A job of the managed workflows keeps the tool directory in the cache of GitHub Actions, and saves
// a new cache for each change of the files of its key. [PruneCaches] deletes each cache that a newer
// cache of the same job replaced, through a [CacheForge]. ergon tool ci prune calls it.
//
// # Errors
//
// Run returns an error that wraps [ErrUnknown] for a tool that the section does not name, which
// lists the tools of the section, and [ErrInstall] for a tool that does not install. Prune returns
// an error that wraps ErrInstall when a section has no plugins at the key of the tag plugins of a
// tool, and the error of a file that it cannot remove. PruneCaches returns the error of its
// CacheForge.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/ergon/core/language],
// [go.dokimi.dev/ergon/core/option], [go.dokimi.dev/ergon/service/forge] for the type of a cache,
// and github.com/ulikunitz/xz for the .tar.xz of a release binary. internal/cli of the root module
// imports it.
package tool
