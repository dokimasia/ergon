// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baselinetest

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/options"
	"go.stealthscale.io/cli"
	"go.yaml.in/yaml/v3"
)

// Version is the release of ergon that the lock of [New] records.
const Version = "1.0.0"

// The values that [Keys] sets.
const (
	// sample is the value of each string and of each key of a map. Its upper-case letters show a
	// decode that changes the case of a key.
	sample = "Sample-Key"

	// sampleInteger is the value of each int.
	sampleInteger = 7
)

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

// Keys checks that go-cli decodes every key that [options.Write] writes into the section name of
// .ergon.yaml for the options of c. It sets every value of new options of c:
//
//   - each string and each key of a map to Sample-Key
//   - each bool to true
//   - each int to 7
//   - each list and each map to one element
//
// It writes the section, decodes it with cli.Decode into the baseline options of c, and compares
// the result with the values that it set. A field whose yaml key differs from the key that go-cli
// reads fails the decode.
//
// Keys stops the test at the first of these:
//
//   - options that [options.Fields] refuses
//   - a value of another kind than a string, a bool, an int, a list, a map or a struct
//   - an error of the write, of the parse or of the decode
//   - a value that the decode changes
func Keys(tb assert.TB, name string, c language.Configurable) {
	tb.Helper()
	want := c.Options()
	_, err := options.Fields(want)
	assert.NoError(tb, err, "the fields of the options of "+name)
	assert.NoError(tb, fill(reflect.ValueOf(want).Elem()), "the values of the options of "+name)
	file, _, err := options.Write(nil, []options.Section{{Name: name, Options: want}}, nil)
	assert.NoError(tb, err, "Write of the section "+name)
	var tree map[string]any
	assert.NoError(tb, yaml.Unmarshal(file, &tree), "the parse of the section "+name)
	got := c.Options()
	assert.NoError(tb, cli.Decode(tree, name, got), "Decode of the section "+name)
	assert.Equal(tb, got, want, "the options of "+name+" after the decode")
}

// fill sets v and every value inside it to the value of its kind in the list of [Keys]. It fills
// each exported field of a struct. It returns an error for a value of another kind, such as a float
// or a pointer.
func fill(v reflect.Value) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(sample)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int:
		v.SetInt(sampleInteger)
	case reflect.Slice:
		list := reflect.MakeSlice(v.Type(), 1, 1)
		if err := fill(list.Index(0)); err != nil {
			return err
		}
		v.Set(list)
	case reflect.Map:
		key, value := reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem()
		if err := fill(key); err != nil {
			return err
		}
		if err := fill(value); err != nil {
			return err
		}
		v.Set(reflect.MakeMapWithSize(v.Type(), 1))
		v.SetMapIndex(key, value)
	case reflect.Struct:
		for i := range v.NumField() {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			if err := fill(v.Field(i)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("baselinetest: no sample of the type %s", v.Type())
	}
	return nil
}
