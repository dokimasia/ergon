// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/release"
)

// The repository of the provider of the cases, and the name of its checksums at 1.2.0.
const (
	providerRepo = "example/terraform-provider-demo"
	providerSums = "terraform-provider-demo_1.2.0_SHA256SUMS"
)

// The identifiers of the assets of the first draft of the cases. The assets of each later draft
// have the identifiers of the constants plus 100 for each draft before it.
const (
	linuxZip int64 = iota + 1
	darwinZip
	manifestAsset
	sumsAsset
	bundleAsset
	sbomAsset
	signatureAsset
)

// The errors of a witness and a signer that fail. errHost is the error of a host that fails.
var (
	errWitness = fmt.Errorf("%w: the bundle names another workflow", release.ErrVerify)
	errSigner  = errors.New("gpg failed")
)

// signForge is a [release.SignForge] with the drafts of drafts, the commits of the tags of tags and
// the content of each asset in contents, which records each upload and each publish. Each field fail
// is the error of the method that it names.
type signForge struct {
	failDrafts, failTag, failDownload, failUpload, failPublish error

	// tags are the commits of the tags of the repository.
	tags map[string]string

	// contents are the contents of the assets, by their identifiers.
	contents map[int64]string

	// drafts are the drafts of the repository.
	drafts []forge.Release

	// calls are the uploads, each with the content of its file, and the publishes, each as a line.
	calls []string
}

var _ release.SignForge = (*signForge)(nil)

// Drafts returns a copy of the drafts of f for the repository of the cases.
func (f *signForge) Drafts(_ context.Context, repo string) ([]forge.Release, error) {
	if f.failDrafts != nil || repo != providerRepo {
		return nil, cmp.Or(f.failDrafts, errHost)
	}
	return slices.Clone(f.drafts), nil
}

// Tag returns the commit of the tag name, and reports whether f has it.
func (f *signForge) Tag(_ context.Context, _, name string) (string, bool, error) {
	commit, ok := f.tags[name]
	return commit, ok, f.failTag
}

// DownloadAsset returns the content of the asset id, and errHost for an asset without a content.
func (f *signForge) DownloadAsset(_ context.Context, _ string, id int64) ([]byte, error) {
	content, ok := f.contents[id]
	if f.failDownload != nil || !ok {
		return nil, cmp.Or(f.failDownload, errHost)
	}
	return []byte(content), nil
}

