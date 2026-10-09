// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/tool"
)

// The addresses of the jars of the cases in Maven Central.
const (
	pmdJar    = "/maven2/net/sourceforge/pmd/pmd-java/7.28.0/pmd-java-7.28.0.jar"
	ktlintJar = "/maven2/com/pinterest/ktlint/ktlint-cli/1.8.0/ktlint-cli-1.8.0-all.jar"
)

func TestDownload(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		runs := []struct {
			name    string
			file    string
			program string
			content []byte
		}{
			{name: "runs the program of a .tar.gz", file: "tool.tar.gz", program: entry, content: toolTarGz},
			{
				name:    "runs the program of a .tar.xz",
				file:    "tool.tar.xz",
				program: entry,
				content: archive(".tar.xz", entry, self),
			},
			{name: "runs the program of a .zip", file: "tool.zip", program: "tool.exe", content: toolZip},
			{name: "runs an asset that is the program", file: program, content: self},
		}
		for _, tt := range runs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o, served := demo(), map[string][]byte{}
				o.Tools.Tool = release(served, tt.file, tt.program, tt.content)
				r, out, _ := runner(t, served)
				code, err := r.Run(t.Context(), section, o, "tool", []string{"check"})
				assert.NoError(t, err, "Run")
				assert.Equal(t, code, 0, "the exit status")
				assert.Equal(t, out.String(), "tool check\n", "the output of the tool")
			})
		}

		t.Run("runs the uv of the section on Windows from its .zip", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.UV = uv(served, option.WindowsAMD64)
			r, out, _ := runner(t, served)
			r.Platform = option.WindowsAMD64
			_, err := r.Run(t.Context(), section, o, "uv", []string{"--version"})
			assert.NoError(t, err, "Run")
			assert.Equal(t, out.String(), "uv --version\n", "the output of uv")
		})

		failures := []struct {
			name string
			give func(served map[string][]byte) binary
		}{
			{
				name: "returns ErrInstall for a platform without an asset",
				give: func(map[string][]byte) binary {
					return binary{SHA256: map[option.Platform]string{host(): digest(nil)}}
				},
			},
			{
				name: "returns ErrInstall for a platform without a digest",
				give: func(served map[string][]byte) binary {
					b := release(served, "tool.tar.gz", entry, toolTarGz)
					b.SHA256 = map[option.Platform]string{"plan9/amd64": digest(toolTarGz)}
					return b
				},
			},
			{
				name: "returns ErrInstall for a download of another digest",
				give: func(served map[string][]byte) binary {
					b := release(served, "tool.tar.gz", entry, toolTarGz)
					b.SHA256[host()] = digest([]byte("other"))
					return b
				},
			},
			{
				name: "returns ErrInstall for an asset that the server does not have",
				give: func(served map[string][]byte) binary {
					b := release(served, "tool.tar.gz", entry, toolTarGz)
					clear(served)
					return b
				},
			},
			{
				name: "returns ErrInstall for an address that is no URL",
				give: func(served map[string][]byte) binary {
					b := release(served, "tool.tar.gz", entry, toolTarGz)
					b.asset.URL = "https://example.com/\x7f"
					return b
				},
			},
			{
				name: "returns ErrInstall for a .tar.gz without the program",
				give: func(served map[string][]byte) binary {
					return release(
						served,
						"tool.tar.gz",
						"tool-1.0/other",
						archive(".tar.gz", "tool-1.0/tool", []byte("x")),
					)
				},
			},
			{
				name: "returns ErrInstall for a .zip without the program",
				give: func(served map[string][]byte) binary {
					return release(served, "tool.zip", "other.exe", archive(".zip", "tool.exe", []byte("x")))
				},
			},
			{
				name: "returns ErrInstall for a .tar.gz that does not read",
				give: func(served map[string][]byte) binary {
					return release(served, "tool.tar.gz", "tool-1.0/tool", []byte("no archive"))
				},
			},
			{
				name: "returns ErrInstall for a .tar.xz that does not read",
				give: func(served map[string][]byte) binary {
					return release(served, "tool.tar.xz", "tool-1.0/tool", toolTarGz)
				},
			},
			{
				name: "returns ErrInstall for a .zip that does not read",
				give: func(served map[string][]byte) binary {
					return release(served, "tool.zip", "tool.exe", []byte("no archive"))
				},
			},
			{
				name: "returns ErrInstall for a .tar.gz that ends inside the program",
				give: func(served map[string][]byte) binary {
					return release(served, "tool.tar.gz", "tool", truncated())
				},
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o, served := demo(), map[string][]byte{}
				o.Tools.Tool = tt.give(served)
				r, _, _ := runner(t, served)
				_, err := r.Run(t.Context(), section, o, "tool", nil)
				assert.ErrorIs(t, err, tool.ErrInstall, "Run")
			})
		}

		t.Run("returns ErrInstall for a request that fails", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.Tool = release(served, "tool.tar.gz", entry, toolTarGz)
			r, _, _ := runner(t, served)
			r.Client.Transport.(redirect).target.Host = "127.0.0.1:1"
			_, err := r.Run(t.Context(), section, o, "tool", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})

		t.Run("returns ErrInstall for a cache that does not create", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			o.Tools.Tool = release(served, "tool.tar.gz", entry, toolTarGz)
			r, _, _ := runner(t, served)
			r.Cache = filepath.Join(r.Cache, "file")
			assert.NoError(t, os.WriteFile(r.Cache, nil, 0o644), "WriteFile of the cache")
			_, err := r.Run(t.Context(), section, o, "tool", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})

		t.Run("runs the jar of a Maven artifact with java", func(t *testing.T) {
			t.Parallel()
			served := map[string][]byte{pmdJar: []byte("jar"), pmdJar + ".sha256": []byte(digest([]byte("jar")))}
			r, out, _ := runner(t, served)
			_, err := r.Run(t.Context(), section, demo(), "pmd", []string{"check"})
			assert.NoError(t, err, "Run")
			want := filepath.Join(r.Cache, "maven", "net.sourceforge.pmd", "pmd-java", "7.28.0", "pmd-java-7.28.0.jar")
			assert.Equal(t, out.String(), "java -jar "+want+" check\n", "the output of java")
		})

		t.Run("runs the jar of the classifier of a Maven artifact", func(t *testing.T) {
			t.Parallel()
			served := map[string][]byte{
				ktlintJar:             []byte("jar"),
				ktlintJar + ".sha256": []byte(strings.ToUpper(digest([]byte("jar"))) + "  ktlint-cli-1.8.0-all.jar\n"),
			}
			r, out, _ := runner(t, served)
			_, err := r.Run(t.Context(), section, demo(), "ktlint", nil)
			assert.NoError(t, err, "Run")
			assert.HasSuffix(t, out.String(), "ktlint-cli-1.8.0-all.jar\n", "the output of java")
		})

		t.Run("runs a jar of the cache without a download", func(t *testing.T) {
			t.Parallel()
			served := map[string][]byte{pmdJar: []byte("jar"), pmdJar + ".sha256": []byte(digest([]byte("jar")))}
			r, out, _ := runner(t, served)
			_, err := r.Run(t.Context(), section, demo(), "pmd", nil)
			assert.NoError(t, err, "the first Run")
			clear(served)
			_, err = r.Run(t.Context(), section, demo(), "pmd", nil)
			assert.NoError(t, err, "the second Run")
			assert.Equal(t, strings.Count(out.String(), "java -jar"), 2, "the runs of java")
		})

		jars := []struct {
			name   string
			served map[string][]byte
		}{
			{
				name:   "returns ErrInstall for a jar without a digest file",
				served: map[string][]byte{pmdJar: []byte("jar")},
			},
			{
				name:   "returns ErrInstall for a digest file without a digest",
				served: map[string][]byte{pmdJar: []byte("jar"), pmdJar + ".sha256": []byte("none\n")},
			},
			{
				name:   "returns ErrInstall for a jar of another digest",
				served: map[string][]byte{pmdJar: []byte("jar"), pmdJar + ".sha256": []byte(digest([]byte("other")))},
			},
		}
		for _, tt := range jars {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r, _, _ := runner(t, tt.served)
				_, err := r.Run(t.Context(), section, demo(), "pmd", nil)
				assert.ErrorIs(t, err, tool.ErrInstall, "Run")
			})
		}
	})
}

// uv returns the release binary of uv 0.12.23 for the platform p, whose asset the server of the
// cases serves with the test binary as the program, and whose pin states its digest. It adds the
// asset to served.
func uv(served map[string][]byte, p option.Platform) option.UV {
	u := option.UV{Version: "0.12.23"}
	asset, _ := u.Asset(p)
	suffix := ".tar.gz"
	if strings.HasSuffix(asset.URL, ".zip") {
		suffix = ".zip"
	}
	content := archive(suffix, asset.Program, self)
	served[strings.TrimPrefix(asset.URL, "https://github.com")] = content
	u.SHA256 = map[option.Platform]string{p: digest(content)}
	return u
}

// truncated returns a .tar.gz whose entry tool states 100 bytes and has 3.
func truncated() []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "tool", Mode: 0o755, Size: 100})
	_, _ = tw.Write([]byte("abc"))
	_ = gz.Close()
	return b.Bytes()
}
