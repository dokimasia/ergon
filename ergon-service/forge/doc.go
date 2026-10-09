// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package forge is the GitHub client of ergon: the links of a commit and of a pull request for the
// changelog of GitHub, the branch, the signed commits, the commit status and the pull request of a
// version pull request, the tags, the draft releases and the assets of a publish, the casks of a
// tap, the passed runs of a workflow, the trees and the parents of commits, the statuses of commits
// and the heads of pull requests that a publish and a skipped run of the gate verify, and the
// releases of a repository with the digests of their assets for a baseline update.
//
// # Requests
//
// A [Client] sends the requests of one repository host with the token of its [Config]: the REST
// requests with the version 2022-11-28 of the API, and the GraphQL requests to the GraphQL
// address. It reads a response of at most 10 MiB. Every method takes the repository as owner/name.
// [Client.UploadAsset] sends a file to the upload URL that GitHub states in the release, and reads
// the whole file before the request.
//
// # Releases
//
// A publish creates a release with [Client.CreateDraft], uploads its assets with
// [Client.UploadAsset], and then publishes it with [Client.Publish], in the order that GitHub
// requires for immutable releases. [Client.ReleaseOf] finds the draft that an earlier publish left.
//
// # Signed commits
//
// [Client.Commit] commits through the GraphQL mutation createCommitOnBranch, whose commits GitHub
// signs and marks as verified. A commit of the REST endpoints of the Git database is unsigned.
// [Client.PutFile] commits one file to the default branch this way.
//
// # Errors
//
// Each error of a request wraps [ErrGitHub], with the method, the path and the status or the
// message of GitHub. [New] returns [ErrToken] for a configuration without a token.
//
// # Concurrency
//
// A Client is safe for concurrent use, as its [http.Client] is.
//
// # Dependency position
//
// Imports the standard library alone. internal/cli of the root module joins it to the package
// release, which declares the interfaces that a Client implements and reads the types of its
// releases, and [go.dokimi.dev/ergon/service/pin] reads its releases.
package forge
