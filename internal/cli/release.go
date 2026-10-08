// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.dokimi.dev/ergon/core/changeset"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/forge"
	"go.dokimi.dev/ergon/service/release"
	"go.dokimi.dev/ergon/service/vcs"
)

// The flags of the release commands.
const (
	sinceFlag    = "since"
	outputFlag   = "output"
	verboseFlag  = "verbose"
	dryRunFlag   = "dry-run"
	planFlag     = "from-publish-plan"
	packDirFlag  = "from-pack-dir"
	outDirFlag   = "out-dir"
	noTagFlag    = "no-git-tag"
	messageFlag  = "message"
	bumpFlag     = "bump"
	emptyFlag    = "empty"
	openFlag     = "open"
	titleFlag    = "title"
	workflowFlag = "workflow"
)

// The variables of the environment that the release commands read, as GitHub Actions sets them, and
// the editors of ergon release add --open.
const (
	authEnv       = "GITHUB_TOKEN"
	repositoryEnv = "GITHUB_REPOSITORY"
	apiEnv        = "GITHUB_API_URL"
	graphqlEnv    = "GITHUB_GRAPHQL_URL"
	serverEnv     = "GITHUB_SERVER_URL"
	outputEnv     = "GITHUB_OUTPUT"
	actionsEnv    = "GITHUB_ACTIONS"
	visualEnv     = "VISUAL"
	editorEnv     = "EDITOR"
)

// The outputs of the release commands in GitHub Actions, as changesets/action names the outputs
// mode, published and published-packages.
const (
	modeOutput      = "mode"
	publishedOutput = "published"
	packagesOutput  = "published-packages"
	pullOutput      = "pull-request-number"
)

// apiTimeout bounds a request to the API of GitHub.
const apiTimeout = time.Minute

// remote is the remote that ergon release publish and ergon release git-tag push the tags of a
// workstation to.
const remote = "origin"

// outputPerm is the mode of a file that a release command writes, before the umask.
const outputPerm fs.FileMode = 0o644

// uncoveredHint follows the packages that ergon release status reports as changed without a
// changeset.
const uncoveredHint = "add a changeset with ergon release add, or with --empty for a change that releases nothing"

// skippedLine is the line of a command that proposes a pull request for the commit of HEAD, and
// skips it because the base branch, the second argument, moved past the commit, the first.
const skippedLine = "skipped %s, which is no longer the head of %s\n"

// The defaults of the flags of the release commands.
const (
	// defaultTitle is the title of a version pull request and the message of its commits, which
	// the commit rules of the baseline accept.
	defaultTitle = "chore: version packages"

	// defaultPackDir is the directory of the artifacts of ergon release pack and publish.
	defaultPackDir = "dist"

	// defaultWorkflow is the workflow of the gate, whose passed run ergon release ci verify finds.
	defaultWorkflow = "ci.yml"
)

