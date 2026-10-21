// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tool_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
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

		trees := []struct {
			name        string
			giveFile    string
			giveProgram string
			giveContent []byte
		}{
			{
				name:        "runs the program of a .tar.xz with its tree",
				giveFile:    "tool.tar.xz",
				giveProgram: treeDir + program,
				giveContent: tree(".tar.xz", member{name: treeDir}, member{name: treeDir + program, content: self},
					member{name: "tool-1.0/lib/"}, member{name: treeLibrary, content: []byte(treeContent)}),
			},
			{
				name:        "runs the program of a .zip with its tree",
				giveFile:    "tool.zip",
				giveProgram: "tool-1.0/tool.exe",
				giveContent: tree(".zip", member{name: treeDir}, member{name: "tool-1.0/tool.exe", content: self},
					member{name: "tool-1.0/lib/"}, member{name: treeLibrary, content: []byte(treeContent)}),
			},
		}
		for _, tt := range trees {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o, served := demo(), map[string][]byte{}
				o.Tools.Tool = release(served, tt.giveFile, tt.giveProgram, tt.giveContent)
				o.Tools.Tool.asset.Tree = true
				r, out, _ := runner(t, served)
				_, err := r.Run(t.Context(), section, o, "tool", []string{"check"})
				assert.NoError(t, err, "Run")
				assert.Equal(t, out.String(), "tool check\n", "the output of the tool")
				root := filepath.Join(r.Cache, "release", "tool", "1.0", strings.ReplaceAll(string(host()), "/", "-"),
					digest(tt.giveContent))
				files.HasContent(t, filepath.Join(root, filepath.FromSlash(treeLibrary)), treeContent,
					"the library beside the program")
			})
		}

		t.Run("runs a tree of the cache without a download", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			content := tree(".tar.xz", member{name: treeDir + program, content: self})
			o.Tools.Tool = release(served, "tool.tar.xz", treeDir+program, content)
			o.Tools.Tool.asset.Tree = true
			r, out, _ := runner(t, served)
			_, err := r.Run(t.Context(), section, o, "tool", nil)
			assert.NoError(t, err, "the first Run")
			clear(served)
			_, err = r.Run(t.Context(), section, o, "tool", []string{"again"})
			assert.NoError(t, err, "the second Run")
			assert.Equal(t, out.String(), "tool\n"+"tool again\n", "the output of both runs")
		})

		refused := []struct {
			name        string
			giveFile    string
			giveContent []byte
		}{
			{
				name:     "returns ErrInstall for a tree with a path outside its directory",
				giveFile: "tool.tar.xz",
				giveContent: tree(".tar.xz", member{name: treeDir + program, content: self},
					member{name: "../escape", content: []byte("x")}),
			},
			{
				name:     "returns ErrInstall for a .tar.xz tree with a link",
				giveFile: "tool.tar.xz",
				giveContent: tree(".tar.xz", member{name: treeDir + program, content: self},
					member{name: "tool-1.0/link", link: program}),
			},
			{
				name:     "returns ErrInstall for a .zip tree with a link",
				giveFile: "tool.zip",
				giveContent: tree(".zip", member{name: treeDir + program, content: self},
					member{name: "tool-1.0/link", link: program}),
			},
			{
				name:        "returns ErrInstall for a tree without the program",
				giveFile:    "tool.tar.xz",
				giveContent: tree(".tar.xz", member{name: treeLibrary, content: []byte(treeContent)}),
			},
			{
				name:     "returns ErrInstall for a tree with a file below a file",
				giveFile: "tool.tar.xz",
				giveContent: tree(".tar.xz", member{name: treeDir + program, content: self},
					member{name: treeDir + program + "/below", content: []byte("x")}),
			},
			{
				name:     "returns ErrInstall for a .zip tree with a file below a file",
				giveFile: "tool.zip",
				giveContent: tree(".zip", member{name: treeDir + program, content: self},
					member{name: treeDir + program + "/below", content: []byte("x")}),
			},
			{
				name:        "returns ErrInstall for a .tar.xz tree that does not read",
				giveFile:    "tool.tar.xz",
				giveContent: []byte("no archive"),
			},
			{
				name:        "returns ErrInstall for a .zip tree that does not read",
				giveFile:    "tool.zip",
				giveContent: []byte("no archive"),
			},
			{
				name:     "returns ErrInstall for a .tar.xz tree whose tar does not read",
				giveFile: "tool.tar.xz",
				giveContent: func() []byte {
					var b bytes.Buffer
					w, _ := xz.NewWriter(&b)
					_, _ = w.Write([]byte("no tar"))
					_ = w.Close()
					return b.Bytes()
				}(),
			},
		}
		for _, tt := range refused {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				o, served := demo(), map[string][]byte{}
				o.Tools.Tool = release(served, tt.giveFile, treeDir+program, tt.giveContent)
				o.Tools.Tool.asset.Tree = true
				r, _, _ := runner(t, served)
				_, err := r.Run(t.Context(), section, o, "tool", nil)
				assert.ErrorIs(t, err, tool.ErrInstall, "Run")
			})
		}

		t.Run("returns ErrInstall for a tree whose directory does not rename", func(t *testing.T) {
			t.Parallel()
			o, served := demo(), map[string][]byte{}
			content := tree(".tar.xz", member{name: treeDir + program, content: self})
			o.Tools.Tool = release(served, "tool.tar.xz", treeDir+program, content)
			o.Tools.Tool.asset.Tree = true
			r, _, _ := runner(t, served)
			platform := strings.ReplaceAll(string(host()), "/", "-")
			root := path.Join("release", "tool", "1.0", platform, digest(content))
			files.Write(t, r.Cache, files.Tree{root: files.Text("")})
			_, err := r.Run(t.Context(), section, o, "tool", nil)
			assert.ErrorIs(t, err, tool.ErrInstall, "Run")
		})

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
			files.Write(t, r.Cache, files.Tree{"file": files.Text("")})
			r.Cache = filepath.Join(r.Cache, "file")
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
