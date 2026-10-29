// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/service/forge"
)

// ErrVerify is the error of [Sign] for a draft that fails a check. Sign returns it for:
//
//   - a draft without its tag or without the cosign bundle of its checksums
//   - a bundle that the [Witness] rejects
//   - an asset whose digest differs from the checksums, and a zip without a line
//   - a GPG signature of another key or of an algorithm that the Terraform Registry refuses
var ErrVerify = errors.New("release: verification failed")

// ErrNoDraft is the error of [Sign] for a tag of [Signing].Tags without a draft that waits for a
// signature.
var ErrNoDraft = errors.New("release: no draft waits for a signature")

// The names of the files of a release that the Terraform Registry reads, by their suffixes.
const (
	// sumsSuffix ends the name of the checksums of a release, <name>_<version>_SHA256SUMS.
	sumsSuffix = "_SHA256SUMS"

	// bundleSuffix ends the name of the cosign bundle of a file.
	bundleSuffix = ".sigstore.json"

	// signatureSuffix ends the name of the binary GPG signature of a file.
	signatureSuffix = ".sig"

	// zipSuffix ends the name of an archive of a provider.
	zipSuffix = ".zip"
)

// versionPrefix starts the tag of a release that the Terraform Registry reads, v<version>.
const versionPrefix = "v"

// releaseWorkflow is the file of the release workflow of a repository, whose job pack signs the
// checksums of each release with cosign.
const releaseWorkflow = ".github/workflows/release.yml"

// registryAlgorithms are the public key algorithms of a GPG signature that the Terraform Registry
// accepts, by their numbers in RFC 9580: RSA, RSA for signatures alone, and DSA.
var registryAlgorithms = []int{1, 3, 17}

// SignForge is the host of the drafts that [Sign] completes. *forge.Client implements it. Every
// method takes the repository as owner/name and returns the error of the host.
type SignForge interface {
	// Drafts returns the draft releases of repo.
	Drafts(ctx context.Context, repo string) ([]forge.Release, error)

	// Tag returns the commit of the tag name of repo, and reports whether repo has the tag.
	Tag(ctx context.Context, repo, name string) (string, bool, error)

	// DownloadAsset returns the content of the asset id of repo.
	DownloadAsset(ctx context.Context, repo string, id int64) ([]byte, error)

	// UploadAsset uploads the file at path as an asset of the release r.
	UploadAsset(ctx context.Context, r *forge.Release, path string) error

	// Publish publishes the draft release id of repo.
	Publish(ctx context.Context, repo string, id int64) error
}

// Workflow is the run of a workflow that signed a cosign bundle. It has the repository on a server,
// the file of the workflow, the branch of its push and the commit of the run.
type Workflow struct {
	// Server is the address of GitHub, such as https://github.com.
	Server string

	// Repo is the repository, as owner/name.
	Repo string

	// File is the path of the workflow in the repository, such as .github/workflows/release.yml.
	File string

	// Branch is the branch whose push starts the run, such as main.
	Branch string

	// Commit is the commit of the run.
	Commit string
}

// Witness verifies the cosign bundle of a file. [Cosign] implements it.
type Witness interface {
	// Verify returns nil when bundle, a path, is a cosign signature of the file at path by the run
	// of w. It returns an error that wraps [ErrVerify] for a bundle that it rejects.
	Verify(ctx context.Context, path, bundle string, w *Workflow) error
}

// Signature is the key of a GPG signature that gpg verified.
type Signature struct {
	// Fingerprint is the fingerprint of the key that made the signature, in hexadecimal.
	Fingerprint string

	// Primary is the fingerprint of the primary key of that key, in hexadecimal, or empty when gpg
	// names none.
	Primary string

	// Algorithm is the number of the public key algorithm of the key, such as 1 for RSA.
	Algorithm int
}

// Signer signs files with the GPG key of a person and verifies a signature. [GPG] implements it.
type Signer interface {
	// Sign writes the binary detached signature of the file at path, made with the key of the
	// fingerprint key, to the path signature.
	Sign(ctx context.Context, key, path, signature string) error

	// Verify verifies the detached signature at signature of the file at path, and returns its key.
	// It returns an error that wraps [ErrVerify] for a signature that it cannot verify.
	Verify(ctx context.Context, path, signature string) (Signature, error)
}

