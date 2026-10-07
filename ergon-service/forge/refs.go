// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"net/http"
)

// The prefixes of the refs of branches and tags.
const (
	branchRefs = "refs/heads/"
	tagRefs    = "refs/tags/"
)

// newRef is the body of a request that creates a ref.
type newRef struct {
	// Ref is the full name of the ref, such as refs/tags/v1.0.0.
	Ref string `json:"ref"`

	// SHA is the commit of the ref.
	SHA string `json:"sha"`
}

// object is the object of a ref or of an annotated tag: its type and its hash.
type object struct {
	// Type is commit for a lightweight tag and a branch, and tag for an annotated tag.
	Type string `json:"type"`

	// SHA is the hash of the object.
	SHA string `json:"sha"`
}

// Branch returns the commit of the branch name of repo, and reports whether repo has the branch.
// It returns an error that wraps [ErrGitHub] for a request that fails.
func (c *Client) Branch(ctx context.Context, repo, name string) (string, bool, error) {
	var ref struct {
		Object object `json:"object"`
	}
	status, err := c.rest(ctx, http.MethodGet, "/repos/"+repo+"/git/ref/heads/"+name, nil, &ref, http.StatusNotFound)
	if err != nil || status == http.StatusNotFound {
		return "", false, err
	}
	return ref.Object.SHA, true, nil
}

// SetBranch points the branch name of repo at the commit sha: it moves an existing branch, also
// when sha does not descend from its commit, and creates a missing one. It returns an error that
// wraps [ErrGitHub] for a request that fails.
func (c *Client) SetBranch(ctx context.Context, repo, name, sha string) error {
	_, found, err := c.Branch(ctx, repo, name)
	if err != nil {
		return err
	}
	if found {
		update := struct {
			SHA   string `json:"sha"`
			Force bool   `json:"force"`
		}{sha, true}
		_, err = c.rest(ctx, http.MethodPatch, "/repos/"+repo+"/git/refs/heads/"+name, update, nil)
		return err
	}
	_, err = c.rest(ctx, http.MethodPost, "/repos/"+repo+"/git/refs", newRef{Ref: branchRefs + name, SHA: sha}, nil)
	return err
}

// Tag returns the commit of the tag name of repo, through the tag object of an annotated tag, and
// reports whether repo has the tag. It returns an error that wraps [ErrGitHub] for a request that
// fails.
func (c *Client) Tag(ctx context.Context, repo, name string) (string, bool, error) {
	var ref struct {
		Object object `json:"object"`
	}
	status, err := c.rest(ctx, http.MethodGet, "/repos/"+repo+"/git/ref/tags/"+name, nil, &ref, http.StatusNotFound)
	if err != nil || status == http.StatusNotFound {
		return "", false, err
	}
	if ref.Object.Type == "tag" {
		var tag struct {
			Object object `json:"object"`
		}
		if _, err := c.rest(ctx, http.MethodGet, "/repos/"+repo+"/git/tags/"+ref.Object.SHA, nil, &tag); err != nil {
			return "", false, err
		}
		return tag.Object.SHA, true, nil
	}
	return ref.Object.SHA, true, nil
}

// CreateTag creates the lightweight tag name of repo at the commit sha, as changesets/action creates
// the tags of a publish. It returns an error that wraps [ErrGitHub] for a request that fails, such
// as for a tag that repo already has.
func (c *Client) CreateTag(ctx context.Context, repo, name, sha string) error {
	_, err := c.rest(ctx, http.MethodPost, "/repos/"+repo+"/git/refs", newRef{Ref: tagRefs + name, SHA: sha}, nil)
	return err
}
