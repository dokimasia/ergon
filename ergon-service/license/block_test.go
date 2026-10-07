// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package license_test

import (
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/service/license"
	"go.dokimi.dev/ergon/service/vcs/vcstest"
)

func TestBlock(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			path string
			give string
			want license.Finding
		}{
			{
				name: "returns Missing for a file whose first comment documents its package",
				path: "main.go",
				give: "// Package main runs the demo.\npackage main\n",
				want: license.Finding{Path: "main.go", Kind: license.Missing},
			},
			{
				name: "returns Outdated for a header of another owner",
				path: "main.go",
				give: "// Copyright ThesmOS B.V. 2024\n// SPDX-License-Identifier: MIT\n\npackage main\n",
				want: license.Finding{Path: "main.go", Kind: license.Outdated},
			},
			{
				name: "returns Outdated for a header of a tag of SPDX alone",
				path: "main.go",
				give: "// SPDX-License-Identifier: Apache-2.0\n\npackage main\n",
				want: license.Finding{Path: "main.go", Kind: license.Outdated},
			},
			{
				name: "returns Outdated for a header of another license of the same prefix",
				path: "main.go",
				give: "// Copyright Dokimasia B.V. 2024\n// SPDX-License-Identifier: MIT-0\n\npackage main\n",
				want: license.Finding{Path: "main.go", Kind: license.Outdated},
			},
			{
				name: "returns Conflict at the first line of a header that is other text",
				path: "main.go",
				give: "// Copyright ThesmOS B.V. 2024\n//\n// Licensed under the Apache License.\n\npackage main\n",
				want: license.Finding{Path: "main.go", Kind: license.Conflict, Line: 3},
			},
			{
				name: "returns Conflict for a block comment with other text",
				path: "style.css",
				give: "/*\n * (c) ThesmOS B.V. 2024\n * All styles of the site.\n */\nbody {}\n",
				want: license.Finding{Path: "style.css", Kind: license.Conflict, Line: 3},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := vcstest.Repository(t, files.Tree{tt.path: files.Text(tt.give)})
				report, err := license.Check(t.Context(), dir, config())
				assert.NoError(t, err, "Check")
				assert.Equal(t, report.Findings, []license.Finding{tt.want}, "the findings")
			})
		}
	})

	t.Run("Fix", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			path string
			give string
			want string
		}{
			{
				name: "replaces an outdated header and keeps its range of years",
				path: "main.go",
				give: "// Copyright ThesmOS B.V. 2020-2024\n// SPDX-License-Identifier: Apache-2.0\n\npackage main\n",
				want: "// Copyright Dokimasia B.V. 2020-2024\n// SPDX-License-Identifier: MIT\n\npackage main\n",
			},
			{
				name: "replaces an outdated header and keeps its list of years",
				path: "main.go",
				give: "// Copyright ThesmOS B.V. 2024, 2026\n// SPDX-License-Identifier: MIT\n\npackage main\n",
				want: "// Copyright Dokimasia B.V. 2024, 2026\n// SPDX-License-Identifier: MIT\n\npackage main\n",
			},
			{
				name: "replaces a header without years with the year",
				path: "Makefile",
				give: "# Copyright ThesmOS B.V.\n# SPDX-License-Identifier: MIT\n\nall:\n",
				want: hashtags + "all:\n",
			},
			{
				name: "replaces a header below the build constraint of Go and writes the new one on top",
				path: "main.go",
				give: "//go:build linux\n\n// Copyright ThesmOS B.V. 2025\n// SPDX-License-Identifier: MIT\n\npackage main\n",
				want: "// Copyright Dokimasia B.V. 2025\n// SPDX-License-Identifier: MIT\n\n//go:build linux\n\npackage main\n",
			},
			{
				name: "replaces a block comment",
				path: "style.css",
				give: "/*\n * Copyright ThesmOS B.V. 2026\n * SPDX-License-Identifier: MIT\n */\n\nbody {}\n",
				want: asterisks + "body {}\n",
			},
			{
				name: "replaces a block comment of one line",
				path: "style.css",
				give: "/* Copyright ThesmOS B.V. 2026 */\nbody {}\n",
				want: asterisks + "body {}\n",
			},
			{
				name: "writes the header above a block comment without its end",
				path: "style.css",
				give: "/* Copyright ThesmOS B.V. 2024\nbody {}\n",
				want: asterisks + "/* Copyright ThesmOS B.V. 2024\nbody {}\n",
			},
			{
				name: "replaces a header after a shebang",
				path: "run.sh",
				give: "#!/bin/sh\n# Copyright ThesmOS B.V. 2026\n# SPDX-License-Identifier: MIT\n\necho\n",
				want: "#!/bin/sh\n" + hashtags + "echo\n",
			},
			{
				name: "replaces a header after the front matter of Markdown",
				path: "docs/guide.md",
				give: "---\ntitle: Guide\n---\n\n<!-- Copyright ThesmOS B.V. 2025 -->\n\n# Guide\n",
				want: "---\ntitle: Guide\n---\n\n<!--\n  ~ Copyright Dokimasia B.V. 2025\n  ~ SPDX-License-Identifier: MIT\n-->\n\n" +
					"# Guide\n",
			},
			{
				name: "writes the header after the shebang and the coding line of Python",
				path: "tool.py",
				give: "#!/usr/bin/env python3\n# -*- coding: utf-8 -*-\nprint(1)\n",
				want: "#!/usr/bin/env python3\n# -*- coding: utf-8 -*-\n" + hashtags + "print(1)\n",
			},
			{
				name: "writes the header after a shebang that ends the file",
				path: "run.sh",
				give: "#!/bin/sh",
				want: "#!/bin/sh\n" + hashtags,
			},
			{
				name: "wraps the header in the tags of PHP in a file without them",
				path: "page.php",
				give: "<p>hi</p>\n",
				want: "<?php\n" + slashes + "?><p>hi</p>\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := vcstest.Repository(t, files.Tree{tt.path: files.Text(tt.give)})
				report, err := license.Fix(t.Context(), dir, config(), 2026)
				assert.NoError(t, err, "Fix")
				assert.Empty(t, report.Findings, "the findings")
				files.Contains(t, os.DirFS(dir), files.Tree{tt.path: files.Text(tt.want)}, "the fixed file")
			})
		}
	})
}
