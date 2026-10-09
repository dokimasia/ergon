---
"go.dokimi.dev/ergon": minor
"go.dokimi.dev/ergon/core": minor
"go.dokimi.dev/ergon/service": minor
"go.dokimi.dev/ergon/lang/go": minor
---

Build, sign and attest the binaries of the commands of a Go module in each release of the module. The key binaries of the section go lists the commands. ergon init renders the configuration of GoReleaser of each module with a command. The job pack of release.yml builds the archives, the deb, rpm and apk packages, the casks, the checksums with their cosign signature and the SBOMs, and attests them. UPX packs each Linux binary with LZMA, and ergon tool run unpacks the .tar.xz of a release binary such as UPX. ergon release publish attaches the assets to a draft release before it publishes the release. ergon release ci homebrew then commits the casks to the tap of the key homebrew. The managed .gitignore ignores publish-plan.json and /dist/, so that Go writes the version of the release into each binary.
