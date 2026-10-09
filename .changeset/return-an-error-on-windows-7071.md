---
"go.dokimi.dev/ergon/service": patch
"go.dokimi.dev/ergon": patch
---

Return an error on Windows for a file in place of the directory of the assets or the casks of a release, as on Linux and macOS. ergon release publish and ergon release ci homebrew read the directory.
