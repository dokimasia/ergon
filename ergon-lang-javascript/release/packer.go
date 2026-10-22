// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
)

// DistDir is the directory of a pack directory with the tarballs of the packages, as js/<name>/,
// which [Packer.Pack] writes and [Publisher.Publish] reads.
const DistDir = "js"

// tarballSuffix is the suffix of the file of a tarball that npm and pnpm pack.
const tarballSuffix = ".tgz"

// scope is the character that starts the scope of the name of an npm package.
const scope = "@"

// packDirMode is the mode of a directory of tarballs that Pack creates.
const packDirMode = 0o755

// Packer is the [language.Packer] of the js toolchain: it packs the tarball of each package with
// the package manager of the repository.
//
// # Concurrency
//
// A Packer is safe for concurrent use when its Command is. Two calls of Pack must not write one pack
// directory at once.
type Packer struct {
	// Command runs the program name of the PATH of the process with args in dir, with env added to
	// the environment of the process. Pack runs npm pack or pnpm pack through it. It must not be nil.
	Command func(ctx context.Context, dir, name string, args, env []string) error
}

var _ language.Packer = Packer{}

// Pack packs the tarball of each package of pkgs from the repository at root into
// dir/js/<name>/, one package at a time in the order of pkgs, and creates that directory first. It
// runs pnpm pack in a repository with pnpm-lock.yaml and npm pack in any other, in the directory of
// the package with --pack-destination. The package manager runs the scripts prepack and postpack of
// the package, and pnpm replaces each requirement of the workspace protocol with the range that it
// publishes. Pack then checks that the directory has the tarball of the package at its Version.
//
// It returns an error that wraps [ErrTarball] for a pack that does not write the tarball of the
// package at its Version. It also returns the error of creating the directory and of the package
// manager.
func (p Packer) Pack(ctx context.Context, root string, pkgs []workspace.Package, dir string) error {
	program := npmProgram
	if _, err := os.Stat(filepath.Join(root, PNPMLockFile)); err == nil {
		program = pnpmProgram
	}
	for i := range pkgs {
		pkg := &pkgs[i]
		out := filepath.Join(dir, DistDir, filepath.FromSlash(pkg.Name))
		if err := os.MkdirAll(out, packDirMode); err != nil {
			return fmt.Errorf("release: %s: %w", pkg.Name, err)
		}
		args := []string{"pack", "--pack-destination", out}
		if err := p.Command(ctx, filepath.Join(root, filepath.FromSlash(pkg.Dir)), program, args, nil); err != nil {
			return fmt.Errorf("release: pack %s: %w", pkg.Name, err)
		}
		if _, err := tarball(dir, pkg); err != nil {
			return err
		}
	}
	return nil
}

// tarball returns the path of the tarball of p at its Version in the pack directory dir, as
// dir/js/<name>/<file>. npm and pnpm name the file after the name of p without the @ of its scope
// and with its / as -, and after the version of p without build metadata, as in probe-a-1.2.0.tgz
// for @probe/a. tarball returns an error that wraps [ErrTarball] for a directory without the file.
func tarball(dir string, p *workspace.Package) (string, error) {
	v := version.Version{Major: p.Version.Major, Minor: p.Version.Minor, Patch: p.Version.Patch, Pre: p.Version.Pre}
	name := strings.ReplaceAll(strings.TrimPrefix(p.Name, scope), "/", "-") + "-" + v.String() + tarballSuffix
	file := filepath.Join(dir, DistDir, filepath.FromSlash(p.Name), name)
	if _, err := os.Stat(file); err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrTarball, file, err)
	}
	return file, nil
}
