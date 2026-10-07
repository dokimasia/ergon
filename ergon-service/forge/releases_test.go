// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge_test

import (
	"net/http"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/forge"
)

// releaseRoute is the route of the releases of the repository of the cases.
const releaseRoute = "POST /repos/" + repo + "/releases"

func TestReleases(t *testing.T) {
	t.Parallel()

	t.Run("CreateRelease", func(t *testing.T) {
		t.Parallel()

		t.Run("creates the release of a tag with its body", func(t *testing.T) {
			t.Parallel()
			c, fake := serve(t, map[string][]response{releaseRoute: {{status: http.StatusCreated}}})
			assert.NoError(
				t,
				c.CreateRelease(t.Context(), repo, "v1.0.0-rc.1", "### Major Changes", true),
				"CreateRelease",
			)
			assert.Equal(
				t,
				fake.recorded()[0].body,
				`{"tag_name":"v1.0.0-rc.1","name":"v1.0.0-rc.1","body":"### Major Changes","prerelease":true}`,
				"the release",
			)
		})

		t.Run("returns ErrGitHub for a tag that has a release", func(t *testing.T) {
			t.Parallel()
			exists := response{status: http.StatusUnprocessableEntity, body: `{"message":"Validation Failed"}`}
			c, _ := serve(t, map[string][]response{releaseRoute: {exists}})
			err := c.CreateRelease(t.Context(), repo, "v1.0.0", "", false)
			assert.ErrorIs(t, err, forge.ErrGitHub, "CreateRelease")
		})
	})
}