// The help texts of the release commands.
const (
	releaseShort = "Plan, write and publish the releases of the packages of the repository"
	releaseLong  = `ergon release releases the packages of the repository with the changesets of
.changeset and the configuration .changeset/config.json, in the format of
changesets: a pull request adds a changeset that names the packages it changes
with their bumps, the version pull request writes the new versions and the
changelogs, and its merge publishes and tags the packages. The packages are the
packages that each toolchain of ergon discovers, such as the modules of go.work.`

	changesetShort = "Add a changeset"
	changesetLong  = `ergon release add writes a changeset into .changeset: the packages that --bump
names, each with its bump, and the summary of --message as the entry of their
changelogs. --empty writes a changeset without packages, which satisfies ergon
release status for a change that releases nothing. --open opens the changeset
in the editor of VISUAL or EDITOR after it writes it.`
	changesetExample = `  ergon release add --bump go.dokimi.dev/ergon/core=minor -m "Add the Unit type."`

	statusShort = "Report the changes of the branch that no changeset releases"
	statusLong  = `ergon release status writes the release plan of the changesets that the branch
added since --since, the base branch by default. It reports each package that
the branch changed without a changeset that names it, and each requirement of a
package on another package of the repository that excludes the current version
of the other. The exit status is 1 when it reports either. --output writes the
plan as JSON into a file.`

	versionShort = "Write the new versions, the requirements and the changelogs"
	versionLong  = `ergon release version writes the release plan of the changesets into the
repository: the new version of each package, the rewritten requirements of its
dependents, the entries of the changelogs and the refreshed lockfiles. It
removes the changesets that it consumed, and restores every file when a step
fails. Without changesets it rewrites the lockfiles that record the earlier
content of a package whose release waits for its publish. --dry-run writes the
plan and changes nothing.`

	publishPlanShort = "Write the packages that a publish uploads or tags"
	publishPlanLong  = `ergon release publish-plan writes the publish plan as JSON: each package whose
registry lacks its version, and each package whose tag is its release and is
missing, in chunks of dependency order. --output writes the plan into a file.`

	packShort = "Build the artifacts of the packages of a publish plan"
	packLong  = `ergon release pack builds the artifacts of each package of the publish plan
that its registry receives as an artifact, into --out-dir.`

	publishShort = "Publish and tag the packages of a publish plan"
	publishLong  = `ergon release publish uploads each package of the publish plan whose registry
lacks its version, and then tags each package at HEAD with the section of its
changelog as the notes. On a workstation it creates annotated tags and pushes
them to origin in one push. In GitHub Actions it creates the tags and the
GitHub Releases through the API of GitHub. It refuses a plan whose lockfiles
record other content of its packages than the working tree. --no-git-tag
creates no tag.`

	gitTagShort = "Tag the packages whose tags are missing"
	gitTagLong  = `ergon release git-tag creates the annotated tag of each package whose tag at
its version is missing, at HEAD, and pushes the tags to origin in one push. It
refuses a plan whose lockfiles record other content of its packages than the
working tree.`

	ciShort = "Run the jobs of the release workflow"
	ciLong  = `ergon release ci runs the steps of the jobs of the release workflow in GitHub
Actions, and writes their outputs into the file of GITHUB_OUTPUT.`

	selectModeShort = "Choose the job that the release workflow runs"
	selectModeLong  = `ergon release ci select-mode writes the mode of the release workflow: version
for a repository with changesets or with lockfiles that record the earlier
content of a package of the publish plan, publish for a publish plan with a
package, and none otherwise. It writes each such lockfile. --output writes the
publish plan into a file for the job publish.`

	ciVersionShort = "Write the release and open the version pull request"
	ciVersionLong  = `ergon release ci version writes the release plan of the changesets as ergon
release version does. It then commits the changes on the branch
ergon-release/<base> through the API of GitHub, which signs the commits, and
opens or updates the version pull request into the base branch. Without
changesets it proposes the lockfiles that ergon release version rewrites, and
otherwise changes nothing. It skips a commit of HEAD that is no longer the head
of the base branch, and leaves the pull request to the run of the newer head.`

	ciVerifyShort = "Verify that a CI run passed on the content of the commit"
	ciVerifyLong  = `ergon release ci verify finds a run of the workflow --workflow that passed on the
content of the commit of HEAD, through the API of GitHub: a run of the commit
itself, such as the run of its push or of its merge group, or a run of the head
of a pull request that merged the commit with the same tree. It reads each run
once and waits for none. The exit status is 1 without such a run.`
)

// releaseRepository is a repository that a release command works on.
type releaseRepository struct {
	// graph is the graph of the packages of the repository.
	graph *release.Graph

	// root is the root of the repository.
	root string

	// config is the configuration of the release.
	config release.Config
}

// untagged is the releaser of a publish without tags: it reads no tag and creates none.
type untagged struct{}

