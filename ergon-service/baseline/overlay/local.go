// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package overlay

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Dir is the directory of the local files: .ergon/local/<path> extends the managed file at <path>.
const Dir = ".ergon/local"

// The lines of a license header, such as the header that ergon license writes on the first lines of
// a local file.
var (
	// copyrightNotice matches a line with a copyright notice: Copyright, ©, or (c), in any case.
	copyrightNotice = regexp.MustCompile(`(?i)(?:\bcopyright\b|©|\(c\))`)

	// spdxTag matches a line with a tag of SPDX, such as SPDX-License-Identifier:.
	spdxTag = regexp.MustCompile(`SPDX-[A-Za-z-]+:`)
)

// The headers of a managed file, which each %s fills with the path of the file.
const (
	header       = "Managed by ergon init. Add repository settings to " + Dir + "/%s and run ergon init sync."
	mergedHeader = "Managed by ergon init, with " + Dir + "/%s merged in. " +
		"Change the repository's settings there and run ergon init sync."
)

// Local is a local file of a repository.
type Local struct {
	// Path is the path of the managed file that the local file extends, relative to the root of the
	// repository and slash-separated.
	Path string

	// Content is the content of the local file.
	Content []byte
}

// Header returns the header that the template of the managed file at path writes in a comment on
// its first line: "Managed by ergon init. Add repository settings to .ergon/local/<path> and run
// ergon init sync."
func Header(path string) string {
	return fmt.Sprintf(header, path)
}

// MergedHeader returns the header of the managed file at path with its local file merged in:
// "Managed by ergon init, with .ergon/local/<path> merged in. Change the repository's settings
// there and run ergon init sync."
func MergedHeader(path string) string {
	return fmt.Sprintf(mergedHeader, path)
}

// Read returns the local files of fsys, the file system of a repository: every file below [Dir],
// sorted by path. It returns no local file for a repository without Dir, and a local file whose
// Path is Dir for a Dir that is a file. It returns an error with the path of a directory or a file
// below Dir that does not read.
func Read(fsys fs.FS) ([]Local, error) {
	if _, err := fs.Stat(fsys, Dir); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	var locals []Local
	err := fs.WalkDir(fsys, Dir, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("overlay: read %s: %w", name, err)
		}
		if d.IsDir() {
			return nil
		}
		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("overlay: read %s: %w", name, err)
		}
		locals = append(locals, Local{Path: strings.TrimPrefix(name, Dir+"/"), Content: content})
		return nil
	})
	slices.SortFunc(locals, func(a, b Local) int { return strings.Compare(a.Path, b.Path) })
	return locals, err
}

// Apply returns the rendering content of the managed file at path with local merged into it: by
// [Merge] for a YAML file, and appended after a newline that content lacks for any other file. An
// appended local file loses the license header on its first lines: the lines that each have a
// copyright notice or a tag of SPDX, when one of them has a tag of SPDX. Apply replaces the
// [Header] of path on the first line of the result with the [MergedHeader] of path, and leaves a
// first line without the header unchanged. It returns the error of Merge, after the path of the
// local file, for a YAML file of which a document does not parse.
func Apply(path string, content, local []byte) ([]byte, error) {
	var merged []byte
	if IsYAML(path) {
		var err error
		if merged, err = Merge(content, local); err != nil {
			return nil, fmt.Errorf("overlay: merge %s/%s: %w", Dir, path, err)
		}
	} else {
		merged = slices.Clone(content)
		if len(merged) > 0 && !bytes.HasSuffix(merged, []byte("\n")) {
			merged = append(merged, '\n')
		}
		merged = append(merged, withoutHeader(local)...)
	}
	first, _, _ := bytes.Cut(merged, []byte("\n"))
	before, after, found := bytes.Cut(first, []byte(Header(path)))
	if !found {
		return merged, nil
	}
	return slices.Concat(before, []byte(MergedHeader(path)), after, merged[len(first):]), nil
}

// withoutHeader returns local without the license header on its first lines: the lines that each
// have a copyright notice or a tag of SPDX, when one of them has a tag of SPDX. The rest keeps its
// first line, such as the empty line after the header, which then separates the local file from the
// rendering before it. It returns local itself for a local file without such a header.
func withoutHeader(local []byte) []byte {
	end, tagged := 0, false
	for line := range bytes.Lines(local) {
		if !copyrightNotice.Match(line) && !spdxTag.Match(line) {
			break
		}
		tagged = tagged || spdxTag.Match(line)
		end += len(line)
	}
	if !tagged {
		return local
	}
	return local[end:]
}

// IsYAML reports whether name is a YAML file, by its extension .yml or .yaml.
func IsYAML(name string) bool {
	ext := path.Ext(name)
	return ext == ".yml" || ext == ".yaml"
}
