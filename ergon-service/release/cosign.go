// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"
	"path/filepath"
)

// actionsIssuer is the issuer of the OIDC tokens of GitHub Actions, and branchRef is the prefix of
// the ref of a branch. Both are parts of the identity of a run in a certificate of Fulcio.
const (
	actionsIssuer = "https://token.actions.githubusercontent.com"
	branchRef     = "refs/heads/"
)

// Cosign verifies cosign bundles with cosign verify-blob through its function Run.
//
// # Concurrency
//
// A Cosign is safe for concurrent use when its Run is.
type Cosign struct {
	// Run runs cosign with args, as ergon tool run go.cosign does, and returns its exit status. It
	// returns an error for a cosign that does not run, such as one that does not install.
	Run func(ctx context.Context, args []string) (int, error)
}

var _ Witness = Cosign{}

// Verify runs cosign verify-blob for the file at path with the bundle at bundle. It requires the
// identity <server>/<repo>/<file>@refs/heads/<branch> of w, the issuer of GitHub Actions, the
// repository of w and the commit of w. It returns an error that wraps [ErrVerify] for an exit status
// other than 0, with the base name of path, and the error of Run.
func (c Cosign) Verify(ctx context.Context, path, bundle string, w *Workflow) error {
	identity := w.Server + "/" + w.Repo + "/" + w.File + "@" + branchRef + w.Branch
	status, err := c.Run(ctx, []string{
		"verify-blob",
		"--bundle", bundle,
		"--certificate-identity", identity,
		"--certificate-oidc-issuer", actionsIssuer,
		"--certificate-github-workflow-repository", w.Repo,
		"--certificate-github-workflow-sha", w.Commit,
		path,
	})
	if err != nil {
		return err
	}
	if status != 0 {
		return fmt.Errorf("%w: cosign verify-blob of %s exited with %d", ErrVerify, filepath.Base(path), status)
	}
	return nil
}
