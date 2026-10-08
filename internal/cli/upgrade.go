// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/lock"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/pin"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/tool"
	"go.dokimi.dev/ergon/service/vcs"
)

// majorFlag lets an upgrade take a release of a later major version.
const majorFlag = "major"

// The variables of the environment that an upgrade reads.
const (
	// upgradeEnv names the release of ergon that an upgrade started. That release upgrades the
	// repository itself, and starts no other release.
	upgradeEnv = "ERGON_UPGRADE"

	// proxyEnv is the list of the module proxies of Go, from which an upgrade reads the newest
	// release of ergon.
	proxyEnv = "GOPROXY"
)

// The release of ergon that an upgrade installs.
const (
	// ergonModule is the root module of ergon, whose tag every release of ergon has.
	ergonModule = "go.dokimi.dev/ergon"

	// ergonRepository is the repository on GitHub whose releases publish the archives of ergon.
	ergonRepository = "dokimasia/ergon"

	// defaultProxy is the module proxy of an environment whose GOPROXY names no URL.
	defaultProxy = "https://proxy.golang.org"

	// checksumsFile is the file of a release of ergon with the SHA-256 of each archive.
	checksumsFile = "checksums.txt"

	// development is the release of a build of ergon without a release.
	development = "dev"

	// windows is the system whose programs end in .exe.
	windows = "windows"
)

// The pull request of ergon init ci upgrade.
const (
	// baselineBranch is the prefix of its branch, before the name of the base branch, as in
	// ergon-baseline/main.
	baselineBranch = "ergon-baseline/"

	// upgradeTitle is its title and the message of its commits, before the release.
	upgradeTitle = "build: upgrade ergon to "

	// upgradeIntro opens its body, with the release.
	upgradeIntro = "This pull request was opened by `ergon init ci upgrade`. Merging it moves the managed files " +
		"to the baseline of ergon %s."

	// overridesIntro opens the list of the overrides of its body.
	overridesIntro = ".ergon.yaml sets these pins to another version than the baseline. Remove a key to take " +
		"the baseline."

	// conflictsIntro opens the list of the conflicts of its body.
	conflictsIntro = "ergon init sync left these managed files, which were edited by hand. Move each edit into " +
		"the local file of its path under .ergon/local, and run ergon init sync --force."

	// upgradeSummary is the summary of the changeset without packages of the pull request, before the
	// release.
	upgradeSummary = "Upgrade ergon to "
)

// The help texts of the upgrade commands.
const (
	upgradeShort = "Move the repository to the baseline of the newest release of ergon"
	upgradeLong  = `ergon init upgrade moves the repository to the baseline of the newest release
of ergon: the highest release of go.dokimi.dev/ergon that the first module
proxy of GOPROXY that is a URL lists, of the major version of this ergon, or of
any major version with --major. When it is newer than this ergon, the command
installs it into the cache of ergon tool run, checks its archive against the
checksums.txt of the release, and runs its ergon init upgrade, whose exit
status it returns.

The newest release runs ergon init sync, and then prints each pin of
.ergon.yaml whose version differs from the baseline. A build of ergon without a
release runs the upgrade itself, and a release refuses a lock that such a build
wrote.`

	initCIShort = "Run the job of the workflow baseline.yml"
	initCILong  = `ergon init ci runs the job of the workflow baseline.yml in GitHub Actions.`

	ciUpgradeShort = "Upgrade ergon and open the pull request of the change"
	ciUpgradeLong  = `ergon init ci upgrade upgrades the repository as ergon init upgrade does. When
the sync changes a file, it writes a changeset without packages, commits the
changes on the branch ergon-baseline/<base> through the API of GitHub, which
signs the commits, and opens or updates the pull request into the base branch
of .changeset/config.json. The exit status is 1 after it proposed the files of a
sync that left a managed file that was edited by hand.`
)

// ergonBinary is a release binary of ergon: the archive ergon_<version>_<os>_<arch>.tar.gz of its
// release on a server of GitHub, with the program ergon, which is ergon.exe on Windows.
type ergonBinary struct {
	option.Binary `yaml:",inline"`

	// server is the address of the server of GitHub without a slash at its end, such as
	// https://github.com.
	server string
}

var _ option.Release = ergonBinary{}

// Asset returns the archive of b for p. It returns no error.
func (b ergonBinary) Asset(p option.Platform) (option.Asset, error) {
	name := program
	if p.OS() == windows {
		name += ".exe"
	}
	archive := program + "_" + b.Version + "_" + p.OS() + "_" + p.Arch() + ".tar.gz"
	return option.Asset{
		URL:     b.server + "/" + b.Repository() + "/releases/download/v" + b.Version + "/" + archive,
		Program: name,
	}, nil
}

// Repository returns dokimasia/ergon, the repository whose releases publish ergon.
func (ergonBinary) Repository() string {
	return ergonRepository
}

// upgraded is what the sync of an upgrade did.
type upgraded struct {
	// changes are the files that the sync wrote and removed.
	changes []baseline.Change

	// overrides are the pins of .ergon.yaml whose version differs from the baseline.
	overrides []baseline.Override

	// conflicts are the managed files that the sync left, because they were edited by hand.
	conflicts []string
}

