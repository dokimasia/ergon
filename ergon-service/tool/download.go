// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
	"go.dokimi.dev/ergon/core/option"
)

// The suffixes of the archives of a release binary.
const (
	tarGz  = ".tar.gz"
	tarXz  = ".tar.xz"
	zipped = ".zip"
)

// central is the address of the repository of Maven Central.
const central = "https://repo1.maven.org/maven2/"

// The modes of the programs and the directories of the cache.
const (
	programPerm fs.FileMode = 0o755
	dirPerm     fs.FileMode = 0o755
)

// release installs the release binary rel of the tool name into the cache, unless the cache has it,
// and returns its program, which [Runner.releaseProgram] names. It downloads the asset of the
// platform, checks it against the digest of the pin, and unpacks the program from a .tar.gz, a
// .tar.xz or a .zip, or takes the asset as the program. It unpacks the whole archive of an asset
// with Tree. It returns the error of releaseProgram, and the errors of the download.
func (r *Runner) release(ctx context.Context, rel option.Release, name string) (string, error) {
	program, install, asset, err := r.releaseProgram(rel, name)
	if err != nil {
		return "", err
	}
	if exists(program) {
		return program, nil
	}
	return program, r.download(ctx, asset, rel.Pin().SHA256[r.Platform], install)
}

// releaseProgram returns the program of the release binary rel of the tool name in the cache, its
// install, and the asset of the platform. The install is the program, or the directory of the
// whole archive of an asset with Tree, which has the program at its path in the archive. The cache
// keeps the install under the name, the version, the platform and the digest, so a pin with
// another digest installs again. It returns an error that wraps [ErrInstall] for a platform without
// an asset or a digest.
func (r *Runner) releaseProgram(rel option.Release, name string) (string, string, option.Asset, error) {
	pin := rel.Pin()
	asset, err := rel.Asset(r.Platform)
	if err != nil {
		return "", "", option.Asset{}, fmt.Errorf("%w: %s %s: %w", ErrInstall, name, pin.Version, err)
	}
	digest, ok := pin.SHA256[r.Platform]
	if !ok {
		return "", "", option.Asset{}, fmt.Errorf("%w: %s %s has no digest for %s", ErrInstall, name,
			pin.Version, r.Platform)
	}
	dir := filepath.Join(r.Cache, "release", name, pin.Version, r.platform(), digest)
	if asset.Tree {
		return filepath.Join(dir, filepath.FromSlash(asset.Program)), dir, asset, nil
	}
	base := path.Base(asset.Program)
	if asset.Program == "" {
		base = path.Base(asset.URL)
	}
	program := filepath.Join(dir, base)
	return program, program, asset, nil
}

// maven installs the jar of the Maven artifact m, with the classifier, into the cache unless the
// cache has it, and returns java and the arguments that run it. It checks the jar against the
// .sha256 file beside it in Maven Central. It returns the errors of the download.
func (r *Runner) maven(ctx context.Context, m option.Maven, classifier string) (string, []string, error) {
	address, jar := r.mavenJar(m, classifier)
	if exists(jar) {
		return "java", []string{"-jar", jar}, nil
	}
	digest, err := r.digest(ctx, address+".sha256")
	if err != nil {
		return "", nil, err
	}
	if err := r.download(ctx, option.Asset{URL: address}, digest, jar); err != nil {
		return "", nil, err
	}
	return "java", []string{"-jar", jar}, nil
}

// mavenJar returns the address of the jar of the Maven artifact m with the classifier in Maven
// Central, and the path of the jar in the cache, under the group, the artifact and the version. An
// empty classifier names the jar of the artifact without a classifier.
func (r *Runner) mavenJar(m option.Maven, classifier string) (string, string) {
	group, artifact, _ := strings.Cut(m.Package(), ":")
	file := artifact + "-" + m.Version()
	if classifier != "" {
		file += "-" + classifier
	}
	file += ".jar"
	address := central + strings.ReplaceAll(group, ".", "/") + "/" + artifact + "/" + m.Version() + "/" + file
	return address, filepath.Join(r.Cache, "maven", group, artifact, m.Version(), file)
}

