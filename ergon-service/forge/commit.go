// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"encoding/base64"
	"maps"
	"slices"
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
