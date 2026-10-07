// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// pull is the title and the body of a pull request, and the branches of a new one.
type pull struct {
	// Title is the title of the pull request.
	Title string `json:"title"`

	// Body is the description of the pull request in Markdown.
	Body string `json:"body"`

	// Head is the branch of the changes of a new pull request, and empty for an update.
	Head string `json:"head,omitempty"`

	// Base is the branch that a new pull request merges into, and empty for an update.
	Base string `json:"base,omitempty"`
}

// number is the number of a pull request in a response.
type number struct {
	// Number is the number of the pull request.
	Number int `json:"number"`
}

// PullRequest returns the number of the open pull request of repo from the branch head into the
// branch base, and reports whether one is open. It returns an error that wraps [ErrGitHub] for a
// request that fails.
func (c *Client) PullRequest(ctx context.Context, repo, head, base string) (int, bool, error) {
	owner, _, _ := strings.Cut(repo, "/")
	query := url.Values{"state": {"open"}, "head": {owner + ":" + head}, "base": {base}}
	var pulls []number
	if _, err := c.rest(ctx, http.MethodGet, "/repos/"+repo+"/pulls?"+query.Encode(), nil, &pulls); err != nil {
		return 0, false, err
	}
	if len(pulls) == 0 {
		return 0, false, nil
	}
	return pulls[0].Number, true, nil
}

// CreatePullRequest opens a pull request of repo from the branch head into the branch base, with
// title and the Markdown body, and returns its number. It returns an error that wraps [ErrGitHub]
// for a request that fails.
func (c *Client) CreatePullRequest(ctx context.Context, repo, head, base, title, body string) (int, error) {
	var created number
	in := pull{Title: title, Body: body, Head: head, Base: base}
	if _, err := c.rest(ctx, http.MethodPost, "/repos/"+repo+"/pulls", in, &created); err != nil {
		return 0, err
	}
	return created.Number, nil
}

// UpdatePullRequest sets the title and the Markdown body of the pull request number of repo. It
// returns an error that wraps [ErrGitHub] for a request that fails.
func (c *Client) UpdatePullRequest(ctx context.Context, repo string, n int, title, body string) error {
	_, err := c.rest(
		ctx,
		http.MethodPatch,
		"/repos/"+repo+"/pulls/"+strconv.Itoa(n),
		pull{Title: title, Body: body},
		nil,
	)
	return err
}
