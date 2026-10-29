// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"errors"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/release"
)

// errCosign is the error of a cosign that does not run.
var errCosign = errors.New("cosign does not install")

// releaseRun is the run of the release workflow of the cases of the checks of a draft.
var releaseRun = release.Workflow{
	Server: "https://github.com",
	Repo:   "example/terraform-provider-demo",
	File:   ".github/workflows/release.yml",
	Branch: "main",
	Commit: commitA,
}

func TestCosign(t *testing.T) {
	t.Parallel()

	t.Run("Verify", func(t *testing.T) {
		t.Parallel()

		t.Run("runs cosign verify-blob with the identity, the repository and the commit of the run",
			func(t *testing.T) {
				t.Parallel()
				var got []string
				c := release.Cosign{Run: func(_ context.Context, args []string) (int, error) {
					got = args
					return 0, nil
				}}
				assert.NoError(t, c.Verify(t.Context(), "sums", "sums.sigstore.json", &releaseRun), "Verify")
				assert.Equal(t, got, []string{
					"verify-blob",
					"--bundle", "sums.sigstore.json",
					"--certificate-identity",
					"https://github.com/example/terraform-provider-demo/.github/workflows/release.yml@refs/heads/main",
					"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
					"--certificate-github-workflow-repository", "example/terraform-provider-demo",
					"--certificate-github-workflow-sha", commitA,
					"sums",
				}, "the arguments of cosign")
			})

		t.Run("returns ErrVerify for a bundle that cosign rejects", func(t *testing.T) {
			t.Parallel()
			c := release.Cosign{Run: func(context.Context, []string) (int, error) { return 1, nil }}
			err := c.Verify(t.Context(), "sums", "sums.sigstore.json", &releaseRun)
			assert.ErrorIs(t, err, release.ErrVerify, "Verify")
			assert.Contains(t, err.Error(), "cosign verify-blob of sums exited with 1", "the error")
		})

		t.Run("returns the error of a cosign that does not run", func(t *testing.T) {
			t.Parallel()
			c := release.Cosign{Run: func(context.Context, []string) (int, error) { return 0, errCosign }}
			assert.ErrorIs(t, c.Verify(t.Context(), "sums", "sums.sigstore.json", &releaseRun), errCosign, "Verify")
		})
	})
}
