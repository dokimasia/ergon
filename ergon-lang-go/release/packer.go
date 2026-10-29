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
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"

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

// cgoEnabled is the variable of the environment of a build of GoReleaser with cgo.
const cgoEnabled = "CGO_ENABLED=1"

// The tool of the section go that compiles the C code of a build with cgo, and the variables of
// the environment of GoReleaser that the builds with cgo of the managed configuration read: the
// program of zig, and the directory of the text stubs of the libraries of macOS.
const (
	zigTool       = "zig"
	zigVariable   = "ERGON_ZIG"
	stubsVariable = "ERGON_DARWIN_STUBS"
)

// The variables of the environment of zig that name its caches. A pack points both at its
// temporary directory, because the module cache is read-only and zig writes its cache beside a C
// source by default.
const (
	zigGlobalCache = "ZIG_GLOBAL_CACHE_DIR"
	zigLocalCache  = "ZIG_LOCAL_CACHE_DIR"
)

// The directories of the builds with cgo in the temporary directory of a pack: one directory for
// each build, with the stubs of macOS and the cache of zig.
const (
	cgoDir      = "cgo"
	stubsDir    = "stubs"
	zigCacheDir = "zig"
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

// configuration has the keys of a configuration of GoReleaser that [Packer.Pack] reads: the dist
// directory, the environment of each build, the extra files of the checksums, and the draft of the
// release.
type configuration struct {
	// Dist is the directory of the artifacts of GoReleaser, relative to root.
	Dist string `yaml:"dist"`

	// Builds are the builds of the commands.
	Builds []struct {
		// Env are the variables of the environment of the build, such as CGO_ENABLED=0.
		Env []string `yaml:"env"`
	} `yaml:"builds"`

	// Checksum has the extra files of the checksums.
	Checksum struct {
		// ExtraFiles are the files of the repository that the checksums list beside the artifacts.
		ExtraFiles []struct {
			// Glob is the path of the file, relative to root.
			Glob string `yaml:"glob"`

			// NameTemplate is the name of the file in the checksums, a template of the version, such
			// as demo_{{ .Version }}_manifest.json, or empty for the base name of the file.
			NameTemplate string `yaml:"name_template"`
		} `yaml:"extra_files"`
	} `yaml:"checksum"`

	// Release has the draft of the release.
	Release struct {
		// Draft reports that the release of the tag remains a draft after the publish.
		Draft bool `yaml:"draft"`
	} `yaml:"release"`
}

// extraFile is an extra file of the checksums of a configuration, which the release of its tag
// attaches.
type extraFile struct {
	// path is the path of the file, relative to the root of the repository and slash-separated.
	path string

	// name is the name of the file in the checksums and in the release.
	name string
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

	// env are the variables of the builds with cgo of the configuration, which [Packer.cgo] returns,
	// and nil for a configuration without such a build.
	env []string

	// extras are the extra files of the checksums of the configuration.
	extras []extraFile

	// cgo reports that a build of the configuration sets CGO_ENABLED=1.
	cgo bool

	// draft reports that the release of the tag remains a draft after the publish.
	draft bool
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

	// Program installs the tool of the section go for dir unless the cache has it, and returns its
	// program, as ergon tool path does, such as zig, the C compiler of the builds with cgo.
	Program func(ctx context.Context, dir, tool string) (string, error)
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
//   - For a configuration with a build that sets CGO_ENABLED=1, it installs zig with Program and
//     writes text stubs of the libraries of macOS that the Go toolchain of the module imports, as
//     [writeStubs] states, into the temporary directory. ERGON_ZIG is the program of zig, which the
//     builds run as their C compiler. ERGON_DARWIN_STUBS is the directory of the stubs, which the
//     builds of macOS link against. ZIG_GLOBAL_CACHE_DIR and ZIG_LOCAL_CACHE_DIR name a cache of
//     zig in the temporary directory.
//   - It runs goreleaser release --snapshot --clean with the configuration in root, outside the
//     workspace, with the environment of a build against that proxy. The go command then resolves
//     the modules of the repository at the versions that go install resolves, and writes those
//     versions into the binaries. ERGON_VERSION is the Version, which the template of the version
//     of the configuration reads. ERGON_MODULES are the paths of the modules of pkgs in order,
//     separated by commas, which the managed configuration passes to syft as GOPRIVATE.
//   - It copies the archives, the packages, the source archive, the checksums, their signature and
//     the SBOMs into dir/assets/<tag>/, and the casks into dir/casks/<tag>/, as artifacts.json in
//     the dist directory of the configuration records them, and then removes that dist directory.
//   - It copies each extra file of the checksums of the configuration into dir/assets/<tag>/ under
//     its name template, with the Version as .Version, because GoReleaser lists such a file in the
//     checksums alone. It writes the empty file dir/drafts/<tag> for a configuration whose release
//     is a draft, whose release the publish then leaves as a draft.
//
// It returns an error that wraps [ErrTag] for a tag at another commit than HEAD, an error for a
// module of pkgs that the repository does not have, and the error of a function of p, of the
// proxy, of go env, of the stubs, of reading the configuration or artifacts.json, of a name
// template with a field other than .Version, of copying an artifact or an extra file, and of
// writing the draft, each with the module or the file.
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
		var c configuration
		err = yaml.Unmarshal(data, &c)
		if err != nil {
			return fmt.Errorf("release: read %s: %w", config, err)
		}
		if c.Dist == "" {
			return fmt.Errorf("release: %s states no dist directory", config)
		}
		tag := Tagger{}.Tag(pkg, pkg.Version)
		err = p.tagHead(ctx, root, tag)
		if err != nil {
			return err
		}
		cgo := false
		for _, b := range c.Builds {
			cgo = cgo || slices.Contains(b.Env, cgoEnabled)
		}
		extras, err := extraFilesOf(&c, pkg.Version.String())
		if err != nil {
			return fmt.Errorf("release: read %s: %w", config, err)
		}
		builds = append(builds, build{
			pkg: pkg, config: config, dist: c.Dist, tag: tag, extras: extras, cgo: cgo, draft: c.Release.Draft,
		})
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
	for i := range builds {
		if !builds[i].cgo {
			continue
		}
		if builds[i].env, err = p.cgo(ctx, root, &builds[i], filepath.Join(tmp, cgoDir, strconv.Itoa(i))); err != nil {
			return err
		}
	}
	env, err := buildEnv(ctx, r, pkgs, filepath.Join(tmp, modcacheDir))
	if err != nil {
		return err
	}
	for _, b := range builds {
		args := []string{"release", "--snapshot", "--clean", "--config", b.config}
		vars := slices.Concat([]string{versionVariable + "=" + b.pkg.Version.String()}, env, b.env)
		if err := p.Run(ctx, root, goreleaserTool, args, vars); err != nil {
			return fmt.Errorf("release: GoReleaser of %s: %w", b.pkg.Name, err)
		}
		if err := collect(root, filepath.Join(root, filepath.FromSlash(b.dist)), dir, b.tag); err != nil {
			return fmt.Errorf("release: the assets of %s: %w", b.pkg.Name, err)
		}
		if err := attach(root, dir, &b); err != nil {
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

// cgo returns the variables of the environment of GoReleaser for the builds with cgo of b: the
// program of zig, which Program installs, the directory of the stubs of macOS, which [writeStubs]
// writes for the Go toolchain of the module of b, and a cache of zig. It writes the stubs and the
// cache below dir. go env GOROOT in the directory of the module, outside the workspace, reports
// the root of the toolchain that the builds use. It returns the errors of Program, of go env and
// of writeStubs, each with the module.
func (p Packer) cgo(ctx context.Context, root string, b *build, dir string) ([]string, error) {
	zig, err := p.Program(ctx, root, zigTool)
	if err != nil {
		return nil, fmt.Errorf("release: install zig for the builds with cgo of %s: %w", b.pkg.Name, err)
	}
	cmd := exec.CommandContext(ctx, "go", "env", "GOROOT")
	cmd.Dir = filepath.Join(root, filepath.FromSlash(b.pkg.Dir))
	cmd.Env = append(os.Environ(), "GOWORK=off")
	goroot, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("release: go env GOROOT for the builds with cgo of %s: %w", b.pkg.Name, err)
	}
	stubs, cache := filepath.Join(dir, stubsDir), filepath.Join(dir, zigCacheDir)
	if err := writeStubs(strings.TrimSpace(string(goroot)), stubs); err != nil {
		return nil, fmt.Errorf("release: the stubs of macOS for the builds with cgo of %s: %w", b.pkg.Name, err)
	}
	return []string{
		zigVariable + "=" + zig,
		stubsVariable + "=" + stubs,
		zigGlobalCache + "=" + cache,
		zigLocalCache + "=" + cache,
	}, nil
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

// extraFilesOf returns the extra files of the checksums of c, each named after its name template
// with version as .Version, as GoReleaser names it in snapshot mode, or after the base name of its
// path for an empty template. It returns the error of a name template that does not parse or that
// names a field other than .Version, with the path of the file.
func extraFilesOf(c *configuration, version string) ([]extraFile, error) {
	extras := make([]extraFile, 0, len(c.Checksum.ExtraFiles))
	for _, f := range c.Checksum.ExtraFiles {
		name := path.Base(f.Glob)
		if f.NameTemplate != "" {
			var b strings.Builder
			t, err := template.New(f.Glob).Option("missingkey=error").Parse(f.NameTemplate)
			if err == nil {
				err = t.Execute(&b, map[string]string{"Version": version})
			}
			if err != nil {
				return nil, fmt.Errorf("the name of the extra file %s: %w", f.Glob, err)
			}
			name = b.String()
		}
		extras = append(extras, extraFile{path: f.Glob, name: name})
	}
	return extras, nil
}

// attach copies the extra files of the checksums of b from the repository at root into
// dir/assets/<tag>/ under their names, and writes the empty file dir/drafts/<tag> for a
// configuration whose release is a draft. It returns the error of copying a file and of writing the
// draft.
func attach(root, dir string, b *build) error {
	for _, f := range b.extras {
		into := filepath.Join(dir, language.AssetsDir, filepath.FromSlash(b.tag), f.name)
		if err := copyFile(filepath.Join(root, filepath.FromSlash(f.path)), into); err != nil {
			return err
		}
	}
	if !b.draft {
		return nil
	}
	draft := filepath.Join(dir, language.DraftsDir, filepath.FromSlash(b.tag))
	err := os.MkdirAll(filepath.Dir(draft), dirPerm)
	if err == nil {
		err = os.WriteFile(draft, nil, filePerm)
	}
	if err != nil {
		return fmt.Errorf("write the draft of %s: %w", b.tag, err)
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
