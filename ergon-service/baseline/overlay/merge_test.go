// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package overlay_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/baseline/overlay"
)

// workflow is the rendering of the cases: a comment, a scalar, a map and a list.
const workflow = "# managed\nname: ci\njobs:\n  common:\n    steps:\n      - run: make\n"

func TestMerge(t *testing.T) {
	t.Parallel()

	t.Run("Merge", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			base string
			over string
			want string
		}{
			{
				name: "merges the maps key by key",
				base: workflow,
				over: "jobs:\n  local:\n    steps: []\n",
				want: "# managed\nname: ci\njobs:\n  common:\n    steps:\n      - run: make\n  local:\n    steps: []\n",
			},
			{
				name: "appends the items of a list",
				base: workflow,
				over: "jobs:\n  common:\n    steps:\n      - run: lint\n",
				want: "# managed\nname: ci\njobs:\n  common:\n    steps:\n      - run: make\n      - run: lint\n",
			},
			{
				name: "replaces a scalar with the value of the second document",
				base: workflow,
				over: "name: custom\n",
				want: "# managed\nname: custom\njobs:\n  common:\n    steps:\n      - run: make\n",
			},
			{
				name: "replaces a map with a list of the second document",
				base: "a:\n  b: 1\n",
				over: "a: [1]\n",
				want: "a: [1]\n",
			},
			{
				name: "replaces a scalar with a map of the second document",
				base: "a: 1\n",
				over: "a:\n  b: 2\n",
				want: "a:\n  b: 2\n",
			},
			{
				name: "replaces a list with a scalar of the second document",
				base: "a: [1]\n",
				over: "a: x\n",
				want: "a: x\n",
			},
			{
				name: "keeps the comments of the second document",
				base: "a: 1\n",
				over: "# b is local\nb: 2\n",
				want: "a: 1\n# b is local\nb: 2\n",
			},
			{name: "returns the first document for an empty second document", base: workflow, over: "", want: workflow},
			{name: "returns the second document for an empty first document", base: "", over: "a: 1\n", want: "a: 1\n"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := overlay.Merge([]byte(tt.base), []byte(tt.over))
				assert.NoError(t, err, "Merge")
				assert.Equal(t, string(got), tt.want, "the merged document")
			})
		}

		t.Run("returns an error for a first document that does not parse", func(t *testing.T) {
			t.Parallel()
			_, err := overlay.Merge([]byte("a: [\n"), []byte("a: 1\n"))
			assert.HasError(t, err, "Merge")
			assert.HasPrefix(t, err.Error(), "parse: ", "the error")
		})

		t.Run("returns an error for a second document that does not parse", func(t *testing.T) {
			t.Parallel()
			_, err := overlay.Merge([]byte("a: 1\n"), []byte("a: [\n"))
			assert.HasError(t, err, "Merge")
			assert.HasPrefix(t, err.Error(), "parse: ", "the error")
		})
	})
}