// download downloads the asset into a temporary file of the directory of target, checks it
// against digest, the SHA-256 of the asset in lowercase hexadecimal, and writes its program to
// target, as [unpack] states. For an asset with Tree, it writes the whole archive to the directory
// target, as [unpackTree] states. It removes the temporary file. It returns an error that wraps
// [ErrInstall] for a directory that does not create, a request that fails, a status other than
// 200, a content of another digest, and the errors of unpack and unpackTree.
func (r *Runner) download(ctx context.Context, asset option.Asset, digest, target string) error {
	body, err := r.get(ctx, asset.URL)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()
	dir := filepath.Dir(target)
	err = os.MkdirAll(dir, dirPerm)
	var archive *os.File
	if err == nil {
		archive, err = os.CreateTemp(dir, ".download-*")
	}
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInstall, asset.URL, err)
	}
	defer func() { _ = archive.Close(); _ = os.Remove(archive.Name()) }()
	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(archive, h), body)
	if got := hex.EncodeToString(h.Sum(nil)); err == nil && got != digest {
		err = fmt.Errorf("the digest %s, where the pin states %s", got, digest)
	}
	if err == nil {
		_, err = archive.Seek(0, io.SeekStart)
	}
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInstall, asset.URL, err)
	}
	if asset.Tree {
		return unpackTree(archive, size, asset, target)
	}
	return unpack(archive, size, asset, target)
}

// digest returns the digest that the file at address states: the first word of its content, which
// is a SHA-256 in hexadecimal, such as the .sha256 file of a jar of Maven Central. It returns an
// error that wraps [ErrInstall] for a request that fails, a status other than 200, and a file whose
// first word is no SHA-256.
func (r *Runner) digest(ctx context.Context, address string) (string, error) {
	body, err := r.get(ctx, address)
	if err != nil {
		return "", err
	}
	defer func() { _ = body.Close() }()
	content, err := io.ReadAll(io.LimitReader(body, 1024))
	words := strings.Fields(string(content))
	if err != nil || len(words) == 0 || len(words[0]) != hex.EncodedLen(sha256.Size) {
		return "", fmt.Errorf("%w: %s states no digest", ErrInstall, address)
	}
	return strings.ToLower(words[0]), nil
}

// get returns the body of the response to a request of address. It returns an error that wraps
// [ErrInstall] for an address that is no URL, a request that fails, and a status other than 200.
func (r *Runner) get(ctx context.Context, address string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInstall, err)
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInstall, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: %s: %s", ErrInstall, address, resp.Status)
	}
	return resp.Body, nil
}

// unpack writes the program of asset, from archive, the downloaded asset of size bytes, to
// program, with the mode of a program: the entry asset.Program of a .tar.gz, a .tar.xz or a .zip,
// or the whole archive for an asset that is the program. It writes a temporary file of the
// directory of program and renames it. It returns an error that wraps [ErrInstall] for an archive
// that does not read, an archive without the program, and a program that does not write.
func unpack(archive *os.File, size int64, asset option.Asset, program string) error {
	var source io.Reader = archive
	var err error
	if strings.HasSuffix(asset.URL, tarGz) || strings.HasSuffix(asset.URL, tarXz) {
		source, err = fromTar(archive, asset.URL, asset.Program)
	} else if strings.HasSuffix(asset.URL, zipped) {
		source, err = fromZip(archive, size, asset.Program)
	}
	var f *os.File
	if err == nil {
		f, err = os.CreateTemp(filepath.Dir(program), ".program-*")
	}
	if err == nil {
		_, err = io.Copy(f, source)
		err = errors.Join(err, f.Chmod(programPerm), f.Close())
		if err == nil {
			err = os.Rename(f.Name(), program)
		}
		if err != nil {
			_ = os.Remove(f.Name())
		}
	}
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInstall, asset.URL, err)
	}
	return nil
}

// unpackTree writes every entry of archive, the downloaded asset of size bytes, below a temporary
// directory beside dir through an [os.Root], and renames the temporary directory to dir, so a run
// reads a whole tree. The program of asset has the mode of a program, and every other file the
// mode of a file. A rename that fails passes when dir has the program, because another process
// renamed its tree first. It returns an error that wraps [ErrInstall] for an archive that does not
// read, an entry whose path leaves the directory, an entry that is neither a directory nor a
// regular file, a tree without the program, and a tree that does not write or rename.
func unpackTree(archive *os.File, size int64, asset option.Asset, dir string) error {
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".tree-*")
	var root *os.Root
	if err == nil {
		defer func() { _ = os.RemoveAll(tmp) }()
		root, err = os.OpenRoot(tmp)
	}
	if err == nil {
		if strings.HasSuffix(asset.URL, zipped) {
			err = treeFromZip(root, archive, size)
		} else {
			err = treeFromTar(root, archive, asset.URL)
		}
		if err == nil {
			err = root.Chmod(asset.Program, programPerm)
		}
		err = errors.Join(err, root.Close())
	}
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInstall, asset.URL, err)
	}
	if err := os.Rename(tmp, dir); err != nil && !exists(filepath.Join(dir, filepath.FromSlash(asset.Program))) {
		return fmt.Errorf("%w: %s: %w", ErrInstall, asset.URL, err)
	}
	return nil
}

