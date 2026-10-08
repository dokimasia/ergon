// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline

import (
	"embed"
	"fmt"
	"io/fs"
	"path"

	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/licenses"
)

// Name is the name of the producer of the license files in the lock, and of its section of
// .ergon.yaml.
const Name = "license"

// The names of the license files of a directory, as the templates name them at the root.
const (
	licenseFile = "LICENSE"
	noticeFile  = "NOTICE"
)

// templates are the templates of the license files, under managed/.
//
//go:embed all:templates
var templates embed.FS

// texts are the license files that the templates read as .Data.
type texts struct {
	// Text is the text of LICENSE.
	Text string

	// Notice is the text of NOTICE, or empty for a license without one.
	Notice string
}

// Producer renders the license files of a repository: LICENSE, and NOTICE for Apache-2.0, at the
// root and in each directory of the section license. Its zero value is ready to use, and it is safe
// for concurrent use.
type Producer struct{}

var (
	_ language.Producer     = Producer{}
	_ language.Configurable = Producer{}
	_ language.Calculator   = Producer{}
	_ language.Contributor  = Producer{}
	_ language.Placer       = Producer{}
)

// Templates returns the templates of the license files: LICENSE and NOTICE under managed/. The
// template of NOTICE renders no byte for a license without one, so the repository has none.
func (Producer) Templates() fs.FS {
	// templates has the directory templates, so Sub returns no error.
	sub, _ := fs.Sub(templates, "templates")
	return sub
}

// Options returns the section license at the baseline: no parameter of BUSL-1.1, no directory, no
// comment style, no excluded path, and a limit of 10 minutes for the job license. ergon init writes
// the owner and the license from its answers.
func (Producer) Options() language.Options {
	c := &licenses.Config{}
	c.CI.Timeout = 10
	return c
}

// Data returns the texts of LICENSE and NOTICE for a and o, as [licenses.Text] renders them, which
// the templates read as .Data. It takes the options at the baseline, with the owner and the license
// of a, when o is not the section license. It returns the error of license.Text, such as the error
// for BUSL-1.1 with an empty parameter.
func (Producer) Data(a *language.Answers, o language.Options, _ *workflow.Contribution) (any, error) {
	c := own(o, a)
	text, notice, err := licenses.Text(c, licenses.Holder{
		Owner:      c.Owner,
		Name:       a.Name,
		Repository: string(a.Repository),
		Year:       a.Year,
	})
	if err != nil {
		return nil, err
	}
	return texts{Text: string(text), Notice: string(notice)}, nil
}

// Files returns the LICENSE of each directory of the section license, and its NOTICE for
// Apache-2.0, as [licenses.Text] renders them for the license and the parameters of the directory,
// with the owner of the section and the name, the repository and the year of a. It takes the
// options at the baseline, which list no directory, when o is not the section license. It returns
// the error of licenses.Text, such as the error for BUSL-1.1 with an empty parameter, with the path
// of the directory.
func (Producer) Files(a *language.Answers, o language.Options, _ *workflow.Contribution) ([]language.File, error) {
	c := own(o, a)
	h := licenses.Holder{Owner: c.Owner, Name: a.Name, Repository: string(a.Repository), Year: a.Year}
	files := make([]language.File, 0, len(c.Directories))
	for _, d := range c.Directories {
		text, notice, err := licenses.Text(&licenses.Config{SPDX: d.SPDX, Parameters: d.Parameters}, h)
		if err != nil {
			return nil, fmt.Errorf("baseline: the license of the directory %s: %w", d.Path, err)
		}
		files = append(files, language.File{Path: path.Join(d.Path, licenseFile), Content: text})
		if notice != nil {
			files = append(files, language.File{Path: path.Join(d.Path, noticeFile), Content: notice})
		}
	}
	return files, nil
}

// Contribution returns the job license of ci.yml for o, and for the options at the baseline when o
// is not the section license. The job checks the license headers of the repository with ergon
// license check on the Linux runner, because the result does not depend on the system.
func (Producer) Contribution(o language.Options) workflow.Contribution {
	return workflow.Contribution{Jobs: []workflow.Job{{
		ID:          Name,
		Name:        "License",
		Text:        true,
		Timeout:     own(o, &language.Answers{}).CI.Timeout,
		Permissions: map[string]string{"contents": "read"},
		Ergon:       true,
		Steps:       []workflow.Step{{Name: "Check the license headers", Run: []string{"ergon license check"}}},
	}}}
}

// own returns o as the section license, and otherwise the options at the baseline with the owner and
// the license of a.
func own(o language.Options, a *language.Answers) *licenses.Config {
	if c, ok := o.(*licenses.Config); ok {
		return c
	}
	c, _ := Producer{}.Options().(*licenses.Config)
	c.Owner, c.SPDX = a.Owner, a.License
	return c
}
