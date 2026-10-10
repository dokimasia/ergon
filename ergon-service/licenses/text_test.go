// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package licenses_test

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/service/licenses"
)

// fields are the fields of the texts of choosealicense.com, which Text fills.
var fields = []string{"[year]", "[fullname]", "[project]", "[projecturl]"}

// errText is the error of a text that breaks a rule of every text.
var errText = errors.New("text: breaks a rule of every text")

func TestText(t *testing.T) {
	t.Parallel()

	t.Run("Text", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a text with every field filled and one final newline for each license", func(t *testing.T) {
			t.Parallel()
			assert.Total(t, func(id spdx.ID) error {
				c := config()
				c.SPDX = id
				c.Parameters = parameters()
				text, _, err := licenses.Text(c, holder())
				if err != nil {
					return err
				}
				for _, field := range fields {
					if bytes.Contains(text, []byte(field)) {
						return fmt.Errorf("%w: %s states %s", errText, id, field)
					}
				}
				if !bytes.HasSuffix(text, []byte("\n")) || bytes.HasSuffix(text, []byte("\n\n")) {
					return fmt.Errorf("%w: %s does not end in one newline", errText, id)
				}
				return nil
			}, slices.Collect(spdx.IDs()), "Text of each license")
		})

		goldens := []struct {
			name string
			give spdx.ID
		}{
			{name: "returns the text of MIT with the year and the owner", give: spdx.MIT},
			{name: "returns the text of NCSA with the project and its address", give: spdx.NCSA},
			{name: "returns the text of BUSL-1.1 with its parameters before its terms", give: spdx.BUSL11},
		}
		for _, tt := range goldens {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := config()
				c.SPDX = tt.give
				c.Parameters = parameters()
				text, notice, err := licenses.Text(c, holder())
				assert.NoError(t, err, "Text")
				assert.Nil(t, notice, "the notice")
				golden.Match(t, "text/"+string(tt.give), text, golden.ShouldUpdate())
			})
		}

		t.Run("returns one text for both identifiers of a GNU license", func(t *testing.T) {
			t.Parallel()
			only, orLater := config(), config()
			only.SPDX, orLater.SPDX = spdx.GPL30Only, spdx.GPL30OrLater
			want, _, err := licenses.Text(only, holder())
			assert.NoError(t, err, "Text of GPL-3.0-only")
			got, _, err := licenses.Text(orLater, holder())
			assert.NoError(t, err, "Text of GPL-3.0-or-later")
			assert.Equal(t, got, want, "the text of GPL-3.0-or-later")
		})

		t.Run("returns the notice of Apache-2.0 with the name of the repository", func(t *testing.T) {
			t.Parallel()
			c := config()
			c.SPDX = spdx.Apache20
			_, notice, err := licenses.Text(c, holder())
			assert.NoError(t, err, "Text")
			assert.Equal(t, string(notice), "demo\nCopyright 2026 Dokimasia B.V.\n", "the notice")
		})

		invalid := []struct {
			name string
			give func(*licenses.Config)
		}{
			{
				name: "returns ErrInvalid for a license that ergon does not support",
				give: func(c *licenses.Config) { c.SPDX = "AMD-newlib" },
			},
			{
				name: "returns ErrInvalid for BUSL-1.1 with an empty parameter",
				give: func(c *licenses.Config) {
					c.SPDX = spdx.BUSL11
					c.Parameters = parameters()
					c.Parameters.ChangeLicense = ""
				},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := config()
				tt.give(c)
				_, _, err := licenses.Text(c, holder())
				assert.ErrorIs(t, err, option.ErrInvalid, "Text")
			})
		}
	})
}

// holder returns the holder of the cases: Dokimasia B.V. in 2026, of the repository dokimasia/demo
// named demo.
func holder() licenses.Holder {
	return licenses.Holder{Owner: "Dokimasia B.V.", Name: "demo", Repository: "dokimasia/demo", Year: 2026}
}
