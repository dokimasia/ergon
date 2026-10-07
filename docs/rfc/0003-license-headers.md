---
rfc: 0003
title: Licenses
author: Roy Klopper
status: Accepted
created: 2026-09-24
updated: 2026-10-07
discussion: none
supersedes: none
superseded-by: none
produces-adr: tbd
---

# RFC-0003: Licenses

## Summary

ergon supports 44 licenses by their SPDX identifiers: the licenses for software of GitHub's license list, and BUSL-1.1. `core/spdx` declares the identifiers. `service/license` renders the text of each license, and its producer contributes `LICENSE` to `ergon init`, with a `NOTICE` for Apache-2.0.

`ergon license fix` adds and updates the copyright and SPDX header of every file that has a comment syntax. `ergon license check` verifies the headers and exits 1 on a missing, outdated or conflicting one. ergon builds the command on apache/skywalking-eyes v0.9.0, whose packages `pkg/header` and `pkg/comments` generate, match and insert headers for 76 comment-bearing languages. ergon supplies the parts the library gets wrong:

- the file set
- the comment style of the file types the library maps wrongly
- a match pattern that accepts any year
- the removal of an outdated header
- a silent logger

The command replaces palantir/go-license, which reads Go files only.

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

`ergon license fix` fixed every failing file in six repositories without breaking one. Each run started from a clone that `ergon init new --force` had adopted, so the counts include the files that `init` seeds:

| Repository | Files fixed | Checks run on the result |
|---|---|---|
| techne | 72 | `go list -m all` and `go list ./...` succeed in all 17 modules, the tests of the 10 packages that embed the `.scm` queries pass, 17 YAML files load, `make help` succeeds |
| treesitter | 64 | `go list ./...` succeeds in all 11 modules, the tests of the 10 packages of queries pass, MSBuild loads the `.csproj` that starts with a byte-order mark, `node --check`, `ruby -c`, `py_compile` and `cargo metadata` pass on the oracles, `bash -n` on 3 scripts |
| assert-python | 68 | `py_compile` on 50 files, `pyproject.toml` loads, pytest passes 1,680 tests, `bash -n` on 2 scripts with the shebang on line 1 |
| assert-typescript | 69 | `tsc --noEmit` and the build exit 0, vitest passes the 521 tests that pass at the same commit without the headers, `bash -n` on 2 scripts |
| assert-java | 80 | `./gradlew build` succeeds with its tests and javadoc, `bash -n` on `gradlew` and 2 scripts |
| assert-rust | 62 | `cargo test` passes every target and the doc tests, `bash -n` on 2 scripts |

markdownlint-cli2 0.23.3 reports one finding in the tracked Markdown files of the six repositories: a double blank line that an RFC of assert-rust has without the header.

The repositories differ in their licenses as well as in their headers. Of the 17 repositories that the earlier Go-only ergon set up and that no later repository superseded, 6 are under MIT, 5 under Apache-2.0 and 1 under BUSL-1.1, and 5 have no `LICENSE`.

## Detailed design

### Commands

| Command | Writes | Exit status |
|---|---|---|
| `ergon license fix` | Adds missing headers and rewrites outdated ones | 1 when a file has a conflicting header, 0 otherwise |
| `ergon license check` | Nothing | 1 when any file has a missing, outdated or conflicting header |
| `ergon license check --json` | Nothing | As above, with the report as one JSON object on stdout |

`ergon license` without a subcommand prints the subcommands and exits 2. Running `ergon license fix` and then `ergon license check` exits 0, unless a conflict remains.

Both commands work on the repository of the working directory, or of the nearest of its parents that has `.ergon/init.lock`. They read the section `license` of its `.ergon.yaml` as every command of `ergon init` resolves it.

Each command prints the kind and the path of each finding, with the line of a conflict, and then the number of files that it checked and skipped. `fix` first prints the path of each file that it wrote. Files without a comment syntax are skipped, and they never fail the check. A file with a comment syntax whose content is not text, such as a file with a null byte, is reported `unsupported` and fails neither command.

### Licenses

