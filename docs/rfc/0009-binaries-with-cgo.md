---
rfc: 0009
title: Binaries of commands with cgo
author: Roy Klopper
status: Accepted
created: 2026-10-21
updated: 2026-10-21
discussion: none
supersedes: RFC-0007, in part
superseded-by: none
produces-adr: ADR-0056
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0009: Binaries of commands with cgo

## Summary

A command of `go.binaries` with `cgo: true` builds with cgo for every platform of its release, on the Linux runner of the job `pack`. zig is the C compiler of each target: it links static musl binaries for Linux, mingw binaries for Windows, and Mach-O binaries for macOS. For macOS, `ergon release pack` writes text stubs of the system libraries that Go's standard library imports, derived from the source of the Go toolchain that builds. The pinned UPX packs the Linux binaries, as it packs the binaries without cgo. The job `pack` does not change.

## Motivation

The managed binary release builds every command with `CGO_ENABLED=0`. eidos, techne and dokimi each have a command that imports the C grammars of tree-sitter through cgo. eidos and techne each keep a hand-written `.goreleaser.yml` and a local job `binaries` in `release.yml`:

- The job runs on `macos-26`. darwin builds with the clang of the runner, and Linux and Windows build with zig.
- The job installs an unpinned UPX with `brew install upx`, because UPX does not publish a program for macOS.
- The `.goreleaser.yml` of eidos and that of techne differ only in their names and their version flags. dokimi does not release its command.

A build of darwin needs a macOS host only for the libraries that the external linker resolves. We cross-compiled `cmd/eidos` and `cmd/techne` on Linux with zig 0.16.0 and Go 1.27.3 on 2026-10-21:

- The C sources of tree-sitter compiled for `aarch64-macos` and `x86_64-macos` without a macOS SDK.
- The link failed on `-lresolv`, which `internal/syscall/unix/net_darwin.go` of Go requests on darwin, and for techne on the frameworks CoreFoundation and Security, which `crypto/x509` imports. The archive of zig has stubs of `libSystem` alone.
- With text stubs of `libresolv`, CoreFoundation and Security, both commands linked for both architectures. Each binary binds those libraries and `libSystem` under their paths on macOS, and has the minimum version 13.0, which Go's own linker writes. The arm64 binaries have an ad-hoc code signature, which Apple silicon requires.

Go's standard library imports 3 symbols of `libresolv`, 23 of CoreFoundation and 8 of Security on darwin, through `//go:cgo_import_dynamic` lines with the path of each library.

## Detailed design

### Components

| Component | Where | Responsibility |
|---|---|---|
| Command option | `lang/go/baseline` | The key `cgo` of a command of `go.binaries` |
| zig | `core/option`, `lang/go/baseline` | The release binary `option.Zig` from ziglang.org, pinned in `go.tools.zig` |
| Tree of a release binary | `core/option`, `service/tool` | `Asset.Tree`: the program needs the other files of its archive, so `ergon tool run` unpacks the whole archive |
| Path of a tool | `service/tool`, `internal/cli` | `Runner.Program` and `ergon tool path <section>.<tool>`, which install a tool and print its program |
| GoReleaser configuration | `lang/go/baseline` | The builds of a command with cgo |
| Go packer | `lang/go/release` | The path of zig, the stubs of macOS and the caches of zig in the environment of GoReleaser |
| Resolution of zig | `service/pin`, `internal/rewrite` | The kind `KindZig`, which reads the download index of ziglang.org |

### The option

```yaml
go:
  binaries:
    - name: eidos
      module: .
      main: ./cmd/eidos
      description: Generates code with the declarative plugins of a workspace
      cgo: true
```

```go
// Package baseline (lang/go/baseline).

type Command struct {
	// The other fields are unchanged.

	// CGO reports that the command builds with cgo, such as a command that links C grammars of
	// tree-sitter. zig compiles the C code of each platform.
	CGO bool `yaml:"cgo"`
}
```

`cgo` takes every platform of a release binary. No other command may have the name of the macOS build of a command with cgo, `<name>-darwin`, as no command may have the name of a UPX build.

### The builds

A command with cgo has the builds of a command without cgo, with `CGO_ENABLED=1` and the C compiler of each target:

