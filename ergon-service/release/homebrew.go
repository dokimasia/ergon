// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.dokimi.dev/ergon/core/language"
)

// The casks of a Homebrew tap: the directory of the tap that contains them, and the extension of a
// cask.
const (
	tapCasks = "Casks"
	caskExt  = ".rb"
)

// TapForge is the host of the Homebrew tap that a release commits its casks to. *forge.Client
// implements it.
type TapForge interface {
	// PutFile writes content to path on the default branch of repo in one commit with message, and
	// commits nothing when the file already has content.
	PutFile(ctx context.Context, repo, path, message string, content []byte) error
}

// Homebrew commits the casks of the releases of plan from the pack directory dir to tap, a
// repository as owner/name: each file dir/casks/<tag>/<name>.rb of an entry of plan to
// Casks/<name>.rb of the tap, in one commit with the message <name> <version>, in the order of the
// plan and of the names. A release without the directory of its tag has no cask. It returns the
// paths of the casks in the tap, and an error with the name of the cask for a cask that it cannot
// read or commit, after the casks that it committed before it.
func Homebrew(ctx context.Context, f TapForge, tap string, plan *PublishPlan, dir string) ([]string, error) {
	var committed []string
	for _, chunk := range plan.Plan {
		for k := range chunk {
			e := &chunk[k]
			casksDir := filepath.Join(dir, language.CasksDir, filepath.FromSlash(e.Tag))
			entries, err := os.ReadDir(casksDir)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return committed, fmt.Errorf("release: read the casks of %s: %w", e.Tag, err)
			}
			for _, entry := range entries {
				name, ok := strings.CutSuffix(entry.Name(), caskExt)
				if !ok || !entry.Type().IsRegular() {
					continue
				}
				content, err := os.ReadFile(filepath.Join(casksDir, entry.Name()))
				if err != nil {
					return committed, fmt.Errorf("release: read the cask %s: %w", name, err)
				}
				target := path.Join(tapCasks, entry.Name())
				if err := f.PutFile(ctx, tap, target, name+" "+e.Version.String(), content); err != nil {
					return committed, fmt.Errorf("release: commit the cask %s to %s: %w", name, tap, err)
				}
				committed = append(committed, target)
			}
		}
	}
	return committed, nil
}
