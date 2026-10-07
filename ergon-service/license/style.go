// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package license

import (
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/apache/skywalking-eyes/assets"
	"github.com/apache/skywalking-eyes/pkg/comments"
	"go.yaml.in/yaml/v3"
)

// The comment styles that ergon adds to those of skywalking-eyes. Each is a style of the library
// with the preamble of another.
const (
	// slashesAfterShebang is DoubleSlash after the shebang line of a script, which Hashtag keeps.
	slashesAfterShebang = "DoubleSlashAfterShebang"

	// slashesAfterPHP is DoubleSlash after the <?php line, which PhpTag keeps.
	slashesAfterPHP = "DoubleSlashAfterPhpTag"

	// hashtagAfterDirectives is Hashtag after the parser directives of a Dockerfile, such as
	// # syntax=docker/dockerfile:1.
	hashtagAfterDirectives = "HashtagAfterDirectives"

	// angleAfterFrontMatter is AngleBracket after the YAML front matter that opens a Markdown file,
	// which a reader of the front matter requires on the first line.
	angleAfterFrontMatter = "AngleBracketAfterFrontMatter"
)

// directives matches the parser directives that open a Dockerfile, # syntax=, # escape= and
// # check=, up to the end of the last one.
const directives = `(?mi)\A(?:#[ \t]*(?:syntax|escape|check)[ \t]*=.*\n)*#[ \t]*(?:syntax|escape|check)[ \t]*=.*$`

// frontMatter matches the YAML front matter that opens a Markdown file, from its line --- to the
// next line ---, with the blank lines after it, so the header follows a blank line.
const frontMatter = `(?ms)\A---[ \t]*\n.*?^---[ \t]*$(?:\n[ \t]*$)*`

// overrides are ergon's comment styles of the file types that the table of skywalking-eyes v0.9.0
// maps to another style or to none, by extension or base name, with none for a file type without
// a header:
//
//   - The files of TypeScript, JavaScript, Java, Kotlin, Scala and C# take // after a shebang,
//     as Go, Rust and protobuf do in the library, and PHP takes // after its <?php line.
//   - go.mod and go.work accept // comments. The library writes HTML comments into go.mod.
//   - The queries of tree-sitter take ; comments.
//   - A Dockerfile keeps its parser directives above the header, and a Markdown file its YAML
//     front matter.
//   - A tool writes the lock of Terraform and the wrapper of Gradle, MDX 2 rejects HTML comments,
//     a Go template renders its header itself, and the other files have no comment syntax.
var overrides = map[string]string{
	".ts":                       slashesAfterShebang,
	".tsx":                      slashesAfterShebang,
	".mts":                      slashesAfterShebang,
	".cts":                      slashesAfterShebang,
	".js":                       slashesAfterShebang,
	".jsx":                      slashesAfterShebang,
	".mjs":                      slashesAfterShebang,
	".cjs":                      slashesAfterShebang,
	".java":                     slashesAfterShebang,
	".kt":                       slashesAfterShebang,
	".kts":                      slashesAfterShebang,
	".scala":                    slashesAfterShebang,
	".cs":                       slashesAfterShebang,
	".csx":                      slashesAfterShebang,
	".php":                      slashesAfterPHP,
	".mod":                      "DoubleSlash",
	"go.work":                   "DoubleSlash",
	".scm":                      "Semicolon",
	"Dockerfile":                hashtagAfterDirectives,
	".dockerfile":               hashtagAfterDirectives,
	".md":                       angleAfterFrontMatter,
	".markdown":                 angleAfterFrontMatter,
	".terraform.lock.hcl":       none,
	".mdx":                      none,
	".tmpl":                     none,
	"gradlew":                   none,
	"gradlew.bat":               none,
	"gradle-wrapper.properties": none,
	"go.sum":                    none,
	"go.work.sum":               none,
	".json":                     none,
	".lock":                     none,
	"NOTICE":                    none,
}

// licenseFiles are the starts of the base names of the license files, such as LICENSE.md, which
// have no header.
var licenseFiles = []string{"LICENSE", "COPYING"}

// table is the comment style of each file type: the styles of skywalking-eyes, with ergon's own,
// and the style of each extension and base name of the library's languages.
type table struct {
	// styles are the comment styles, by identifier.
	styles map[string]comments.CommentStyle

	// after are the preambles of the styles, compiled, by the identifier of their style.
	after map[string]*regexp.Regexp

	// types are the identifiers of the styles of the library's file types, by extension or base
	// name. A file type that two styles claim has the empty identifier, which no style has.
	types map[string]string
}

