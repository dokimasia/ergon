// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workspace"
	"go.yaml.in/yaml/v3"
)

// ErrTag is the error of [Packer.Pack] for a tag of a module that the repository has at another
// commit than HEAD.
var ErrTag = errors.New("release: tag at another commit")

// The configuration of GoReleaser of a module, which ergon init renders in the directory of the
// module, and the tool of the section go that runs it.
const (
	goreleaserConfig = ".goreleaser.yaml"
	goreleaserTool   = "goreleaser"
)

// The record of GoReleaser of the artifacts of a run, in its dist directory.
const artifactsFile = "artifacts.json"

// The variables of the environment of GoReleaser that the managed configuration reads: the version
// of the release, such as 0.6.0, which the template of the version of its snapshot mode reads, and
// the paths of the modules of the release, separated by commas, which it passes to syft as
// GOPRIVATE.
const (
	versionVariable = "ERGON_VERSION"
	modulesVariable = "ERGON_MODULES"
)

// assetTypes are the types of the artifacts of GoReleaser that a release attaches.
var assetTypes = []string{"Archive", "Linux Package", "Source", "Checksum", "Signature", "SBOM"}

// caskType is the type of an artifact of GoReleaser that is a cask of Homebrew.
const caskType = "Homebrew Cask"

// artifact is an artifact of a run of GoReleaser, as its artifacts.json records it.
type artifact struct {
	// Name is the name of the artifact, such as ergon_0.6.0_linux_amd64.tar.gz.
	Name string `json:"name"`

	// Path is the path of the artifact, relative to the directory that GoReleaser ran in.
	Path string `json:"path"`

	// Type is the type of the artifact, such as Archive.
	Type string `json:"type"`
}

// build is a module of a release whose directory has .goreleaser.yaml.
type build struct {
	// pkg is the module.
	pkg *workspace.Package

	// config is the path of the configuration of GoReleaser of the module, relative to the root of
	// the repository and slash-separated.
	config string

	// dist is the dist directory of the configuration, relative to the root of the repository.
	dist string

	// tag is the tag of the module at its version.
	tag string
}

// Packer builds the assets of the commands of Go modules with GoReleaser. A module has commands when
// its directory has .goreleaser.yaml, which ergon init renders from the key binaries of the section
// go. Each function is required.
//
// # Concurrency
//
// A Packer is safe for concurrent use when its functions are, but two calls of Pack must not pack
// one repository at once, because both write its tags and its dist directory.
type Packer struct {
	// Snapshot returns the tree of the working tree of a directory as git would commit it, which
	// Pack writes the zips of the released modules from.
	Snapshot func(ctx context.Context, dir string) (string, error)

	// Tags returns the commit of each tag of the repository of dir.
	Tags func(ctx context.Context, dir string) (map[string]string, error)

	// Head returns the commit of HEAD of the repository of dir.
	Head func(ctx context.Context, dir string) (string, error)

	// Tag creates the lightweight tag name at commit in the repository of dir.
	Tag func(ctx context.Context, dir, name, commit string) error

	// Run runs the tool of the section go with args in dir, with env added to the environment of the
	// process, as ergon tool run does, and returns an error for a tool that fails.
	Run func(ctx context.Context, dir, tool string, args, env []string) error
}

var _ language.Packer = Packer{}

// Pack builds the assets of each module of pkgs whose directory under root has .goreleaser.yaml, in
// the order of pkgs, and builds nothing for any other module:
//
//   - It reads the dist directory of each configuration, and creates the tag of each such module at
//     its Version at HEAD, as [Tagger] names it, unless the repository has the tag at HEAD. Go
//     writes the version of the module into its binaries from that tag, and the publish creates the
//     same tag on the host.
//   - It writes the zip of each module of pkgs at its Version into a module proxy in a temporary
//     directory, from a snapshot of the working tree, as [Versioner.Apply] does, and removes the
//     directory afterwards.
//   - It runs goreleaser release --snapshot --clean with the configuration in root, outside the
//     workspace, with the environment of a build against that proxy. The go command then resolves
//     the modules of the repository at the versions that go install resolves, and writes those
//     versions into the binaries. ERGON_VERSION is the Version, which the template of the version
//     of the configuration reads. ERGON_MODULES are the paths of the modules of pkgs in order,
//     separated by commas, which the managed configuration passes to syft as GOPRIVATE.
//   - It copies the archives, the packages, the source archive, checksums.txt, its signature and the
//     SBOMs into dir/assets/<tag>/, and the casks into dir/casks/<tag>/, as artifacts.json in the
//     dist directory of the configuration records them, and then removes that dist directory.
//
// It returns an error that wraps [ErrTag] for a tag at another commit than HEAD, an error for a
// module of pkgs that the repository does not have, and the error of a function of p, of the
// proxy, of go env, of reading the configuration or artifacts.json, and of copying an artifact, each
// with the module or the file.
func (p Packer) Pack(ctx context.Context, root string, pkgs []workspace.Package, dir string) error {
	var builds []build
	for i := range pkgs {
		pkg := &pkgs[i]
		config := path.Join(pkg.Dir, goreleaserConfig)
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(config)))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("release: read %s: %w", config, err)
		}
		var c struct {
			// Dist is the directory of the artifacts of GoReleaser, relative to root.
			Dist string `yaml:"dist"`
		}
		if err := yaml.Unmarshal(data, &c); err != nil {
			return fmt.Errorf("release: read %s: %w", config, err)
		}
		if c.Dist == "" {
			return fmt.Errorf("release: %s states no dist directory", config)
		}
		tag := Tagger{}.Tag(pkg, pkg.Version)
		if err := p.tagHead(ctx, root, tag); err != nil {
			return err
		}
		builds = append(builds, build{pkg: pkg, config: config, dist: c.Dist, tag: tag})
	}
	if len(builds) == 0 {
		return nil
	}
	r, err := newRun(p.Snapshot, root)
	if err != nil {
		return err
	}
	tmp, err := r.open()
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	env, err := buildEnv(ctx, r, pkgs, filepath.Join(tmp, modcacheDir))
	if err != nil {
		return err
	}
	for _, b := range builds {
		args := []string{"release", "--snapshot", "--clean", "--config", b.config}
		vars := slices.Concat([]string{versionVariable + "=" + b.pkg.Version.String()}, env)
		if err := p.Run(ctx, root, goreleaserTool, args, vars); err != nil {
			return fmt.Errorf("release: GoReleaser of %s: %w", b.pkg.Name, err)
		}
		if err := collect(root, filepath.Join(root, filepath.FromSlash(b.dist)), dir, b.tag); err != nil {
			return fmt.Errorf("release: the assets of %s: %w", b.pkg.Name, err)
		}
	}
	return nil
}

