// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package option

import (
	"fmt"
	"slices"
)

// PackageManager is the package manager of the npm packages of a repository: the program that
// installs the packages of package.json and runs their scripts, and that ergon tool run starts an
// [NPM] tool with.
type PackageManager string

// The package managers of npm packages.
const (
	// PackageManagerNPM installs the packages of package-lock.json with npm, and ergon tool run
	// starts an npm package with npx.
	PackageManagerNPM PackageManager = "npm"

	// PackageManagerPNPM installs the packages of pnpm-lock.yaml with pnpm, and ergon tool run
	// starts an npm package with pnpm dlx.
	PackageManagerPNPM PackageManager = "pnpm"
)

// packageManagers are the package managers.
var packageManagers = []PackageManager{PackageManagerNPM, PackageManagerPNPM}

// Validate returns an error that wraps [ErrInvalid] for an m other than npm and pnpm.
func (m PackageManager) Validate() error {
	if !slices.Contains(packageManagers, m) {
		return fmt.Errorf("%w: package manager %q, which is neither npm nor pnpm", ErrInvalid, m)
	}
	return nil
}

// PackageManaged is the options of the section that sets the package manager of the npm packages of
// a repository, as the section js does. ergon tool run starts the npm packages of every section
// with that package manager.
type PackageManaged interface {
	// Manager returns the package manager of the npm packages of the repository.
	Manager() PackageManager
}