| Build | Targets | `CC` | Other |
|---|---|---|---|
| `<name>-upx` | linux/amd64, linux/arm64 | `zig cc -target x86_64-linux-musl`, `aarch64-linux-musl` | UPX packs each binary |
| `<name>` | windows/amd64, windows/arm64 | `zig cc -target x86_64-windows-gnu`, `aarch64-windows-gnu` | |
| `<name>-darwin` | darwin/amd64, darwin/arm64 | `zig cc -target x86_64-macos.13.0`, `aarch64-macos.13.0` | `CGO_LDFLAGS` puts the stubs on the paths of libraries and frameworks |

`CC` is `"{{ .Env.ERGON_ZIG }}" cc -target <triple>`. The quotes keep a path with a space in one argument, because the go command splits `CC` at the spaces outside quotes. The architecture of the triple follows `{{ .Arch }}` of GoReleaser.

The triple of macOS states the version 13.0, the oldest macOS that Go supports. Go's linker writes that minimum into a binary that it links itself, but it passes no minimum to an external linker. zig 0.17.0 writes 15.0 for a triple without a version, so the binaries would refuse macOS 13 and 14. The archives, the packages, the SBOMs, the checksums, the signature and the cask of a command with cgo are the same as those of a command without cgo.

### zig

```go
// Package option (core/option).

// Asset gains the tree of its archive.
type Asset struct {
	// The other fields are unchanged.

	// Tree reports that the program needs the other files of its archive, such as the library of
	// zig beside its program. ergon tool run then unpacks the whole archive.
	Tree bool
}

// Zig is the release binary of zig, which ziglang.org publishes for every platform. Its asset is
// the archive zig-<arch>-<os>-<version>, a .tar.xz on Linux and macOS and a .zip on Windows, whose
// tree the program needs.
type Zig struct {
	Binary `yaml:",inline"`
}

// Asset returns the archive of zig for p, with the program zig-<arch>-<os>-<version>/zig, or
// zig.exe on Windows, and Tree set.
func (z Zig) Asset(p Platform) (Asset, error)

// Repository returns the empty string, because no repository on GitHub publishes the releases of
// zig. A baseline update reads the download index of ziglang.org.
func (Zig) Repository() string
```

`go.tools.zig` pins zig 0.17.0 with the SHA-256 of the archive of each of the six platforms, from `https://ziglang.org/download/index.json`.

`ergon tool run` unpacks the whole archive of an asset with `Tree` into its cache directory through an `os.Root`, and runs the program inside it. It refuses an entry whose path leaves the directory, and an entry that is neither a directory nor a regular file. It writes the tree into a temporary directory and renames it, so a run reads a whole tree.

### The path of a tool

```go
// Package tool (service/tool).

// Program installs the tool name of the options o of section, unless the cache has it, and returns
// its program. It returns an error that wraps ErrUnknown for options without the tool, and
// ErrInstall for a tool that does not install, or that runs through another program, such as an
// npm package that npx runs.
func (r *Runner) Program(ctx context.Context, section string, o language.Options, name string) (string, error)
```

`ergon tool path <section>.<tool>` prints that program. A Makefile or a script can then pass the path of a tool to a program that starts it, such as `CC`.

### The packer

`golang.Register` takes the tools of the section `go` as a struct, because the packer needs the path of zig:

```go
// Package golang (lang/go).

// Tools runs and finds the tools of the section go of .ergon.yaml, as ergon tool run and ergon
// tool path do.
type Tools struct {
	// Run runs the tool with args in dir, with env added to the environment of the process.
	Run func(ctx context.Context, dir, tool string, args, env []string) error

	// Program installs the tool unless the cache has it, and returns its program.
	Program func(ctx context.Context, dir, tool string) (string, error)
}
```

`Packer.Pack` reads the builds of each configuration. For a configuration with a build that sets `CGO_ENABLED=1`, it adds these variables to the environment of its run of GoReleaser:

- `ERGON_ZIG` is the program of `go.tools.zig`, which `Program` installs.
- `ERGON_DARWIN_STUBS` is a directory with a text stub of each library of macOS other than `libSystem` that the standard library of the Go toolchain imports: `lib<name>.tbd` for a library under `/usr/lib`, and `<Name>.framework/<Name>.tbd` for a framework. Each stub states the path of the library and the symbols that the `//go:cgo_import_dynamic` lines of the Go files under `src` of the toolchain import from it. `go env GOROOT` in the directory of the module, outside the workspace, reports the root of the toolchain that the builds use.
- `ZIG_GLOBAL_CACHE_DIR` and `ZIG_LOCAL_CACHE_DIR` are a directory of the temporary directory of the pack, because the module cache is read-only and zig writes its cache beside a C source by default.

