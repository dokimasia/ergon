// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/go/release"
)

func TestTagger(t *testing.T) {
	t.Parallel()

	t.Run("Tag", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			dir  string
			want string
		}{
			{name: "returns v and the version for the module at the root", dir: ".", want: "v1.2.0-rc.1"},
			{
				name: "returns the directory before v and the version for a module in a directory",
				dir:  "ergon-lang-go",
				want: "ergon-lang-go/v1.2.0-rc.1",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := release.Tagger{}.Tag(&workspace.Package{Name: pathA, Dir: tt.dir}, parse(t, "1.2.0-rc.1"))
				assert.Equal(t, got, tt.want, "the tag")
			})
		}
	})
}