// Tag reports that the repository has no tag.
func (untagged) Tag(context.Context, string) (string, bool, error) {
	return "", false, nil
}

// Release creates nothing.
func (untagged) Release(context.Context, string, string, string, bool) error {
	return nil
}

// Finish does nothing.
func (untagged) Finish(context.Context) error {
	return nil
}

// releaseCommand returns ergon release with its subcommands, which work on the repository of the
// working directory of s under ctx.
func releaseCommand(ctx context.Context, s *session) *cobra.Command {
	ci := group(s, "ci", ciShort, ciLong, selectModeCommand(ctx, s), ciVerifyCommand(ctx, s), ciVersionCommand(ctx, s))
	return group(s, "release", releaseShort, releaseLong, changesetCommand(ctx, s), statusCommand(ctx, s),
		versionCommand(ctx, s), publishPlanCommand(ctx, s), packCommand(ctx, s), publishCommand(ctx, s),
		gitTagCommand(ctx, s), ci)
}

// changesetCommand returns ergon release add, which writes a changeset of the packages of --bump
// with the summary of --message, and opens it in the editor for --open. It returns a [usageError]
// for a --bump that is not name=level and for a command line without --bump and without --empty,
// and the error of [release.NewPlan] for the changeset, which names the file and the line of a
// package that the repository does not have.
func changesetCommand(ctx context.Context, s *session) *cobra.Command {
	var bumps []string
	var summary string
	var empty, open bool
	cmd := &cobra.Command{
		Use:     "add",
		Short:   changesetShort,
		Long:    changesetLong,
		Example: changesetExample,
		Args:    usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(bumps) == 0 && !empty {
				return usageError{err: errors.New("cli: name the packages with --bump, or pass --empty")}
			}
			c := changeset.Changeset{Summary: strings.TrimSpace(summary)}
			for k, b := range bumps {
				name, level, ok := strings.Cut(b, "=")
				bump, err := version.ParseBump(level)
				if !ok || err != nil {
					return usageError{
						err: fmt.Errorf("cli: --bump %q, which is not <name>=<major|minor|patch|none>", b),
					}
				}
				// The front matter opens on the first line, so the release k is on the line k+2.
				c.Releases = append(c.Releases, changeset.Release{Name: name, Bump: bump, Line: k + 2})
			}
			r, err := s.releaseRepository(ctx, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if c.ID, err = release.ChangesetID(c.Summary, s.random); err != nil {
				return err
			}
			if _, err = release.NewPlan(r.graph, &r.config, []changeset.Changeset{c}); err != nil {
				return err
			}
			file, err := release.AddChangeset(r.root, &c)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", baseline.Wrote, file)
			if !open {
				return nil
			}
			return s.edit(ctx, cmd, filepath.Join(r.root, filepath.FromSlash(file)))
		},
	}
	flags := cmd.Flags()
	flags.StringArrayVar(&bumps, bumpFlag, nil, "release `name=level`, where level is major, minor, patch or none")
	flags.StringVarP(&summary, messageFlag, "m", "", "the `summary` of the changeset")
	flags.BoolVar(&empty, emptyFlag, false, "write a changeset without packages")
	flags.BoolVar(&open, openFlag, false, "open the changeset in the editor")
	return cmd
}

