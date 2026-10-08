// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package overlay_test

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/baseline/overlay"
)

// The headers of the cases, pinned because a person reads them: the header of notes.txt that a
// template writes on its first line, and the header that names its merged local file.
const (
	notesHeader = "Managed by ergon init. Add repository settings to .ergon/local/notes.txt and run ergon init sync."
	notesMerged = "Managed by ergon init, with .ergon/local/notes.txt merged in. " +
		"Change the repository's settings there and run ergon init sync."
	toolsHeader = "Managed by ergon init. Add repository settings to .ergon/local/tools.yml and run ergon init sync."
	toolsMerged = "Managed by ergon init, with .ergon/local/tools.yml merged in. " +
		"Change the repository's settings there and run ergon init sync."
)

// dir pins the directory of the local files.
const dir = ".ergon/local"

// errFault is the error of an Open that a failing file system fails.
var errFault = errors.New("fault: injected")

// failing is a file system whose Open fails for name. It hides the ReadDir and ReadFile of the file
// system it wraps, so every read opens the file.
type failing struct {
	fs.FS

	// name is the path whose Open fails.
	name string
}

// Open fails for f.name, and opens any other file of the file system, with its error wrapped.
func (f failing) Open(name string) (fs.File, error) {
	if name == f.name {
		return nil, errFault
	}
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failing: %w", err)
	}
	return file, nil
}

