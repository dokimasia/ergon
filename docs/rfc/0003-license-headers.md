---
rfc: 0003
title: License headers
author: Roy Klopper
status: Draft
created: 2026-09-24
updated: 2026-10-05
discussion: none
supersedes: none
superseded-by: none
produces-adr: tbd
---

# RFC-0003: License headers

## Summary

`ergon license fix` adds and updates the copyright and SPDX header of every file that has a comment syntax. `ergon license check` verifies the headers and exits 1 on a missing, outdated or conflicting one. ergon builds the command on apache/skywalking-eyes v0.9.0, whose packages `pkg/header` and `pkg/comments` generate, match and insert headers for 76 comment-bearing languages. ergon supplies the parts the library gets wrong:

- the file set
- the comment style of the file types the library maps wrongly
- a match pattern that accepts any year
- the removal of an outdated header
- a silent logger The command replaces palantir/go-license, which reads Go files only.

## Motivation

techne, treesitter and assert-go apply and verify headers with go-license and this `.go-license.yml`:

```yaml
header: |
  // Copyright ThesmOS B.V. {{YEAR}}
  // SPDX-License-Identifier: MIT
```

eidos uses the same template with `Apache-2.0`. go-license filters the walk to `.go` files. Its template contains the `//` markers, so it cannot describe a file whose comments start with `#`, and the rest of each repository has no header check. Across techne, eidos, treesitter, the assert repositories, interface and stealth, git tracks 1,183 Go files and 2,635 files of other types with a comment syntax, not counting MDX and Go templates. They include 1,902 TypeScript files, 289 Markdown files, 90 YAML files, 57 Python files and 57 Java files.

skywalking-eyes is the one polyglot header tool written in Go. None of its packages is under `internal/`, so ergon can import every part of it. Run as its own CLI runs it, it is not safe on these repositories. A header fix of a clone of techne, with `header.Check` followed by `header.Fix`, measured:

- It wrote HTML comments, `<!-- ~ … -->`, into all 17 `go.mod` files, because four Linguist languages claim `.mod`. Every `go.mod` then failed to parse with `unknown directive: <!--`, and `go list -m all` failed for the workspace.
- It refused 21 tree-sitter query files, `go.work` and `go.work.sum` as unsupported.
- Its global logrus logger wrote 54,353 lines at debug level to stdout.

The same library with ergon's layer on top fixed every failing file in five repositories without breaking one:

| Repository | Files fixed | Checks run on the result |
|---|---|---|
| techne | 72 | every `go.mod` parses, `go list -m all` and `go list ./...` succeed, 8 YAML files load, `make -n help` succeeds |
| assert-python | 66 | `py_compile` on 50 files, `pyproject.toml` loads, `bash -n` on 2 scripts with the shebang on line 1 |
| assert-typescript | 66 | `tsc --noEmit` exits 0, `bash -n` on 2 scripts |
| assert-java | 80 | `bash -n` on `gradlew` and 2 scripts with the shebang on line 1 |
| assert-rust | 58 | `cargo check` passes, `bash -n` on 2 scripts |

In all five repositories, no file resolved to two different comment styles across 20 lookups.

## Detailed design

### Commands

| Command | Writes | Exit status |
|---|---|---|
| `ergon license fix` | Adds missing headers and rewrites outdated ones | 1 when a file has a conflicting header, 0 otherwise |
| `ergon license check` | Nothing | 1 when any file has a missing, outdated or conflicting header |
| `ergon license check --json` | Nothing | As above, with every finding as JSON on stdout |

`ergon license` without a subcommand prints the subcommands and exits 2. Running `ergon license fix` and then `ergon license check` exits 0, unless a conflict remains.

Files without a comment syntax are skipped. `check` reports how many it skipped, and they never fail the check.

### Configuration

The header is configured in `.ergon.yaml`, without comment markers:

```yaml
license:
  owner: ThesmOS B.V.
  spdx: MIT
  exclude:
    - "**/*.gen.go"
    - "testdata/**"
  styles:
    ".sql": none
```

