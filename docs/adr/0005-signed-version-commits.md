---
adr: 0005
title: Version commits through createCommitOnBranch, split by size
status: Accepted
date: 2026-09-24
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0005: Version commits through createCommitOnBranch, split by size

## Status

Accepted

## Context

The CI version job commits the new versions, changelogs and lockfiles to `ergon-release/<base>` and opens the version pull request. The job has no signing key. GitHub's GraphQL mutation `createCommitOnBranch` makes a commit that GitHub signs and marks verified. The Git Database REST API makes a commit from a blob, a tree and a commit object.

ergon's version commit includes lockfiles, and changesets' does not. The largest lockfile in these repositories is `dokimasia/stealth/bun.lock`, at 674,822 bytes. GitHub documents no request size limit for the GraphQL API, and it ends any request after 10 seconds.

A test on a scratch branch measured both paths with a user token:

| File size | base64 payload | `createCommitOnBranch` | Verified | Git Database REST | Verified |
|---|---|---|---|---|---|
| 674,822 bytes | 899,764 bytes | 2.1 s | yes | 3.2 s | no |
| 2,000,000 bytes | 2,666,668 bytes | 2.7 s | yes | 3.2 s | no |
| 5,000,000 bytes | 6,666,668 bytes | 3.4 s | yes | 4.6 s | no |
| 10,000,000 bytes | 13,333,336 bytes | 7.0 s | yes | 5.2 s | no |

## Decision

We will commit the version pull request through `createCommitOnBranch`, in commits whose base64 payload is at most 6,666,668 bytes each, because GitHub signs those commits and a 13,333,336-byte request already took 7.0 of the 10 seconds GitHub allows.

## Alternatives Considered

### The Git Database REST API

Each file is its own blob request, so no single request contains the whole change. It lost because GitHub left all four REST commits unsigned, with reason `unsigned`.

### One commit regardless of size

One `createCommitOnBranch` call contains the whole change. It lost because the time per request grows with its size, from 2.1 seconds at 899,764 bytes to 7.0 seconds at 13,333,336 bytes, and a request past 10 seconds fails.

## Consequences

**Positive:**

- Every commit on the version branch is signed by GitHub and marked verified.
- A version pull request with lockfiles of any size can be committed.

**Negative:**

- A large change becomes several commits on the version branch.
- The 6,666,668-byte bound comes from one measurement with a user token, not from a documented limit.
- The commits are signed by GitHub, not by a maintainer's key.

**Neutral:**

- The mutation appends to an existing branch, so `ci version` first points `ergon-release/<base>` at the base commit through the REST refs endpoint.

## References

| What | Where |
|---|---|
| `createCommitOnBranch` and GitHub's signature | GitHub GraphQL schema, mutation description |
| GraphQL limits, with the 10-second timeout | https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api |
| The measurement | `dokimasia/techne`, branch `ergon-test/commit-size`, 2026-09-24, deleted after the run |