// upgradeCommand returns ergon init upgrade, which moves the repository of s to the newest release
// of ergon under ctx: it starts that release with [session.launch], or upgrades the repository
// itself with [session.upgrade].
func upgradeCommand(ctx context.Context, s *session) *cobra.Command {
	var major, force bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: upgradeShort,
		Long:  upgradeLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			launched, err := s.launch(ctx, cmd, major)
			if err != nil || launched {
				return err
			}
			_, err = s.upgrade(cmd.OutOrStdout(), force)
			return err
		},
	}
	cmd.Flags().BoolVar(&major, majorFlag, false, "take a release of a later major version")
	cmd.Flags().BoolVar(&force, forceFlag, false, "overwrite the managed files that were edited by hand")
	return cmd
}

// ciUpgradeCommand returns ergon init ci upgrade, which upgrades the repository of s as ergon init
// upgrade does under ctx, and proposes the change through the API of GitHub: a changeset without
// packages, and the pull request from ergon-baseline/<base> into the base branch. It returns the
// errors of the environment, of the upgrade and of the proposal, and the error of a sync that left a
// managed file, which wraps [baseline.ErrConflict], after it proposed the change.
func ciUpgradeCommand(ctx context.Context, s *session) *cobra.Command {
	var major, force bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: ciUpgradeShort,
		Long:  ciUpgradeLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			launched, err := s.launch(ctx, cmd, major)
			if err != nil || launched {
				return err
			}
			client, repo, err := s.forge()
			if err != nil {
				return err
			}
			config, err := os.ReadFile(filepath.Join(s.dir, filepath.FromSlash(release.ConfigPath)))
			if err != nil {
				return fmt.Errorf("cli: read %s, which ergon init seeds: %w", release.ConfigPath, err)
			}
			base, err := release.ParseBaseBranch(config)
			if err != nil {
				return err
			}
			head, err := vcs.Head(ctx, s.dir)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			u, synced := s.upgrade(out, force)
			if synced != nil && !errors.Is(synced, baseline.ErrConflict) {
				return synced
			}
			if len(u.changes) == 0 {
				return synced
			}
			c := changeset.Changeset{Summary: upgradeSummary + s.release + "."}
			if c.ID, err = release.ChangesetID(c.Summary, s.random); err != nil {
				return err
			}
			file, err := release.AddChangeset(s.dir, &c)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "%s %s\n", baseline.Wrote, file)
			files, deleted, err := release.ChangedFiles(ctx, s.dir)
			if err != nil {
				return err
			}
			p := release.Proposal{
				Files: files, Repo: repo, Base: base, Branch: baselineBranch + base, Head: head,
				Title: upgradeTitle + s.release, Body: upgradeBody(s.release, &u), Deleted: deleted,
			}
			number, err := release.Propose(ctx, client, &p)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "pull request %d\n", number)
			return synced
		},
	}
	cmd.Flags().BoolVar(&major, majorFlag, false, "take a release of a later major version")
	cmd.Flags().BoolVar(&force, forceFlag, false, "overwrite the managed files that were edited by hand")
	return cmd
}

// launch starts the newest release of ergon with the arguments of the process of s, when it is newer
// than the running ergon, and reports whether it started one. A release of a later major version
// counts only for major. The release runs through [tool.Runner.RunRelease] in the cache of ergon
// tool run, with upgradeEnv set to the release, and s records its exit status. launch starts
// nothing for a build of ergon without a release, and for a release that an upgrade started. It
// returns the error of the lookup of the release, of its checksums.txt, and of the runner, which
// wraps [tool.ErrInstall] for an archive that does not install.
func (s *session) launch(ctx context.Context, cmd *cobra.Command, major bool) (bool, error) {
	if s.release == development || s.getenv(upgradeEnv) != "" {
		return false, nil
	}
	client := &http.Client{Transport: s.transport, Timeout: downloadTimeout}
	resolver := pin.Resolver{
		Client: client, Now: s.now, Registries: pin.Registries{GoProxy: proxy(s.getenv(proxyEnv))}, Major: major,
	}
	res, err := resolver.Resolve(ctx, &pin.Pin{
		Kind: pin.KindModule, Key: ergonModule, Name: ergonModule, Version: "v" + s.release,
	})
	if err != nil {
		return false, err
	}
	if res.Major != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s: ergon %s is a release of a later major version, which --%s takes\n",
			program, strings.TrimPrefix(res.Major.Version, "v"), majorFlag)
	}
	if res.Next == nil {
		return false, nil
	}
	server := cmp.Or(s.getenv(serverEnv), forge.DefaultServer)
	newest := ergonBinary{Version: strings.TrimPrefix(res.Next.Version, "v"), server: strings.TrimSuffix(server, "/")}
	digest, err := checksum(ctx, client, &newest, s.platform)
	if err != nil {
		return false, err
	}
	newest.SHA256 = map[option.Platform]string{s.platform: digest}
	cache, err := s.cacheDir()
	if err != nil {
		return false, fmt.Errorf("cli: find the cache directory: %w", err)
	}
	runner := tool.Runner{
		Client:   client,
		Stdin:    cmd.InOrStdin(),
		Stdout:   cmd.OutOrStdout(),
		Stderr:   cmd.ErrOrStderr(),
		Cache:    filepath.Join(cache, program, "tools"),
		Dir:      s.dir,
		Platform: s.platform,
		Env:      append(append([]string{}, s.env...), upgradeEnv+"="+newest.Version),
	}
	s.status, err = runner.RunRelease(ctx, program, newest, s.args)
	return true, err
}

