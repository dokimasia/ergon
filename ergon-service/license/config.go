// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package license

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
)

// none is the value of [Config.Styles] for a file type without a header.
const none = "none"

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

// Config is the section license of .ergon.yaml: the owner and the license of the headers and of the
// license files, the parameters of BUSL-1.1, the comment styles, the files without a header, and
// the limit of the job license of ci.yml. ergon init writes Owner and SPDX from its answers, and a
// value that differs from the answer fails every command that reads the section.
type Config struct {
	// Owner is the copyright holder, such as "Dokimasia B.V.".
	Owner string `yaml:"owner" answer:"owner" doc:"The copyright holder of the headers and of the license files. ergon init writes it from --owner, and ergon init sync --owner changes it."`

	// SPDX is the identifier of the license, such as MIT.
	SPDX spdx.ID `yaml:"spdx" answer:"license" doc:"The SPDX identifier of the license of the headers and of the license files. ergon init writes it from --license, and ergon init sync --license changes it."`

	// Parameters are the parameters of BUSL-1.1.
	Parameters Parameters `yaml:"parameters" doc:"The parameters of BUSL-1.1, which LICENSE states before its terms. Every other license ignores them."`

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
//   - a parameter that spans lines, and for BUSL-1.1 a parameter that is empty, which the error
//     names with every other empty one
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
	if err := c.Parameters.check(c.SPDX); err != nil {
		return err
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

// check returns an error that wraps [option.ErrInvalid] for a parameter of p that spans lines, and
// for the license BUSL-1.1 when a parameter is empty. The error names every empty parameter.
func (p Parameters) check(id spdx.ID) error {
	params := []struct{ key, value string }{
		{key: "licensed-work", value: p.LicensedWork},
		{key: "additional-use-grant", value: p.AdditionalUseGrant},
		{key: "change-date", value: p.ChangeDate},
		{key: "change-license", value: p.ChangeLicense},
	}
	var empty []string
	for _, param := range params {
		if strings.ContainsAny(param.value, "\r\n") {
			return fmt.Errorf("%w: parameters %s, which spans lines", option.ErrInvalid, param.key)
		}
		if strings.TrimSpace(param.value) == "" {
			empty = append(empty, param.key)
		}
	}
	if id == spdx.BUSL11 && len(empty) > 0 {
		return fmt.Errorf("%w: parameters %s, which BUSL-1.1 requires and which are empty", option.ErrInvalid,
			strings.Join(empty, ", "))
	}
	return nil
}