- ergon renders the header as `Copyright <owner> <year>` and `SPDX-License-Identifier: <spdx>`, which is the current go-license template.
- `spdx` is an SPDX license expression on one line.
- `exclude` takes doublestar globs.
- `styles` maps an extension or a base name to one of skywalking-eyes' style identifiers, such as `DoubleSlash`, `Hashtag` or `Semicolon`, or to `none`. A mapping replaces ergon's built-in overrides and the library's lookup for that key.

### The library's part and ergon's part

| Concern | skywalking-eyes v0.9.0 | ergon |
|---|---|---|
| Comment styles | 20 styles and 76 languages from Linguist data, in `pkg/comments` | Overrides the file types the library maps wrongly, through `comments.OverrideLanguageCommentStyle` |
| Header text | `header.GenerateLicenseHeader` renders it in a style | Sets `LicenseConfig.Content` with the `[owner]` and `[year]` placeholders |
| Matching | `header.CheckFile` normalizes the file and matches the header, or `LicenseConfig.Pattern` | Sets a pattern that accepts any year expression |
| Insertion | `header.InsertComment` inserts after the style's `After` match | Chooses the style, and removes an outdated header first |
| File set | `header.Check` opens `./` with go-git and walks every file | Lists files with `git ls-files` and calls `CheckFile` per file |
| Logging | `logger.Log` writes to stdout at debug level | Sets `logger.Log` to discard, and reports findings itself |

ergon's built-in overrides for v0.9.0:

| Key | Library's style | ergon's style | Reason |
|---|---|---|---|
| `.ts`, `.tsx`, `.mts`, `.cts`, `.js`, `.jsx`, `.mjs`, `.cjs`, `.java`, `.kt`, `.kts`, `.scala`, `.cs`, `.csx` | `SlashAsterisk` | `DoubleSlash` | Every C-family header uses `//`, as Go, Rust and protobuf headers already do in the library |
| `.php` | `PhpTag` | `DoubleSlash`, after the `<?php` line | The same rule for PHP. The `<?php` line remains the first line |
| `.fcgi` | `PhpTag` or `Hashtag`, by map order | none | Two languages claim the extension, and the library picks one at random |
| `.terraform.lock.hcl` | `Hashtag` | none | `terraform init` writes the dependency lock file |
| `.mod`, `go.work` | `AngleBracket`, or none | `DoubleSlash` | `go.mod` and `go.work` accept `//` comments. The library's choice breaks the file |
| `.scm` | none | `Semicolon` | tree-sitter queries use `;` comments |
| `.mdx` | `AngleBracket` | none | MDX 2 and later reject HTML comments |
| `.tmpl` | varies | none | Go templates render their header through `{{template "header" .}}` |
| `gradlew`, `gradlew.bat` | `Hashtag`, `Remark` | none | The Gradle wrapper generates them. A `rem` header above `@rem` also prints on every run |
| `go.sum`, `go.work.sum`, `*.json`, `*.lock`, `LICENSE*`, `COPYING*` | varies | none | These files have no comment syntax |

The library writes TypeScript, TSX, JavaScript, Java, Kotlin, Scala and C# headers as a `/* */` block, from `SlashAsterisk` in its `assets/languages.yaml`, and PHP headers as a `/* */` block after `<?php`. The assert-java run confirmed it for `.java` and `.kt` files. CSS keeps `SlashAsterisk`, because CSS has no line comment. Terraform (`.tf`, `.tfvars`) and Bash (`.sh`, `.bash`, `.bats`) keep the library's `Hashtag`.

The `.mod`, `go.work` and `.scm` overrides correct defects in the library. Each of those overrides is deleted when the pinned version fixes its defect.

### Calling the library

