// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package workspace_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	coreworkspace "go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/python/workspace"
)

// manifestFile is the name of the file in the errors of the cases.
const manifestFile = "pyproject.toml"

// project is a pyproject.toml with every key that a release reads.
const project = `[project]
name = "Probe_B"
version = '0.1.0'  # the version
classifiers = ["Programming Language :: Python"]
dependencies = [
  "probe-a>=0.1.0",
  "tqdm (>=4)",
]
optional-dependencies = { cli = ["probe-a[cli]"] }

[dependency-groups]
dev = ["probe-a", {include-group = "lint"}]

[tool.uv]
dev-dependencies = ["probe-c==0.1.0"]
workspace.members = ["packages/*"]
workspace.exclude = ["packages/seeds"]

[[tool.uv.index]]
name = "internal"
`

func TestManifest(t *testing.T) {
	t.Parallel()

	t.Run("ParseManifest", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the keys that a release reads", func(t *testing.T) {
			t.Parallel()
			got, err := workspace.ParseManifest(manifestFile, []byte(project))
			assert.NoError(t, err, "ParseManifest")
			start := strings.Index(project, `'0.1.0'`)
			entry := func(text, kind string) workspace.Entry {
				r, err := workspace.ParseRequirement(text)
				assert.NoError(t, err, "ParseRequirement of "+text)
				at := strings.Index(project, `"`+text+`"`)
				return workspace.Entry{
					Requirement: r, Text: text, Kind: coreworkspace.Kind(kind),
					Span: workspace.Span{Start: at, End: at + len(text) + 2},
				}
			}
			assert.Equal(t, got, &workspace.Manifest{
				Requirements: []workspace.Entry{
					entry("probe-a>=0.1.0", "runtime"),
					entry("tqdm (>=4)", "runtime"),
					entry("probe-a[cli]", "optional"),
					entry("probe-a", "dev"),
					entry("probe-c==0.1.0", "dev"),
				},
				Members:     []string{"packages/*"},
				Exclude:     []string{"packages/seeds"},
				Name:        "Probe_B",
				Version:     "0.1.0",
				VersionSpan: workspace.Span{Start: start, End: start + len(`'0.1.0'`)},
				Project:     true,
				Workspace:   true,
			}, "the manifest")
		})

		private := []struct {
			name     string
			giveText string
		}{
			{name: "reports a private project for tool.uv.package false", giveText: "[tool.uv]\npackage = false\n"},
			{
				name:     "reports a private project for the classifier that PyPI refuses",
				giveText: "[project]\nclassifiers = [\"Private :: Do Not Upload\"]\n",
			},
		}
		for _, tt := range private {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := workspace.ParseManifest(manifestFile, []byte(tt.giveText))
				assert.NoError(t, err, "ParseManifest")
				assert.True(t, got.Private, "the private project")
			})
		}

		t.Run("reports a dynamic version", func(t *testing.T) {
			t.Parallel()
			got, err := workspace.ParseManifest(
				manifestFile,
				[]byte("[project]\nname = \"a\"\ndynamic = [\"version\"]\n"),
			)
			assert.NoError(t, err, "ParseManifest")
			assert.True(t, got.Dynamic, "the dynamic version")
		})

		t.Run("reports a workspace of an inline table", func(t *testing.T) {
			t.Parallel()
			got, err := workspace.ParseManifest(manifestFile, []byte("tool = { uv = { workspace = {} } }\n"))
			assert.NoError(t, err, "ParseManifest")
			assert.True(t, got.Workspace, "the workspace")
		})

		invalid := []struct {
			name     string
			giveText string
		}{
			{name: "returns ErrManifest for a file that does not parse", giveText: "[project\n"},
			{name: "returns ErrManifest for a name that is no string", giveText: "[project]\nname = 1\n"},
			{
				name:     "returns ErrManifest for classifiers that are no list",
				giveText: "[project]\nclassifiers = \"a\"\n",
			},
			{
				name:     "returns ErrManifest for a list with a value that is no string",
				giveText: "[project]\ndynamic = [1]\n",
			},
			{
				name:     "returns ErrManifest for dependencies that are no list",
				giveText: "[project]\ndependencies = \"a\"\n",
			},
			{
				name:     "returns ErrManifest for a dependency that is no string",
				giveText: "[project]\ndependencies = [{include-group = \"a\"}]\n",
			},
			{
				name:     "returns ErrManifest for a requirement that does not parse",
				giveText: "[project]\ndependencies = [\">=1\"]\n",
			},
			{
				name:     "returns ErrManifest for a package of uv that is no boolean",
				giveText: "[tool.uv]\npackage = \"no\"\n",
			},
			{
				name:     "returns ErrManifest for a value of the wrong type in an inline table",
				giveText: "[project]\noptional-dependencies = { cli = \"probe-a\" }\n",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := workspace.ParseManifest(manifestFile, []byte(tt.giveText))
				assert.ErrorIs(t, err, workspace.ErrManifest, "ParseManifest")
				assert.Contains(t, err.Error(), manifestFile, "the error")
			})
		}
	})
}