// tagHead creates the lightweight tag name at HEAD of the repository at root, and leaves a tag that
// is at HEAD. It returns an error that wraps [ErrTag] for a tag at another commit, and the errors
// of the functions of p.
func (p Packer) tagHead(ctx context.Context, root, name string) error {
	head, err := p.Head(ctx, root)
	if err != nil {
		return err
	}
	tags, err := p.Tags(ctx, root)
	if err != nil {
		return err
	}
	commit, found := tags[name]
	if !found {
		return p.Tag(ctx, root, name, head)
	}
	if commit != head {
		return fmt.Errorf("%w: %s is at %s, and HEAD is at %s", ErrTag, name, commit, head)
	}
	return nil
}

// buildEnv serves the zip of each module of pkgs at its Version from the proxy of r, and returns
// the variables of the environment of GoReleaser. These are the variables of a build against the
// proxy with the module cache modcache, as [run.environment] returns them, and ERGON_MODULES. It
// returns an error for a module of pkgs that the repository does not have, and the errors of
// [run.serve] and of run.environment.
func buildEnv(ctx context.Context, r *run, pkgs []workspace.Package, modcache string) ([]string, error) {
	paths := make([]string, 0, len(pkgs))
	for i := range pkgs {
		name := pkgs[i].Name
		if err := r.checkModule(name); err != nil {
			return nil, err
		}
		r.released[name] = "v" + pkgs[i].Version.String()
		if err := r.serve(ctx, r.modules[name]); err != nil {
			return nil, err
		}
		paths = append(paths, name)
	}
	env, err := r.environment(ctx, modcache, buildFlags)
	if err != nil {
		return nil, err
	}
	return append(env, modulesVariable+"="+strings.Join(paths, ",")), nil
}

// collect copies the assets and the casks that artifacts.json of the dist directory dist records,
// with paths relative to root, into dir/assets/<tag>/ and dir/casks/<tag>/, and then removes dist.
// It returns the error of reading artifacts.json and of copying an artifact.
func collect(root, dist, dir, tag string) error {
	data, err := os.ReadFile(filepath.Join(dist, artifactsFile))
	if err != nil {
		return fmt.Errorf("read the artifacts of GoReleaser: %w", err)
	}
	var artifacts []artifact
	if err := json.Unmarshal(data, &artifacts); err != nil {
		return fmt.Errorf("decode the artifacts of GoReleaser: %w", err)
	}
	for _, a := range artifacts {
		target := language.AssetsDir
		switch {
		case slices.Contains(assetTypes, a.Type):
		case a.Type == caskType:
			target = language.CasksDir
		default:
			continue
		}
		into := filepath.Join(dir, target, filepath.FromSlash(tag), a.Name)
		if err := copyFile(filepath.Join(root, filepath.FromSlash(a.Path)), into); err != nil {
			return err
		}
	}
	if err := os.RemoveAll(dist); err != nil {
		return fmt.Errorf("remove the dist directory of GoReleaser: %w", err)
	}
	return nil
}

// copyFile copies the file at from to the path into, and creates the directory of into. It returns
// the error of the file system with the path.
func copyFile(from, into string) error {
	if err := os.MkdirAll(filepath.Dir(into), dirPerm); err != nil {
		return fmt.Errorf("copy an artifact: %w", err)
	}
	source, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("copy an artifact: %w", err)
	}
	defer func() { _ = source.Close() }()
	target, err := os.OpenFile(into, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, filePerm)
	if err != nil {
		return fmt.Errorf("copy an artifact: %w", err)
	}
	_, err = io.Copy(target, source)
	if err = errors.Join(err, target.Close()); err != nil {
		return fmt.Errorf("copy %s: %w", from, err)
	}
	return nil
}
