// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package overlay_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/baseline/overlay"
)

// ruff is the rendering of the TOML cases: a comment, a key of the top level, and two tables.
const ruff = "# managed\n\nline-length = 88\n\n[lint]\nselect = [\"ALL\"]\nignore = [\"COM812\"]\n\n" +
	"[lint.per-file-ignores]\n\"**/tests/**\" = [\"S101\"]\n"

func TestTOML(t *testing.T) {
	t.Parallel()

	t.Run("MergeTOML", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			base string
			over string
			want string
		}{
			{
				name: "adds a key that the table of the first document lacks",
				base: ruff,
				over: "[lint.per-file-ignores]\n\"src/**\" = [\"N803\"]\n",
				want: "# managed\n\nline-length = 88\n\n[lint]\nselect = [\"ALL\"]\nignore = [\"COM812\"]\n\n" +
					"[lint.per-file-ignores]\n\"**/tests/**\" = [\"S101\"]\n\"src/**\" = [\"N803\"]\n",
			},
			{
				name: "appends the items of an array",
				base: ruff,
				over: "[lint]\nignore = [\"D203\"]\n",
				want: "# managed\n\nline-length = 88\n\n[lint]\nselect = [\"ALL\"]\nignore = [\"COM812\", \"D203\"]\n\n" +
					"[lint.per-file-ignores]\n\"**/tests/**\" = [\"S101\"]\n",
			},
			{
				name: "replaces a value with the value of the second document",
				base: ruff,
				over: "line-length = 100\n",
				want: "# managed\n\nline-length = 100\n\n[lint]\nselect = [\"ALL\"]\nignore = [\"COM812\"]\n\n" +
					"[lint.per-file-ignores]\n\"**/tests/**\" = [\"S101\"]\n",
			},
			{
				name: "replaces an array with a value of another type",
				base: "a = [1]\n",
				over: "a = \"x\"\n",
				want: "a = \"x\"\n",
			},
			{
				name: "merges the inline tables key by key",
				base: "a = {x = 1, y = [1]}\n",
				over: "a = {y = [2], z = 3}\n",
				want: "a = {x = 1, y = [1, 2], z = 3}\n",
			},
			{
				name: "replaces an inline table with a value of another type",
				base: "a = {x = 1}\n",
				over: "a = 1\n",
				want: "a = 1\n",
			},
			{
				name: "adds a key of the top level that the first document lacks",
				base: "a = 1\n\n[lint]\nselect = [\"ALL\"]\n",
				over: "b = 2\n",
				want: "a = 1\nb = 2\n\n[lint]\nselect = [\"ALL\"]\n",
			},
			{
				name: "merges a key into a table that the first document defines with dotted keys",
				base: "lint.select = [\"ALL\"]\n",
				over: "[lint]\nselect = [\"E\"]\n",
				want: "lint.select = [\"ALL\", \"E\"]\n",
			},
			{
				name: "adds a table that the first document lacks after its tables",
				base: ruff,
				over: "[format]\nquote-style = \"single\"\n",
				want: ruff + "\n[format]\nquote-style = \"single\"\n",
			},
			{
				name: "adds an empty table that the first document lacks",
				base: "a = 1\n",
				over: "[b]\n",
				want: "a = 1\n\n[b]\n",
			},
			{
				name: "adds a table of an array of tables after the tables of that array",
				base: "[[x]]\nk = 1\n\n[y]\nz = 1\n",
				over: "[[x]]\nk = 2\n",
				want: "[[x]]\nk = 1\n\n[[x]]\nk = 2\n\n[y]\nz = 1\n",
			},
			{
				name: "adds a table of an array of tables that the first document lacks after its tables",
				base: "[y]\nz = 1\n",
				over: "[[x]]\nk = 2\n",
				want: "[y]\nz = 1\n\n[[x]]\nk = 2\n",
			},
			{
				name: "adds a table of an array of tables with the tables inside it",
				base: "[[x]]\nk = 1\n\n[x.y]\nz = 1\n\n[w]\nv = 1\n",
				over: "[[x]]\nk = 2\n\n[x.y]\nz = 2\n",
				want: "[[x]]\nk = 1\n\n[x.y]\nz = 1\n\n[[x]]\nk = 2\n\n[x.y]\nz = 2\n\n[w]\nv = 1\n",
			},
			{
				name: "keeps the comments of the second document",
				base: "[lint]\nselect = [\"ALL\"]\n",
				over: "[lint]\n# local\nfixable = [\"I\"]\n",
				want: "[lint]\nselect = [\"ALL\"]\n\n# local\nfixable = [\"I\"]\n",
			},
			{
				name: "keeps the comments of both keys of a merged value",
				base: "# managed\nignore = [\"A\"]\n",
				over: "# local\nignore = [\"B\"]\n",
				want: "# managed\n# local\nignore = [\"A\", \"B\"]\n",
			},
			{name: "returns the first document for an empty second document", base: ruff, over: "", want: ruff},
			{
				name: "returns the second document for an empty first document",
				base: "",
				over: "a = 1\n",
				want: "a = 1\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := overlay.MergeTOML([]byte(tt.base), []byte(tt.over))
				assert.NoError(t, err, "MergeTOML")
				assert.Equal(t, string(got), tt.want, "the merged document")
			})
		}

		refusals := []struct {
			name string
			base string
			over string
			want string
		}{
			{
				name: "returns an error for a first document that does not parse",
				base: "a = [\n",
				over: "a = 1\n",
				want: "parse: ",
			},
			{
				name: "returns an error for a second document that does not parse",
				base: "a = 1\n",
				over: "a = [\n",
				want: "parse: ",
			},
			{
				name: "returns an error for a second document that defines a key twice",
				base: "a = 1\n",
				over: "b = 1\nb = 2\n",
				want: "invalid: ",
			},
			{
				name: "returns an error for a table that the first document defines inline",
				base: "[lint]\nper-file-ignores = {x = [1]}\n",
				over: "[lint.per-file-ignores]\ny = [2]\n",
				want: "conflict: ",
			},
			{
				name: "returns an error for an inline table that the first document defines as a table",
				base: "[lint.per-file-ignores]\nx = [1]\n",
				over: "[lint]\nper-file-ignores = {y = [2]}\n",
				want: "conflict: ",
			},
			{
				name: "returns an error for a key of an array of tables of the first document",
				base: "[[x]]\nk = 1\n",
				over: "x.k = 2\n",
				want: "conflict: ",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := overlay.MergeTOML([]byte(tt.base), []byte(tt.over))
				assert.HasError(t, err, "MergeTOML")
				assert.HasPrefix(t, err.Error(), tt.want, "the error")
			})
		}
	})
}