// styles returns the table of the process, which it reads from the assets of skywalking-eyes on its
// first call. The table is safe for concurrent use, because no caller writes it.
var styles = sync.OnceValue(func() *table {
	// The package comments of skywalking-eyes decodes the same two assets when the process starts,
	// and panics on an error, so these reads and decodes return none.
	languages, _ := assets.Asset("languages.yaml")
	list, _ := assets.Asset("styles.yaml")
	var langs map[string]comments.Language
	_ = yaml.Unmarshal(languages, &langs)
	var library []comments.CommentStyle
	_ = yaml.Unmarshal(list, &library)
	t := &table{
		styles: map[string]comments.CommentStyle{},
		after:  map[string]*regexp.Regexp{},
		types:  map[string]string{},
	}
	for _, s := range library {
		t.styles[s.ID] = s
	}
	for _, l := range langs {
		if l.CommentStyleID == "" {
			continue
		}
		for _, key := range slices.Concat(l.Extensions, l.Filenames) {
			if id, ok := t.types[key]; ok && id != l.CommentStyleID {
				t.types[key] = ""
				continue
			}
			t.types[key] = l.CommentStyleID
		}
	}
	slashes, hashtag, php, angle := t.styles["DoubleSlash"], t.styles["Hashtag"], t.styles["PhpTag"],
		t.styles["AngleBracket"]
	t.styles[slashesAfterShebang] = comments.CommentStyle{
		ID:     slashesAfterShebang,
		After:  hashtag.After,
		Start:  slashes.Start,
		Middle: slashes.Middle,
		End:    slashes.End,
	}
	t.styles[slashesAfterPHP] = comments.CommentStyle{
		ID:           slashesAfterPHP,
		After:        php.After,
		Start:        slashes.Start,
		Middle:       slashes.Middle,
		End:          slashes.End,
		EnsureAfter:  php.EnsureAfter,
		EnsureBefore: php.EnsureBefore,
	}
	t.styles[hashtagAfterDirectives] = comments.CommentStyle{
		ID:     hashtagAfterDirectives,
		After:  directives,
		Start:  hashtag.Start,
		Middle: hashtag.Middle,
		End:    hashtag.End,
	}
	t.styles[angleAfterFrontMatter] = comments.CommentStyle{
		ID:     angleAfterFrontMatter,
		After:  frontMatter,
		Start:  angle.Start,
		Middle: angle.Middle,
		End:    angle.End,
	}
	for id, s := range t.styles {
		if s.After != "" {
			t.after[id] = regexp.MustCompile(s.After)
		}
	}
	return t
})

// has reports whether id is the identifier of a comment style of the table.
func (t *table) has(id string) bool {
	_, ok := t.styles[id]
	return ok
}

// resolve returns the comment style of the header of the file at name, a slash-separated path, and
// reports whether the file has a header. The first of these that matches the base name of name,
// exactly or by its longest extension, states the style:
//
//  1. own, the styles of the configuration
//  2. none, for a license file, such as LICENSE.md
//  3. overrides, ergon's styles
//  4. the table of skywalking-eyes, which states none for a file type that two styles claim
//
// It reports false for a file of none, and for a file that nothing matches.
func (t *table) resolve(name string, own map[string]string) (comments.CommentStyle, bool) {
	base := path.Base(name)
	id, ok := lookup(own, base)
	if !ok && slices.ContainsFunc(licenseFiles, func(start string) bool { return strings.HasPrefix(base, start) }) {
		id, ok = none, true
	}
	if !ok {
		id, ok = lookup(overrides, base)
	}
	if !ok {
		id, _ = lookup(t.types, base)
	}
	style, found := t.styles[id]
	return style, found
}

// lookup returns the value of base in types: the value of the key that equals base, or else of the
// longest extension of base that is a key, and reports whether a key matches. An extension of base
// is the suffix that starts at one of its dots, so .lock.hcl and .hcl are the extensions of
// .terraform.lock.hcl.
func lookup(types map[string]string, base string) (string, bool) {
	if id, ok := types[base]; ok {
		return id, true
	}
	for i := range len(base) {
		if base[i] != '.' {
			continue
		}
		if id, ok := types[base[i:]]; ok {
			return id, true
		}
	}
	return "", false
}
