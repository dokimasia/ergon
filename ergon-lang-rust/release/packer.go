// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
)

// DistDir is the directory of a pack directory with the .crate files of the crates, as
// rust/<name>/<name>-<version>.crate, which [Packer.Pack] writes and [Publisher.Publish] reads.
const DistDir = "rust"

// crateSuffix is the suffix of the .crate file of a crate, the package that crates.io receives.
const crateSuffix = ".crate"

// The command and the flags of cargo that package the crates.
const (
	packageCommand = "package"
	targetDirFlag  = "--target-dir"
	packageFlag    = "--package"
)

// The target directory of cargo package below the root of the repository, which the .gitignore of
// Rust ignores, and its directory of .crate files.
const (
	targetDir      = "target"
	packTargetDir  = "ergon"
	packageDirName = "package"
)

// The modes of the directories and the .crate files that Pack creates in the pack directory.
const (
	dirMode   = 0o755
	crateMode = 0o644
)

// Packer is the [language.Packer] of the toolchain rust: it packages and verifies the crates with
// cargo package.
//
// # Concurrency
//
// A Packer is safe for concurrent use when its Command is. Two calls of Pack must not write one pack
// directory or one repository at once.
type Packer struct {
	// Command runs the program name of the PATH with args in dir, with env added to the environment
	// of the process. Pack runs cargo package through it. It must not be nil.
	Command func(ctx context.Context, dir, name string, args, env []string) error
}

var _ language.Packer = Packer{}

// Pack packages the crates of pkgs from the workspace at root with one run of cargo package with a
// flag --package for each crate, into the target directory target/ergon of root. cargo package
// builds each crate from its package, and a crate that requires a new version of another crate of
// the run builds against the package of that crate. Pack then creates dir when it does not exist,
// and copies the .crate file of each crate to dir/rust/<name>/<name>-<version>.crate. It runs
// nothing for no crate. cargo package of Cargo 1.90 or later packages crates that require each
// other in one run.
//
// It returns an error that wraps [ErrCrate] for a crate whose .crate file cargo package does not
// write, and the errors of cargo package, of creating dir and of writing the pack directory.
func (p Packer) Pack(ctx context.Context, root string, pkgs []workspace.Package, dir string) error {
	if len(pkgs) == 0 {
		return nil
	}
	target := filepath.Join(root, targetDir, packTargetDir)
	args := []string{packageCommand, targetDirFlag, target}
	for i := range pkgs {
		args = append(args, packageFlag, pkgs[i].Name)
	}
	if err := p.Command(ctx, root, cargoProgram, args, nil); err != nil {
		return fmt.Errorf("release: package the crates: %w", err)
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("release: create the pack directory: %w", err)
	}
	pack, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("release: open the pack directory: %w", err)
	}
	defer func() { _ = pack.Close() }()
	for i := range pkgs {
		name := pkgs[i].Name + "-" + pkgs[i].Version.String() + crateSuffix
		var data []byte
		if data, err = os.ReadFile(filepath.Join(target, packageDirName, name)); err != nil {
			return fmt.Errorf("%w: cargo package wrote no %s: %w", ErrCrate, name, err)
		}
		// The root refuses a name of a crate that leaves the pack directory.
		out := filepath.Join(DistDir, pkgs[i].Name)
		if err = pack.MkdirAll(out, dirMode); err != nil {
			return fmt.Errorf("release: create the directory of %s: %w", pkgs[i].Name, err)
		}
		if err = pack.WriteFile(filepath.Join(out, name), data, crateMode); err != nil {
			return fmt.Errorf("release: write %s: %w", name, err)
		}
	}
	return nil
}
