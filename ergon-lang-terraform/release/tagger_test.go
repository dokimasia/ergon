// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/lang/terraform/release"
)

func TestTagger(t *testing.T) {
	t.Parallel()

	t.Run("Tag", func(t *testing.T) {
		t.Parallel()

		t.Run("returns v and the version", func(t *testing.T) {
			t.Parallel()
			v := version.Version{Major: 1, Minor: 2, Pre: "rc.1"}
			assert.Equal(t, release.Tagger{}.Tag(&pkg, v), "v1.2.0-rc.1", "the tag")
		})
	})
}