// statusCommand returns ergon release status, which writes the release plan of [release.NewStatus].
// It returns an error for a changed package without a changeset and for a requirement that excludes
// the current version of its package.
func statusCommand(ctx context.Context, s *session) *cobra.Command {
	var since, output string
	var verbose bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: statusShort,
		Long:  statusLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, err := s.releaseRepository(ctx, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			sets, err := release.ReadChangesets(r.root)
			if err != nil {
				return err
			}
			if since == "" {
				since = r.config.BaseBranch
			}
			st, err := release.NewStatus(ctx, r.root, r.graph, &r.config, sets, since)
			if err != nil {
				return err
			}
			writePlan(cmd.OutOrStdout(), &st.Plan, verbose)
			if err := writeJSON(output, &st.Plan); err != nil {
				return err
			}
			problems := make([]string, 0, len(st.Uncovered)+1+len(st.Broken))
			for _, name := range st.Uncovered {
				problems = append(problems, name+" changed without a changeset that names it")
			}
			if len(problems) > 0 {
				problems = append(problems, uncoveredHint)
			}
			problems = append(problems, st.Broken...)
			if len(problems) > 0 {
				return fmt.Errorf("cli: %s", strings.Join(problems, "; "))
			}
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&since, sinceFlag, "", "compare against `ref`, the base branch by default")
	flags.StringVar(&output, outputFlag, "", "write the release plan as JSON into `file`")
	flags.BoolVar(&verbose, verboseFlag, false, "list the changesets of each release")
	return cmd
}

// versionCommand returns ergon release version, which writes the release plan of the changesets
// with [release.Version], or writes the plan for --dry-run.
func versionCommand(ctx context.Context, s *session) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: versionShort,
		Long:  versionLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, plan, err := s.plan(ctx, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if dryRun {
				writePlan(cmd.OutOrStdout(), &plan, true)
				return nil
			}
			_, err = s.version(ctx, cmd, r, &plan)
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, dryRunFlag, false, "write the release plan and change nothing")
	return cmd
}

// publishPlanCommand returns ergon release publish-plan, which writes the publish plan of the
// repository as JSON into --output or the standard output.
func publishPlanCommand(ctx context.Context, s *session) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "publish-plan",
		Short: publishPlanShort,
		Long:  publishPlanLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, plan, err := s.publishPlan(ctx, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if output == "" {
				return writeJSONTo(cmd.OutOrStdout(), &plan)
			}
			return writeJSON(output, &plan)
		},
	}
	cmd.Flags().StringVar(&output, outputFlag, "", "write the publish plan into `file`")
	return cmd
}

// packCommand returns ergon release pack, which builds the artifacts of the publish plan of
// --from-publish-plan, or of the repository, into --out-dir with [release.Pack].
func packCommand(ctx context.Context, s *session) *cobra.Command {
	var from, outDir string
	cmd := &cobra.Command{
		Use:   "pack",
		Short: packShort,
		Long:  packLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, plan, err := s.planOf(ctx, cmd.ErrOrStderr(), from)
			if err != nil {
				return err
			}
			return release.Pack(ctx, r.root, r.graph, &plan, outDir)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&from, planFlag, "", "read the publish plan from `file`")
	flags.StringVar(&outDir, outDirFlag, defaultPackDir, "build the artifacts into `dir`")
	return cmd
}

// publishCommand returns ergon release publish, which publishes the publish plan of
// --from-publish-plan, or of the repository, with [release.Publish].
func publishCommand(ctx context.Context, s *session) *cobra.Command {
	var from, packDir, output string
	var noTag bool
	cmd := &cobra.Command{
		Use:   "publish",
		Short: publishShort,
		Long:  publishLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, plan, err := s.planOf(ctx, cmd.ErrOrStderr(), from)
			if err != nil {
				return err
			}
			releaser, err := s.releaser(cmd, r.root, noTag)
			if err != nil {
				return err
			}
			return s.publish(ctx, cmd, r, &plan, packDir, output, releaser)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&from, planFlag, "", "read the publish plan from `file`")
	flags.StringVar(&packDir, packDirFlag, defaultPackDir, "upload the artifacts of `dir`")
	flags.StringVar(&output, outputFlag, "", "write the released packages as JSON into `file`")
	flags.BoolVar(&noTag, noTagFlag, false, "create no tag")
	return cmd
}

