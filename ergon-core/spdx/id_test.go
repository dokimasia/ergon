// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package spdx_test

import (
	"fmt"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/spdx"
)

// spellings pins the spelling of every identifier, in the order of the SPDX License List 3.29.0,
// which sorts them without regard to case.
var spellings = []spdx.ID{
	"0BSD", "AFL-3.0", "AGPL-3.0-only", "AGPL-3.0-or-later", "Apache-2.0", "Artistic-2.0",
	"BlueOak-1.0.0", "BSD-2-Clause", "BSD-2-Clause-Patent", "BSD-3-Clause", "BSD-3-Clause-Clear",
	"BSD-4-Clause", "BSL-1.0", "BUSL-1.1", "CC0-1.0", "CECILL-2.1", "ECL-2.0", "EPL-1.0", "EPL-2.0",
	"EUPL-1.1", "EUPL-1.2", "GPL-2.0-only", "GPL-2.0-or-later", "GPL-3.0-only", "GPL-3.0-or-later",
	"ISC", "LGPL-2.1-only", "LGPL-2.1-or-later", "LGPL-3.0-only", "LGPL-3.0-or-later", "MIT", "MIT-0",
	"MPL-2.0", "MS-PL", "MS-RL", "MulanPSL-2.0", "NCSA", "OSL-3.0", "PostgreSQL", "Unlicense",
	"UPL-1.0", "Vim", "WTFPL", "Zlib",
}

func TestID(t *testing.T) {
	t.Parallel()

	t.Run("IDs", func(t *testing.T) {
		t.Parallel()

		t.Run("yields the 44 identifiers in the order of the SPDX License List", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, slices.Collect(spdx.IDs()), spellings, "the identifiers")
		})

		t.Run("stops when the loop breaks", func(t *testing.T) {
			t.Parallel()
			var first []spdx.ID
			for id := range spdx.IDs() {
				first = append(first, id)
				break
			}
			assert.Equal(t, first, []spdx.ID{spdx.ZeroBSD}, "the identifiers before the break")
		})
	})

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for every identifier of IDs", func(t *testing.T) {
			t.Parallel()
			assert.Total(t, func(id spdx.ID) error {
				if !id.Valid() {
					return fmt.Errorf("spdx: Valid of %s reports false", id)
				}
				return nil
			}, slices.Collect(spdx.IDs()), "Valid of each identifier")
		})

		tests := []struct {
			name string
			give spdx.ID
		}{
			{name: "reports false for the zero value", give: ""},
			{name: "reports false for an identifier in another case", give: "mit"},
			{name: "reports false for an identifier that SPDX deprecates", give: "GPL-3.0"},
			{name: "reports false for an identifier with a trailing space", give: "MIT "},
			{name: "reports false for a license outside the list", give: "AMD-newlib"},
			{name: "reports false for an expression of two licenses", give: "MIT OR Apache-2.0"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.False(t, tt.give.Valid(), "Valid of "+string(tt.give))
			})
		}
	})
}
