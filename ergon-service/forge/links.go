// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
)

// commitQuery reads the address of a commit, its pull requests with their authors, and its author,
// as @changesets/get-github-info reads them.
const commitQuery = `query($owner: String!, $name: String!, $commit: String!) {
  repository(owner: $owner, name: $name) {
    object(expression: $commit) {
      ... on Commit {
        commitUrl
        associatedPullRequests(first: 50) { nodes { number url mergedAt author { login url } } }
        author { user { login url } }
      }
    }
  }
}`

// pullQuery reads the address and the author of a pull request, and its merge commit.
const pullQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      url
      author { login url }
      mergeCommit { commitUrl abbreviatedOid }
    }
  }
}`

// user is an account of GitHub, as a query reads it.
type user struct {
	// Login is the name of the account.
	Login string `json:"login"`

	// URL is the address of the page of the account.
	URL string `json:"url"`
}

// link returns the Markdown link of u, such as [@login](https://github.com/login), and the empty
// string for a nil u.
func (u *user) link() string {
	if u == nil {
		return ""
	}
	return "[@" + u.Login + "](" + u.URL + ")"
}

// pullRequest is a pull request of a commit, as commitQuery reads it.
type pullRequest struct {
	// Author is the author of the pull request, or nil for a deleted account.
	Author *user `json:"author"`

	// URL is the address of the pull request.
	URL string `json:"url"`

	// MergedAt is the time of the merge in RFC 3339, or empty for a pull request that is not
	// merged.
	MergedAt string `json:"mergedAt"`

	// Number is the number of the pull request.
	Number int `json:"number"`
}

// CommitLinks returns the Markdown links of the commit sha of repo, as the changelog of
// @changesets/changelog-github 1.0.1 writes them: the commit, such as [`a085003`](<address>), the
// pull request that merged it, such as [#1613](<address>), and its author, such as
// [@login](<address>). The pull request is the first that GitHub merged among the pull requests of
// the commit, and the author is the author of that pull request, or of the commit for a commit
// without one. It returns empty links for a commit that repo does not have, and the error of the
// query.
func (c *Client) CommitLinks(ctx context.Context, repo, sha string) (commit, pull, author string, err error) {
	owner, name, _ := strings.Cut(repo, "/")
	var data struct {
		Repository struct {
			Object *struct {
				Author struct {
					User *user `json:"user"`
				} `json:"author"`
				CommitURL              string `json:"commitUrl"`
				AssociatedPullRequests struct {
					Nodes []pullRequest `json:"nodes"`
				} `json:"associatedPullRequests"`
			} `json:"object"`
		} `json:"repository"`
	}
	variables := map[string]any{"owner": owner, "name": name, "commit": sha}
	if err := c.graphql(ctx, commitQuery, variables, &data); err != nil {
		return "", "", "", err
	}
	object := data.Repository.Object
	if object == nil {
		return "", "", "", nil
	}
	commit = "[`" + sha[:min(7, len(sha))] + "`](" + object.CommitURL + ")"
	author = object.Author.User.link()
	pulls := object.AssociatedPullRequests.Nodes
	// A pull request that is not merged sorts after every merged one: when one of two is merged,
	// the longer time comes first, as the time of the other is empty. Two merges sort by time.
	slices.SortStableFunc(pulls, func(a, b pullRequest) int {
		if (a.MergedAt == "") != (b.MergedAt == "") {
			return cmp.Compare(len(b.MergedAt), len(a.MergedAt))
		}
		return strings.Compare(a.MergedAt, b.MergedAt)
	})
	if len(pulls) > 0 {
		pull = fmt.Sprintf("[#%d](%s)", pulls[0].Number, pulls[0].URL)
		author = pulls[0].Author.link()
	}
	return commit, pull, author, nil
}

// PullLinks returns the Markdown links of the pull request number of repo, as the changelog of
// @changesets/changelog-github 1.0.1 writes them for a summary that names it: the pull request, its
// merge commit and its author. The links of a merge commit and of an author that the pull request
// lacks are empty. It returns the error of the query, which wraps [ErrGitHub] for a pull request
// that repo does not have.
func (c *Client) PullLinks(ctx context.Context, repo string, number int) (pull, commit, author string, err error) {
	owner, name, _ := strings.Cut(repo, "/")
	var data struct {
		Repository struct {
			PullRequest struct {
				Author      *user `json:"author"`
				MergeCommit *struct {
					CommitURL      string `json:"commitUrl"`
					AbbreviatedOid string `json:"abbreviatedOid"`
				} `json:"mergeCommit"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}
	variables := map[string]any{"owner": owner, "name": name, "number": number}
	if err := c.graphql(ctx, pullQuery, variables, &data); err != nil {
		return "", "", "", err
	}
	pr := &data.Repository.PullRequest
	pull = fmt.Sprintf("[#%d](%s/%s/pull/%d)", number, c.config.Server, repo, number)
	if pr.MergeCommit != nil {
		oid := pr.MergeCommit.AbbreviatedOid
		commit = "[`" + oid[:min(7, len(oid))] + "`](" + pr.MergeCommit.CommitURL + ")"
	}
	return pull, commit, pr.Author.link(), nil
}
