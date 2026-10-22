// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
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
	"go.dokimi.dev/ergon/lang/javascript/release"
)

// versionB is the path of the version of @probe/b at its release in the npm registry.
const versionB = "/@probe%2fb/0.2.0"

// The errors of a request to the registry that fails, and of a body that fails to read.
var (
	errRegistry = errors.New("registry: the connection failed")
	errBody     = errors.New("registry: the body failed")
)

// registry is a fake of the npm registry. The transport records the address of each request. It
// answers a request with err, or with the status of its escaped path in statuses and 404 for any
// other path. The body of a response fails to read with bodyErr when it is set.
type registry struct {
	err      error
	bodyErr  error
	statuses map[string]int
	urls     []string
}

// RoundTrip records the address of r and answers it as registry states.
func (f *registry) RoundTrip(r *http.Request) (*http.Response, error) {
	f.urls = append(f.urls, r.URL.String())
	if f.err != nil {
		return nil, f.err
	}
	status, ok := f.statuses[r.URL.EscapedPath()]
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

// tools is a fake of ergon tool run. It records each call and returns err.
type tools struct {
	err   error
	calls []call
}

// run records the call and returns err.
func (f *tools) run(_ context.Context, dir, tool string, args, _ []string) error {
	f.calls = append(f.calls, call{dir: dir, name: tool, args: args})
	return f.err
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
			{name: "reports true for a version that the registry has", giveStatus: http.StatusOK, want: true},
			{name: "reports false for a version that the registry does not have", giveStatus: http.StatusNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f := &registry{statuses: map[string]int{versionB: tt.giveStatus}}
				got, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &releasedB)
				assert.NoError(t, err, "Published")
				assert.Equal(t, got, tt.want, "Published")
			})
		}

		t.Run("reads the version without build metadata", func(t *testing.T) {
			t.Parallel()
			f := &registry{}
			p := releasedB
			p.Version = version.Version{Major: 1, Pre: "rc.1", Build: "5"}
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &p)
			assert.NoError(t, err, "Published")
			assert.Equal(t, f.urls, []string{"https://registry.npmjs.org/@probe%2fb/1.0.0-rc.1"}, "the requests")
		})

		t.Run("returns ErrRegistry for another status of the registry", func(t *testing.T) {
			t.Parallel()
			f := &registry{statuses: map[string]int{versionB: http.StatusServiceUnavailable}}
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &releasedB)
			assert.ErrorIs(t, err, release.ErrRegistry, "Published")
		})

		t.Run("returns the error of the request", func(t *testing.T) {
			t.Parallel()
			client := &http.Client{Transport: &registry{err: errRegistry}}
			_, err := release.Publisher{Client: client}.Published(t.Context(), &releasedB)
			assert.ErrorIs(t, err, errRegistry, "Published")
		})

		t.Run("returns the error of reading the body", func(t *testing.T) {
			t.Parallel()
			f := &registry{bodyErr: errBody, statuses: map[string]int{versionB: http.StatusOK}}
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &releasedB)
			assert.ErrorIs(t, err, errBody, "Published")
		})

		t.Run("returns the error of a name that is not valid in an address", func(t *testing.T) {
			t.Parallel()
			f := &registry{}
			p := releasedB
			p.Name = "probe%zz"
			_, err := release.Publisher{Client: &http.Client{Transport: f}}.Published(t.Context(), &p)
			ue := assert.ErrorAs[*url.Error](t, err, "Published")
			assert.Equal(t, ue.Op, "parse", "the operation of the error")
			assert.Empty(t, f.urls, "the requests")
		})
	})

	t.Run("Publish", func(t *testing.T) {
		t.Parallel()

		t.Run("runs npm publish with the tarball of each package in the order of the packages", func(t *testing.T) {
			t.Parallel()
			pack := files.Workspace(t, files.Tree{
				"js/@probe/a/" + tarballA: files.Text(""),
				"js/@probe/b/" + tarballB: files.Text(""),
			})
			f := &tools{}
			err := release.Publisher{Tools: f.run}.Publish(t.Context(), t.TempDir(),
				[]workspace.Package{releasedB, releasedA}, pack)
			assert.NoError(t, err, "Publish")
			dist := filepath.Join(pack, release.DistDir, "@probe")
			assert.Equal(t, f.calls, []call{
				{dir: pack, name: "npm", args: []string{"publish", filepath.Join(dist, "b", tarballB)}},
				{dir: pack, name: "npm", args: []string{"publish", filepath.Join(dist, "a", tarballA)}},
			}, "the runs of npm")
		})

		tests := []struct {
			name    string
			givePre string
			wantTag string
		}{
			{
				name:    "publishes a pre-release with the dist-tag of its first identifier",
				givePre: "beta.1",
				wantTag: "beta",
			},
			{
				name:    "publishes a pre-release whose identifier is a range with the dist-tag next",
				givePre: "0",
				wantTag: "next",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				p := releasedB
				p.Version.Pre = tt.givePre
				file := "probe-b-0.2.0-" + tt.givePre + ".tgz"
				pack := files.Workspace(t, files.Tree{"js/@probe/b/" + file: files.Text("")})
				f := &tools{}
				err := release.Publisher{Tools: f.run}.Publish(t.Context(), t.TempDir(), []workspace.Package{p}, pack)
				assert.NoError(t, err, "Publish")
				assert.Equal(t, f.calls, []call{{dir: pack, name: "npm", args: []string{
					"publish", filepath.Join(pack, release.DistDir, "@probe", "b", file), "--tag", tt.wantTag,
				}}}, "the runs of npm")
			})
		}

		t.Run("returns ErrTarball for a package without its tarball in the pack directory", func(t *testing.T) {
			t.Parallel()
			f := &tools{}
			err := release.Publisher{Tools: f.run}.Publish(t.Context(), t.TempDir(), []workspace.Package{releasedB},
				t.TempDir())
			assert.ErrorIs(t, err, release.ErrTarball, "Publish")
			assert.Empty(t, f.calls, "the runs of npm")
		})

		t.Run("returns the error of npm publish", func(t *testing.T) {
			t.Parallel()
			pack := files.Workspace(t, files.Tree{"js/@probe/b/" + tarballB: files.Text("")})
			err := release.Publisher{Tools: (&tools{err: errProgram}).run}.Publish(t.Context(), t.TempDir(),
				[]workspace.Package{releasedB}, pack)
			assert.ErrorIs(t, err, errProgram, "Publish")
		})
	})
}
