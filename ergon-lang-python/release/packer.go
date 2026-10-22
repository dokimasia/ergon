// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	pyworkspace "go.dokimi.dev/ergon/lang/python/workspace"
)

// DistDir is the directory of a pack directory with the distributions of the projects, as
// python/<name>/, which [Packer.Pack] writes and [Publisher.Publish] reads.
const DistDir = "python"

// The suffixes of the files of an sdist and of a wheel.
const (
	sdistSuffix = ".tar.gz"
	wheelSuffix = ".whl"
)

// Packer is the [language.Packer] of the toolchain python: it builds the sdist and the wheel of each
// project with uv build.
//
// # Concurrency
//
// A Packer is safe for concurrent use when its Tools is. Two calls of Pack must not write one pack
// directory at once.
type Packer struct {
	// Tools runs the tool of the section python of .ergon.yaml with args in dir, with env added to
	// the environment of the process, as ergon tool run does. Pack runs uv build through it. It must
	// not be nil.
	Tools func(ctx context.Context, dir, tool string, args, env []string) error
}

var _ language.Packer = Packer{}

// Pack builds the sdist and the wheel of each project of pkgs from the workspace at root into
// dir/python/<name>/, with uv build --package <name>, one project at a time in the order of pkgs.
// It then checks that each sdist and wheel of that directory is a distribution of the project at
// its Version. uv build also writes a .gitignore there, which Pack leaves as it is.
//
// It returns an error that wraps [ErrDistribution] for a build that does not write a distribution,
// and for a distribution of another project or version, such as a build backend that computes
// another dynamic version. It also returns the error of uv build and of reading the directory. It
// returns an error that wraps [pyworkspace.ErrVersion] for a Version that PEP 440 does not have,
// before the build.
func (p Packer) Pack(ctx context.Context, root string, pkgs []workspace.Package, dir string) error {
	for i := range pkgs {
		pkg := &pkgs[i]
		if _, err := pyworkspace.FromSemVer(pkg.Version); err != nil {
			return fmt.Errorf("release: %s: %w", pkg.Name, err)
		}
		out := filepath.Join(dir, DistDir, pkg.Name)
		args := []string{"build", "--package", pkg.Name, "--out-dir", out}
		if err := p.Tools(ctx, root, uvTool, args, nil); err != nil {
			return fmt.Errorf("release: build %s: %w", pkg.Name, err)
		}
		if _, err := distributions(out, pkg); err != nil {
			return err
		}
	}
	return nil
}

// distributions returns the paths of the sdist and the wheel files of the directory dir, in the
// order of their names, and leaves out every other file. Each file is a distribution of p at its
// Version, whose name starts with the name of p, with each - as _, and its version of PEP 440. It
// returns an error that wraps [ErrDistribution] for a directory without a distribution, and for a
// distribution of another project or version. It also returns the error of reading the directory,
// and an error that wraps [pyworkspace.ErrVersion] for a Version that PEP 440 does not have.
func distributions(dir string, p *workspace.Package) ([]string, error) {
	v, err := pyworkspace.FromSemVer(p.Version)
	if err != nil {
		return nil, fmt.Errorf("release: %s: %w", p.Name, err)
	}
	prefix := strings.ReplaceAll(p.Name, "-", "_") + "-" + v.String()
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("release: read the distributions of %s: %w", p.Name, err)
	}
	var paths []string
	for _, e := range entries {
		name := e.Name()
		if !e.Type().IsRegular() || (!strings.HasSuffix(name, sdistSuffix) && !strings.HasSuffix(name, wheelSuffix)) {
			continue
		}
		sdist := name == prefix+sdistSuffix
		wheel := strings.HasPrefix(name, prefix+"-") && strings.HasSuffix(name, wheelSuffix)
		if !sdist && !wheel {
			return nil, fmt.Errorf("%w: %s has %s, which is no distribution of %s %s", ErrDistribution, dir, name,
				p.Name, v)
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: %s has no distribution of %s %s", ErrDistribution, dir, p.Name, v)
	}
	return paths, nil
}
