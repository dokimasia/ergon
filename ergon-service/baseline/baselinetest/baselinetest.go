// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baselinetest

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
	"go.yaml.in/yaml/v3"
)

// Version is the release of ergon that the lock of [New] records.
const Version = "1.0.0"

// trailing matches a line that ends in a space or a tab.
var trailing = regexp.MustCompile(`(?m)[ \t]$`)

// Answers returns new answers of a repository of a test, with languages: the repository
// dokimasia/demo named demo, of Dokimasia B.V. under MIT since 2026, whose security contact is
// security@example.com.
func Answers(languages ...workspace.Language) *language.Answers {
	return &language.Answers{
		Name:            "demo",
		Owner:           "Dokimasia B.V.",
		License:         spdx.MIT,
		Repository:      "dokimasia/demo",
		SecurityContact: "security@example.com",
		Languages:       languages,
		Year:            2026,
	}
}

// New writes the files of ergon init new for a into a new directory of the test, with the
// toolchains and the languages of catalog and the base producers base, and returns the directory.
// Every option is at its baseline. New stops the test at an error of the directory or of ergon init
// new.
func New(tb testing.TB, catalog *language.Catalog, a *language.Answers, base ...baseline.Producer) string {
	tb.Helper()
	dir := tb.TempDir()
	root, err := os.OpenRoot(dir)
	assert.NoError(tb, err, "OpenRoot of the directory of the repository")
	defer func() { _ = root.Close() }()
	r, err := baseline.Open(root, catalog, Version, base...)
	assert.NoError(tb, err, "Open of the repository")
	_, err = r.New(a, baseline.Options{})
	assert.NoError(tb, err, "New of the repository")
	return dir
}

// Hygiene checks every file below dir, the directory of a repository: each ends in one newline,
// no line ends in a space or a tab, and a file with the extension .yml, .yaml, .json or .toml
// parses as YAML, JSON or TOML. It stops the test at the first file that breaks a rule, and at a
// file that does not read.
func Hygiene(tb assert.TB, dir string) {
	tb.Helper()
	fsys := os.DirFS(dir)
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("baselinetest: read %s: %w", name, err)
		}
		text := string(data)
		assert.HasSuffix(tb, text, "\n", name+" ends in a newline")
		assert.False(tb, strings.HasSuffix(text, "\n\n"), name+" ends in one newline")
		assert.False(tb, trailing.MatchString(text), name+" has no line that ends in a space or a tab")
		var value any
		switch path.Ext(name) {
		case ".yml", ".yaml":
			assert.NoError(tb, yaml.Unmarshal(data, &value), name+" parses as YAML")
		case ".json":
			assert.True(tb, json.Valid(data), name+" parses as JSON")
		case ".toml":
			assert.NoError(tb, toml.Unmarshal(data, &value), name+" parses as TOML")
		}
		return nil
	})
	assert.NoError(tb, err, "the walk of "+dir)
}