// UploadAsset records the upload of the file at path to the release r, with the content of the
// file.
func (f *signForge) UploadAsset(_ context.Context, r *forge.Release, path string) error {
	if f.failUpload != nil {
		return f.failUpload
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f.calls = append(f.calls, "upload "+r.Tag+" "+filepath.Base(path)+" "+string(data))
	return nil
}

// Publish records the publish of the release id with its mark of the latest release.
func (f *signForge) Publish(_ context.Context, _ string, id int64, latest forge.Latest) error {
	if f.failPublish != nil {
		return f.failPublish
	}
	f.calls = append(f.calls, "publish "+strconv.FormatInt(id, 10)+" make_latest="+string(latest))
	return nil
}

// witness is a [release.Witness] that records the runs that it checks, and returns fail.
type witness struct {
	// fail is the error of Verify, or nil.
	fail error

	// runs are the runs that Verify checked, in their order.
	runs []release.Workflow

	// remove makes Verify remove the file that it checks, as another process could.
	remove bool
}

var _ release.Witness = (*witness)(nil)

// Verify records the run of w, removes the file at path for v.remove, and returns v.fail.
func (v *witness) Verify(_ context.Context, path, _ string, w *release.Workflow) error {
	v.runs = append(v.runs, *w)
	if v.remove {
		return os.Remove(path)
	}
	return v.fail
}

// signer is a [release.Signer] whose Sign writes "signature of <file>" and records the key, and
// whose Verify returns key. Each field fail is the error of the method that it names.
type signer struct {
	failSign, failVerify error

	// key is the key that Verify returns.
	key release.Signature

	// signed are the keys and the base names of the files that Sign signed, each as a line.
	signed []string
}

var _ release.Signer = (*signer)(nil)

// Sign writes the signature of the file at path and records it.
func (s *signer) Sign(_ context.Context, key, path, signature string) error {
	if s.failSign != nil {
		return s.failSign
	}
	s.signed = append(s.signed, key+" "+filepath.Base(path))
	return os.WriteFile(signature, []byte("signature of "+filepath.Base(path)), 0o600)
}

// Verify returns s.key for a signature that exists.
func (s *signer) Verify(_ context.Context, _, signature string) (release.Signature, error) {
	if s.failVerify != nil {
		return release.Signature{}, s.failVerify
	}
	if _, err := os.Stat(signature); err != nil {
		return release.Signature{}, err
	}
	return s.key, nil
}

// providerHost returns the host of the drafts of the provider of the cases, one for each version of
// versions, with the IDs 7, 6 and so on, each at the tag of commitA. Each draft has two zips, a
// manifest, an SBOM, its checksums with the digests of the zips and the manifest, and the cosign
// bundle of the checksums.
func providerHost(versions ...string) *signForge {
	f := &signForge{tags: map[string]string{}, contents: map[int64]string{}}
	for k, v := range versions {
		base := int64(100 * k)
		prefix := "terraform-provider-demo_" + v + "_"
		names := map[int64]string{
			linuxZip:      prefix + "linux_amd64.zip",
			darwinZip:     prefix + "darwin_arm64.zip",
			manifestAsset: prefix + "manifest.json",
			sumsAsset:     prefix + "SHA256SUMS",
			bundleAsset:   prefix + "SHA256SUMS.sigstore.json",
			sbomAsset:     prefix + "linux_amd64.zip.sbom.json",
		}
		contents := map[int64]string{
			linuxZip:      "the zip of linux at " + v,
			darwinZip:     "the zip of darwin at " + v,
			manifestAsset: `{"version":1}`,
			bundleAsset:   "the bundle of " + v,
			sbomAsset:     "the SBOM of " + v,
		}
		var sums strings.Builder
		for _, id := range []int64{darwinZip, linuxZip, manifestAsset} {
			fmt.Fprintf(&sums, "%s  %s\n", digestOf(contents[id]), names[id])
		}
		contents[sumsAsset] = sums.String()
		draft := forge.Release{Tag: "v" + v, ID: int64(7 - k), Draft: true}
		for _, id := range []int64{linuxZip, darwinZip, manifestAsset, sumsAsset, bundleAsset, sbomAsset} {
			asset := forge.Asset{Name: names[id], ID: base + id, Digest: digestOf(contents[id])}
			draft.Assets = append(draft.Assets, asset)
			f.contents[base+id] = contents[id]
		}
		f.drafts = append(f.drafts, draft)
		f.tags["v"+v] = commitA
	}
	return f
}

// digestOf returns the SHA-256 of content in lowercase hexadecimal.
func digestOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// signingOf returns the run of Sign of the provider of the cases with f, w and s, the key
// signingKey and a directory of the test tb.
func signingOf(tb testing.TB, f *signForge, w *witness, s *signer) *release.Signing {
	tb.Helper()
	return &release.Signing{
		Forge: f, Witness: w, Signer: s, Repo: providerRepo, Server: "https://github.com", Branch: "main",
		Key: signingKey, Dir: tb.TempDir(),
	}
}

func TestSign(t *testing.T) {
	t.Parallel()

	t.Run("Sign", func(t *testing.T) {
		t.Parallel()

		t.Run("signs the checksums of a draft, uploads the signature and publishes the draft", func(t *testing.T) {
			t.Parallel()
			f, w := providerHost("1.2.0"), &witness{}
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			got, err := release.Sign(t.Context(), signingOf(t, f, w, s))
			assert.NoError(t, err, "Sign")
			expect.Equal(t, got, []string{"v1.2.0"}, "the published tags")
			expect.Equal(t, w.runs, []release.Workflow{releaseRun}, "the run of the bundle")
			expect.Equal(t, s.signed, []string{signingKey + " " + providerSums}, "the signatures")
			expect.Equal(t, f.calls, []string{
				"upload v1.2.0 " + providerSums + ".sig signature of " + providerSums,
				"publish 7 make_latest=true",
			}, "the calls of the host")
		})

		t.Run("completes the drafts in ascending order of their versions", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0", "1.1.0")
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			got, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
			assert.NoError(t, err, "Sign")
			expect.Equal(t, got, []string{"v1.1.0", "v1.2.0"}, "the published tags")
			sums := "terraform-provider-demo_1.1.0_SHA256SUMS"
			expect.Equal(t, f.calls, []string{
				"upload v1.1.0 " + sums + ".sig signature of " + sums,
				"publish 6 make_latest=true",
				"upload v1.2.0 " + providerSums + ".sig signature of " + providerSums,
				"publish 7 make_latest=true",
			}, "the calls of the host")
		})

		t.Run("does not mark a prerelease as the latest release", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.3.0-rc.1")
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			_, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
			assert.NoError(t, err, "Sign")
			sums := "terraform-provider-demo_1.3.0-rc.1_SHA256SUMS"
			assert.Equal(t, f.calls, []string{
				"upload v1.3.0-rc.1 " + sums + ".sig signature of " + sums,
				"publish 7 make_latest=false",
			}, "the calls of the host")
		})

		t.Run("publishes a draft with a signature whose primary key is the key", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0")
			key := release.Signature{Fingerprint: primaryKey, Primary: strings.ToLower(signingKey), Algorithm: 17}
			s := &signer{key: key}
			_, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
			assert.NoError(t, err, "Sign")
			assert.Equal(t, f.calls[len(f.calls)-1], "publish 7 make_latest=true", "the last call of the host")
		})

		t.Run("publishes a draft that has its signature after it verifies the signature", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0")
			signature := forge.Asset{Name: providerSums + ".sig", ID: signatureAsset}
			f.drafts[0].Assets = append(f.drafts[0].Assets, signature)
			f.contents[signatureAsset] = "an earlier signature"
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			got, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
			assert.NoError(t, err, "Sign")
			expect.Equal(t, got, []string{"v1.2.0"}, "the published tags")
			expect.Empty(t, s.signed, "the signatures")
			expect.Equal(t, f.calls, []string{"publish 7 make_latest=true"}, "the calls of the host")
		})

		t.Run("leaves a draft without the checksums of the version of its tag", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0")
			f.drafts = append(f.drafts,
				forge.Release{Tag: "v2.0.0", ID: 8, Draft: true, Assets: []forge.Asset{{Name: "notes.txt"}}},
				forge.Release{Tag: "lint/v1.0.0", ID: 9, Draft: true, Assets: []forge.Asset{{Name: providerSums}}},
				forge.Release{Tag: "vnext", ID: 10, Draft: true, Assets: []forge.Asset{{Name: "x_next_SHA256SUMS"}}},
			)
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			got, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
			assert.NoError(t, err, "Sign")
			assert.Equal(t, got, []string{"v1.2.0"}, "the published tags")
		})

		t.Run("completes the drafts of Tags alone", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0", "1.1.0")
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			run := signingOf(t, f, &witness{}, s)
			run.Tags = []string{"v1.1.0"}
			got, err := release.Sign(t.Context(), run)
			assert.NoError(t, err, "Sign")
			expect.Equal(t, got, []string{"v1.1.0"}, "the published tags")
			expect.Equal(t, f.calls[len(f.calls)-1], "publish 6 make_latest=true", "the last call of the host")
			expect.Length(t, f.calls, 2, "the calls of the host")
		})

		t.Run("returns ErrNoDraft for a tag without a draft that waits before it signs", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0")
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			run := signingOf(t, f, &witness{}, s)
			run.Tags = []string{"v1.2.0", "v9.0.0"}
			got, err := release.Sign(t.Context(), run)
			assert.ErrorIs(t, err, release.ErrNoDraft, "Sign")
			expect.Contains(t, err.Error(), "no draft waits for a signature: v9.0.0", "the error")
			expect.Empty(t, got, "the published tags")
			expect.Empty(t, s.signed, "the signatures")
			expect.Empty(t, f.calls, "the calls of the host")
		})

		checks := []struct {
			name   string
			change func(f *signForge, s *signer)
			want   string
		}{
			{
				name:   "returns ErrVerify for a draft without its tag",
				change: func(f *signForge, _ *signer) { clear(f.tags) },
				want:   "the draft v1.2.0 has no tag",
			},
			{
				name: "returns ErrVerify for a draft without the cosign bundle of its checksums",
				change: func(f *signForge, _ *signer) {
					f.drafts[0].Assets = slices.DeleteFunc(f.drafts[0].Assets, func(a forge.Asset) bool {
						return a.ID == bundleAsset
					})
				},
				want: "the draft v1.2.0 has no " + providerSums + ".sigstore.json",
			},
			{
				name:   "returns ErrVerify for an asset whose digest differs from the checksums",
				change: func(f *signForge, _ *signer) { f.drafts[0].Assets[0].Digest = digestOf("another zip") },
				want:   "the asset terraform-provider-demo_1.2.0_linux_amd64.zip of the draft v1.2.0 has the SHA-256",
			},
			{
				name:   "returns ErrVerify for an asset without a digest",
				change: func(f *signForge, _ *signer) { f.drafts[0].Assets[2].Digest = "" },
				want:   `has the SHA-256 ""`,
			},
			{
				name: "returns ErrVerify for checksums that list a file that the draft lacks",
				change: func(f *signForge, _ *signer) {
					f.contents[sumsAsset] += digestOf("x") + "  terraform-provider-demo_1.2.0_freebsd_amd64.zip\n"
				},
				want: "lists terraform-provider-demo_1.2.0_freebsd_amd64.zip, which the draft v1.2.0 lacks",
			},
			{
				name: "returns ErrVerify for a zip that the checksums do not list",
				change: func(f *signForge, _ *signer) {
					f.drafts[0].Assets = append(f.drafts[0].Assets,
						forge.Asset{Name: "terraform-provider-demo_1.2.0_windows_amd64.zip", ID: 99})
				},
				want: "does not list the zip terraform-provider-demo_1.2.0_windows_amd64.zip of the draft v1.2.0",
			},
			{
				name:   "returns ErrVerify for checksums without a line",
				change: func(f *signForge, _ *signer) { f.contents[sumsAsset] = "" },
				want:   providerSums + " lists no file",
			},
			{
				name:   "returns ErrVerify for a line that is not a digest and a name",
				change: func(f *signForge, _ *signer) { f.contents[sumsAsset] = "a b c\n" },
				want:   `the line "a b c" of ` + providerSums + ", which is not a digest and a name",
			},
			{
				name:   "returns ErrVerify for a signature of another key",
				change: func(_ *signForge, s *signer) { s.key = release.Signature{Fingerprint: primaryKey, Algorithm: 1} },
				want:   "the key " + primaryKey + " signed " + providerSums + ", and not " + signingKey,
			},
			{
				name:   "returns ErrVerify for a key of an algorithm other than RSA or DSA",
				change: func(_ *signForge, s *signer) { s.key.Algorithm = 22 },
				want:   "has the algorithm 22, which is not RSA or DSA",
			},
		}
		for _, tt := range checks {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f := providerHost("1.2.0")
				s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
				tt.change(f, s)
				got, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
				assert.ErrorIs(t, err, release.ErrVerify, "Sign")
				expect.Contains(t, err.Error(), tt.want, "the error")
				expect.Empty(t, got, "the published tags")
				expect.Empty(t, f.calls, "the calls of the host")
			})
		}

		t.Run("reads a line of the binary mode of sha256sum", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0")
			f.contents[sumsAsset] = strings.ReplaceAll(f.contents[sumsAsset], "  terraform", " *terraform")
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			_, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
			assert.NoError(t, err, "Sign")
		})

		failures := []struct {
			want   error
			change func(f *signForge, w *witness, s *signer)
			name   string
		}{
			{
				name:   "returns the error of the drafts of the host",
				change: func(f *signForge, _ *witness, _ *signer) { f.failDrafts = errHost },
				want:   errHost,
			},
			{
				name:   "returns the error of the tag of the host",
				change: func(f *signForge, _ *witness, _ *signer) { f.failTag = errHost },
				want:   errHost,
			},
			{
				name:   "returns the error of the download of the checksums",
				change: func(f *signForge, _ *witness, _ *signer) { f.failDownload = errHost },
				want:   errHost,
			},
			{
				name:   "returns the error of the download of the bundle",
				change: func(f *signForge, _ *witness, _ *signer) { delete(f.contents, bundleAsset) },
				want:   errHost,
			},
			{
				name:   "returns the error of the witness before the signature",
				change: func(_ *signForge, w *witness, _ *signer) { w.fail = errWitness },
				want:   errWitness,
			},
			{
				name:   "returns the error of the signer",
				change: func(_ *signForge, _ *witness, s *signer) { s.failSign = errSigner },
				want:   errSigner,
			},
			{
				name:   "returns the error of the verification of the signature",
				change: func(_ *signForge, _ *witness, s *signer) { s.failVerify = errSigner },
				want:   errSigner,
			},
			{
				name:   "returns the error of the upload of the signature",
				change: func(f *signForge, _ *witness, _ *signer) { f.failUpload = errHost },
				want:   errHost,
			},
			{
				name:   "returns the error of the publish",
				change: func(f *signForge, _ *witness, _ *signer) { f.failPublish = errHost },
				want:   errHost,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f, w := providerHost("1.2.0"), &witness{}
				s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
				tt.change(f, w, s)
				got, err := release.Sign(t.Context(), signingOf(t, f, w, s))
				assert.ErrorIs(t, err, tt.want, "Sign")
				assert.Empty(t, got, "the published tags")
			})
		}

		t.Run("returns the error of the download of an existing signature", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0")
			signature := forge.Asset{Name: providerSums + ".sig", ID: signatureAsset}
			f.drafts[0].Assets = append(f.drafts[0].Assets, signature)
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			_, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
			assert.ErrorIs(t, err, errHost, "Sign")
			assert.Empty(t, f.calls, "the calls of the host")
		})

		t.Run("returns the error of checksums that it cannot read", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0")
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			_, err := release.Sign(t.Context(), signingOf(t, f, &witness{remove: true}, s))
			assert.ErrorIs(t, err, fs.ErrNotExist, "Sign")
			assert.Contains(t, err.Error(), "read "+providerSums, "the error")
		})

		t.Run("returns the error of a download that it cannot write", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0")
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			run := signingOf(t, f, &witness{}, s)
			run.Dir = filepath.Join(run.Dir, "absent")
			_, err := release.Sign(t.Context(), run)
			assert.HasError(t, err, "Sign")
			assert.Contains(t, err.Error(), "write the download of "+providerSums, "the error")
		})

		t.Run("returns the tags that it published before an error", func(t *testing.T) {
			t.Parallel()
			f := providerHost("1.2.0", "1.1.0")
			delete(f.tags, "v1.2.0")
			s := &signer{key: release.Signature{Fingerprint: signingKey, Algorithm: 1}}
			got, err := release.Sign(t.Context(), signingOf(t, f, &witness{}, s))
			assert.ErrorIs(t, err, release.ErrVerify, "Sign")
			assert.Equal(t, got, []string{"v1.1.0"}, "the published tags")
		})
	})
}