The licenses are the licenses for software of GitHub's license list, choosealicense.com at commit f717b23, under their current SPDX identifiers, and BUSL-1.1. The 44 identifiers are `0BSD`, `AFL-3.0`, `AGPL-3.0-only`, `AGPL-3.0-or-later`, `Apache-2.0`, `Artistic-2.0`, `BlueOak-1.0.0`, `BSD-2-Clause`, `BSD-2-Clause-Patent`, `BSD-3-Clause`, `BSD-3-Clause-Clear`, `BSD-4-Clause`, `BSL-1.0`, `BUSL-1.1`, `CC0-1.0`, `CECILL-2.1`, `ECL-2.0`, `EPL-1.0`, `EPL-2.0`, `EUPL-1.1`, `EUPL-1.2`, `GPL-2.0-only`, `GPL-2.0-or-later`, `GPL-3.0-only`, `GPL-3.0-or-later`, `ISC`, `LGPL-2.1-only`, `LGPL-2.1-or-later`, `LGPL-3.0-only`, `LGPL-3.0-or-later`, `MIT`, `MIT-0`, `MPL-2.0`, `MS-PL`, `MS-RL`, `MulanPSL-2.0`, `NCSA`, `OSL-3.0`, `PostgreSQL`, `Unlicense`, `UPL-1.0`, `Vim`, `WTFPL` and `Zlib`.

- GitHub's list also names nine licenses for works other than software: CC-BY-4.0, CC-BY-SA-4.0, CERN-OHL-P-2.0, CERN-OHL-S-2.0, CERN-OHL-W-2.0, GFDL-1.3, LPPL-1.3c, ODbL-1.0 and OFL-1.1. ergon sets up software repositories, so it leaves them out.
- GitHub's list names each GNU license by its version, such as `GPL-3.0`, an identifier that the SPDX License List deprecates. ergon accepts the two current identifiers of each version, `-only` and `-or-later`, and renders one text for both. The header of each file states the choice.
- ergon supports no license outside GitHub's list except BUSL-1.1.

The text of a license is the body of its file in choosealicense.com: the text after the front matter, without the blank lines before it, and with one newline at its end. That is the text that GitHub's license API serves, and the `LICENSE` that GitHub writes for a new repository. On 2026-10-07 the API served the same text for 37 of the 38 files. Its zlib text had two spaces after a full stop where the commit has one. The `LICENSE` files of core, kanon and thesmos/service match the Apache-2.0 text byte for byte.

ergon fills the fields of a text from the answers of `ergon init`:

| Field | Value |
|---|---|
| `[year]` | The year |
| `[fullname]` | The owner |
| `[project]` | The name of the repository |
| `[projecturl]` | `https://github.com/<repository>` |

No other field occurs in the 38 texts. A text without a field, such as the Apache License 2.0 or the GPL, names no owner. Apache-2.0 also gets a `NOTICE` that contains the name of the repository and `Copyright <year> <owner>`.

ergon takes the text of BUSL-1.1, which GitHub's list lacks, from the SPDX License List 3.29.0, whose text contains the terms. Each licensor states its parameters before the terms, so ergon renders the title of the text, then the parameters from the `license` section, then the rest of the text:

```text
Business Source License 1.1

Parameters

Licensor:             <owner>
Licensed Work:        <licensed-work>
Additional Use Grant: <additional-use-grant>
Change Date:          <change-date>
Change License:       <change-license>

License text copyright © 2017 MariaDB Corporation Ab, All Rights Reserved.
```

`ergon init` fails for BUSL-1.1 until the section sets each parameter, and the error names each empty one.

### Configuration

The `license` section of `.ergon.yaml` configures the headers and the license files, without comment markers:

```yaml
license:
  owner: ThesmOS B.V.
  spdx: MIT
  parameters:
    licensed-work: ""
    additional-use-grant: ""
    change-date: ""
    change-license: ""
  styles:
    ".sql": none
  exclude:
    - "**/*.gen.go"
    - "testdata/**"
  ci:
    actions: {}
    timeout: 10
```

