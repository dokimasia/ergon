// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/common"
	"go.dokimi.dev/ergon/service/baseline/github"
	"go.dokimi.dev/ergon/service/baseline/lock"
	licensefiles "go.dokimi.dev/ergon/service/licenses/baseline"
)

// Producers returns the producers of the files that ergon init renders in every repository, before
// the languages: the common files, the GitHub files and the license files, in that order. Each call
// returns a new slice.
func Producers() []baseline.Producer {
	return []baseline.Producer{
		{Name: common.Name, Producer: common.Producer{}},
		{Name: github.Name, Producer: github.Producer{}},
		{Name: licensefiles.Name, Producer: licensefiles.Producer{}},
	}
}

// open opens the repository in dir, with the [Producers] before the languages of the catalog of s,
// runs run on it, and closes it. It returns the errors of the open, of run and of the close, joined.
func (s *session) open(dir string, run func(*baseline.Repository) error) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("cli: open the repository: %w", err)
	}
	r, err := baseline.Open(root, s.catalog, s.release, Producers()...)
	if err == nil {
		err = run(r)
	}
	return errors.Join(err, root.Close())
}

// root returns the directory of the repository of the working directory of s: the working
// directory or the nearest of its parents that has the lock of ergon init, and the working
// directory when none has one. A target of the Makefile runs ergon in a directory of the repository,
// such as a module of Go, so ergon tool run and ergon license find the options there.
func (s *session) root() string {
	for dir := s.dir; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(lock.Path))); err == nil {
			return dir
		}
		if filepath.Dir(dir) == dir {
			return s.dir
		}
	}
}
