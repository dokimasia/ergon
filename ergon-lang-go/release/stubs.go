// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"bytes"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// importDynamic is the directive of the Go source that imports a symbol of a dynamic library, as
// //go:cgo_import_dynamic <local> <remote> "<library>".
const importDynamic = "//go:cgo_import_dynamic"

// The libraries of macOS: the directory of the system libraries, the directory of the frameworks,
// and libSystem, whose stub zig has.
const (
	systemLibraries = "/usr/lib/"
	frameworks      = "/System/Library/Frameworks/"
	libSystem       = "/usr/lib/libSystem.B.dylib"
)

// stubSuffix ends the name of a stub, and frameworkSuffix ends the name of the directory of a
// framework.
const (
	stubSuffix      = ".tbd"
	frameworkSuffix = ".framework"
)

// stubFormat is a text stub of a library of macOS in version 4 of the format of Apple's TAPI: the
// path of the library, then its symbols on both architectures of macOS, each with the underscore
// of a C symbol of Mach-O.
const stubFormat = `--- !tapi-tbd
tbd-version: 4
targets: [ x86_64-macos, arm64-macos ]
install-name: '%s'
current-version: 1
compatibility-version: 1
exports:
  - targets: [ x86_64-macos, arm64-macos ]
    symbols: [ %s ]
...
`

// writeStubs writes into dir a text stub of each library of macOS other than libSystem that a
// //go:cgo_import_dynamic line of a Go file under goroot/src imports a symbol from: a library
// under /usr/lib as lib<name>.tbd, and a framework under /System/Library/Frameworks as
// <Name>.framework/<Name>.tbd, where the linker of zig finds them with -L and -F on dir. Each stub
// states the path of its library and the symbols that the lines import from it, sorted and once
// each. writeStubs reads every Go file but the test files through an [os.Root] of goroot/src, so
// the stubs cover a build of darwin on either architecture. It returns the error of reading the
// source and of writing a stub, each with goroot.
func writeStubs(goroot, dir string) error {
	src, err := os.OpenRoot(filepath.Join(goroot, "src"))
	if err != nil {
		return fmt.Errorf("the Go toolchain at %s: %w", goroot, err)
	}
	defer func() { _ = src.Close() }()
	fsys := src.FS()
	imports := map[string][]string{}
	err = fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return err
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if !bytes.Contains(data, []byte(importDynamic)) {
			return nil
		}
		for line := range strings.Lines(string(data)) {
			f := strings.Fields(line)
			if len(f) != 4 || f[0] != importDynamic || f[2] == "_" {
				continue
			}
			library := strings.Trim(f[3], `"`)
			macOS := strings.HasPrefix(library, systemLibraries) || strings.HasPrefix(library, frameworks)
			if macOS && library != libSystem {
				imports[library] = append(imports[library], "_"+f[2])
			}
		}
		return nil
	})
	for _, library := range slices.Sorted(maps.Keys(imports)) {
		var stub string
		if framework, ok := strings.CutPrefix(library, frameworks); ok {
			bundle, _, _ := strings.Cut(framework, "/")
			stub = filepath.Join(dir, bundle, strings.TrimSuffix(bundle, frameworkSuffix)+stubSuffix)
		} else {
			name, _, _ := strings.Cut(path.Base(library), ".")
			stub = filepath.Join(dir, name+stubSuffix)
		}
		symbols := slices.Compact(slices.Sorted(slices.Values(imports[library])))
		if err == nil {
			err = os.MkdirAll(filepath.Dir(stub), dirPerm)
		}
		if err == nil {
			err = os.WriteFile(stub, fmt.Appendf(nil, stubFormat, library, strings.Join(symbols, ", ")), filePerm)
		}
	}
	if err != nil {
		return fmt.Errorf("the Go toolchain at %s: %w", goroot, err)
	}
	return nil
}
