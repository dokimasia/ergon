// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/python/release"
	pyworkspace "go.dokimi.dev/ergon/lang/python/workspace"
)

// releaseB is the path of the JSON API of PyPI for probe-b at 0.2.0.
const releaseB = "/pypi/probe-b/0.2.0/json"

// The errors of a request to the index that fails, and of a body that fails to read.
var (
	errIndex = errors.New("index: the connection failed")
	errBody  = errors.New("index: the body failed")
)

// index is a fake of PyPI. The transport records the address of each request. It answers a request
// with err, or with the status of its path in statuses and 404 for any other path. The body of a
// response fails to read with bodyErr when it is set.
type index struct {
	err      error
	bodyErr  error
	statuses map[string]int
	urls     []string
}

// RoundTrip records the address of r and answers it as index states.
func (f *index) RoundTrip(r *http.Request) (*http.Response, error) {
	f.urls = append(f.urls, r.URL.String())
	if f.err != nil {
		return nil, f.err
	}
	status, ok := f.statuses[r.URL.Path]
	if !ok {
		status = http.StatusNotFound
	}
	var body io.Reader = strings.NewReader("{}")
	if f.bodyErr != nil {
		body = iotest.ErrReader(f.bodyErr)
	}
	return &http.Response{
		StatusCode: status,
		Status:     strconv.Itoa(status) + " " + http.StatusText(status),
		Header:     http.Header{},
		Body:       io.NopCloser(body),
		Request:    r,
	}, nil
}

func TestPublisher(t *testing.T) {
	t.Parallel()

	t.Run("Published", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			giveStatus int
			want       bool
		}{
			{name: "reports true for a release that PyPI has", giveStatus: http.StatusOK, want: true},
			{name: "reports false for a release that PyPI does not have", giveStatus: http.StatusNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f := &index{statuses: map[string]int{releaseB: tt.giveStatus}}
				got, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &releasedB)
				assert.NoError(t, err, "Published")
				assert.Equal(t, got, tt.want, "Published")
			})
		}

		t.Run("reads the JSON API of PyPI at the version of PEP 440", func(t *testing.T) {
			t.Parallel()
			f := &index{}
			p := releasedB
			p.Version = version.Version{Major: 1, Pre: "rc.1"}
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &p)
			assert.NoError(t, err, "Published")
			assert.Equal(t, f.urls, []string{"https://pypi.org/pypi/probe-b/1.0.0rc1/json"}, "the requests")
		})

		t.Run("returns ErrIndex for another status of PyPI", func(t *testing.T) {
			t.Parallel()
			f := &index{statuses: map[string]int{releaseB: http.StatusServiceUnavailable}}
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &releasedB)
			assert.ErrorIs(t, err, release.ErrIndex, "Published")
		})

		t.Run("returns the error of the request", func(t *testing.T) {
			t.Parallel()
			client := &http.Client{Transport: &index{err: errIndex}}
			_, err := release.Publisher{Client: client}.Published(t.Context(), &releasedB)
			assert.ErrorIs(t, err, errIndex, "Published")
		})

		t.Run("returns the error of reading the body", func(t *testing.T) {
			t.Parallel()
			f := &index{bodyErr: errBody, statuses: map[string]int{releaseB: http.StatusOK}}
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &releasedB)
			assert.ErrorIs(t, err, errBody, "Published")
		})

		t.Run("returns the error of a name that is not valid in an address", func(t *testing.T) {
			t.Parallel()
			f := &index{}
			p := releasedB
			p.Name = "probe%zz"
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &p)
			ue := assert.ErrorAs[*url.Error](t, err, "Published")
			assert.Equal(t, ue.Op, "parse", "the operation of the error")
			assert.Empty(t, f.urls, "the requests")
		})

		t.Run("returns ErrVersion for a version that PEP 440 does not have", func(t *testing.T) {
			t.Parallel()
			f := &index{}
			p := releasedB
			p.Version.Build = "1"
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &p)
			assert.ErrorIs(t, err, pyworkspace.ErrVersion, "Published")
			assert.Empty(t, f.urls, "the requests")
		})
	})

	t.Run("Publish", func(t *testing.T) {
		t.Parallel()

		t.Run("runs uv publish with the distributions of each package in their order", func(t *testing.T) {
			t.Parallel()
			pack := t.TempDir()
			files.Write(t, pack, files.Tree{
				"python/probe-a/.gitignore":                     files.Text("*"),
				"python/probe-a/probe_a-0.3.0.tar.gz":           files.Text(""),
				"python/probe-a/probe_a-0.3.0-py3-none-any.whl": files.Text(""),
				"python/probe-b/.gitignore":                     files.Text("*"),
				"python/probe-b/" + sdistB:                      files.Text(""),
				"python/probe-b/" + wheelB:                      files.Text(""),
			})
			f := &tools{}
			p := release.Publisher{Tools: f.run}
			err := p.Publish(t.Context(), t.TempDir(), []workspace.Package{releasedB, releasedA}, pack)
			assert.NoError(t, err, "Publish")
			dist := filepath.Join(pack, release.DistDir)
			assert.Equal(t, f.calls, []call{
				{dir: pack, tool: "uv", args: []string{
					"publish", filepath.Join(dist, "probe-b", wheelB), filepath.Join(dist, "probe-b", sdistB),
				}},
				{dir: pack, tool: "uv", args: []string{
					"publish",
					filepath.Join(dist, "probe-a", "probe_a-0.3.0-py3-none-any.whl"),
					filepath.Join(dist, "probe-a", "probe_a-0.3.0.tar.gz"),
				}},
			}, "the runs of uv")
		})

		t.Run("returns ErrDistribution for a package without distributions in the pack directory", func(t *testing.T) {
			t.Parallel()
			f := &tools{}
			err := release.Publisher{Tools: f.run}.Publish(t.Context(), t.TempDir(), []workspace.Package{releasedB},
				t.TempDir())
			assert.ErrorIs(t, err, release.ErrDistribution, "Publish")
			assert.Empty(t, f.calls, "the runs of uv")
		})

		t.Run("returns an error for a project path that is a file", func(t *testing.T) {
			t.Parallel()
			pack := t.TempDir()
			files.Write(t, pack, files.Tree{"python/probe-b": files.Text("")})
			f := &tools{}
			err := release.Publisher{
				Tools: f.run,
			}.Publish(
				t.Context(),
				t.TempDir(),
				[]workspace.Package{releasedB},
				pack,
			)
			assert.HasError(t, err, "Publish")
			assert.Empty(t, f.calls, "the runs of uv")
		})

		t.Run("returns ErrVersion for a version that PEP 440 does not have", func(t *testing.T) {
			t.Parallel()
			f := &tools{}
			p := releasedB
			p.Version.Build = "1"
			err := release.Publisher{
				Tools: f.run,
			}.Publish(
				t.Context(),
				t.TempDir(),
				[]workspace.Package{p},
				t.TempDir(),
			)
			assert.ErrorIs(t, err, pyworkspace.ErrVersion, "Publish")
			assert.Empty(t, f.calls, "the runs of uv")
		})

		t.Run("returns the error of uv publish", func(t *testing.T) {
			t.Parallel()
			pack := t.TempDir()
			files.Write(t, pack, files.Tree{"python/probe-b/" + sdistB: files.Text("")})
			p := release.Publisher{Tools: (&tools{err: errUV}).run}
			err := p.Publish(t.Context(), t.TempDir(), []workspace.Package{releasedB}, pack)
			assert.ErrorIs(t, err, errUV, "Publish")
		})
	})
}
