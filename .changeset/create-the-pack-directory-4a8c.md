---
"go.dokimi.dev/ergon/lang/rust": patch
---

Fix the Cargo packer to create the pack directory, so `ergon release pack --out-dir dist` packs the crates of a Cargo workspace.
