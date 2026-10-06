// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package common

import (
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.dokimi.dev/ergon/core/language"
)

// Name is the name of the producer of the common files in the lock.
const Name = "common"

// The licenses that the common files render, by their SPDX identifier.
const (
	// MIT is the MIT license, whose text names the owner and the year.
	MIT = "MIT"

	// Apache is the Apache License 2.0, whose NOTICE names the owner and the year.
	Apache = "Apache-2.0"
)

// Commitlint is the module and the release of commitlint, which checks each commit message against
// .commitlint.yaml: in the commit-msg hook of .pre-commit-config.yaml, and in the job commits of
// the workflow of the gate.
const Commitlint = "github.com/conventionalcommit/commitlint@v0.12.0"

// The paths of the common files that [Initializer] renders apart from its table of templates.
const (
	license = "LICENSE"
	notice  = "NOTICE"
	config  = ".ergon.yaml"
)

// ErrInvalidAnswer is the error for an answer that the common files cannot render: a name, an
// owner or a security contact that is empty or longer than one line, a license other than MIT and
// Apache-2.0, a year outside 1 to 9999, or a repository that is not owner/name on GitHub.
var ErrInvalidAnswer = errors.New("common: invalid answer")

// The templates of the common files, which mirror their paths in the repository.
var (
	//go:embed templates/managed/.commitlint.yaml.tmpl
	commitlint []byte

	//go:embed templates/managed/.editorconfig.tmpl
	editorconfig []byte

	//go:embed templates/managed/.gitattributes.tmpl
	gitattributes []byte

	//go:embed templates/managed/.gitignore.tmpl
	gitignore []byte

	//go:embed templates/managed/.markdownlint.yml.tmpl
	markdownlint []byte

	//go:embed templates/managed/.pre-commit-config.yaml.tmpl
	precommit []byte

	//go:embed templates/managed/Makefile.tmpl
	makefile []byte

	//go:embed templates/managed/CODE_OF_CONDUCT.md.tmpl
	conduct []byte

	//go:embed templates/seeded/.changeset/config.json.tmpl
	changesetConfig []byte

	//go:embed templates/seeded/.changeset/README.md.tmpl
	changesetReadme []byte

	//go:embed templates/seeded/README.md.tmpl
	readme []byte

	//go:embed templates/seeded/CONTRIBUTING.md.tmpl
	contributing []byte

	//go:embed templates/seeded/SECURITY.md.tmpl
	security []byte

	//go:embed templates/seeded/docs/README.md.tmpl
	docs []byte

	//go:embed templates/seeded/docs/adr/README.md.tmpl
	adr []byte

	//go:embed templates/seeded/docs/architecture/README.md.tmpl
	architecture []byte

	//go:embed templates/seeded/docs/rfc/README.md.tmpl
	rfc []byte

	//go:embed templates/seeded/docs/roadmap/README.md.tmpl
	roadmap []byte

	//go:embed templates/licenses/MIT.tmpl
	mit []byte

	//go:embed templates/licenses/Apache-2.0.tmpl
	apache []byte

	//go:embed templates/licenses/NOTICE.tmpl
	noticeText []byte
)

