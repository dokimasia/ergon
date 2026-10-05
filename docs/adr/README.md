# Decision records

Each ADR records one decision and what it cost. Once an ADR is accepted, do not change its argument. A later decision supersedes it, and both records link to each other.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-one-module-per-language.md) | One module per language | Accepted, superseded in part by ADR-0008 |
| [0002](0002-independent-module-versions.md) | Independent module versions, with the root module as an entry point | Accepted |
| [0003](0003-follow-changesets.md) | Follow changesets' file format, configuration and planning rules | Accepted |
| [0004](0004-go-releases-in-one-commit.md) | Go releases in one commit through a file proxy | Accepted |
| [0005](0005-signed-version-commits.md) | Version commits through createCommitOnBranch, split by size | Accepted |
| [0006](0006-publish-credentials.md) | OIDC for npm, PyPI and crates.io, and environment secrets for Maven Central | Accepted |
| [0007](0007-toolchains-and-languages.md) | Toolchains and languages are separate catalog entries | Accepted, superseded in part by ADR-0008 |
| [0008](0008-toolchains-in-language-modules.md) | The npm and Gradle toolchains are in the JavaScript and Java modules | Accepted |