// gitTagCommand returns ergon release git-tag, which tags each package of the publish plan of the
// repository with the git of the repository, and pushes the tags. git runs with the standard input
// and the standard error of the command, on which its programs prompt for the PIN and the touch of
// a key.
func gitTagCommand(ctx context.Context, s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "git-tag",
		Short: gitTagShort,
		Long:  gitTagLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, plan, err := s.publishPlan(ctx, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			for _, chunk := range plan.Plan {
				for k := range chunk {
					chunk[k].Kind = release.KindTagOnly
				}
			}
			releaser := &release.GitReleaser{
				Terminal: vcs.Terminal{Stdin: cmd.InOrStdin(), Stderr: cmd.ErrOrStderr()},
				Root:     r.root,
				Remote:   remote,
			}
			return s.publish(ctx, cmd, r, &plan, "", "", releaser)
		},
	}
}

// selectModeCommand returns ergon release ci select-mode, which writes the stale lockfiles of
// [release.Stale], the mode of the release workflow, its output mode, and the publish plan into
// --output.
func selectModeCommand(ctx context.Context, s *session) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "select-mode",
		Short: selectModeShort,
		Long:  selectModeLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, plan, err := s.publishPlan(ctx, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			sets, err := release.ReadChangesets(r.root)
			if err != nil {
				return err
			}
			stale, err := release.Stale(ctx, r.root, r.graph, &plan)
			if err != nil {
				return err
			}
			if err := writeJSON(output, &plan); err != nil {
				return err
			}
			for _, file := range stale {
				fmt.Fprintf(cmd.OutOrStdout(), "stale %s\n", file)
			}
			mode := release.SelectMode(sets, &plan, stale)
			fmt.Fprintf(cmd.OutOrStdout(), "mode %s\n", mode)
			return s.output(modeOutput, mode)
		},
	}
	cmd.Flags().StringVar(&output, outputFlag, "", "write the publish plan into `file`")
	return cmd
}

// ciVersionCommand returns ergon release ci version, which writes the release plan as ergon
// release version does, and proposes it with [release.Propose] through the API of GitHub. For a
// repository without changesets it proposes the lockfiles that [release.Lock] rewrites, and nothing
// when it rewrites none. For the [release.ErrMoved] of Propose it writes that it skipped the commit,
// and returns no error.
func ciVersionCommand(ctx context.Context, s *session) *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "version",
		Short: ciVersionShort,
		Long:  ciVersionLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, repo, err := s.forge()
			if err != nil {
				return err
			}
			r, plan, err := s.plan(ctx, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			head, err := vcs.Head(ctx, r.root)
			if err != nil {
				return err
			}
			wrote, err := s.version(ctx, cmd, r, &plan)
			if err != nil || !wrote {
				return err
			}
			p, err := release.NewProposal(ctx, r.root, r.config.BaseBranch, r.graph, &plan)
			if err != nil {
				return err
			}
			p.Repo, p.Head, p.Title = repo, head, title
			number, err := release.Propose(ctx, client, &p)
			if errors.Is(err, release.ErrMoved) {
				fmt.Fprintf(cmd.OutOrStdout(), skippedLine, head, p.Base)
				return nil
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pull request %d\n", number)
			return s.output(pullOutput, strconv.Itoa(number))
		},
	}
	cmd.Flags().StringVar(&title, titleFlag, defaultTitle, "the `title` of the pull request and its commits")
	return cmd
}

// ciVerifyCommand returns ergon release ci verify, which finds with [release.Gate] a run of the
// workflow of --workflow that passed on the content of the commit of HEAD, and writes its page. It
// returns the error of [release.Gate.Verify], which wraps [release.ErrGate] when no run passed.
func ciVerifyCommand(ctx context.Context, s *session) *cobra.Command {
	var workflow string
	cmd := &cobra.Command{
		Use:   "verify",
		Short: ciVerifyShort,
		Long:  ciVerifyLong,
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, repo, err := s.forge()
			if err != nil {
				return err
			}
			head, err := vcs.Head(ctx, s.root())
			if err != nil {
				return err
			}
			gate := release.Gate{Forge: client, Repo: repo, Workflow: workflow}
			page, err := gate.Verify(ctx, head)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s passed: %s\n", workflow, page)
			return nil
		},
	}
	cmd.Flags().StringVar(&workflow, workflowFlag, defaultWorkflow, "find a passed run of the workflow `file`")
	return cmd
}