// templates are the common files that every repository has, in the order of their paths in
// templates/. The template of a file is its Content, and the template of the first fragment of a
// shared file is its Fragment.
var templates = []language.File{
	{Path: ".commitlint.yaml", Class: language.Managed, Content: commitlint},
	{Path: language.EditorConfig, Class: language.Managed, Fragment: editorconfig},
	{Path: language.GitAttributes, Class: language.Managed, Fragment: gitattributes},
	{Path: language.GitIgnore, Class: language.Managed, Fragment: gitignore},
	{Path: ".markdownlint.yml", Class: language.Managed, Content: markdownlint},
	{Path: ".pre-commit-config.yaml", Class: language.Managed, Content: precommit},
	{Path: language.Makefile, Class: language.Managed, Fragment: makefile},
	{Path: "CODE_OF_CONDUCT.md", Class: language.Managed, Content: conduct},
	{Path: ".changeset/config.json", Class: language.Seeded, Content: changesetConfig},
	{Path: ".changeset/README.md", Class: language.Seeded, Content: changesetReadme},
	{Path: "README.md", Class: language.Seeded, Content: readme},
	{Path: "CONTRIBUTING.md", Class: language.Seeded, Content: contributing},
	{Path: "SECURITY.md", Class: language.Seeded, Content: security},
	{Path: "docs/README.md", Class: language.Seeded, Content: docs},
	{Path: "docs/adr/README.md", Class: language.Seeded, Content: adr},
	{Path: "docs/architecture/README.md", Class: language.Seeded, Content: architecture},
	{Path: "docs/rfc/README.md", Class: language.Seeded, Content: rfc},
	{Path: "docs/roadmap/README.md", Class: language.Seeded, Content: roadmap},
}

// Initializer renders the common files of a repository. Its zero value is ready to use, and it is
// safe for concurrent use.
type Initializer struct{}

var _ language.Initializer = Initializer{}

// Files returns the common files for a: the files of the templates, the LICENSE of a.License,
// the NOTICE of the Apache License 2.0, and the name and the license of .ergon.yaml. It returns an
// error that wraps [ErrInvalidAnswer] for an answer that the files cannot render.
func (Initializer) Files(a *language.Answers) ([]language.File, error) {
	if err := validate(a); err != nil {
		return nil, err
	}
	replacer := strings.NewReplacer(
		"{{name}}", a.Name,
		"{{owner}}", a.Owner,
		"{{license}}", a.License,
		"{{year}}", strconv.Itoa(a.Year),
		"{{repository}}", string(a.Repository),
		"{{security-contact}}", a.SecurityContact,
		"{{commitlint}}", Commitlint,
	)
	// fill returns the template text with the answers in place, and nil for no template.
	fill := func(text []byte) []byte {
		if text == nil {
			return nil
		}
		return []byte(replacer.Replace(string(text)))
	}
	files := make([]language.File, 0, len(templates)+3)
	for _, t := range templates {
		t.Content, t.Fragment = fill(t.Content), fill(t.Fragment)
		files = append(files, t)
	}
	text := mit
	if a.License == Apache {
		text = apache
		files = append(files, language.File{Path: notice, Class: language.Managed, Content: fill(noticeText)})
	}
	files = append(files,
		language.File{Path: license, Class: language.Managed, Content: fill(text)},
		language.File{Path: config, Class: language.Configured, Content: []byte(
			"name: " + strconv.Quote(a.Name) + "\nlicense:\n" +
				"  owner: " + strconv.Quote(a.Owner) + "\n" +
				"  spdx: " + strconv.Quote(a.License) + "\n",
		)},
	)
	return files, nil
}

// validate returns an error that wraps [ErrInvalidAnswer] for the first answer of a that the
// common files cannot render, and nil when they render every answer.
func validate(a *language.Answers) error {
	lines := []struct{ name, value string }{
		{name: "name", value: a.Name},
		{name: "owner", value: a.Owner},
		{name: "security contact", value: a.SecurityContact},
	}
	for _, l := range lines {
		if strings.TrimSpace(l.value) == "" || strings.ContainsAny(l.value, "\r\n") {
			return fmt.Errorf("%w: the %s must be one line of text", ErrInvalidAnswer, l.name)
		}
	}
	if a.License != MIT && a.License != Apache {
		return fmt.Errorf("%w: license %q, which is neither %s nor %s", ErrInvalidAnswer, a.License, MIT, Apache)
	}
	if a.Year < 1 || a.Year > 9999 {
		return fmt.Errorf("%w: year %d, which is not between 1 and 9999", ErrInvalidAnswer, a.Year)
	}
	if !a.Repository.Valid() {
		return fmt.Errorf("%w: repository %q, which is not owner/name", ErrInvalidAnswer, a.Repository)
	}
	return nil
}
