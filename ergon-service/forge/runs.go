// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"net/http"
	"net/url"
)

// success is the conclusion of a run whose jobs passed, which the runs of [Client.Passed] have.
const success = "success"

// Passed returns the address of the page of the newest run of the workflow file of repo, such as
// ci.yml, for the commit sha that completed with the conclusion success, and reports whether such a
// run exists. A run of any event counts, such as push, pull_request or merge_group. It returns an
// error that wraps [ErrGitHub] for a request that fails, such as for a workflow that repo does not
// have.
func (c *Client) Passed(ctx context.Context, repo, workflow, sha string) (string, bool, error) {
	query := url.Values{"head_sha": {sha}, "status": {success}, perPage: {"1"}}
	var runs struct {
		Runs []struct {
			Page string `json:"html_url"`
		} `json:"workflow_runs"`
	}
	path := "/repos/" + repo + "/actions/workflows/" + url.PathEscape(workflow) + "/runs?" + query.Encode()
	if _, err := c.rest(ctx, http.MethodGet, path, nil, &runs); err != nil {
		return "", false, err
	}
	if len(runs.Runs) == 0 {
		return "", false, nil
	}
	return runs.Runs[0].Page, true, nil
}