// Signing configures a run of [Sign] with the host of the drafts, the checks of their checksums,
// the key that signs them and the directory of the files of the run.
type Signing struct {
	// Forge is the host of the drafts.
	Forge SignForge

	// Witness verifies the cosign bundle of the checksums of a draft.
	Witness Witness

	// Signer signs the checksums of a draft and verifies the signature.
	Signer Signer

	// Repo is the repository of the drafts, as owner/name.
	Repo string

	// Server is the address of GitHub, such as https://github.com, which the identity of the release
	// workflow names.
	Server string

	// Branch is the base branch of the repository, on whose pushes the release workflow runs.
	Branch string

	// Key is the fingerprint of the GPG key that signs the checksums, in hexadecimal.
	Key string

	// Dir is the directory of the downloads and the signatures of the run, which exists.
	Dir string

	// Tags are the tags of the drafts to complete, or empty for every draft that waits for a
	// signature.
	Tags []string
}

// Sign completes the drafts of the repository of s that wait for a signature, or the drafts of
// s.Tags, in the order of the host, and returns the tags of the releases that it published. A draft
// waits for a signature when its tag is v<version> and its assets include <name>_<version>_SHA256SUMS,
// the checksums of a provider of the Terraform Registry. For each such draft, Sign:
//
//   - downloads the checksums and their cosign bundle <checksums>.sigstore.json into s.Dir
//   - verifies the bundle with s.Witness against the run of .github/workflows/release.yml on a push
//     to s.Branch at the commit of the tag
//   - checks each line of the checksums against the asset of the draft of the same name, whose
//     SHA-256 on the host must be the digest of the line, and checks that each zip of the draft has
//     a line
//   - signs the checksums with the key s.Key into <checksums>.sig, unless the draft has that
//     signature, which it downloads in its place
//   - verifies the signature with s.Signer, whose key is s.Key or has s.Key as its primary key, and
//     whose algorithm is RSA or DSA
//   - uploads the signature, unless the draft has it, and publishes the draft
//
// It returns an error that wraps [ErrNoDraft] for a tag of s.Tags without a draft that waits for a
// signature, before it signs anything. It returns an error that wraps [ErrVerify] for a draft that
// fails a check, before it uploads or publishes anything of that draft. It returns the errors of
// s.Forge, of s.Witness, of s.Signer and of writing a download. On an error it also returns the tags
// that it published before the error.
func Sign(ctx context.Context, s *Signing) ([]string, error) {
	drafts, err := s.Forge.Drafts(ctx, s.Repo)
	if err != nil {
		return nil, err
	}
	var waiting []*forge.Release
	var checksums []*forge.Asset
	for k := range drafts {
		d := &drafts[k]
		if len(s.Tags) > 0 && !slices.Contains(s.Tags, d.Tag) {
			continue
		}
		if sums := checksumsOf(d); sums != nil {
			waiting = append(waiting, d)
			checksums = append(checksums, sums)
		}
	}
	var missing []string
	for _, tag := range s.Tags {
		if !slices.ContainsFunc(waiting, func(d *forge.Release) bool { return d.Tag == tag }) {
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoDraft, strings.Join(missing, ", "))
	}
	published := make([]string, 0, len(waiting))
	for k, d := range waiting {
		if err := s.complete(ctx, d, checksums[k]); err != nil {
			return published, err
		}
		published = append(published, d.Tag)
	}
	return published, nil
}

// checksumsOf returns the checksums <name>_<version>_SHA256SUMS of the draft d, and nil for a draft
// whose tag is not v<version> or that has no such asset.
func checksumsOf(d *forge.Release) *forge.Asset {
	v, ok := strings.CutPrefix(d.Tag, versionPrefix)
	if !ok {
		return nil
	}
	if _, err := version.Parse(v); err != nil {
		return nil
	}
	for k := range d.Assets {
		if strings.HasSuffix(d.Assets[k].Name, "_"+v+sumsSuffix) {
			return &d.Assets[k]
		}
	}
	return nil
}

