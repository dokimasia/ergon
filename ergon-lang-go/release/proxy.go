// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
)

// The modes of the files and the directories that Apply creates, before the umask.
const (
	filePerm fs.FileMode = 0o644
	dirPerm  fs.FileMode = 0o755
)

// proxy is a module proxy in a directory, in the layout that GOPROXY=file:// reads: for each module,
// the file @v/list with its versions, and the files .info, .mod and .zip of each version.
type proxy struct {
	// versions are the versions of each module path that the proxy serves, in the order of add.
	versions map[string][]string

	// dir is the directory of the proxy.
	dir string
}

// url returns the file URL of p, which GOPROXY takes, with a slash before the volume of a
// directory of Windows, as in file:///C:/proxy.
func (p *proxy) url() string {
	dir := "/" + strings.TrimPrefix(filepath.ToSlash(p.dir), "/")
	return (&url.URL{Scheme: "file", Path: dir}).String()
}

// add writes the version m of a module into p: data, the zip of the module, mod, its go.mod, an
// .info that states the version, and the list of the versions of the module. m is a version that
// golang.org/x/mod/zip wrote data for, so its path and its version escape. It returns the error of
// the file system.
func (p *proxy) add(m module.Version, data, mod []byte) error {
	// zip.Create checked the path and the version of m, and a struct of one string encodes.
	escaped, _ := module.EscapePath(m.Path)
	file, _ := module.EscapeVersion(m.Version)
	info, _ := json.Marshal(&struct {
		Version string `json:"Version"`
	}{m.Version})
	p.versions[m.Path] = append(p.versions[m.Path], m.Version)
	list := strings.Join(p.versions[m.Path], "\n") + "\n"
	dir := filepath.Join(p.dir, filepath.FromSlash(escaped), "@v")
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("release: serve %s: %w", m, err)
	}
	files := []struct {
		name string
		data []byte
	}{{file + ".zip", data}, {file + ".mod", mod}, {file + ".info", info}, {"list", []byte(list)}}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.name), f.data, filePerm); err != nil {
			return fmt.Errorf("release: serve %s: %w", m, err)
		}
	}
	return nil
}