```go
// Package license adds and verifies copyright and SPDX headers.
package license

// Config is the license section of .ergon.yaml.
type Config struct {
	// Owner is the copyright holder, such as "ThesmOS B.V.".
	Owner string

	// SPDX is a single-line SPDX license expression, such as "MIT".
	SPDX string

	// Exclude lists doublestar globs of repository-relative paths.
	Exclude []string

	// Styles maps an extension or a base name to a skywalking-eyes style
	// identifier, or to "none" to leave the file without a header.
	Styles map[string]string
}

// Kind classifies a finding.
type Kind int

const (
	Missing Kind = iota
	Outdated
	Conflict
	Unsupported
)

// Finding is one file whose header does not match the configuration.
type Finding struct {
	// Path is repository-relative and slash-separated.
	Path string

	// Kind is what is wrong with the header.
	Kind Kind

	// Line is the first line that is not header text, for a Conflict.
	// It is 0 for every other kind.
	Line int
}

// Check reports every file under root whose header does not match cfg.
// It writes nothing.
func Check(ctx context.Context, root string, cfg Config) ([]Finding, error)

// Fix adds missing headers and rewrites outdated ones. It returns the
// findings it could not fix: conflicts and unsupported files.
func Fix(ctx context.Context, root string, cfg Config) ([]Finding, error)
```

`Check` and `Fix` run these steps:

1. Set `logger.Log` to discard, and apply the overrides and `cfg.Styles` with `comments.OverrideLanguageCommentStyle`.
2. Build a `header.ConfigHeader` whose `License.Content` is the rendered header and whose `License.Pattern` is `Copyright <owner> \d{4}(?:\s*[-,]\s*\d{4})*\s+SPDX-License-Identifier: <spdx>`.
3. List the files with `git ls-files --cached --others --exclude-standard`, and drop the excluded, generated and style-less files.
4. Call `header.CheckFile` for each file.
5. For each failing file, `Fix` looks for an existing header block in the file's style. It removes a block whose lines are all copyright lines, SPDX tags or empty comment lines. It reports a block with any other line as a conflict and leaves the file unchanged. It then calls `header.InsertComment` with the style ergon resolved.

The overrides change a package-level table in `pkg/comments`, and `logger.Log` is a package-level variable. ergon runs one license configuration per process, so this global state is set once.

### Outdated headers

`header.InsertComment` removes an existing header only when the new pattern matches it, and it runs that removal over the whole file. With a changed license, a test file with `SPDX-License-Identifier: Apache-2.0` came out with the MIT header above the Apache-2.0 header. ergon removes the old block before it calls `InsertComment`, so a file never gets two headers.

### Years

- `check` accepts one year, a list such as `2024, 2026`, or a range such as `2020-2026`.
- `ergon license fix` writes the current year into a new header, and keeps the years of a header it rewrites.
- Neither command reads git history.

A header written in 2026 passes in 2027. The library's own check compares the year literally, and its maintainers declined to change that. The pattern avoids that check.

### Preamble lines

The library's styles keep a shebang in `#` files, a shebang and a PEP 263 line in Python, and `<?xml … ?>` and `<?php` in their styles. ergon adds a style with an `After` pattern where the library has none, for the file types in these repositories:

| Case | ergon's rule |
|---|---|
| A UTF-8 byte-order mark | Removed before `InsertComment` and written back after it |
| `#!` in a JavaScript or TypeScript file | A copy of the library's style with `After` set to the shebang line |
| `<?php` in a PHP file with a `//` header | A copy of `DoubleSlash` with `After` set to the `<?php` line |
| Dockerfile `# syntax=`, `# escape=` and `# check=` | A copy of `Hashtag` with `After` set to the directive lines |

Two of these cases occur in the repositories today. `treesitter/oracles/csharp/csharp.csproj` starts with a byte-order mark. Eleven JavaScript and TypeScript files start with `#!`: `treesitter/oracles/typescript/oracle.mjs`, and ten scripts under `stealth/tools`. No repository tracks a Dockerfile. Each rule has a fixture that proves the file still parses or runs after `ergon license fix`.

### Packages

| Package | Contains | Imports |
|---|---|---|
| `ergon-service/license` | `Config`, `Check`, `Fix`, the overrides, the header-block removal | `core/*`, `service/vcs`, `github.com/apache/skywalking-eyes/pkg/header`, `pkg/comments`, `pkg/logger` |

No language module takes part, and `ergon-lang` gains nothing. The library's table is keyed by file type, and most file types in it have no language module.

