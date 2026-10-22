// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/lang/java/baseline"
)

// notPath ends the error of a rule set that is not a relative path of the characters of a rule set.
const notPath = ", which is not a relative path of segments of letters, digits, . _ and -"

func TestRulesets(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		valid := []struct {
			name string
			give baseline.Rulesets
		}{
			{name: "returns nil for the base ruleset of PMD", give: baseline.Rulesets{"rulesets/java/quickstart.xml"}},
			{
				name: "returns nil for a rule of a category of PMD",
				give: baseline.Rulesets{"category/java/bestpractices.xml/UnusedPrivateField"},
			},
			{
				name: "returns nil for a file of the repository beside a rule set of PMD",
				give: baseline.Rulesets{"config/pmd_rules-1.xml", "category/java/errorprone.xml"},
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, tt.give.Validate(), "Validate")
			})
		}

		invalid := []struct {
			name string
			give baseline.Rulesets
			want string
		}{
			{name: "returns ErrInvalid for an empty list", give: baseline.Rulesets{}, want: ": rulesets is empty"},
			{name: "returns ErrInvalid for no list", give: nil, want: ": rulesets is empty"},
			{
				name: "returns ErrInvalid for an empty rule set",
				give: baseline.Rulesets{""},
				want: `: rule set ""` + notPath,
			},
			{
				name: "returns ErrInvalid for an absolute path",
				give: baseline.Rulesets{"/etc/pmd.xml"},
				want: `: rule set "/etc/pmd.xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for a path that ends with a slash",
				give: baseline.Rulesets{"config/"},
				want: `: rule set "config/"` + notPath,
			},
			{
				name: "returns ErrInvalid for an empty segment",
				give: baseline.Rulesets{"config//pmd.xml"},
				want: `: rule set "config//pmd.xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for the segment of the current directory",
				give: baseline.Rulesets{"./pmd.xml"},
				want: `: rule set "./pmd.xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for the segment of the parent directory",
				give: baseline.Rulesets{"config/../pmd.xml"},
				want: `: rule set "config/../pmd.xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for a backslash",
				give: baseline.Rulesets{`config\pmd.xml`},
				want: `: rule set "config\\pmd.xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for a quote",
				give: baseline.Rulesets{`pmd".xml`},
				want: `: rule set "pmd\".xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for a dollar sign",
				give: baseline.Rulesets{"$pmd.xml"},
				want: `: rule set "$pmd.xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for a comma",
				give: baseline.Rulesets{"a.xml,b.xml"},
				want: `: rule set "a.xml,b.xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for a space",
				give: baseline.Rulesets{"pmd rules.xml"},
				want: `: rule set "pmd rules.xml"` + notPath,
			},
			{
				name: "returns ErrInvalid for a rule set that appears twice",
				give: baseline.Rulesets{"a.xml", "b.xml", "a.xml"},
				want: `: rule set "a.xml", which appears twice`,
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.give.Validate()
				assert.ErrorIs(t, err, option.ErrInvalid, "Validate")
				assert.HasSuffix(t, err.Error(), tt.want, "the error")
			})
		}
	})
}
