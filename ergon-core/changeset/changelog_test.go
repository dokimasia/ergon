// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package changeset_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/version"
)

func TestChangelog(t *testing.T) {
	t.Parallel()

	t.Run("Latest", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveData string
			want     version.Version
		}{
			{
				name:     "returns the highest version of the headings",
				giveData: "# a\n\n## 1.2.0\n\n- b\n\n## 1.10.0\n\n- c\n",
				want:     version.Version{Major: 1, Minor: 10},
			},
			{
				name:     "returns the version of a heading with a v before it",
				giveData: "# a\n\n## v0.3.0\n",
				want:     version.Version{Minor: 3},
			},
			{
				name:     "returns a pre-release above the releases below it",
				giveData: "## 1.1.0\n\n## 1.2.0-rc.1\n",
				want:     version.Version{Major: 1, Minor: 2, Pre: "rc.1"},
			},
			{
				name:     "returns the zero version for headings without a version",
				giveData: "# a\n\n## not a version\n\n## 1.2\n",
			},
			{
				name:     "returns the zero version for a version outside a heading",
				giveData: "# a\n\n1.2.0\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, changeset.Latest([]byte(tt.giveData)), tt.want, "the version")
			})
		}
	})
}
