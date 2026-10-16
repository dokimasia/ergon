// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package licenses

import (
	"fmt"
	"io/fs"
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
)

// none is the value of [Config.Styles] for a file type without a header.
const none = "none"

// The characters that a path of [Directory] does not contain: the line breaks, the separator of
// Windows, and the characters of a pattern of doublestar, because a directory is no glob.
const (
	breaks    = "\r\n"
	backslash = `\`
	pattern   = "*?[{"
)

// unlicensed are the directories in which ergon writes no license file: the repository of git, and
// the lock and the local files of ergon init.
var unlicensed = []string{".git", ".ergon"}

// Parameters are the parameters of BUSL-1.1, which its LICENSE states before its terms. The owner
// of [Config] is the Licensor. Every other license ignores them.
type Parameters struct {
	// LicensedWork is the work that the license covers, such as the name and the version of the
	// software.
	LicensedWork string `yaml:"licensed-work" doc:"The Licensed Work of BUSL-1.1, such as the name and the release of the software."`

	// AdditionalUseGrant is the production use that the Licensor permits, or None.
	AdditionalUseGrant string `yaml:"additional-use-grant" doc:"The Additional Use Grant of BUSL-1.1, the production use that the Licensor permits, or None."`

	// ChangeDate is the date on which the Change License applies, such as 2030-01-01.
	ChangeDate string `yaml:"change-date" doc:"The Change Date of BUSL-1.1, on which the Change License applies, such as 2030-01-01."`

	// ChangeLicense is the license that applies on the Change Date, such as GPL-2.0-or-later.
	ChangeLicense string `yaml:"change-license" doc:"The Change License of BUSL-1.1, the license that applies on the Change Date, such as GPL-2.0-or-later."`
}

// Directory is a directory of the repository under a license of its own. The header of a file of
// the directory states its license, and the producer of the license files of ergon init renders its
// LICENSE, with a NOTICE for Apache-2.0. A file takes the license of the deepest directory of
// [Config.Directories] that contains it.
type Directory struct {
	// Path is the directory, relative to the root of the repository and slash-separated, such as
	// enterprise or sdk/go.
	Path string `yaml:"path"`

	// SPDX is the identifier of the license of the directory, such as BUSL-1.1.
	SPDX spdx.ID `yaml:"spdx"`

	// Parameters are the parameters of BUSL-1.1 for the directory. Every other license ignores
	// them, and the section omits them where every parameter is empty.
	Parameters Parameters `yaml:"parameters,omitempty"`
}

// Config is the section license of .ergon.yaml: the owner and the license of the headers and of the
// license files, the parameters of BUSL-1.1, the directories under a license of their own, the
// comment styles, the files without a header, and the limit of the job license of ci.yml. ergon
// init writes Owner and SPDX from its answers, and a value that differs from the answer fails every
// command that reads the section.
type Config struct {
	// Owner is the copyright holder, such as "Dokimasia B.V.".
	Owner string `yaml:"owner" answer:"owner" doc:"The copyright holder of the headers and of the license files. ergon init writes it from --owner, and ergon init sync --owner changes it."`

	// SPDX is the identifier of the license, such as MIT.
	SPDX spdx.ID `yaml:"spdx" answer:"license" doc:"The SPDX identifier of the license of the headers and of the license files. ergon init writes it from --license, and ergon init sync --license changes it."`

	// Parameters are the parameters of BUSL-1.1.
	Parameters Parameters `yaml:"parameters" doc:"The parameters of BUSL-1.1, which LICENSE states before its terms. Every other license ignores them."`

	// Directories are the directories of the repository under a license of their own.
	Directories []Directory `yaml:"directories" doc:"The directories under a license of their own: the path of each, relative and slash-separated, its SPDX identifier, and its parameters of BUSL-1.1. A file states the license of the deepest directory that contains it, and each directory gets its LICENSE, with a NOTICE for Apache-2.0."`

	// Styles maps an extension, such as .sql, or a base name, such as Dockerfile, to the comment
	// style of its header, or to none for a file without a header.
	Styles map[string]string `yaml:"styles" doc:"The comment style of the header of each extension, such as .sql, or base name, such as Dockerfile: a style of skywalking-eyes, such as DoubleSlash, Hashtag or Semicolon, or none for no header. A key here replaces the style that ergon resolves."`

	// Exclude are the doublestar globs of the paths of the repository whose files have no header.
	Exclude []string `yaml:"exclude" doc:"The doublestar globs of the paths whose files have no header, such as testdata/**."`

	// CI is the limit of the job license of ci.yml.
	CI option.CI[struct{}] `yaml:"ci" doc:"The limit in minutes of the job license of ci.yml, which runs ergon license check."`
}

// Validate returns an error that wraps [option.ErrInvalid] for the first value of c that the
// headers or the license files cannot state:
//
//   - an Owner that is blank or spans lines
//   - an SPDX that is not an identifier of [spdx.IDs]
//   - a parameter that spans lines
//   - a directory that is not valid, as [Directory] states its rules, which the error names
//   - a key of Styles that is empty, spans lines or has a slash, and a style that is neither a
//     comment style of skywalking-eyes or of ergon nor none
//   - a glob of Exclude that is empty, spans lines or is not a pattern of doublestar
//
// It reads the styles in the order of their keys, so it returns the same error for the same
// configuration.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Owner) == "" || strings.ContainsAny(c.Owner, "\r\n") {
		return fmt.Errorf("%w: owner %q, which is not one line of text", option.ErrInvalid, c.Owner)
	}
	if !c.SPDX.Valid() {
		return fmt.Errorf("%w: spdx %q, which is not the identifier of a license of ergon", option.ErrInvalid, c.SPDX)
	}
	if err := c.Parameters.check(); err != nil {
		return err
	}
	for i := range c.Directories {
		if err := c.Directories[i].check(c.Directories[:i]); err != nil {
			return err
		}
	}
	t := styles()
	for _, key := range slices.Sorted(maps.Keys(c.Styles)) {
		if key == "" || strings.ContainsAny(key, "/\r\n") {
			return fmt.Errorf("%w: styles key %q, which is not an extension or a base name", option.ErrInvalid, key)
		}
		if style := c.Styles[key]; style != none && !t.has(style) {
			return fmt.Errorf(
				"%w: styles %s: %q, which is no comment style and not none",
				option.ErrInvalid,
				key,
				style,
			)
		}
	}
	for _, glob := range c.Exclude {
		if glob == "" || strings.ContainsAny(glob, "\r\n") || !doublestar.ValidatePattern(glob) {
			return fmt.Errorf("%w: exclude %q, which is not a doublestar glob on one line", option.ErrInvalid, glob)
		}
	}
	return nil
}

// check returns an error that wraps [option.ErrInvalid] for a directory that a header and a
// LICENSE cannot state, and names its path:
//
//   - a path that is empty, spans lines, starts with / or .., is not clean, is ., has a backslash
//     or a character of a glob, or is in .git or .ergon
//   - a path of one of earlier, the directories before d
//   - an SPDX that is not an identifier of [spdx.IDs]
//   - a parameter that spans lines
func (d *Directory) check(earlier []Directory) error {
	p := d.Path
	inside := fs.ValidPath(p) && p != "." && !strings.ContainsAny(p, breaks+backslash+pattern)
	within := func(dir string) bool { return p == dir || strings.HasPrefix(p, dir+"/") }
	if !inside || slices.ContainsFunc(unlicensed, within) {
		return fmt.Errorf("%w: directories %q, which is not a clean relative path of a directory outside .git and "+
			".ergon", option.ErrInvalid, p)
	}
	if slices.ContainsFunc(earlier, func(e Directory) bool { return e.Path == p }) {
		return fmt.Errorf("%w: directories %q, which two entries name", option.ErrInvalid, p)
	}
	if !d.SPDX.Valid() {
		return fmt.Errorf("%w: directories %q: spdx %q, which is not the identifier of a license of ergon",
			option.ErrInvalid, p, d.SPDX)
	}
	if err := d.Parameters.check(); err != nil {
		return fmt.Errorf("%w, in the directory %s", err, p)
	}
	return nil
}

// All returns an iterator over the parameters of p in the order of LICENSE. It yields the key of
// each parameter under parameters in .ergon.yaml, such as change-date, and its value.
func (p Parameters) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, param := range [...][2]string{
			{"licensed-work", p.LicensedWork},
			{"additional-use-grant", p.AdditionalUseGrant},
			{"change-date", p.ChangeDate},
			{"change-license", p.ChangeLicense},
		} {
			if !yield(param[0], param[1]) {
				return
			}
		}
	}
}

// check returns an error that wraps [option.ErrInvalid] for a parameter of p that spans lines. A
// parameter may be empty, and its line of LICENSE then has its label alone.
func (p Parameters) check() error {
	for key, value := range p.All() {
		if strings.ContainsAny(value, breaks) {
			return fmt.Errorf("%w: parameters %s, which spans lines", option.ErrInvalid, key)
		}
	}
	return nil
}