// writePlan writes a line of each release of plan to w, with its level and its versions, and with
// verbose the changesets of the release.
func writePlan(w io.Writer, plan *release.Plan, verbose bool) {
	if len(plan.Releases) == 0 {
		fmt.Fprintln(w, "no release")
	}
	for k := range plan.Releases {
		r := &plan.Releases[k]
		fmt.Fprintf(w, "%s %s %s -> %s\n", r.Name, r.Bump, r.Old, r.New)
		if verbose {
			for _, id := range r.Changesets {
				fmt.Fprintf(w, "  %s\n", id)
			}
		}
	}
}

// writeJSONTo writes v to w as indented JSON. It returns the error of the write.
func writeJSONTo(w io.Writer, v any) error {
	// The plans of the package release encode.
	data, _ := json.MarshalIndent(v, "", "  ")
	if _, err := w.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("cli: write the JSON: %w", err)
	}
	return nil
}

// writeJSON writes v as indented JSON into the file named file, and nothing for an empty file. It
// returns the error of the write.
func writeJSON(file string, v any) error {
	// The plans of the package release encode.
	data, _ := json.MarshalIndent(v, "", "  ")
	return writeFile(file, append(data, '\n'))
}

// writeFile writes data into the file named file, and nothing for an empty file. It returns the
// error of the write.
func writeFile(file string, data []byte) error {
	if file == "" {
		return nil
	}
	if err := os.WriteFile(file, data, outputPerm); err != nil {
		return fmt.Errorf("cli: write %s: %w", file, err)
	}
	return nil
}

// relock rewrites the lockfiles of r that record the earlier content of a package of its publish
// plan with [release.Lock], writes the path of each file that it changed to the output of cmd, and
// reports whether it changed one. It writes the message of changesets when no lockfile is stale. It
// returns the error of the publish plan and of Lock.
func relock(ctx context.Context, cmd *cobra.Command, r *releaseRepository) (bool, error) {
	plan, err := publishPlanOf(ctx, r)
	if err != nil {
		return false, err
	}
	written, err := release.Lock(ctx, r.root, r.graph, &plan)
	if err != nil {
		return false, err
	}
	if len(written) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No unreleased changesets found.")
		return false, nil
	}
	for _, file := range written {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", baseline.Wrote, file)
	}
	return true, nil
}

// publishPlanOf returns the publish plan of r against the tags of git. It returns the error of git
// and of [release.NewPublishPlan].
func publishPlanOf(ctx context.Context, r *releaseRepository) (release.PublishPlan, error) {
	tags, err := vcs.Tags(ctx, r.root)
	if err != nil {
		return release.PublishPlan{}, err
	}
	return release.NewPublishPlan(ctx, r.graph, &r.config, tags)
}

// releaseRepository returns the repository of the working directory of s with its packages and
// .changeset/config.json, and writes each warning of the configuration to warnings. It returns the
// error of the discovery, of reading the configuration and of [release.ParseConfig].
func (s *session) releaseRepository(ctx context.Context, warnings io.Writer) (*releaseRepository, error) {
	root := s.root()
	g, err := release.Discover(ctx, s.catalog, root)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(release.ConfigPath)))
	if err != nil {
		return nil, fmt.Errorf("cli: read %s, which ergon init seeds: %w", release.ConfigPath, err)
	}
	c, err := release.ParseConfig(data, g)
	if err != nil {
		return nil, err
	}
	for _, w := range c.Warnings {
		fmt.Fprintf(warnings, "%s: warning: %s\n", program, w)
	}
	return &releaseRepository{graph: g, root: root, config: c}, nil
}

