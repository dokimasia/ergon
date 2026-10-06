// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package github

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/service/baseline/common"
)

// Name is the name of the producer of the GitHub files in the lock.
const Name = "github"

// The actions that the workflows run besides [language.Checkout], each pinned to the commit of a
// release, with the release in a comment.
const (
	setupGo           = "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0"
	markdownlint      = "DavidAnson/markdownlint-cli2-action@21c1be1b93ad9ed58fa840aacc3f279cde2a72ff # v24.2.0"
	dependencyReview  = "actions/dependency-review-action@a1d282b36b6f3519aa1f3fc636f609c47dddb294 # v5.0.0"
	scorecard         = "ossf/scorecard-action@2d1146689b8cda280b9bc96326124645441f03bc # v2.4.4"
	codeqlInit        = "github/codeql-action/init@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2 # v4.38.2"
	codeqlAnalyze     = "github/codeql-action/analyze@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2 # v4.38.2"
	codeqlUploadSarif = "github/codeql-action/upload-sarif@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2 # v4.38.2"
)

// makeVersion is the release of GNU make that the action setup-make installs from Chocolatey on a
// Windows runner.
const makeVersion = "4.4.1"

// ErrInvalidAnswer is the error for an answer that the GitHub files cannot render: a repository
// that is not owner/name on GitHub.
var ErrInvalidAnswer = errors.New("github: invalid answer")

// The templates of the GitHub files, which mirror their paths under .github in the repository.
var (
	//go:embed templates/managed/ISSUE_TEMPLATE/bug.yml.tmpl
	bug []byte

	//go:embed templates/managed/ISSUE_TEMPLATE/config.yml.tmpl
	issueConfig []byte

	//go:embed templates/managed/ISSUE_TEMPLATE/feature.yml.tmpl
	feature []byte

	//go:embed templates/managed/PULL_REQUEST_TEMPLATE.md.tmpl
	pullRequest []byte

	//go:embed templates/managed/actions/setup-ergon/action.yml.tmpl
	setupErgon []byte

	//go:embed templates/managed/actions/setup-make/action.yml.tmpl
	setupMake []byte

	//go:embed templates/managed/dependabot.yml.tmpl
	dependabot []byte

	//go:embed templates/managed/workflows/baseline.yml.tmpl
	baselineWorkflow []byte

	//go:embed templates/managed/workflows/ci.yml.tmpl
	ci []byte

	//go:embed templates/managed/workflows/codeql.yml.tmpl
	codeql []byte

	//go:embed templates/managed/workflows/security.yml.tmpl
	security []byte

	//go:embed templates/seeded/CODEOWNERS.tmpl
	codeowners []byte
)

// templates are the GitHub files that every repository has, in the order of their paths in
// templates/. The template of a file is its Content, and the template of the first fragment of a
// shared file is its Fragment.
var templates = []language.File{
	{Path: ".github/ISSUE_TEMPLATE/bug.yml", Class: language.Managed, Content: bug},
	{Path: ".github/ISSUE_TEMPLATE/config.yml", Class: language.Managed, Content: issueConfig},
	{Path: ".github/ISSUE_TEMPLATE/feature.yml", Class: language.Managed, Content: feature},
	{Path: ".github/PULL_REQUEST_TEMPLATE.md", Class: language.Managed, Content: pullRequest},
	{Path: ".github/actions/setup-ergon/action.yml", Class: language.Managed, Content: setupErgon},
	{Path: ".github/actions/setup-make/action.yml", Class: language.Managed, Content: setupMake},
	{Path: language.Dependabot, Class: language.Managed, Fragment: dependabot},
	{Path: ".github/workflows/baseline.yml", Class: language.Managed, Content: baselineWorkflow},
	{Path: language.CI, Class: language.Managed, Fragment: ci},
	{Path: ".github/workflows/codeql.yml", Class: language.Managed, Content: codeql},
	{Path: language.Security, Class: language.Managed, Fragment: security},
	{Path: ".github/CODEOWNERS", Class: language.Seeded, Content: codeowners},
}

// Initializer renders the GitHub files of a repository. Its zero value is ready to use, and it is
// safe for concurrent use.
type Initializer struct{}

var _ language.Initializer = Initializer{}

// Files returns the GitHub files for a: the workflows, the actions setup-ergon and setup-make,
// the configuration of Dependabot, the issue forms, the template of a pull request, and
// CODEOWNERS. The workflows and the configuration of Dependabot are shared files, to which the
// languages add their jobs and their package managers.
//
// A template names the repository as {{repository}}, an action as {{checkout}}, {{setup-go}},
// {{markdownlint}}, {{dependency-review}}, {{scorecard}}, {{codeql-init}}, {{codeql-analyze}} or
// {{codeql-upload-sarif}}, the runner of Linux as {{linux}}, the matrix of the three systems as
// {{runners}}, the release of Go that installs commitlint, [language.Go], as {{go-version}}, the
// release of commitlint as {{commitlint}}, and the release of GNU make as {{make-version}}. Files
// returns an error that wraps [ErrInvalidAnswer] for a repository that is not owner/name.
func (Initializer) Files(a *language.Answers) ([]language.File, error) {
	if !a.Repository.Valid() {
		return nil, fmt.Errorf("%w: repository %q, which is not owner/name", ErrInvalidAnswer, a.Repository)
	}
	replacer := strings.NewReplacer(
		"{{repository}}", string(a.Repository),
		"{{checkout}}", language.Checkout,
		"{{linux}}", language.Linux,
		"{{runners}}", language.Runners,
		"{{make-version}}", makeVersion,
		"{{setup-go}}", setupGo,
		"{{markdownlint}}", markdownlint,
		"{{dependency-review}}", dependencyReview,
		"{{scorecard}}", scorecard,
		"{{codeql-init}}", codeqlInit,
		"{{codeql-analyze}}", codeqlAnalyze,
		"{{codeql-upload-sarif}}", codeqlUploadSarif,
		"{{go-version}}", language.Go,
		"{{commitlint}}", common.Commitlint,
	)
	// fill returns the template text with the placeholders in place, and nil for no template.
	fill := func(text []byte) []byte {
		if text == nil {
			return nil
		}
		return []byte(replacer.Replace(string(text)))
	}
	files := make([]language.File, 0, len(templates))
	for _, t := range templates {
		t.Content, t.Fragment = fill(t.Content), fill(t.Fragment)
		files = append(files, t)
	}
	return files, nil
}