// complete completes the draft d with its checksums sums by the steps of [Sign].
func (s *Signing) complete(ctx context.Context, d *forge.Release, sums *forge.Asset) error {
	commit, found, err := s.Forge.Tag(ctx, s.Repo, d.Tag)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: the draft %s has no tag", ErrVerify, d.Tag)
	}
	bundle := assetOf(d, sums.Name+bundleSuffix)
	if bundle == nil {
		return fmt.Errorf("%w: the draft %s has no %s", ErrVerify, d.Tag, sums.Name+bundleSuffix)
	}
	sumsPath, err := s.download(ctx, sums)
	if err != nil {
		return err
	}
	bundlePath, err := s.download(ctx, bundle)
	if err != nil {
		return err
	}
	w := Workflow{Server: s.Server, Repo: s.Repo, File: releaseWorkflow, Branch: s.Branch, Commit: commit}
	err = s.Witness.Verify(ctx, sumsPath, bundlePath, &w)
	if err != nil {
		return err
	}
	err = listed(d, sumsPath)
	if err != nil {
		return err
	}
	signature := assetOf(d, sums.Name+signatureSuffix)
	signaturePath := filepath.Join(s.Dir, sums.Name+signatureSuffix)
	if signature != nil {
		_, err = s.download(ctx, signature)
	} else {
		err = s.Signer.Sign(ctx, s.Key, sumsPath, signaturePath)
	}
	if err != nil {
		return err
	}
	got, err := s.Signer.Verify(ctx, sumsPath, signaturePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got.Fingerprint, s.Key) && !strings.EqualFold(got.Primary, s.Key) {
		return fmt.Errorf("%w: the key %s signed %s, and not %s", ErrVerify, got.Fingerprint, sums.Name, s.Key)
	}
	if !slices.Contains(registryAlgorithms, got.Algorithm) {
		return fmt.Errorf("%w: the key of the signature of %s has the algorithm %d, which is not RSA or DSA",
			ErrVerify, sums.Name, got.Algorithm)
	}
	if signature == nil {
		if err := s.Forge.UploadAsset(ctx, d, signaturePath); err != nil {
			return err
		}
	}
	return s.Forge.Publish(ctx, s.Repo, d.ID)
}

// download writes the content of the asset a into s.Dir under its name, and returns the path of
// the file. It returns the error of the host and of the write.
func (s *Signing) download(ctx context.Context, a *forge.Asset) (string, error) {
	data, err := s.Forge.DownloadAsset(ctx, s.Repo, a.ID)
	if err != nil {
		return "", err
	}
	file := filepath.Join(s.Dir, a.Name)
	if err := os.WriteFile(file, data, filePerm); err != nil {
		return "", fmt.Errorf("release: write the download of %s: %w", a.Name, err)
	}
	return file, nil
}

// listed checks the checksums at sums against the assets of the draft d. Each line of the checksums
// has a digest and the name of an asset of d, whose SHA-256 on the host is the digest, and each zip
// of d has a line. It returns an error that wraps [ErrVerify] when a check fails, such as for
// checksums without a line or for a line that is not a digest and a name. It also returns the error
// of reading sums.
func listed(d *forge.Release, sums string) error {
	data, err := os.ReadFile(sums)
	if err != nil {
		return fmt.Errorf("release: read %s: %w", filepath.Base(sums), err)
	}
	var names []string
	for line := range strings.Lines(string(data)) {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("%w: the line %q of %s, which is not a digest and a name", ErrVerify,
				strings.TrimSpace(line), filepath.Base(sums))
		}
		digest, name := fields[0], strings.TrimPrefix(fields[1], "*")
		a := assetOf(d, name)
		switch {
		case a == nil:
			return fmt.Errorf("%w: %s lists %s, which the draft %s lacks", ErrVerify, filepath.Base(sums), name, d.Tag)
		case !strings.EqualFold(a.Digest, digest):
			return fmt.Errorf("%w: the asset %s of the draft %s has the SHA-256 %q, and %s lists %s", ErrVerify, name,
				d.Tag, a.Digest, filepath.Base(sums), digest)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return fmt.Errorf("%w: %s lists no file", ErrVerify, filepath.Base(sums))
	}
	for _, a := range d.Assets {
		if strings.HasSuffix(a.Name, zipSuffix) && !slices.Contains(names, a.Name) {
			return fmt.Errorf("%w: %s does not list the zip %s of the draft %s", ErrVerify, filepath.Base(sums), a.Name,
				d.Tag)
		}
	}
	return nil
}

// assetOf returns the asset of the draft d named name, and nil for a draft without it.
func assetOf(d *forge.Release, name string) *forge.Asset {
	i := slices.IndexFunc(d.Assets, func(a forge.Asset) bool { return a.Name == name })
	if i < 0 {
		return nil
	}
	return &d.Assets[i]
}
