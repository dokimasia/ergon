// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"net/http"
	"net/url"
)

// Run returns the newest run of the workflow file of repo, such as ci.yml, for the commit sha and
// the event, such as push: its status, such as in_progress or completed, its conclusion, such as
// success, and the address of its page. The conclusion is empty until the run completes, and every
// result is empty for a commit without such a run. It returns an error that wraps [ErrGitHub] for a
// request that fails, such as for a workflow that repo does not have.
func (c *Client) Run(
	ctx context.Context,
	repo, workflow, sha, event string,
) (status, conclusion, page string, err error) {
	query := url.Values{"head_sha": {sha}, "event": {event}, "per_page": {"1"}}
	var runs struct {
		Runs []struct {
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			Page       string `json:"html_url"`
		} `json:"workflow_runs"`
	}
	path := "/repos/" + repo + "/actions/workflows/" + url.PathEscape(workflow) + "/runs?" + query.Encode()
	if _, err := c.rest(ctx, http.MethodGet, path, nil, &runs); err != nil {
		return "", "", "", err
	}
	if len(runs.Runs) == 0 {
		return "", "", "", nil
	}
	return runs.Runs[0].Status, runs.Runs[0].Conclusion, runs.Runs[0].Page, nil
}
