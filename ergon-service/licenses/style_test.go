// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package licenses_test

import (
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/licenses"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

// The headers of Dokimasia B.V. under MIT in 2026, in the comment styles of the cases.
const (
	slashes   = "// Copyright Dokimasia B.V. 2026\n// SPDX-License-Identifier: MIT\n\n"
	hashtags  = "# Copyright Dokimasia B.V. 2026\n# SPDX-License-Identifier: MIT\n\n"
	asterisks = "/*\n * Copyright Dokimasia B.V. 2026\n * SPDX-License-Identifier: MIT\n */\n\n"
	angles    = "<!--\n  ~ Copyright Dokimasia B.V. 2026\n  ~ SPDX-License-Identifier: MIT\n-->\n\n"
)

func TestStyle(t *testing.T) {
	t.Parallel()

	t.Run("Fix", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			path string
			give string
			want string
		}{
			{
				name: "writes // into a file of Go",
				path: "main.go",
				give: "package main\n",
				want: slashes + "package main\n",
			},
			{
				name: "writes // into a file of TypeScript",
				path: "app.ts",
				give: "export {};\n",
				want: slashes + "export {};\n",
			},
			{
				name: "writes // after the shebang of a module of JavaScript",
				path: "tool.mjs",
				give: "#!/usr/bin/env node\nconsole.log(1);\n",
				want: "#!/usr/bin/env node\n" + slashes + "console.log(1);\n",
			},
			{
				name: "writes // after the tag of PHP",
				path: "index.php",
				give: "<?php\necho 1;\n",
				want: "<?php\n" + slashes + "echo 1;\n",
			},
			{
				name: "writes // into go.mod",
				path: "go.mod",
				give: "module example.com/demo\n",
				want: slashes + "module example.com/demo\n",
			},
			{name: "writes // into go.work", path: "go.work", give: "go 1.27\n", want: slashes + "go 1.27\n"},
			{
				name: "writes ; into a query of tree-sitter",
				path: "queries/highlights.scm",
				give: "(identifier) @variable\n",
				want: "; Copyright Dokimasia B.V. 2026\n; SPDX-License-Identifier: MIT\n\n(identifier) @variable\n",
			},
			{
				name: "writes # after the parser directives of a Dockerfile",
				path: "Dockerfile",
				give: "# syntax=docker/dockerfile:1\n# escape=`\nFROM scratch\n",
				want: "# syntax=docker/dockerfile:1\n# escape=`\n" + hashtags + "FROM scratch\n",
			},
			{
				name: "writes # after the shebang of a script",
				path: "scripts/run.sh",
				give: "#!/bin/sh\necho hi\n",
				want: "#!/bin/sh\n" + hashtags + "echo hi\n",
			},
			{
				name: "writes an HTML comment into Markdown",
				path: "README.md",
				give: "# Title\n",
				want: angles + "# Title\n",
			},
			{
				name: "writes an HTML comment after the front matter of Markdown and its blank line",
				path: "docs/rfc/0001-design.md",
				give: "---\nrfc: 0001\nstatus: Draft\n---\n\n# Design\n",
				want: "---\nrfc: 0001\nstatus: Draft\n---\n\n" + angles + "# Design\n",
			},
			{
				name: "writes an HTML comment after front matter without a blank line",
				path: "notes.markdown",
				give: "---\ntitle: Notes\n---\n# Notes\n",
				want: "---\ntitle: Notes\n---\n" + angles + "# Notes\n",
			},
			{
				name: "writes a block comment into CSS",
				path: "web/style.css",
				give: "body {}\n",
				want: asterisks + "body {}\n",
			},
			{name: "writes # by a style of the configuration", path: "a.foo", give: "a\n", want: hashtags + "a\n"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := vcstest.Repository(t, files.Tree{tt.path: files.Text(tt.give)})
				report, err := licenses.Fix(t.Context(), dir, styled(), 2026)
				assert.NoError(t, err, "Fix")
				assert.Equal(t, report.Fixed, []string{tt.path}, "the fixed files")
				files.Contains(t, os.DirFS(dir), files.Tree{tt.path: files.Text(tt.want)}, "the fixed file")
			})
		}

		skipped := []struct {
			name string
			path string
		}{
			{name: "skips a file of MDX", path: "docs/page.mdx"},
			{name: "skips a Go template", path: "views/page.tmpl"},
			{name: "skips the wrapper of Gradle", path: "gradlew"},
			{name: "skips the properties of the wrapper of Gradle", path: "gradle/wrapper/gradle-wrapper.properties"},
			{name: "skips the lock of Terraform", path: ".terraform.lock.hcl"},
			{name: "skips a license file of Markdown", path: "LICENSE.md"},
			{name: "skips a file of JSON", path: "package.json"},
			{name: "skips go.sum", path: "go.sum"},
			{name: "skips an extension that two styles claim", path: "cgi-bin/run.fcgi"},
			{name: "skips a file that the configuration gives none", path: "schema.sql"},
			{name: "skips a file without a comment style", path: "notes.txt"},
		}
		for _, tt := range skipped {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := vcstest.Repository(t, files.Tree{tt.path: files.Text("content\n")})
				report, err := licenses.Fix(t.Context(), dir, styled(), 2026)
				assert.NoError(t, err, "Fix")
				assert.Equal(t, report.Skipped, 1, "the skipped files")
				files.Contains(t, os.DirFS(dir), files.Tree{tt.path: files.Text("content\n")}, "the skipped file")
			})
		}
	})
}

// styled returns the configuration of the cases with the styles of a repository: none for .sql,
// and # for .foo.
func styled() *licenses.Config {
	c := config()
	c.Styles = map[string]string{".sql": "none", ".foo": "Hashtag"}
	return c
}
