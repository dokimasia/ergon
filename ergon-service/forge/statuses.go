// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"net/http"
	"net/url"
)

// maxStatuses is the most statuses of a commit that one request of [Client.Status] reads, the most
// that GitHub returns on one page.
const maxStatuses = "100"

// status is a commit status of the REST API.
type status struct {
	// State is the state of the status: error, failure, pending or success.
	State string `json:"state"`

	// Context tells the status apart from the other statuses of the commit, such as ergon/version.
	Context string `json:"context"`

	// Description is the short description of the status, or empty.
	Description string `json:"description,omitempty"`

	// Target is the address of the page of the status, such as the page of a run, or empty.
	Target string `json:"target_url,omitempty"`
}

// SetStatus sets the status of the commit sha of repo whose context is name, with state, such as
// success, the short description and the address target of its page. A later status of the same
// context replaces it. It returns an error that wraps [ErrGitHub] for a request that fails, such
// as for a token without the permission statuses: write.
func (c *Client) SetStatus(ctx context.Context, repo, sha, state, name, description, target string) error {
	in := status{State: state, Context: name, Description: description, Target: target}
	_, err := c.rest(ctx, http.MethodPost, "/repos/"+repo+"/statuses/"+sha, in, nil)
	return err
}

// Status returns the state of the newest status of the commit sha of repo whose context is name,
// and reports whether the commit has such a status. It reads the combined status of the commit,
// which lists the newest status of each of its first 100 contexts. It returns an error that wraps
// [ErrGitHub] for a request that fails.
func (c *Client) Status(ctx context.Context, repo, sha, name string) (string, bool, error) {
	var combined struct {
		Statuses []status `json:"statuses"`
	}
	path := "/repos/" + repo + "/commits/" + sha + "/status?" + url.Values{"per_page": {maxStatuses}}.Encode()
	if _, err := c.rest(ctx, http.MethodGet, path, nil, &combined); err != nil {
		return "", false, err
	}
	for _, s := range combined.Statuses {
		if s.Context == name {
			return s.State, true, nil
		}
	}
	return "", false, nil
}