- `owner` and `spdx` are answers of `ergon init`, which writes them into the section. `ergon init sync --owner` and `--license` change them, as [Changing the license](#changing-the-license) specifies.
- `spdx` is one of the identifiers of [Licenses](#licenses).
- ergon renders the header as `Copyright <owner> <year>` and `SPDX-License-Identifier: <spdx>`, which is the current go-license template.
- `parameters` are the parameters of BUSL-1.1, and every other license ignores them.
- `styles` maps an extension or a base name to one of skywalking-eyes' style identifiers, such as `DoubleSlash`, `Hashtag` or `Semicolon`, or to `none`. A mapping replaces every other rule of ergon for that key.
- `exclude` takes doublestar globs.
- `ci.timeout` is the limit in minutes of the job `license` of `ci.yml`. The job runs no action of its own, so `ci.actions` is empty.

The section is the options of the license producer: the struct `Config` of `service/license`, which a strict decode reads. A key that the struct does not have is an error, as it is in every section of `.ergon.yaml`.

### Changing the license

`ergon init sync --license <identifier>` records the new license in the lock, writes it into `license.spdx` and renders `LICENSE` again. It writes `NOTICE` for Apache-2.0 and removes it for every other license, and it fails for BUSL-1.1 until `parameters` states each parameter. `ergon init sync --owner <holder>` changes the owner the same way. A text without a field of the owner, such as the GPL, does not change.

`ergon license check` then reports each header that states the earlier license or owner as `outdated`. `ergon license fix` rewrites each one and keeps its years.

The lock records the answers of `ergon init`, so `owner` and `spdx` change only through those flags. A hand edit of either key fails every command that reads `.ergon.yaml`, with an error that names the key, its value, the flag and the answer:

```text
ergon: options: invalid .ergon.yaml: license.spdx "Apache-2.0", which ergon init sync --license sets, differs from the answer "MIT" of ergon init
```

`ergon init sync` with that flag and the value of the key records the new answer. While `sync` changes an answer, the key may still state the answer that the lock records. `ergon init new` writes its answers over the keys of an existing section.

### The library's part and ergon's part

| Concern | skywalking-eyes v0.9.0 | ergon |
|---|---|---|
| Comment styles | 20 styles and 76 languages from Linguist data, in `pkg/comments` | Builds a table of its own from the same assets, with overrides for the file types the library maps wrongly, and resolves a file by its base name and then by its longest extension |
| Header text | `header.GenerateLicenseHeader` renders it in a style | Sets `LicenseConfig.Content` with the `[owner]` and `[year]` placeholders |
| Matching | `header.CheckFile` normalizes the file and matches the header, or `LicenseConfig.Pattern` | Matches the file in memory, normalized by `license.NormalizeHeader`, against a pattern that accepts any year expression, normalized by `ConfigHeader.NormalizedPattern` |
| Insertion | `header.InsertComment` removes each header that the pattern matches, and inserts after the style's `After` match | Removes an outdated header itself, and inserts the header where the library inserts it, in one write that keeps a byte-order mark and the mode of the file |
| File set | `header.Check` opens `./` with go-git and walks every file | Lists files with `git ls-files` and reads each file once |
| Logging | `logger.Log` writes to stdout at debug level | Sets `logger.Log` to discard, and reports findings itself |

ergon's built-in overrides for v0.9.0:

| Key | Library's style | ergon's style | Reason |
|---|---|---|---|
| `.ts`, `.tsx`, `.mts`, `.cts`, `.js`, `.jsx`, `.mjs`, `.cjs`, `.java`, `.kt`, `.kts`, `.scala`, `.cs`, `.csx` | `SlashAsterisk` | `DoubleSlash`, after a shebang | Every C-family header uses `//`, as Go, Rust and protobuf headers already do in the library |
| `.php` | `PhpTag` | `DoubleSlash`, after the `<?php` line | The same rule for PHP. The `<?php` line remains the first line |
| An extension or a base name that two styles claim, such as `.fcgi` | `PhpTag` or `Hashtag`, by map order | none | The library picks one of the two at random |
| `.terraform.lock.hcl` | `Hashtag` | none | `terraform init` writes the dependency lock file |
| `.mod`, `go.work` | `AngleBracket`, or none | `DoubleSlash` | `go.mod` and `go.work` accept `//` comments. The library's choice breaks the file |
| `.scm` | none | `Semicolon` | tree-sitter queries use `;` comments |
| `.mdx` | `AngleBracket` | none | MDX 2 and later reject HTML comments |
| `.tmpl` | varies | none | Go templates render their header through `{{template "header" .}}` |
| `gradlew`, `gradlew.bat`, `gradle-wrapper.properties` | `Hashtag`, `Remark`, `Hashtag` | none | `gradle wrapper` generates them, and writes them again without a header. A `rem` header above `@rem` also prints on every run |
| `go.sum`, `go.work.sum`, `*.json`, `*.lock`, `LICENSE*`, `COPYING*`, `NOTICE` | varies | none | These files have no comment syntax |

The keys of the preamble rules of [Preamble lines](#preamble-lines) are overrides too: `Dockerfile` and `.dockerfile`, and `.md` and `.markdown`. A license file, such as `LICENSE.md`, has no header whatever its extension. A key of `styles` in the section replaces every rule for that key, the license files included.

The library writes TypeScript, TSX, JavaScript, Java, Kotlin, Scala and C# headers as a `/* */` block, from `SlashAsterisk` in its `assets/languages.yaml`, and PHP headers as a `/* */` block after `<?php`. The assert-java run confirmed it for `.java` and `.kt` files. CSS keeps `SlashAsterisk`, because CSS has no line comment. Terraform (`.tf`, `.tfvars`) and Bash (`.sh`, `.bash`, `.bats`) keep the library's `Hashtag`.

The `.mod`, `go.work` and `.scm` overrides correct defects in the library. Each of those overrides is deleted when the pinned version fixes its defect.

### Calling the library

```go
// Package license declares the licenses that ergon supports, renders their
// texts, and adds and verifies copyright and SPDX headers.
package license

// Config is the license section of .ergon.yaml.
type Config struct {
	// Owner is the copyright holder, such as "ThesmOS B.V.".
	Owner string

	// SPDX is one of the identifiers of core/spdx, such as "MIT".
	SPDX spdx.ID

	// Parameters are the parameters of BUSL-1.1.
	Parameters Parameters

	// Styles maps an extension or a base name to a skywalking-eyes style
	// identifier, or to "none" to leave the file without a header.
	Styles map[string]string

	// Exclude lists doublestar globs of repository-relative paths.
	Exclude []string

	// CI is the limit of the job license of ci.yml.
	CI option.CI[struct{}]
}

// Parameters are the parameters that the LICENSE of BUSL-1.1 states
// before its terms. The owner is the Licensor.
type Parameters struct {
	LicensedWork       string
	AdditionalUseGrant string
	ChangeDate         string
	ChangeLicense      string
}

// Holder is what the fields of a license text name.
type Holder struct {
	Owner      string
	Name       string
	Repository string
	Year       int
}

// Text returns the LICENSE of the license of c for h, and the NOTICE,
// which only Apache-2.0 has.
func Text(c *Config, h Holder) (text, notice []byte, err error)

// Kind is what is wrong with the header of a file.
type Kind string

const (
	Missing     Kind = "missing"
	Outdated    Kind = "outdated"
	Conflict    Kind = "conflict"
	Unsupported Kind = "unsupported"
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

// Report is what Check or Fix found in a repository.
type Report struct {
	// Findings are the files whose header does not match the
	// configuration. Fix reports the files that it leaves: the conflicts
	// and the unsupported files.
	Findings []Finding

	// Fixed are the files that Fix wrote.
	Fixed []string

	// Checked and Skipped count the files with and without a header.
	Checked, Skipped int
}

// Check reports every file under root whose header does not match c.
// It writes nothing.
func Check(ctx context.Context, root string, c *Config) (Report, error)

// Fix adds the missing headers with year, and rewrites the outdated ones
// with their years. It reports the files that it could not fix.
func Fix(ctx context.Context, root string, c *Config, year int) (Report, error)
```

`Check` and `Fix` run these steps:

1. Set `logger.Log` to discard, once per process. Resolve the style of each file from `c.Styles`, then none for a license file, then ergon's overrides, then the library's table, each by the base name and then by the longest extension.
2. Build a `header.ConfigHeader` whose `License.Content` is the rendered header and whose pattern is `Copyright <owner> \d{4}(?:\s*[-,]\s*\d{4})*\s+SPDX-License-Identifier: <spdx>(?:\s|$)`, normalized by `NormalizedPattern`.
3. List the files with `git ls-files --cached --others --exclude-standard`, and drop the excluded files, the generated files, the files without a style and every path that is no regular file. A file is generated when its first line contains `Code generated … DO NOT EDIT` or `Managed by ergon init`.
4. Normalize each file with `license.NormalizeHeader`, and match the pattern.
5. For each failing file, `Fix` looks for an existing header block in the file's style, after its preamble: the first comment block with a copyright line or an SPDX tag, among the comment blocks and blank lines before the first other line. It removes a block whose lines are all copyright lines, SPDX tags or empty comment lines, and keeps its years. It reports a block with any other line as a conflict and leaves the file unchanged. It then inserts the text of `header.GenerateLicenseHeader` after the preamble of the style, as the library inserts it, in one write.

`logger.Log` is a package-level variable of the library, so ergon sets it to discard every message, once per process. ergon resolves the styles from a table of its own and changes no table of the library.

### Outdated headers

`header.InsertComment` removes an existing header only when the new pattern matches it, and it runs that removal over the whole file. With a changed license, a test file with `SPDX-License-Identifier: Apache-2.0` came out with the MIT header above the Apache-2.0 header. ergon removes the old block itself and inserts the new header without that removal, so a file never gets two headers.

### Years

- `check` accepts one year, a list such as `2024, 2026`, or a range such as `2020-2026`.
- `ergon license fix` writes the current year into a new header, and keeps the years of a header it rewrites.
- Neither command reads git history.

A header written in 2026 passes in 2027. The library's own check compares the year literally, and its maintainers declined to change that. The pattern avoids that check.

### Preamble lines

The library's styles keep a shebang in `#` files, a shebang and a PEP 263 line in Python, and `<?xml … ?>` and `<?php` in their styles. ergon adds a style with an `After` pattern where the library has none, for the file types in these repositories:

| Case | ergon's rule |
|---|---|
| A UTF-8 byte-order mark | Removed before the insertion and written back after it |
| `#!` in a JavaScript or TypeScript file | A copy of the library's style with `After` set to the shebang line |
| `<?php` in a PHP file with a `//` header | A copy of `DoubleSlash` with `After` set to the `<?php` line |
| Dockerfile `# syntax=`, `# escape=` and `# check=` | A copy of `Hashtag` with `After` set to the directive lines |
| YAML front matter that opens a Markdown file | A copy of `AngleBracket` with `After` set to the front matter, from its line `---` to the next line `---`, and the blank lines after it |

Three of these cases occur in the repositories today. `treesitter/oracles/csharp/csharp.csproj` starts with a byte-order mark. Eleven JavaScript and TypeScript files start with `#!`: `treesitter/oracles/typescript/oracle.mjs`, and ten scripts under `stealth/tools`. The RFCs of ergon open with YAML front matter. No repository tracks a Dockerfile. Each rule has a fixture that proves the file still parses or runs after `ergon license fix`, and markdownlint-cli2 0.23.3 accepts the 24 Markdown files of ergon after it.

### The license files of ergon init

The license producer is a base producer of `ergon init`, beside the producers of the common files and the GitHub files. It renders `LICENSE` as a managed file. For Apache-2.0 it also renders `NOTICE`. Its options are the `license` section. Its templates call `license.Text`. Its job `license` in the workflow of the gate runs `ergon license check`.

### Packages

| Package | Contains | Imports |
|---|---|---|
| `ergon-core/spdx` | `ID`, the 44 identifiers and `ID.Valid` | the standard library |
| `ergon-service/license` | `Config`, `Text` and the texts of the 44 licenses, `Check`, `Fix`, the table of styles with the overrides, and the header-block removal | `core/*`, `service/vcs`, skywalking-eyes `assets`, `pkg/comments`, `pkg/header`, `pkg/license` and `pkg/logger`, logrus, `github.com/bmatcuk/doublestar/v4` |
| `ergon-service/license/baseline` | The license producer of `ergon init` | `core/*`, `service/license` |

No language module takes part, and `ergon-lang` gains nothing. The library's table is keyed by file type, and most file types in it have no language module.

### Migration from go-license

1. Delete `.go-license.yml`, and the `license` section of the `.ergon.yaml` of the earlier ergon. The license producer has none of its keys `config_file`, `exclude_dirs` and `exclude_files`, so every command of `ergon init` fails on the section. On 2026-10-07 each of the 20 `.ergon.yaml` files of the earlier ergon on the development machine had it.
2. Adopt `ergon init` as RFC-0004 specifies, with the owner and the identifier of `.go-license.yml` as `--owner` and `--license`. `init` writes both into the new `license` section.
3. Add each file that the repository copies from another project to `license.exclude`, such as the queries that treesitter vendors from its grammars. A header names the repository's owner.
4. Add each glob of `exclude_files` to `license.exclude` as a doublestar glob, such as `**/*.gen.go`, unless its files open with a `Code generated … DO NOT EDIT` line, which ergon skips.
5. Run `ergon license check`. Against each repository's own template, 1,121 of the 1,183 Go files pass unchanged: all of treesitter's and assert-go's, 319 of techne's 328 and 593 of eidos's 646. The other 62 have their header below a `//go:build` line or have none. Files of other types report `missing` until `ergon license fix` runs.
6. Run `ergon license fix`, and review the diff.

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

### E. Every license of the SPDX License List

The texts and templates of the SPDX License List 3.29.0 cover its 708 current licenses.

**Why not:** many of those licenses name the copyright holder of one project in their text, such as `AMD-newlib`, whose notice is `Copyright 1990 Advanced Micro Devices, Inc.`. The SPDX texts also mark the copyright notice inconsistently: the template of BSD-2-Clause states it as `<owner>.`, while its text has `<owner>` without the full stop, and the text of zlib omits the notice that its template declares. GitHub's texts state each notice as a field.

## Drawbacks

- ergon depends on a v0.x library whose API may change. ergon pins v0.9.0, and each upgrade needs the six-repository run again.
- The library brings go-git, logrus and 4.57 MB of embedded assets. The probe binary that imports it is 14,999,624 bytes.
- `logger.Log` is package-level state of the library, which discards every message for the whole process once ergon has set it.
- Files without a comment syntax, 538 of them JSON, contain no license information.
- `check` accepts any year, so it does not detect a year that is out of date.
- A license outside the 44 needs a change to ergon.
- The texts follow GitHub's formatting, which differs in places from the text of a license's author: GitHub's Apache License 2.0 lacks the blank line that opens the text at apache.org.
- A Markdown file that opens with a thematic break `---` and has a second one reads as front matter, so its header follows the second break.
- A file that the repository copies from another project gets a header that names the repository's owner, unless `license.exclude` names the file. In treesitter, `ergon license fix` put that header on the 11 queries that the repository vendors from its grammars. `fix` also replaces a header of the other project whose lines are copyright lines and SPDX tags alone.

## Unresolved and future work

- Annotating files without a comment syntax through `REUSE.toml`, so that `reuse lint` passes, is not proposed.
- Fixing the `.mod`, `.scm`, logger and lookup defects in skywalking-eyes itself is not proposed here. Each fix upstream removes one override from ergon.
- Updating a year from git history is not proposed.
- Writing SPDX SBOMs or listing the licenses of dependencies is not proposed. skywalking-eyes has a `pkg/deps` package for that.
- A repository under two licenses, such as `MIT OR Apache-2.0`, is not proposed.

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
| The probe of the library on techne | `/tmp/ergon-eyes/main.go`, run on 2026-09-24 against a local clone |
| The six-repository run | `~/.cache/ergon-build/proper.sh`, run on 2026-10-07 against local clones |
| GitHub's license list | https://github.com/github/choosealicense.com/tree/f717b235f404c60b656bfbc1643f623d02ec10a7/_licenses |
| GitHub's license API | https://docs.github.com/rest/licenses/licenses |
| The SPDX License List 3.29.0, and the text of BUSL-1.1 | https://github.com/spdx/license-list-data/tree/v3.29.0, `text/BUSL-1.1.txt` |
| The survey of the repositories of the earlier ergon | `~/.cache/ergon-migration/`, run on 2026-10-07 |
