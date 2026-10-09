// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"crypto/sha1" //nolint:gosec // git names a blob by the SHA-1 of its content, which PutFile compares.
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// commitMutation commits the changes of files on a branch whose head is the expected commit.
const commitMutation = `mutation($input: CreateCommitOnBranchInput!) {
  createCommitOnBranch(input: $input) { commit { oid } }
}`

// commitInput is the input of commitMutation, as the GraphQL schema of GitHub names its fields.
type commitInput struct {
	// Branch names the repository and the branch.
	Branch struct {
		Repository string `json:"repositoryNameWithOwner"`
		Name       string `json:"branchName"`
	} `json:"branch"`

	// Message is the headline and the body of the commit.
	Message struct {
		Headline string `json:"headline"`
		Body     string `json:"body"`
	} `json:"message"`

	// ExpectedHead is the commit that the branch must point at.
	ExpectedHead string `json:"expectedHeadOid"`

	// FileChanges are the files that the commit adds or changes, and the files that it deletes.
	FileChanges struct {
		Additions []fileAddition `json:"additions"`
		Deletions []fileDeletion `json:"deletions"`
	} `json:"fileChanges"`
}

// fileAddition is a file that a commit adds or changes, with its content in base64.
type fileAddition struct {
	// Path is the path of the file, relative to the root of the repository and slash-separated.
	Path string `json:"path"`

	// Contents is the content of the file in standard base64.
	Contents string `json:"contents"`
}

// fileDeletion is a file that a commit deletes.
type fileDeletion struct {
	// Path is the path of the file, relative to the root of the repository and slash-separated.
	Path string `json:"path"`
}

// Commit commits changes on the branch of repo whose head is the commit head, and returns the new
// commit: the content of each path of files, and the deletion of each path of deleted, each
// relative to the root of the repository and slash-separated. message is the headline of the
// commit, then a blank line and its body. GitHub signs the commit and marks it as verified.
//
// It returns an error that wraps [ErrGitHub] for a branch whose head is not head, and for a
// request that fails, such as one over the size that GitHub accepts in 10 seconds.
func (c *Client) Commit(
	ctx context.Context, repo, branch, head, message string, files map[string][]byte, deleted []string,
) (string, error) {
	var input commitInput
	input.Branch.Repository, input.Branch.Name = repo, branch
	input.Message.Headline, input.Message.Body, _ = strings.Cut(message, "\n\n")
	input.ExpectedHead = head
	input.FileChanges.Additions = make([]fileAddition, 0, len(files))
	for _, path := range slices.Sorted(maps.Keys(files)) {
		contents := base64.StdEncoding.EncodeToString(files[path])
		input.FileChanges.Additions = append(input.FileChanges.Additions, fileAddition{Path: path, Contents: contents})
	}
	input.FileChanges.Deletions = make([]fileDeletion, 0, len(deleted))
	for _, path := range deleted {
		input.FileChanges.Deletions = append(input.FileChanges.Deletions, fileDeletion{Path: path})
	}
	var data struct {
		CreateCommitOnBranch struct {
			Commit struct {
				OID string `json:"oid"`
			} `json:"commit"`
		} `json:"createCommitOnBranch"`
	}
	if err := c.graphql(ctx, commitMutation, map[string]any{"input": &input}, &data); err != nil {
		return "", err
	}
	return data.CreateCommitOnBranch.Commit.OID, nil
}

// PutFile writes content to path on the default branch of repo in one commit with message, which
// [Client.Commit] creates, and GitHub signs. path is relative to the root of the repository and
// slash-separated. PutFile commits nothing when the file at path already has content, so a second
// call with the same content changes nothing. It returns an error that wraps [ErrGitHub] for a
// repository without its default branch, and the errors of the requests.
func (c *Client) PutFile(ctx context.Context, repo, path, message string, content []byte) error {
	var meta struct {
		DefaultBranch string `json:"default_branch"`
	}
	if _, err := c.rest(ctx, http.MethodGet, "/repos/"+repo, nil, &meta); err != nil {
		return err
	}
	branch := meta.DefaultBranch
	head, found, err := c.Branch(ctx, repo, branch)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %s has no default branch %q", ErrGitHub, repo, branch)
	}
	var file struct {
		SHA string `json:"sha"`
	}
	escaped := (&url.URL{Path: path}).EscapedPath()
	status, err := c.rest(ctx, http.MethodGet, "/repos/"+repo+"/contents/"+escaped+"?ref="+url.QueryEscape(head), nil,
		&file, http.StatusNotFound)
	if err != nil {
		return err
	}
	// git names a blob by the SHA-1 of its header, blob <length> and a NUL byte, and its content.
	blob := sha1.New() //nolint:gosec // the name of a git blob, as git computes it.
	blob.Write([]byte("blob " + strconv.Itoa(len(content)) + "\x00"))
	blob.Write(content)
	if status != http.StatusNotFound && file.SHA == hex.EncodeToString(blob.Sum(nil)) {
		return nil
	}
	_, err = c.Commit(ctx, repo, branch, head, message, map[string][]byte{path: content}, nil)
	return err
}

// Tree returns the tree of the commit sha of repo. It returns an error that wraps [ErrGitHub] for a
// request that fails, such as for a commit that repo does not have.
func (c *Client) Tree(ctx context.Context, repo, sha string) (string, error) {
	var commit struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	if _, err := c.rest(ctx, http.MethodGet, "/repos/"+repo+"/git/commits/"+sha, nil, &commit); err != nil {
		return "", err
	}
	return commit.Tree.SHA, nil
}

// Parent returns the first parent of the commit sha of repo, and reports whether the commit has a
// parent. The first parent of a merge commit is the commit of the branch that it merged into. It
// returns an error that wraps [ErrGitHub] for a request that fails, such as for a commit that repo
// does not have.
func (c *Client) Parent(ctx context.Context, repo, sha string) (string, bool, error) {
	var commit struct {
		Parents []struct {
			SHA string `json:"sha"`
		} `json:"parents"`
	}
	if _, err := c.rest(ctx, http.MethodGet, "/repos/"+repo+"/git/commits/"+sha, nil, &commit); err != nil {
		return "", false, err
	}
	if len(commit.Parents) == 0 {
		return "", false, nil
	}
	return commit.Parents[0].SHA, true, nil
}