func TestLocal(t *testing.T) {
	t.Parallel()

	t.Run("Dir", func(t *testing.T) {
		t.Parallel()

		t.Run("is local in the directory .ergon", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, overlay.Dir, dir, "Dir")
		})
	})

	t.Run("Header", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the header that names the local file of the path", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, overlay.Header("notes.txt"), notesHeader, "Header of notes.txt")
		})
	})

	t.Run("MergedHeader", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the header that states the merge of the local file", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, overlay.MergedHeader("notes.txt"), notesMerged, "MergedHeader of notes.txt")
		})
	})

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		t.Run("returns every file below the directory sorted by path", func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{
				dir + "/b.txt":                        {Data: []byte("b\n")},
				dir + "/.github/workflows/ci.yml":     {Data: []byte("jobs: {}\n")},
				dir + "/a.txt":                        {Data: []byte("a\n")},
				"README.md":                           {Data: []byte("# demo\n")},
				".ergon/init.lock":                    {Data: []byte("{}\n")},
				dir + "/.github/workflows/README.txt": {Data: []byte("r\n")},
			}
			got, err := overlay.Read(fsys)
			assert.NoError(t, err, "Read")
			assert.Equal(t, got, []overlay.Local{
				{Path: ".github/workflows/README.txt", Content: []byte("r\n")},
				{Path: ".github/workflows/ci.yml", Content: []byte("jobs: {}\n")},
				{Path: "a.txt", Content: []byte("a\n")},
				{Path: "b.txt", Content: []byte("b\n")},
			}, "the local files")
		})

		t.Run("returns a file of a directory after a file whose path is lower", func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{
				dir + "/a/b.txt": {Data: []byte("b\n")},
				dir + "/a-c.txt": {Data: []byte("c\n")},
			}
			got, err := overlay.Read(fsys)
			assert.NoError(t, err, "Read")
			assert.Equal(t, got, []overlay.Local{
				{Path: "a-c.txt", Content: []byte("c\n")},
				{Path: "a/b.txt", Content: []byte("b\n")},
			}, "the local files, in the order of their paths and not of the walk")
		})

		t.Run("returns no local file for a repository without the directory", func(t *testing.T) {
			t.Parallel()
			got, err := overlay.Read(fstest.MapFS{"README.md": {Data: []byte("# demo\n")}})
			assert.NoError(t, err, "Read")
			assert.Empty(t, got, "the local files")
		})

		t.Run("returns the directory as a local file when it is a file", func(t *testing.T) {
			t.Parallel()
			got, err := overlay.Read(fstest.MapFS{dir: {Data: []byte("x\n")}})
			assert.NoError(t, err, "Read")
			assert.Equal(t, got, []overlay.Local{{Path: dir, Content: []byte("x\n")}}, "the local files")
		})

		t.Run("returns an error for a directory that does not read", func(t *testing.T) {
			t.Parallel()
			fsys := failing{FS: fstest.MapFS{dir + "/sub/a.txt": {Data: []byte("a\n")}}, name: dir + "/sub"}
			_, err := overlay.Read(fsys)
			assert.ErrorIs(t, err, errFault, "Read")
		})

		t.Run("returns an error for a file that does not read", func(t *testing.T) {
			t.Parallel()
			fsys := failing{FS: fstest.MapFS{dir + "/a.txt": {Data: []byte("a\n")}}, name: dir + "/a.txt"}
			_, err := overlay.Read(fsys)
			assert.ErrorIs(t, err, errFault, "Read")
		})
	})

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			path    string
			content string
			local   string
			want    string
		}{
			{
				name:    "appends a local file to a file that is not YAML",
				path:    ".gitignore",
				content: "bin/\n",
				local:   "local/\n",
				want:    "bin/\nlocal/\n",
			},
			{
				name:    "adds a newline before the local file when the rendering lacks one",
				path:    "notes.txt",
				content: "first",
				local:   "second\n",
				want:    "first\nsecond\n",
			},
			{
				name:    "returns the local file for an empty rendering",
				path:    "notes.txt",
				content: "",
				local:   "only\n",
				want:    "only\n",
			},
			{
				name:    "appends a local file without the license header on its first lines",
				path:    "Makefile",
				content: "check:\n",
				local:   "# Copyright Example B.V. 2026\n# SPDX-License-Identifier: MIT\n\nlocal:\n",
				want:    "check:\n\nlocal:\n",
			},
			{
				name:    "appends a local file without a license header whose tag of SPDX comes first",
				path:    "Makefile",
				content: "check:\n",
				local:   "# SPDX-License-Identifier: MIT\n# Copyright Example B.V. 2026\n\nlocal:\n",
				want:    "check:\n\nlocal:\n",
			},
			{
				name:    "appends the lines after a license header that no empty line follows",
				path:    "Makefile",
				content: "check:\n",
				local:   "# SPDX-License-Identifier: MIT\nlocal:\n",
				want:    "check:\nlocal:\n",
			},
			{
				name:    "appends nothing of a local file that is a license header alone",
				path:    "Makefile",
				content: "check:\n",
				local:   "# Copyright Example B.V. 2026\n# SPDX-License-Identifier: MIT",
				want:    "check:\n",
			},
			{
				name:    "appends a first line with a copyright notice and without a tag of SPDX",
				path:    ".gitignore",
				content: "bin/\n",
				local:   "# Copyright notices of the vendored files\nvendor/\n",
				want:    "bin/\n# Copyright notices of the vendored files\nvendor/\n",
			},
			{
				name:    "merges a YAML file without the license header of its local file",
				path:    "tools.yml",
				content: "a: 1\n",
				local:   "# Copyright Example B.V. 2026\n# SPDX-License-Identifier: MIT\n\nb: 2\n",
				want:    "a: 1\nb: 2\n",
			},
			{
				name:    "names the local file in the header of a file that is not YAML",
				path:    "notes.txt",
				content: "# " + notesHeader + "\nfirst\n",
				local:   "second\n",
				want:    "# " + notesMerged + "\nfirst\nsecond\n",
			},
			{
				name:    "names the local file in the header of a merged YAML file",
				path:    "tools.yml",
				content: "# " + toolsHeader + "\na: 1\n",
				local:   "b: 2\n",
				want:    "# " + toolsMerged + "\na: 1\nb: 2\n",
			},
			{
				name:    "leaves a first line without the header unchanged",
				path:    "tools.yaml",
				content: "# notes\na: 1\n",
				local:   "a: 2\n",
				want:    "# notes\na: 2\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := overlay.Apply(tt.path, []byte(tt.content), []byte(tt.local))
				assert.NoError(t, err, "Apply")
				assert.Equal(t, string(got), tt.want, "the merged file")
			})
		}

		t.Run("returns an error that names the local file of a YAML file that does not parse", func(t *testing.T) {
			t.Parallel()
			_, err := overlay.Apply("tools.yml", []byte("a: 1\n"), []byte("a: [\n"))
			assert.HasError(t, err, "Apply")
			assert.Contains(t, err.Error(), "overlay: merge .ergon/local/tools.yml: parse: ", "the error")
		})
	})

	t.Run("IsYAML", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want bool
		}{
			{name: "reports true for the extension yml", give: ".github/workflows/ci.yml", want: true},
			{name: "reports true for the extension yaml", give: ".pre-commit-config.yaml", want: true},
			{name: "reports false for another extension", give: "ruff.toml", want: false},
			{name: "reports false for a name without an extension", give: "Makefile", want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, overlay.IsYAML(tt.give), tt.want, "IsYAML of "+tt.give)
			})
		}
	})
}