// upgrade brings the repository of s to the baseline of the running ergon with
// [baseline.Repository.Sync] and force, and writes a line of each file that it wrote or removed and
// of each override to w. It returns them with the managed files that the sync left. It returns an
// error for a lock that a build of ergon without a release wrote while a release runs, the error of
// the sync, which is a [baseline.ConflictError] for a managed file that was edited by hand, and the
// error of the overrides.
func (s *session) upgrade(w io.Writer, force bool) (upgraded, error) {
	var u upgraded
	err := s.open(s.dir, func(r *baseline.Repository) error {
		if s.release != development {
			wrote, err := lockRelease(s.dir)
			if err != nil {
				return err
			}
			if wrote == development {
				return fmt.Errorf("cli: a build of ergon without a release wrote %s, so run ergon init sync with that "+
					"build", lock.Path)
			}
		}
		var synced error
		u.changes, synced = r.Sync(nil, baseline.Options{Force: force})
		for _, c := range u.changes {
			fmt.Fprintf(w, "%s %s\n", c.Action, c.Path)
		}
		if conflict, ok := errors.AsType[*baseline.ConflictError](synced); ok {
			u.conflicts = conflict.Paths
		} else if synced != nil {
			return synced
		}
		var err error
		if u.overrides, err = r.Overrides(); err != nil {
			return err
		}
		for _, o := range u.overrides {
			fmt.Fprintf(w, "override %s %s, the baseline is %s\n", o.Key, o.Version, o.Baseline)
		}
		return synced
	})
	return u, err
}

// checksum returns the SHA-256 of the asset of b for platform, which the checksums.txt of the release
// of b states, through client under ctx. It returns an error for a request that fails, a status other
// than 200, and a checksums.txt without the asset.
func checksum(ctx context.Context, client *http.Client, b *ergonBinary, platform option.Platform) (string, error) {
	asset, _ := b.Asset(platform)
	name := path.Base(asset.URL)
	address := strings.TrimSuffix(asset.URL, name) + checksumsFile
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", fmt.Errorf("cli: GET %s: %w", address, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cli: GET %s: %w", address, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cli: GET %s: %s", address, resp.Status)
	}
	lines := bufio.NewScanner(resp.Body)
	for lines.Scan() {
		// A line without the separator cuts to an empty file, which is never the name of an asset.
		if digest, file, _ := strings.Cut(lines.Text(), "  "); file == name {
			return digest, nil
		}
	}
	if err := lines.Err(); err != nil {
		return "", fmt.Errorf("cli: read %s: %w", address, err)
	}
	return "", fmt.Errorf("cli: %s has no line for %s", address, name)
}

// proxy returns the first module proxy of goproxy, the list of GOPROXY, that is a URL of http or
// https, without a slash at its end, and proxy.golang.org for a list without one.
func proxy(goproxy string) string {
	for entry := range strings.FieldsFuncSeq(goproxy, func(r rune) bool { return r == ',' || r == '|' }) {
		if strings.HasPrefix(entry, "https://") || strings.HasPrefix(entry, "http://") {
			return strings.TrimSuffix(entry, "/")
		}
	}
	return defaultProxy
}

// lockRelease returns the release of ergon that wrote the lock of the repository at root, and the
// empty string for a repository without a lock, which the sync reports. It returns the error of a
// lock that does not read, and of [lock.Decode].
func lockRelease(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(lock.Path)))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("cli: read %s: %w", lock.Path, err)
	}
	l, err := lock.Decode(data)
	return l.Ergon, err
}

// upgradeBody returns the body of the pull request of an upgrade to release: the files that the sync
// of u changed, and the overrides and the conflicts of u, each list under its heading.
func upgradeBody(release string, u *upgraded) string {
	lines := []string{fmt.Sprintf(upgradeIntro, release), "", "# Files", ""}
	for _, c := range u.changes {
		lines = append(lines, "- "+string(c.Action)+" `"+c.Path+"`")
	}
	if len(u.overrides) > 0 {
		lines = append(lines, "", "# Overrides", "", overridesIntro, "")
		for _, o := range u.overrides {
			lines = append(lines, "- `"+o.Key+"`: "+o.Version+", where the baseline has "+o.Baseline)
		}
	}
	if len(u.conflicts) > 0 {
		lines = append(lines, "", "# Conflicts", "", conflictsIntro, "")
		for _, path := range u.conflicts {
			lines = append(lines, "- `"+path+"`")
		}
	}
	return strings.Join(lines, "\n")
}