// plan returns the repository of s and the release plan of its changesets.
func (s *session) plan(ctx context.Context, warnings io.Writer) (*releaseRepository, release.Plan, error) {
	r, err := s.releaseRepository(ctx, warnings)
	if err != nil {
		return nil, release.Plan{}, err
	}
	sets, err := release.ReadChangesets(r.root)
	if err != nil {
		return nil, release.Plan{}, err
	}
	plan, err := release.NewPlan(r.graph, &r.config, sets)
	return r, plan, err
}

// publishPlan returns the repository of s and its publish plan against the tags of git.
func (s *session) publishPlan(ctx context.Context, warnings io.Writer) (
	*releaseRepository, release.PublishPlan, error,
) {
	r, err := s.releaseRepository(ctx, warnings)
	if err != nil {
		return nil, release.PublishPlan{}, err
	}
	plan, err := publishPlanOf(ctx, r)
	return r, plan, err
}

// planOf returns the repository of s and the publish plan of the JSON file from, or the publish
// plan of the repository for an empty from. It returns the error of reading and of decoding from.
func (s *session) planOf(ctx context.Context, warnings io.Writer, from string) (
	*releaseRepository, release.PublishPlan, error,
) {
	if from == "" {
		return s.publishPlan(ctx, warnings)
	}
	r, err := s.releaseRepository(ctx, warnings)
	if err != nil {
		return nil, release.PublishPlan{}, err
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return nil, release.PublishPlan{}, fmt.Errorf("cli: read the publish plan: %w", err)
	}
	var plan release.PublishPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, release.PublishPlan{}, fmt.Errorf("cli: read the publish plan %s: %w", from, err)
	}
	return r, plan, nil
}

// version writes plan into the repository r with [release.Version], with the host of GitHub of the
// environment of s for the links of the changelog of GitHub, writes a line of each file that it wrote
// and then of each file that it removed, and reports whether it wrote the plan. For a plan without
// changesets it rewrites the stale lockfiles of r with [relock]. It returns the error of the host,
// of Version and of relock.
func (s *session) version(
	ctx context.Context, cmd *cobra.Command, r *releaseRepository, plan *release.Plan,
) (bool, error) {
	host, err := s.host()
	if err != nil {
		return false, err
	}
	written, removed, err := release.Version(ctx, r.root, r.graph, &r.config, plan, host)
	if errors.Is(err, release.ErrNoChangesets) {
		return relock(ctx, cmd, r)
	}
	out := cmd.OutOrStdout()
	for _, file := range written {
		fmt.Fprintf(out, "%s %s\n", baseline.Wrote, file)
	}
	for _, file := range removed {
		fmt.Fprintf(out, "%s %s\n", baseline.Removed, file)
	}
	return err == nil, err
}

// publish publishes plan from the repository r at HEAD with releaser and the artifacts of packDir.
// It writes a line of each released package, the outputs published and published-packages, and the
// JSON of the released packages into output when it is not empty.
func (s *session) publish(
	ctx context.Context, cmd *cobra.Command, r *releaseRepository, plan *release.PublishPlan, packDir, output string,
	releaser release.Releaser,
) error {
	head, err := vcs.Head(ctx, r.root)
	if err != nil {
		return err
	}
	released, err := release.Publish(ctx, r.root, r.graph, plan, head, packDir, releaser)
	for _, p := range released {
		fmt.Fprintf(cmd.OutOrStdout(), "released %s@%s\n", p.Name, p.Version)
	}
	// A list of packages encodes, and a list of none encodes as [], as changesets/action writes it.
	packages, _ := json.Marshal(append([]release.Published{}, released...))
	return errors.Join(err,
		s.output(publishedOutput, strconv.FormatBool(len(released) > 0)),
		s.output(packagesOutput, string(packages)),
		writeFile(output, append(packages, '\n')))
}