// treeFromTar writes each entry of the tar archive below root, as [tarReader] reads the archive
// for address. It returns an error for an archive that does not read, an entry outside root, an
// entry that is neither a directory nor a regular file, and a file that does not write.
func treeFromTar(root *os.Root, archive io.Reader, address string) error {
	t, err := tarReader(archive, address)
	if err != nil {
		return err
	}
	for {
		h, err := t.Next()
		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return fmt.Errorf("read the archive: %w", err)
		case h.Typeflag == tar.TypeDir:
			err = root.MkdirAll(h.Name, dirPerm)
		case h.Typeflag == tar.TypeReg:
			err = writeEntry(root, h.Name, t)
		default:
			err = fmt.Errorf("the entry %s is neither a directory nor a regular file", h.Name)
		}
		if err != nil {
			return err
		}
	}
}

// treeFromZip writes each entry of the zip archive of size bytes below root. It returns an error
// for an archive that does not read, an entry outside root, an entry that is neither a directory
// nor a regular file, and a file that does not write.
func treeFromZip(root *os.Root, archive io.ReaderAt, size int64) error {
	z, err := zip.NewReader(archive, size)
	if err != nil {
		return fmt.Errorf("read the archive: %w", err)
	}
	for _, f := range z.File {
		switch mode := f.Mode(); {
		case mode.IsDir():
			err = root.MkdirAll(f.Name, dirPerm)
		case mode.IsRegular():
			var content io.ReadCloser
			if content, err = f.Open(); err == nil {
				err = errors.Join(writeEntry(root, f.Name, content), content.Close())
			}
		default:
			err = fmt.Errorf("the entry %s is neither a directory nor a regular file", f.Name)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// writeEntry creates the directories of the path of the file name below root, and writes content
// to the file with the mode of a file. It returns an error for a name outside root, a directory of
// the path that is a file, and a file that does not write.
func writeEntry(root *os.Root, name string, content io.Reader) error {
	err := root.MkdirAll(path.Dir(name), dirPerm)
	var f *os.File
	if err == nil {
		f, err = root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	}
	if err == nil {
		_, err = io.Copy(f, content)
		err = errors.Join(err, f.Close())
	}
	return err
}

// fromTar returns the content of the entry name of the tar archive, as [tarReader] reads the
// archive for address. It returns an error for an archive that does not read, and for an archive
// without the entry.
func fromTar(archive io.Reader, address, name string) (io.Reader, error) {
	t, err := tarReader(archive, address)
	if err != nil {
		return nil, err
	}
	for {
		h, err := t.Next()
		if err != nil {
			return nil, fmt.Errorf("find %s in the archive: %w", name, err)
		}
		if path.Clean(h.Name) == name {
			return t, nil
		}
	}
}

// tarReader returns a reader of the tar archive, which xz compresses for an address that ends in
// .tar.xz, and gzip for any other. It returns an error for an archive that does not read.
func tarReader(archive io.Reader, address string) (*tar.Reader, error) {
	var stream io.Reader
	var err error
	if strings.HasSuffix(address, tarXz) {
		stream, err = xz.NewReader(archive)
	} else {
		stream, err = gzip.NewReader(archive)
	}
	if err != nil {
		return nil, fmt.Errorf("read the archive: %w", err)
	}
	return tar.NewReader(stream), nil
}

// fromZip returns the content of the entry name of the zip archive of size bytes. It returns an
// error for an archive that does not read, and for an archive without the entry.
func fromZip(archive io.ReaderAt, size int64, name string) (io.Reader, error) {
	z, err := zip.NewReader(archive, size)
	if err != nil {
		return nil, fmt.Errorf("read the archive: %w", err)
	}
	f, err := z.Open(name)
	if err != nil {
		return nil, fmt.Errorf("find %s in the archive: %w", name, err)
	}
	return f, nil
}
