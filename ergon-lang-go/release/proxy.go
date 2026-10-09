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

// The modes of the files and the directories that Apply and Pack create, before the umask.
const (
	filePerm fs.FileMode = 0o644
	dirPerm  fs.FileMode = 0o755
)

// The extensions of the files of a module version in a proxy.
const (
	zipExt  = ".zip"
	modExt  = ".mod"
	infoExt = ".info"
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
	// A struct of one string encodes.
	info, _ := json.Marshal(&struct {
		Version string `json:"Version"`
	}{m.Version})
	p.versions[m.Path] = append(p.versions[m.Path], m.Version)
	list := strings.Join(p.versions[m.Path], "\n") + "\n"
	zipFile := p.file(m, zipExt)
	dir := filepath.Dir(zipFile)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("release: serve %s: %w", m, err)
	}
	files := []struct {
		name string
		data []byte
	}{{zipFile, data}, {p.file(m, modExt), mod}, {p.file(m, infoExt), info}, {filepath.Join(dir, "list"), []byte(list)}}
	for _, f := range files {
		if err := os.WriteFile(f.name, f.data, filePerm); err != nil {
			return fmt.Errorf("release: serve %s: %w", m, err)
		}
	}
	return nil
}

// file returns the path of the file of the version m in p with the extension ext, such as .zip. m
// is a version that golang.org/x/mod/zip wrote a zip for, so its path and its version escape.
func (p *proxy) file(m module.Version, ext string) string {
	// zip.Create checked the path and the version of m.
	escaped, _ := module.EscapePath(m.Path)
	version, _ := module.EscapeVersion(m.Version)
	return filepath.Join(p.dir, filepath.FromSlash(escaped), "@v", version+ext)
}
