// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/baseline"
)

// localDir is the directory of the local files of a repository.
const localDir = ".ergon/local/"

func TestMerge(t *testing.T) {
	t.Parallel()

	t.Run("Repository", func(t *testing.T) {
		t.Parallel()

		t.Run("New", func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name  string
				local string
				want  string
			}{
				{
					name:  "merges the maps of a local YAML file key by key",
					local: "jobs:\n  local:\n    steps: []\n",
					want: "# managed\nname: ci\njobs:\n  common:\n    steps:\n      - run: make\n" +
						"  local:\n    steps: []\n",
				},
				{
					name:  "appends the lists of a local YAML file",
					local: "jobs:\n  common:\n    steps:\n      - run: lint\n",
					want:  "# managed\nname: ci\njobs:\n  common:\n    steps:\n      - run: make\n      - run: lint\n",
				},
				{
					name:  "replaces a scalar with the value of a local YAML file",
					local: "name: custom\n",
					want:  "# managed\nname: custom\njobs:\n  common:\n    steps:\n      - run: make\n",
				},
				{
					name:  "leaves the rendering for an empty local YAML file",
					local: "",
					want:  workflowContent,
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					root := directory(t)
					put(t, root, localDir+workflow, tt.local)
					_, err := repository(t, root).New(answers(), baseline.Options{})
					assert.NoError(t, err, "New")
					assert.Equal(t, content(t, root, workflow), tt.want, "the merged "+workflow)
				})
			}

			t.Run("returns the local YAML file for an empty rendering", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, localDir+"empty.yml", "a: 1\n")
				r, err := baseline.Open(root, catalog(t), version, common, renders(
					language.File{Path: "empty.yml", Class: language.Managed, Content: []byte{}},
				))
				assert.NoError(t, err, "Open")
				_, err = r.New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Equal(t, content(t, root, "empty.yml"), "a: 1\n", "the merged empty.yml")
			})

			t.Run("returns an error for a local YAML file that does not parse", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, localDir+workflow, "jobs: [\n")
				_, err := repository(t, root).New(answers(), baseline.Options{})
				assert.HasError(t, err, "New")
				assert.Contains(t, err.Error(), "baseline: merge "+localDir+workflow, "the error")
			})

			t.Run("returns an error for a rendering that does not parse", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, localDir+"bad.yml", "a: 1\n")
				r, err := baseline.Open(root, catalog(t), version, common, renders(
					language.File{Path: "bad.yml", Class: language.Managed, Content: []byte("a: [\n")},
				))
				assert.NoError(t, err, "Open")
				_, err = r.New(answers(), baseline.Options{})
				assert.HasError(t, err, "New")
			})

			t.Run("writes the keys of the rendering into an existing configured file", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, config, "owner: us\nname: old\n")
				_, err := repository(t, root).New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Equal(t, content(t, root, config), "owner: us\nname: demo\n", "the configured "+config)
			})

			t.Run("replaces a list of an existing configured file", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, "tools.yaml", "tools:\n  - b\n")
				r, err := baseline.Open(root, catalog(t), version, common, renders(
					language.File{Path: "tools.yaml", Class: language.Configured, Content: []byte("tools:\n  - a\n")},
				))
				assert.NoError(t, err, "Open")
				_, err = r.New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Equal(t, content(t, root, "tools.yaml"), "tools:\n  - a\n", "the configured tools.yaml")
			})

			t.Run("leaves a configured file whose keys have the values of the rendering", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, config, "# our settings\nname:   demo\nother: 1\n")
				changes, err := repository(t, root).New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.NotContains(t, paths(changes), config, "the files that New wrote")
				assert.Equal(t, content(t, root, config), "# our settings\nname:   demo\nother: 1\n", "the "+config)
			})

			t.Run("writes the rendering into an empty configured file", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, config, "")
				_, err := repository(t, root).New(answers(), baseline.Options{})
				assert.NoError(t, err, "New")
				assert.Equal(t, content(t, root, config), "name: demo\n", "the configured "+config)
			})

			t.Run("returns an error for a configured file that does not parse", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, config, "name: [\n")
				_, err := repository(t, root).New(answers(), baseline.Options{})
				assert.HasError(t, err, "New")
				assert.Contains(t, err.Error(), "baseline: configure "+config, "the error")
			})

			t.Run("returns an error for a configured rendering that does not parse", func(t *testing.T) {
				t.Parallel()
				root := directory(t)
				put(t, root, "tools.yaml", "tools: []\n")
				r, err := baseline.Open(root, catalog(t), version, common, renders(
					language.File{Path: "tools.yaml", Class: language.Configured, Content: []byte("tools: [\n")},
				))
				assert.NoError(t, err, "Open")
				_, err = r.New(answers(), baseline.Options{})
				assert.HasError(t, err, "New")
				assert.Contains(t, err.Error(), "rendering", "the error")
			})
		})
	})
}