### Migration from go-license

1. Move the owner and the SPDX identifier from `.go-license.yml` to the `license` section of `.ergon.yaml`.
2. Delete `.go-license.yml`, and remove go-license from the bootstrap tool list.
3. Run `ergon license check`. Against each repository's own template, 1,121 of the 1,183 Go files pass unchanged: all of treesitter's and assert-go's, 319 of techne's 328 and 593 of eidos's 646. The other 62 have their header below a `//go:build` line or have none. Files of other types report `missing` until `ergon license fix` runs.

## Alternatives considered

### A. A comment table and header logic written in ergon

ergon would own about 35 file-type entries in 6 styles, and 9 preamble rules, in `ergon-lang/comment`.

**Why not:** skywalking-eyes already provides the styles, the rendering, the normalized matching and the insertion, for 76 languages. ergon's layer on top of it is the file set, a few overrides and the header-block removal. A table written in ergon would repeat work the library has done and tested.

### B. Run skywalking-eyes as its own CLI

`license-eye header fix` with a `.licenserc.yaml` in each repository.

**Why not:** measured on techne, it broke all 17 `go.mod` files and refused 23 files. Its configuration can map an extension to a style but cannot change the file set or remove an outdated header, and its logger writes to stdout.

### C. Run hawkeye or reuse-tool as a subprocess

hawkeye handles preamble lines most completely. reuse-tool implements the REUSE specification itself.

**Why not:** every repository would need a Rust or Python binary for one check. hawkeye compares the year literally, and reuse-tool deletes other lines from a block it rewrites.

### D. Keep go-license for Go, and add a second tool for the other types

Go files keep the tool they already pass.

**Why not:** two tools would need two templates that agree. go-license also puts a second header on a file whose year is a range.

## Drawbacks

- ergon depends on a v0.x library whose API may change. ergon pins v0.9.0, and each upgrade needs the five-repository run again.
- The library brings go-git, logrus and 4.57 MB of embedded assets. The probe binary that imports it is 14,999,624 bytes.
- The overrides and `logger.Log` are package-level state. Two license configurations cannot run in one process.
- The library's `LicenseConfig.Pattern` removal runs over the whole file. A file that quotes its own header text in a string loses that text on a rewrite.
- Files without a comment syntax, 538 of them JSON, carry no license information.
- `check` accepts any year, so it does not detect a year that is out of date.

## Unresolved and future work

- Annotating files without a comment syntax through `REUSE.toml`, so that `reuse lint` passes, is not proposed.

- Fixing the `.mod`, `.scm`, logger and lookup defects in skywalking-eyes itself is not proposed here. Each fix upstream removes one override from ergon.
- Updating a year from git history is not proposed.
- Writing SPDX SBOMs or listing the licenses of dependencies is not proposed. skywalking-eyes has a `pkg/deps` package for that.

## References

| What | Where |
|---|---|
| skywalking-eyes v0.9.0 | https://github.com/apache/skywalking-eyes/tree/v0.9.0 |
| `header.CheckFile`, `header.Fix`, `header.InsertComment` | `pkg/header/check.go`, `pkg/header/fix.go`, v0.9.0 |
| `comments.OverrideLanguageCommentStyle`, the map lookup | `pkg/comments/config.go`, v0.9.0 |
| `logger.Log` set to stdout | `pkg/logger/log.go:30-33`, v0.9.0 |
| The library's literal year check, declined | https://github.com/apache/skywalking/issues/10223 |
| go-license v1.50.0, Go files only | https://github.com/palantir/go-license/blob/v1.50.0/golicense/golicense.go |
| hawkeye v7.2.0 | https://github.com/fast/hawkeye/blob/v7.2.0/hawkeye/src/engine/analyze.rs |
| reuse-tool deletes other lines in a rewritten block | https://codeberg.org/fsfe/reuse-tool/issues/1315 |
| REUSE specification 3.3 | https://github.com/fsfe/reuse-website/blob/main/site/content/en/spec-3.3.md |
| The five-repository probe | `/tmp/ergon-eyes/main.go`, run on 2026-09-24 against local clones |
