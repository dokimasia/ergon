// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/core/workspace"
	"go.dokimi.dev/ergon/lang/javascript/release"
	jsworkspace "go.dokimi.dev/ergon/lang/javascript/workspace"
)

// The package.json files of @probe/a and @probe/b in the cases of Apply. @probe/b requires @probe/a
// as a dependency and as a peer, with spaces of its own.
const (
	manifestA = `{"name": "@probe/a", "version": "0.1.0"}` + "\n"
	manifestB = `{
  "name": "@probe/b",
  "version" :  "0.2.0",
  "dependencies": {"@probe/a": "^0.1.0", "left-pad": "^1.3.0"},
  "peerDependencies": {"@probe/a":">=0.1.0"}
}
`
)

// The paths of the manifests of @probe/a and @probe/b, relative to the root and slash-separated.
const (
	fileA = "packages/a/package.json"
	fileB = "packages/b/package.json"
)

// The packages of manifestA and manifestB, as their discovery returns them.
var (
	packageA = workspace.Package{
		Name: "@probe/a", Toolchain: pinnedToolchain, Dir: "packages/a", Version: version.Version{Minor: 1},
	}
	packageB = workspace.Package{
		Name:      "@probe/b",
		Toolchain: pinnedToolchain,
		Dir:       "packages/b",
		Version:   version.Version{Minor: 2},
		Deps: []workspace.Dependency{
			{Name: "@probe/a", Kind: workspace.KindRuntime, Req: "^0.1.0"},
			{Name: "@probe/a", Kind: workspace.KindPeer, Req: ">=0.1.0"},
		},
	}
)

// rewriting is an expectation of testdata/rewrites.json: the requirement that changesets writes for
// the requirement and the new version, or null where changesets leaves the requirement as it is.
type rewriting struct {
	Want        *string `json:"want"`
	GiveReq     string  `json:"req"`
	GiveVersion string  `json:"version"`
}

// lockfile is a fake of the refresh of a lockfile by a package manager. It records each call. When
// content is not empty, it writes content into the lockfile in the directory of the call, which file
// names. It removes that lockfile when remove is set, and returns err.
type lockfile struct {
	tb      testing.TB
	err     error
	file    string
	content string
	calls   []call
	remove  bool
}

// run records the call and acts as lockfile states.
func (f *lockfile) run(_ context.Context, dir, name string, args, _ []string) error {
	f.calls = append(f.calls, call{dir: dir, name: name, args: args})
	if f.remove {
		return os.Remove(filepath.Join(dir, f.file))
	}
	if f.content != "" {
		files.Write(f.tb, dir, files.Tree{f.file: files.Text(f.content)})
	}
	return f.err
}

