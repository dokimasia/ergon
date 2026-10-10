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
// .tar.xz or a .zip, or takes the asset as the program. It returns the error of releaseProgram, and
// the errors of the download.
func (r *Runner) release(ctx context.Context, rel option.Release, name string) (string, error) {
	program, asset, err := r.releaseProgram(rel, name)
	if err != nil {
		return "", err
	}
	if exists(program) {
		return program, nil
	}
	return program, r.download(ctx, asset, rel.Pin().SHA256[r.Platform], program)
}

// releaseProgram returns the program of the release binary rel of the tool name in the cache, and
// the asset of the platform. The cache keeps the program under the name, the version, the platform
// and the digest, so a pin with another digest installs again. It returns an error that wraps
// [ErrInstall] for a platform without an asset or a digest.
func (r *Runner) releaseProgram(rel option.Release, name string) (string, option.Asset, error) {
	pin := rel.Pin()
	asset, err := rel.Asset(r.Platform)
	if err != nil {
		return "", option.Asset{}, fmt.Errorf("%w: %s %s: %w", ErrInstall, name, pin.Version, err)
	}
	digest, ok := pin.SHA256[r.Platform]
	if !ok {
		return "", option.Asset{}, fmt.Errorf("%w: %s %s has no digest for %s", ErrInstall, name, pin.Version,
			r.Platform)
	}
	base := path.Base(asset.Program)
	if asset.Program == "" {
		base = path.Base(asset.URL)
	}
	return filepath.Join(r.Cache, "release", name, pin.Version, r.platform(), digest, base), asset, nil
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
// target, as [unpack] states. It removes the temporary file. It returns an error that wraps
// [ErrInstall] for a directory that does not create, a request that fails, a status other than
// 200, a content of another digest, and the errors of unpack.
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

// fromTar returns the content of the entry name of the tar archive, which xz compresses for an
// address that ends in .tar.xz, and gzip for any other. It returns an error for an archive that does
// not read, and for an archive without the entry.
func fromTar(archive io.Reader, address, name string) (io.Reader, error) {
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
	t := tar.NewReader(stream)
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