// releaser returns the releaser of a publish in the repository at root: none for noTag, the API of
// GitHub in GitHub Actions, and otherwise the git of the repository with the remote origin and the
// standard input and the standard error of cmd, on which the programs of git prompt for the PIN and
// the touch of a key.
func (s *session) releaser(cmd *cobra.Command, root string, noTag bool) (release.Releaser, error) {
	switch {
	case noTag:
		return untagged{}, nil
	case s.getenv(actionsEnv) == "true":
		client, repo, err := s.forge()
		if err != nil {
			return nil, err
		}
		return release.ForgeReleaser{Forge: client, Repo: repo}, nil
	}
	return &release.GitReleaser{
		Terminal: vcs.Terminal{Stdin: cmd.InOrStdin(), Stderr: cmd.ErrOrStderr()},
		Root:     root,
		Remote:   remote,
	}, nil
}

// host returns the host of GitHub of the environment of s for the links of the changelog of GitHub,
// and a host without a forge for an environment without GITHUB_TOKEN.
func (s *session) host() (release.Host, error) {
	if s.getenv(authEnv) == "" {
		return release.Host{}, nil
	}
	client, repo, err := s.forge()
	if err != nil {
		return release.Host{}, err
	}
	return release.Host{Forge: client, Repo: repo, Server: client.Server()}, nil
}

// forge returns the client of GitHub of the environment of s and the repository of
// GITHUB_REPOSITORY. It returns an error for an environment without GITHUB_REPOSITORY, and the
// error of [forge.New] for one without GITHUB_TOKEN.
func (s *session) forge() (*forge.Client, string, error) {
	repo := s.getenv(repositoryEnv)
	if repo == "" {
		return nil, "", fmt.Errorf("cli: set %s to the repository on GitHub, as owner/name", repositoryEnv)
	}
	client, err := forge.New(&http.Client{Transport: s.transport, Timeout: apiTimeout}, forge.Config{
		Token: s.getenv(authEnv), API: s.getenv(apiEnv), GraphQL: s.getenv(graphqlEnv), Server: s.getenv(serverEnv),
	})
	return client, repo, err
}

// edit opens file in the editor of VISUAL or EDITOR, with the standard input and output of cmd. It
// returns an error for an environment without an editor, and the error of the editor.
func (s *session) edit(ctx context.Context, cmd *cobra.Command, file string) error {
	editor := strings.Fields(s.getenv(visualEnv))
	if len(editor) == 0 {
		editor = strings.Fields(s.getenv(editorEnv))
	}
	if len(editor) == 0 {
		return fmt.Errorf("cli: set %s or %s to open the changeset", visualEnv, editorEnv)
	}
	program, args := editor[0], slices.Concat(editor[1:], []string{file})
	run := exec.CommandContext(ctx, program, args...)
	run.Stdin, run.Stdout, run.Stderr = cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()
	run.Env = s.env
	if err := run.Run(); err != nil {
		return fmt.Errorf("cli: open the changeset in %s: %w", program, err)
	}
	return nil
}

// output appends the output name=value of a step of GitHub Actions to the file of GITHUB_OUTPUT,
// and writes nothing outside GitHub Actions. It returns the error of the write.
func (s *session) output(name, value string) error {
	file := s.getenv(outputEnv)
	if file == "" {
		return nil
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, outputPerm)
	if err == nil {
		_, err = fmt.Fprintf(f, "%s=%s\n", name, value)
		err = errors.Join(err, f.Close())
	}
	if err != nil {
		return fmt.Errorf("cli: write the output %s: %w", name, err)
	}
	return nil
}

// getenv returns the value of the variable key of the environment of s, the last value for a key
// that it sets twice, and the empty string for a variable that it does not set.
func (s *session) getenv(key string) string {
	value := ""
	for _, entry := range s.env {
		if k, v, ok := strings.Cut(entry, "="); ok && k == key {
			value = v
		}
	}
	return value
}
