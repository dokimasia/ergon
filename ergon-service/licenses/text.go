// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package licenses

import (
	"embed"
	"fmt"
	"strconv"
	"strings"

	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
)

// texts are the texts of the licenses: the body of each license of choosealicense.com at commit
// f717b23, named after its file there, such as gpl-3.0.txt, and the text of BUSL-1.1 of the SPDX
// License List 3.29.0.
//
//go:embed texts
var texts embed.FS

// The variants of the identifiers of the GNU licenses, which name one text.
const (
	only    = "-only"
	orLater = "-or-later"
)

// Holder is what the fields of a license text name.
type Holder struct {
	// Owner is the copyright holder, the [fullname] of a text.
	Owner string

	// Name is the name of the repository, the [project] of a text.
	Name string

	// Repository is the repository on GitHub as owner/name, whose address is the [projecturl] of a
	// text.
	Repository string

	// Year is the year of the copyright notice, the [year] of a text.
	Year int
}

// Text returns the LICENSE of the license of c for h, and the NOTICE, which Apache-2.0 alone has
// and which is nil for any other license.
//
// The text of a license of choosealicense.com is the body of its file, with its fields [year],
// [fullname], [project] and [projecturl] filled from h. A text without a field, such as the Apache
// License 2.0, names no owner. The text of BUSL-1.1 states its title, then the parameters of c with
// the owner as the Licensor, then its terms. The NOTICE of Apache-2.0 states the name of the
// repository and Copyright <year> <owner>.
//
// It returns an error that wraps [option.ErrInvalid] for a license that ergon does not support, and
// for BUSL-1.1 with a parameter that is empty or spans lines. Text reads c and h and modifies
// neither.
func Text(c *Config, h Holder) (text, notice []byte, err error) {
	if !c.SPDX.Valid() {
		return nil, nil, fmt.Errorf("%w: spdx %q, which is not the identifier of a license of ergon", option.ErrInvalid,
			c.SPDX)
	}
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(string(c.SPDX), only), orLater))
	// texts has the text of every identifier of spdx.IDs, which a test of the package checks.
	body, _ := texts.ReadFile("texts/" + name + ".txt")
	year := strconv.Itoa(h.Year)
	if c.SPDX == spdx.BUSL11 {
		if err := c.Parameters.check(c.SPDX); err != nil {
			return nil, nil, err
		}
		title, terms, _ := strings.Cut(string(body), "\n\n")
		parameters := fmt.Sprintf("%s\n\nParameters\n\n"+
			"Licensor:             %s\n"+
			"Licensed Work:        %s\n"+
			"Additional Use Grant: %s\n"+
			"Change Date:          %s\n"+
			"Change License:       %s\n\n",
			title, h.Owner, c.Parameters.LicensedWork, c.Parameters.AdditionalUseGrant, c.Parameters.ChangeDate,
			c.Parameters.ChangeLicense)
		return []byte(parameters + terms), nil, nil
	}
	fields := strings.NewReplacer(
		"[year]", year,
		"[fullname]", h.Owner,
		"[projecturl]", "https://github.com/"+h.Repository,
		"[project]", h.Name,
	)
	text = []byte(fields.Replace(string(body)))
	if c.SPDX == spdx.Apache20 {
		notice = []byte(h.Name + "\nCopyright " + year + " " + h.Owner + "\n")
	}
	return text, notice, nil
}