A pack without a command with cgo does not install zig or write a stub.

### Resolution

```go
// Package pin (service/pin).

// KindZig is a release binary of zig, from the download index of ziglang.org.
const KindZig Kind = 11

// Registries gains the download index of zig.
type Registries struct {
	// The other fields are unchanged.

	// Zig is the download index of zig, such as https://ziglang.org/download/index.json.
	Zig string
}
```

`Find` classifies an `option.Zig` as `KindZig`. `Resolve` reads the index:

- Each stable version of the index is a release. The entry `master` is not.
- The date of a version is its time of publication.
- The `shasum` of the archive of each platform of the pin is the digest of that platform.

The rewrite of a baseline update moves the version and the digests of a pin of `KindZig` as it moves those of a `KindBinary`.

### Failure handling

| Failure | State afterwards | Recovery |
|---|---|---|
| zig does not install | `pack` fails before GoReleaser runs | Fix the pin or the network, and rerun the job |
| A command links a library that no stub covers | GoReleaser fails on the link of darwin, and nothing is published | The command needs a library outside the standard library of Go. Such a command is out of scope |
| A version of Go imports a new library on darwin | The stubs follow the source of that toolchain | None |
| A version of Go raises its oldest macOS | The spec of the builds, which compares the triple with the linker of that Go, fails | Raise the version in the triple of macOS |
| UPX refuses a binary | GoReleaser fails, and nothing is published | The same as for a command without cgo |

Invariants:

- A command with cgo builds every platform of its release in the job `pack`, on its Linux runner.
- A darwin binary binds only libraries that every macOS has under the paths of its stubs.
- Every tool of the build, zig and UPX included, has a version and a SHA-256 per platform in `.ergon.yaml`.

### Out of scope

- **C code that needs Apple's headers or frameworks**: the stubs cover the libraries that Go's standard library imports. A command whose own C code includes a header of the macOS SDK does not build.
- **C++**: `zig c++` compiles it, but no command of these repositories has C++ code. A command that needs it sets `CXX` in its local configuration.
- **A minimum macOS version other than 13.0**: the binaries of a command with cgo state the oldest macOS that Go supports, as the binaries of a command without cgo do.

## Alternatives considered

### A. A macOS runner for darwin

The job `pack` runs on `macos-26` for a repository with a command with cgo, as the jobs of eidos and techne do.

**Why not:** UPX does not publish a program for macOS, so the job either runs an unpinned UPX from Homebrew or does not pack the binaries. macOS minutes also cost more than Linux minutes on a private repository, such as dokimi.

### B. A split of the runners

A Linux job builds Linux and Windows, and a macOS job builds darwin.

**Why not:** GoReleaser without Pro cannot import a binary from another runner. Its builder `prebuilt` is a feature of Pro. The cask needs the archives of darwin and of Linux, and GoReleaser builds a cask only from the archives of its own run, so ergon would write the checksums, the signature and the cask itself.

### C. The container of goreleaser-cross

The job runs in the image `goreleaser/goreleaser-cross`, which has a C cross compiler and a sysroot of each platform.

**Why not:** the image contains the macOS SDK, and its newest tag on 2026-10-21, `v1.26.10-v2.18.3`, is 2.05 GB compressed. That tag has Go 1.26.10, older than the Go 1.27 of the repositories. zig compiles the same C code, and is one pinned archive of about 55 MB.

### D. The action setup-zig

`mlugg/setup-zig` installs zig in the job.

**Why not:** a workstation cannot run the same zig, and the action adds a pin of an action beside the version of zig. `go.tools.zig` pins both in one place.

## Drawbacks

- No run on macOS has checked the darwin binaries that this design builds. The probe checked their load commands, their libraries and their signature.
- Each pack of a repository with a command with cgo downloads zig once, 55 MB, and unpacks 20,000 files.
- The darwin builds depend on the `//go:cgo_import_dynamic` lines of Go's source. A change of their form in a later Go breaks the stubs until the packer follows it.
- `golang.Tools` changes from a function to a struct, which is a breaking change of the module `lang/go`.

## References

| What | Where |
|---|---|
| The limits of cgo in GoReleaser, and zig as its C compiler | https://goreleaser.com/resources/limitations/cgo/ |
| The builder `prebuilt` of GoReleaser Pro | https://goreleaser.com/customization/builds/builders/prebuilt/ |
| The download index of zig | https://ziglang.org/download/index.json |
| The text-based stubs of Apple's linker | https://github.com/apple-oss-distributions/tapi |
