// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baselinetest_test

import (
	"io/fs"
	"os"
	"runtime"
	"strings"
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

// configurable is a producer of the cases whose options are the result of a function.
type configurable func() language.Options

// Options returns the options of c.
func (c configurable) Options() language.Options {
	return c()
}

// accepted gives the options of the cases a Validate method that returns nil.
type accepted struct{}

// Validate returns nil.
func (accepted) Validate() error {
	return nil
}

// Step is a group of the options of the cases, which a list and a map also contain.
type Step struct {
	// Name is a string of the group.
	Name string `yaml:"name"`

	// Run is a list of the group.
	Run []string `yaml:"run"`
}

// keyed is options with a value of each kind that Keys sets, a group, a list and a map of groups, a
// struct inline, an unexported field, and a key that the tag name of go-cli states.
type keyed struct {
	accepted `yaml:",inline"`

	Step `yaml:",inline"`

	Text   string            `yaml:"text"`
	Flag   bool              `yaml:"flag"`
	Count  int               `yaml:"count"`
	List   []string          `yaml:"list"`
	Map    map[string]string `yaml:"map"`
	Group  Step              `yaml:"group"`
	Steps  []Step            `yaml:"steps"`
	ByName map[string]Step   `yaml:"by-name"`
	Lint   string            `yaml:"golangci-lint" name:"golangci-lint"`
	hidden string
}

// lower is a string that its decode turns into lower case.
type lower string

// UnmarshalText sets l to b in lower case.
func (l *lower) UnmarshalText(b []byte) error {
	*l = lower(strings.ToLower(string(b)))
	return nil
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

	t.Run("Keys", func(t *testing.T) {
		t.Parallel()

		t.Run("passes options whose every key go-cli decodes", func(t *testing.T) {
			t.Parallel()
			baselinetest.Keys(t, "demo", configurable(func() language.Options { return &keyed{hidden: "kept"} }))
		})

		tests := []struct {
			name     string
			giveOpts func() language.Options
			want     string
		}{
			{
				name: "rejects a key that go-cli reads under another name",
				giveOpts: func() language.Options {
					return &struct {
						accepted `yaml:",inline"`
						CodeQL   string `yaml:"codeql"`
					}{}
				},
				want: "Decode of the section demo",
			},
			{
				name:     "rejects options that are not a pointer to a struct",
				giveOpts: func() language.Options { return accepted{} },
				want:     "the fields of the options of demo",
			},
			{
				name: "rejects a value of a kind without a sample",
				giveOpts: func() language.Options {
					return &struct {
						accepted `yaml:",inline"`
						Ratio    float64 `yaml:"ratio"`
					}{}
				},
				want: "the values of the options of demo",
			},
			{
				name: "rejects a list of a kind without a sample",
				giveOpts: func() language.Options {
					return &struct {
						accepted `yaml:",inline"`
						Ratios   []float64 `yaml:"ratios"`
					}{}
				},
				want: "the values of the options of demo",
			},
			{
				name: "rejects a map whose key has no sample",
				giveOpts: func() language.Options {
					return &struct {
						accepted `yaml:",inline"`
						Names    map[float64]string `yaml:"names"`
					}{}
				},
				want: "the values of the options of demo",
			},
			{
				name: "rejects a map whose value has no sample",
				giveOpts: func() language.Options {
					return &struct {
						accepted `yaml:",inline"`
						Ratios   map[string]float64 `yaml:"ratios"`
					}{}
				},
				want: "the values of the options of demo",
			},
			{
				name: "rejects a value that the decode changes",
				giveOpts: func() language.Options {
					return &struct {
						accepted `yaml:",inline"`
						Text     lower `yaml:"text"`
					}{}
				},
				want: "the options of demo after the decode",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := assert.Rejects(t, "the options fail the check", func(tb assert.TB) {
					baselinetest.Keys(tb, "demo", configurable(tt.giveOpts))
				})
				assert.Length(t, got, 1, "the failures")
				assert.Equal(t, got[0].Contract, tt.want, "the step of the check that fails")
			})
		}
	})
}
