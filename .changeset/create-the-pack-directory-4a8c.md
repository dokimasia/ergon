---
"go.dokimi.dev/ergon/service": patch
---

Fix `release.Pack` to create the pack directory before the first packer runs, so `ergon release pack --out-dir dist` packs the crates of a Cargo workspace.