func TestVersioner(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			giveReq string
			want    language.Resolution
		}{
			{
				name:    "returns Selected for a range after the workspace protocol that admits the version",
				giveReq: "workspace:^1.2.0",
				want:    language.ResolutionSelected,
			},
			{
				name:    "returns Excluded for a version after the workspace protocol",
				giveReq: "workspace:1.2.0",
				want:    language.ResolutionExcluded,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := release.Versioner{}.Resolve(tt.giveReq, version.Version{Major: 1, Minor: 3})
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, tt.want, "the resolution")
			})
		}
	})

	t.Run("Rewrite", func(t *testing.T) {
		t.Parallel()
		var tests []rewriting
		data := files.Read(t, filepath.Join("testdata", "rewrites.json"))
		assert.NoError(t, json.Unmarshal([]byte(data), &tests), "the decoding of the expectations")
		assert.NotEmpty(t, tests, "the expectations")
		for _, tt := range tests {
			want := tt.GiveReq
			if tt.Want != nil {
				want = *tt.Want
			}
			t.Run(fmt.Sprintf("returns %q for %q at %s", want, tt.GiveReq, tt.GiveVersion), func(t *testing.T) {
				t.Parallel()
				v, err := version.Parse(tt.GiveVersion)
				assert.NoError(t, err, "Parse")
				got, err := release.Versioner{}.Rewrite(tt.GiveReq, v)
				assert.NoError(t, err, "Rewrite")
				assert.Equal(t, got, want, "the requirement")
			})
		}

		t.Run("returns ErrOverflow for an upper bound past the highest component", func(t *testing.T) {
			t.Parallel()
			_, err := release.Versioner{}.Rewrite(">=1.0.0 <2.0.0", version.Version{Major: version.MaxComponent})
			assert.ErrorIs(t, err, version.ErrOverflow, "Rewrite")
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a pre-release", func(t *testing.T) {
			t.Parallel()
			err := release.Versioner{}.Validate(&packageA, version.Version{Major: 1, Pre: "rc.1"})
			assert.NoError(t, err, "Validate")
		})
	})

	t.Run("Apply", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the version and the requirements of each edit in place", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{fileA: files.Text(manifestA), fileB: files.Text(manifestB)})
			f := &lockfile{tb: t}
			edits := []language.Edit{
				{Package: packageA, Version: version.Version{Major: 1}},
				{Package: packageB, Version: version.Version{Minor: 2, Patch: 1}, Requirements: []workspace.Dependency{
					{Name: "@probe/a", Kind: workspace.KindRuntime, Req: "^1.0.0"},
					{Name: "@probe/a", Kind: workspace.KindPeer, Req: ">=1.0.0"},
				}},
			}
			got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, edits)
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{fileA, fileB}, "the changed files")
			files.Contains(t, os.DirFS(root), files.Tree{
				fileA: files.Text(`{"name": "@probe/a", "version": "1.0.0"}` + "\n"),
				fileB: files.Text(`{
  "name": "@probe/b",
  "version" :  "0.2.1",
  "dependencies": {"@probe/a": "^1.0.0", "left-pad": "^1.3.0"},
  "peerDependencies": {"@probe/a":">=1.0.0"}
}
`),
			}, "the manifests")
			assert.Empty(t, f.calls, "the runs of the package manager")
		})

		t.Run("rewrites the requirement of one section alone", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(t, files.Tree{fileB: files.Text(manifestB)})
			edit := language.Edit{Package: packageB, Version: packageB.Version, Requirements: []workspace.Dependency{
				{Name: "@probe/a", Kind: workspace.KindPeer, Req: ">=1.0.0"},
			}}
			got, err := release.Versioner{
				Command: (&lockfile{tb: t}).run,
			}.Apply(
				t.Context(),
				root,
				[]language.Edit{edit},
			)
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{fileB}, "the changed files")
			assert.Contains(t, files.Read(t, filepath.Join(root, filepath.FromSlash(fileB))),
				`"dependencies": {"@probe/a": "^0.1.0", "left-pad": "^1.3.0"},
  "peerDependencies": {"@probe/a":">=1.0.0"}`, "the requirements of the manifest")
		})

		t.Run("leaves a shorthand of the workspace protocol as it is", func(t *testing.T) {
			t.Parallel()
			const manifest = `{"name": "@probe/b", "version": "0.2.0", "dependencies": {"@probe/a": "workspace:*"}, ` +
				`"devDependencies": {"@probe/a": "workspace:^0.1.0"}}`
			root := files.Workspace(t, files.Tree{fileB: files.Text(manifest)})
			p := packageB
			p.Deps = []workspace.Dependency{
				{Name: "@probe/a", Kind: workspace.KindRuntime, Req: "workspace:0.1.0"},
				{Name: "@probe/a", Kind: workspace.KindDev, Req: "workspace:^0.1.0"},
			}
			edit := language.Edit{Package: p, Version: p.Version, Requirements: []workspace.Dependency{
				{Name: "@probe/a", Kind: workspace.KindRuntime, Req: "workspace:1.0.0"},
				{Name: "@probe/a", Kind: workspace.KindDev, Req: "workspace:^1.0.0"},
			}}
			_, err := release.Versioner{Command: (&lockfile{tb: t}).run}.Apply(t.Context(), root, []language.Edit{edit})
			assert.NoError(t, err, "Apply")
			files.HasContent(t, filepath.Join(root, filepath.FromSlash(fileB)),
				`{"name": "@probe/b", "version": "0.2.0", "dependencies": {"@probe/a": "workspace:*"}, `+
					`"devDependencies": {"@probe/a": "workspace:^1.0.0"}}`, "the manifest")
		})

		lockfiles := []struct {
			name     string
			giveFile string
			wantCall call
		}{
			{
				name:     "refreshes pnpm-lock.yaml with pnpm",
				giveFile: release.PNPMLockFile,
				wantCall: call{name: "pnpm", args: []string{"install", "--lockfile-only"}},
			},
			{
				name:     "refreshes package-lock.json with npm",
				giveFile: release.NPMLockFile,
				wantCall: call{name: "npm", args: []string{"install", "--package-lock-only"}},
			},
		}
		for _, tt := range lockfiles {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				root := files.Workspace(t, files.Tree{fileA: files.Text(manifestA), tt.giveFile: files.Text("old\n")})
				f := &lockfile{tb: t, file: tt.giveFile, content: "new\n"}
				edit := language.Edit{Package: packageA, Version: version.Version{Major: 1}}
				got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{edit})
				assert.NoError(t, err, "Apply")
				assert.Equal(t, got, []string{fileA, tt.giveFile}, "the changed files")
				tt.wantCall.dir = root
				assert.Equal(t, f.calls, []call{tt.wantCall}, "the runs of the package manager")
			})
		}

		t.Run("returns no lockfile that the refresh leaves as it is", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(
				t,
				files.Tree{fileA: files.Text(manifestA), release.NPMLockFile: files.Text("{}\n")},
			)
			f := &lockfile{tb: t}
			edit := language.Edit{Package: packageA, Version: version.Version{Major: 1}}
			got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{edit})
			assert.NoError(t, err, "Apply")
			assert.Equal(t, got, []string{fileA}, "the changed files")
			assert.Length(t, f.calls, 1, "the runs of the package manager")
		})

		t.Run("refreshes no lockfile for edits that change no file", func(t *testing.T) {
			t.Parallel()
			root := files.Workspace(
				t,
				files.Tree{fileA: files.Text(manifestA), release.NPMLockFile: files.Text("{}\n")},
			)
			f := &lockfile{tb: t}
			edit := language.Edit{Package: packageA, Version: packageA.Version}
			got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{edit})
			assert.NoError(t, err, "Apply")
			assert.Empty(t, got, "the changed files")
			assert.Empty(t, f.calls, "the runs of the package manager")
		})

		edits := []struct {
			name     string
			giveTree files.Tree
			giveEdit language.Edit
			wantErr  error
		}{
			{
				name:     "returns ErrEdit for a manifest without a version",
				giveTree: files.Tree{fileA: files.Text(`{"name": "@probe/a"}`)},
				giveEdit: language.Edit{Package: packageA, Version: version.Version{Major: 1}},
				wantErr:  release.ErrEdit,
			},
			{
				name:     "returns ErrEdit for a manifest whose requirements changed",
				giveTree: files.Tree{fileB: files.Text(`{"name": "@probe/b", "version": "0.2.0"}`)},
				giveEdit: language.Edit{Package: packageB, Version: version.Version{Minor: 3}},
				wantErr:  release.ErrEdit,
			},
			{
				name:     "returns ErrEdit for a requirement that the package does not have",
				giveTree: files.Tree{fileB: files.Text(manifestB)},
				giveEdit: language.Edit{
					Package: packageB,
					Version: packageB.Version,
					Requirements: []workspace.Dependency{
						{Name: "@probe/a", Kind: workspace.KindDev, Req: "^1.0.0"},
					},
				},
				wantErr: release.ErrEdit,
			},
			{
				name:     "returns ErrManifest for a manifest that does not parse",
				giveTree: files.Tree{fileA: files.Text(`{"name": `)},
				giveEdit: language.Edit{Package: packageA, Version: version.Version{Major: 1}},
				wantErr:  jsworkspace.ErrManifest,
			},
		}
		for _, tt := range edits {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f := &lockfile{tb: t}
				got, err := release.Versioner{Command: f.run}.Apply(t.Context(), files.Workspace(t, tt.giveTree),
					[]language.Edit{tt.giveEdit})
				assert.ErrorIs(t, err, tt.wantErr, "Apply")
				assert.Empty(t, got, "the changed files")
			})
		}

		failures := []struct {
			name     string
			giveTree files.Tree
			giveEdit language.Edit
			wantText string
		}{
			{
				name:     "returns the error of a manifest that does not read",
				giveTree: files.Tree{fileA: files.Dir()},
				giveEdit: language.Edit{Package: packageA, Version: version.Version{Major: 1}},
				wantText: "release: read " + fileA + ": ",
			},
			{
				name:     "returns the error of a manifest that does not write",
				giveTree: files.Tree{fileA: files.Text(manifestA).WithMode(0o444)},
				giveEdit: language.Edit{Package: packageA, Version: version.Version{Major: 1}},
				wantText: "release: write " + fileA + ": ",
			},
			{
				name:     "returns the error of a requirement that is not UTF-8",
				giveTree: files.Tree{fileB: files.Text(manifestB)},
				giveEdit: language.Edit{
					Package: packageB,
					Version: packageB.Version,
					Requirements: []workspace.Dependency{
						{Name: "@probe/a", Kind: workspace.KindRuntime, Req: "^1.0.0\xff"},
					},
				},
				wantText: "release: " + fileB + `: the requirement "^1.0.0\xff": `,
			},
			{
				name:     "returns the error of a lockfile that does not read",
				giveTree: files.Tree{fileA: files.Text(manifestA), release.PNPMLockFile: files.Dir()},
				giveEdit: language.Edit{Package: packageA, Version: version.Version{Major: 1}},
				wantText: "release: read " + release.PNPMLockFile + ": ",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				f := &lockfile{tb: t}
				_, err := release.Versioner{Command: f.run}.Apply(t.Context(), files.Workspace(t, tt.giveTree),
					[]language.Edit{tt.giveEdit})
				assert.HasError(t, err, "Apply")
				assert.HasPrefix(t, err.Error(), tt.wantText, "the text of the error")
			})
		}

		refreshes := []struct {
			name     string
			giveFake lockfile
			wantText string
		}{
			{
				name:     "returns the error of the package manager with the lockfile",
				giveFake: lockfile{err: errProgram},
				wantText: "release: refresh " + release.NPMLockFile + ": ",
			},
			{
				name:     "returns the error of a lockfile that the refresh removes",
				giveFake: lockfile{file: release.NPMLockFile, remove: true},
				wantText: "release: read " + release.NPMLockFile + ": ",
			},
		}
		for _, tt := range refreshes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				root := files.Workspace(
					t,
					files.Tree{fileA: files.Text(manifestA), release.NPMLockFile: files.Text("{}\n")},
				)
				f := tt.giveFake
				f.tb = t
				edit := language.Edit{Package: packageA, Version: version.Version{Major: 1}}
				got, err := release.Versioner{Command: f.run}.Apply(t.Context(), root, []language.Edit{edit})
				assert.HasError(t, err, "Apply")
				assert.HasPrefix(t, err.Error(), tt.wantText, "the text of the error")
				assert.Equal(t, got, []string{fileA, release.NPMLockFile}, "the changed files")
			})
		}

		t.Run("returns the error of a repository that does not open", func(t *testing.T) {
			t.Parallel()
			_, err := release.Versioner{}.Apply(t.Context(), filepath.Join(t.TempDir(), "absent"), nil)
			assert.HasError(t, err, "Apply")
			assert.HasPrefix(t, err.Error(), "release: open the repository: ", "the text of the error")
		})
	})
}
