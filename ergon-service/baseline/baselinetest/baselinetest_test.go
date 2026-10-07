// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baselinetest_test

import (
	"io/fs"
	"os"
	"runtime"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
)

// templates is a producer of the cases whose templates are a map.
type templates fstest.MapFS

// Templates returns the templates.
func (t templates) Templates() fs.FS {
	return fstest.MapFS(t)
}

func TestBaselinetest(t *testing.T) {
	t.Parallel()

	t.Run("Answers", func(t *testing.T) {
		t.Parallel()

		t.Run("returns valid answers with the languages", func(t *testing.T) {
			t.Parallel()
			a := baselinetest.Answers("go", "python")
			assert.NoError(t, a.Validate(), "Validate of the answers")
			assert.Equal(t, a.Languages, []workspace.Language{"go", "python"}, "the languages")
		})
	})

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the files of the producers into a directory of the test", func(t *testing.T) {
			t.Parallel()
			base := baseline.Producer{Name: "demo", Producer: templates{
				"managed/notes.txt.tmpl": {Data: []byte("{{% .Answers.Owner %}}\n")},
			}}
			dir := baselinetest.New(t, new(language.Catalog), baselinetest.Answers(), base)
			files.HasContent(t, dir+"/notes.txt", "Dokimasia B.V.\n", "the rendered notes.txt")
			_, err := os.Stat(dir + "/.ergon/init.lock")
			assert.NoError(t, err, "Stat of the lock")
		})
	})

	t.Run("Hygiene", func(t *testing.T) {
		t.Parallel()

		t.Run("passes files that end in one newline, without trailing blanks, that parse", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{
				"a.yml":       files.Text("a: 1\n"),
				"b.json":      files.Text("{\"b\": 1}\n"),
				"c.toml":      files.Text("c = 1\n"),
				"docs/d.md":   files.Text("# d\n\ntext\n"),
				"Makefile":    files.Text("all:\n\ttrue\n"),
				"empty/.keep": files.Text("\n"),
			})
			baselinetest.Hygiene(t, dir)
		})

		tests := []struct {
			name string
			tree files.Tree
			want string
		}{
			{
				name: "rejects a file without a final newline",
				tree: files.Tree{"a.txt": files.Text("a")},
				want: "a.txt ends in a newline",
			},
			{
				name: "rejects a file with two final newlines",
				tree: files.Tree{"a.txt": files.Text("a\n\n")},
				want: "a.txt ends in one newline",
			},
			{
				name: "rejects a line that ends in a space",
				tree: files.Tree{"a.txt": files.Text("a \nb\n")},
				want: "a.txt has no line that ends in a space or a tab",
			},
			{
				name: "rejects a line that ends in a tab",
				tree: files.Tree{"a.txt": files.Text("a\nb\t\n")},
				want: "a.txt has no line that ends in a space or a tab",
			},
			{
				name: "rejects YAML that does not parse",
				tree: files.Tree{"a.yaml": files.Text("a: [\n")},
				want: "a.yaml parses as YAML",
			},
			{
				name: "rejects JSON that does not parse",
				tree: files.Tree{"a.json": files.Text("{\n")},
				want: "a.json parses as JSON",
			},
			{
				name: "rejects TOML that does not parse",
				tree: files.Tree{"a.toml": files.Text("a = [\n")},
				want: "a.toml parses as TOML",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				dir := files.Workspace(t, tt.tree)
				got := assert.Rejects(t, "the file breaks a rule", func(tb assert.TB) { baselinetest.Hygiene(tb, dir) })
				assert.Length(t, got, 1, "the failures")
				assert.Equal(t, got[0].Contract, tt.want, "the rule that the file breaks")
			})
		}

		t.Run("rejects a directory that does not exist", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir() + "/missing"
			got := assert.Rejects(t, "the walk fails", func(tb assert.TB) { baselinetest.Hygiene(tb, dir) })
			assert.Length(t, got, 1, "the failures")
			assert.Equal(t, got[0].Contract, "the walk of "+dir, "the failure")
		})

		t.Run("rejects a file that does not read", func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == "windows" {
				t.Skip("Windows has no mode that denies the owner reading a file")
			}
			dir := files.Workspace(t, files.Tree{"a.txt": files.Text("a\n").WithMode(0o000)})
			got := assert.Rejects(t, "the read fails", func(tb assert.TB) { baselinetest.Hygiene(tb, dir) })
			assert.Length(t, got, 1, "the failures")
			assert.Equal(t, got[0].Contract, "the walk of "+dir, "the failure")
		})
	})
}
