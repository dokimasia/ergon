// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package licenses renders the texts of the licenses that ergon supports, and adds and checks the
// copyright and SPDX header of every file of a repository that has a comment syntax.
//
// The name of its directory differs from LICENSE in more than case. The go command adds the
// LICENSE of the repository to the zip of the module, and refuses a zip with two paths whose names
// differ in case alone.
//
// [Config] is the section license of .ergon.yaml: the owner and the license, the parameters of
// BUSL-1.1, each [Directory] under a license of its own, the comment styles, the files without a
// header, and the limit of the job license of ci.yml. The producer of the license files of ergon
// init declares it as its options.
//
// # Texts
//
// [Text] returns the LICENSE of a license, and the NOTICE of Apache-2.0. The texts are the bodies of
// the licenses of choosealicense.com at commit f717b23, which GitHub's license API serves, and the
// text of BUSL-1.1 of the SPDX License List 3.29.0. Each GNU license has one text for its
// identifiers -only and -or-later.
//
// # Headers
//
// [Check] reports each file whose header does not match the configuration, and [Fix] writes the
// header of each file that has none or an outdated one. A header states two lines in the comment
// style of its file:
//
//	Copyright <owner> <years>
//	SPDX-License-Identifier: <spdx>
//
// <spdx> is the license of the deepest directory of [Config.Directories] that contains the file,
// or the license of the section for every other file.
//
// The commands work on the files that git tracks or would track, as git ls-files lists them, and
// skip the files that the configuration excludes, the changesets of .changeset, whose body is the
// entry of the changelogs, the changelogs CHANGELOG.md, which ergon release version creates without
// a header, the files that a tool generated or that ergon init manages, and the files without a
// comment style. A header matches with any year, list of years or range of years, so a header of
// 2026 matches in 2027.
//
// # skywalking-eyes
//
// The commands build on apache/skywalking-eyes v0.9.0: its comment styles and its languages, the
// check of its package header, which normalizes a file and matches the header, and its insertion of
// a header after the preamble of a style, such as a shebang line. ergon adds what the library does
// not state:
//
//   - The comment style of each file, which ergon resolves from the library's table by the base
//     name of the file and then by its longest extension, where the library picks a match in the
//     order of a map. A file type that two styles claim, such as .fcgi, has no header.
//   - The styles of the file types that the library maps wrongly, such as // for TypeScript and
//     go.mod, and no header for the files without a comment syntax.
//   - The preambles that the library does not keep: a shebang line before //, the <?php line of
//     PHP, the parser directives of a Dockerfile, the YAML front matter of a Markdown file, and the
//     byte-order mark of UTF-8.
//   - The removal of an outdated header before the insertion, so a file never has two headers.
//
// # Global state
//
// The commands set the logger of skywalking-eyes, a variable of its package logger, to discard
// every message, once per process. They change no other state of the library, so the commands are
// safe for concurrent use.
//
// # Errors
//
// [Config.Validate] and Text return errors that wrap
// [go.dokimi.dev/ergon/core/option.ErrInvalid]. Check and Fix return the errors of git, of the
// files, and of the context.
//
// # Dependency position
//
// Imports the standard library, github.com/apache/skywalking-eyes, github.com/sirupsen/logrus,
// github.com/bmatcuk/doublestar/v4, go.yaml.in/yaml/v3, [go.dokimi.dev/ergon/core/changeset],
// [go.dokimi.dev/ergon/core/option], [go.dokimi.dev/ergon/core/spdx] and
// [go.dokimi.dev/ergon/service/vcs]. The producer of the license files and internal/cli of the
// root module import it.
package licenses
