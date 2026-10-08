// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	goworkspace "go.dokimi.dev/ergon/lang/go/workspace"
	"golang.org/x/mod/module"
)

// modSuffix follows the version of a line of go.sum that records the hash of the go.mod of the
// version, as in v1.2.0/go.mod.
const modSuffix = "/go.mod"

// sum is one line of a go.sum: the hash of the zip of a module version, or of its go.mod.
type sum struct {
	// version is the module version of the line.
	version module.Version

	// hash is the hash that the line records, such as h1: and the base64 of a SHA-256.
	hash string

	// mod reports that the line records the hash of the go.mod of the version.
	mod bool
}

// String returns s as a line of go.sum, without its newline.
func (s sum) String() string {
	v := s.version.Version
	if s.mod {
		v += modSuffix
	}
	return s.version.Path + " " + v + " " + s.hash
}

// readSums returns the lines of the go.sum of m in the repository at root, in their order, and none
// for a module without a go.sum. It skips an empty line, as the go command does. It returns the
// error of reading the go.sum, and an error for a line that is not a module path, a version and a
// hash, with the go.sum and the number of the line.
func readSums(root string, m *goworkspace.Module) ([]sum, error) {
	file := path.Join(m.Dir, goworkspace.SumFile)
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("release: read %s: %w", file, err)
	}
	var sums []sum
	number := 0
	for line := range strings.Lines(string(data)) {
		number++
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 {
			return nil, fmt.Errorf("release: read %s: line %d is not a module path, a version and a hash", file, number)
		}
		v, mod := strings.CutSuffix(fields[1], modSuffix)
		sums = append(sums, sum{version: module.Version{Path: fields[0], Version: v}, hash: fields[2], mod: mod})
	}
	return sums, nil
}

// formatSums returns sums as the content of a go.sum, a line for each.
func formatSums(sums []sum) []byte {
	var b strings.Builder
	for _, s := range sums {
		b.WriteString(s.String())
		b.WriteString("\n")
	}
	return []byte(b.String())
}
