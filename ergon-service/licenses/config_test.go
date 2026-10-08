// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package licenses_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/service/licenses"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give func(*licenses.Config)
		}{
			{name: "returns nil for a configuration of MIT", give: func(*licenses.Config) {}},
			{
				name: "returns nil for BUSL-1.1 with every parameter",
				give: func(c *licenses.Config) {
					c.SPDX = spdx.BUSL11
					c.Parameters = parameters()
				},
			},
			{
				name: "returns nil for MIT with empty parameters",
				give: func(c *licenses.Config) { c.Parameters = licenses.Parameters{} },
			},
			{
				name: "returns nil for styles of the library, of ergon and none",
				give: func(c *licenses.Config) {
					c.Styles = map[string]string{
						".sql":       "none",
						"Dockerfile": "Hashtag",
						".groovy":    "DoubleSlashAfterShebang",
					}
				},
			},
			{
				name: "returns nil for doublestar globs",
				give: func(c *licenses.Config) { c.Exclude = []string{"**/*.gen.go", "testdata/**", "{a,b}/*.sql"} },
			},
			{
				name: "returns nil for a directory inside another directory under another license",
				give: func(c *licenses.Config) {
					c.Directories = []licenses.Directory{
						{Path: "enterprise", SPDX: spdx.BUSL11, Parameters: parameters()},
						{Path: "enterprise/sdk", SPDX: spdx.Apache20},
						{Path: ".github", SPDX: spdx.MIT},
					}
				},
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := config()
				tt.give(c)
				assert.NoError(t, c.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give func(*licenses.Config)
		}{
			{name: "returns ErrInvalid for a blank owner", give: func(c *licenses.Config) { c.Owner = " " }},
			{
				name: "returns ErrInvalid for an owner that spans lines",
				give: func(c *licenses.Config) { c.Owner = "A\nB" },
			},
			{
				name: "returns ErrInvalid for a license that ergon does not support",
				give: func(c *licenses.Config) { c.SPDX = "GPL-3.0" },
			},
			{
				name: "returns ErrInvalid for a parameter that spans lines",
				give: func(c *licenses.Config) { c.Parameters.ChangeDate = "2030-01-01\n2031-01-01" },
			},
			{
				name: "returns ErrInvalid for BUSL-1.1 without a parameter",
				give: func(c *licenses.Config) { c.SPDX = spdx.BUSL11 },
			},
			{
				name: "returns ErrInvalid for an empty key of a style",
				give: func(c *licenses.Config) { c.Styles = map[string]string{"": "Hashtag"} },
			},
			{
				name: "returns ErrInvalid for a key of a style with a slash",
				give: func(c *licenses.Config) { c.Styles = map[string]string{"cmd/main.go": "Hashtag"} },
			},
			{
				name: "returns ErrInvalid for a style that the library does not have",
				give: func(c *licenses.Config) { c.Styles = map[string]string{".sql": "DoubleHash"} },
			},
			{name: "returns ErrInvalid for an empty glob", give: func(c *licenses.Config) { c.Exclude = []string{""} }},
			{
				name: "returns ErrInvalid for a glob that spans lines",
				give: func(c *licenses.Config) { c.Exclude = []string{"a\nb"} },
			},
			{
				name: "returns ErrInvalid for a glob that does not parse",
				give: func(c *licenses.Config) { c.Exclude = []string{"[a"} },
			},
			{
				name: "returns ErrInvalid for a directory under a license that ergon does not support",
				give: func(c *licenses.Config) { c.Directories = []licenses.Directory{{Path: "sdk", SPDX: "GPL-3.0"}} },
			},
			{
				name: "returns ErrInvalid for a directory under BUSL-1.1 without a parameter",
				give: func(c *licenses.Config) { c.Directories = []licenses.Directory{{Path: "sdk", SPDX: spdx.BUSL11}} },
			},
			{
				name: "returns ErrInvalid for a directory whose parameter spans lines",
				give: func(c *licenses.Config) {
					c.Directories = []licenses.Directory{
						{Path: "sdk", SPDX: spdx.MIT, Parameters: licenses.Parameters{ChangeDate: "2030\n"}},
					}
				},
			},
			{
				name: "returns ErrInvalid for a directory that two entries name",
				give: func(c *licenses.Config) {
					c.Directories = []licenses.Directory{
						{Path: "sdk", SPDX: spdx.MIT},
						{Path: "sdk", SPDX: spdx.Apache20},
					}
				},
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := config()
				tt.give(c)
				assert.ErrorIs(t, c.Validate(), option.ErrInvalid, "Validate")
			})
		}

		paths := []struct {
			name string
			give string
		}{
			{name: "returns ErrInvalid for a directory without a path", give: ""},
			{name: "returns ErrInvalid for the root as a directory", give: "."},
			{name: "returns ErrInvalid for an absolute path of a directory", give: "/enterprise"},
			{name: "returns ErrInvalid for a directory outside the repository", give: "../enterprise"},
			{name: "returns ErrInvalid for a path of a directory that is not clean", give: "a/./b"},
			{name: "returns ErrInvalid for a path of a directory with a final slash", give: "enterprise/"},
			{name: "returns ErrInvalid for a path of a directory with a backslash", give: `sdk\go`},
			{name: "returns ErrInvalid for a path of a directory that spans lines", give: "sdk\ngo"},
			{name: "returns ErrInvalid for a glob in place of a directory", give: "sdk/*"},
			{name: "returns ErrInvalid for the repository of git as a directory", give: ".git"},
			{name: "returns ErrInvalid for a directory in the directory of ergon", give: ".ergon/local"},
		}
		for _, tt := range paths {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				c := config()
				c.Directories = []licenses.Directory{{Path: tt.give, SPDX: spdx.MIT}}
				err := c.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.Contains(t, err.Error(), "directories", "the error")
			})
		}

		t.Run("names the directory of a parameter that BUSL-1.1 requires", func(t *testing.T) {
			t.Parallel()
			c := config()
			c.Directories = []licenses.Directory{{Path: "enterprise", SPDX: spdx.BUSL11}}
			err := c.Validate()
			assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
			assert.Contains(t, err.Error(), "in the directory enterprise", "the error")
		})

		t.Run("names every empty parameter of BUSL-1.1", func(t *testing.T) {
			t.Parallel()
			c := config()
			c.SPDX = spdx.BUSL11
			c.Parameters = licenses.Parameters{LicensedWork: "demo", AdditionalUseGrant: "None"}
			err := c.Validate()
			assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
			assert.Contains(t, err.Error(), "change-date, change-license", "the error")
		})
	})
}

// config returns a new valid configuration of the cases: the headers of Dokimasia B.V. under MIT,
// and a limit of 10 minutes of the job license.
func config() *licenses.Config {
	c := &licenses.Config{Owner: "Dokimasia B.V.", SPDX: spdx.MIT}
	c.CI.Timeout = 10
	return c
}

// parameters returns the parameters of BUSL-1.1 of the cases.
func parameters() licenses.Parameters {
	return licenses.Parameters{
		LicensedWork:       "demo 1.0",
		AdditionalUseGrant: "None",
		ChangeDate:         "2030-01-01",
		ChangeLicense:      "GPL-2.0-or-later",
	}
}
