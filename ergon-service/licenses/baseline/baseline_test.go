// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	"go.dokimi.dev/ergon/service/licenses"
	licensebaseline "go.dokimi.dev/ergon/service/licenses/baseline"
)

// name pins the name of the producer of the license files.
const name = "license"

// The key of the directories in the section license that ergon init new writes, and the same key
// with the directory sdk under Apache-2.0.
const (
	noDirectories = "  directories: []\n"
	sdkDirectory  = "  directories:\n    - path: sdk\n      spdx: Apache-2.0\n"
)

// busl are the parameters of BUSL-1.1 of the cases.
var busl = licenses.Parameters{
	LicensedWork:       "demo 1.0",
	AdditionalUseGrant: "None",
	ChangeDate:         "2030-01-01",
	ChangeLicense:      "Apache-2.0",
}

func TestBaseline(t *testing.T) {
	t.Parallel()

	t.Run("Name", func(t *testing.T) {
		t.Parallel()

		t.Run("is license", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, licensebaseline.Name, name, "Name")
		})
	})

	t.Run("Producer", func(t *testing.T) {
		t.Parallel()

		t.Run("Templates", func(t *testing.T) {
			t.Parallel()

			goldens := []struct {
				name string
				give spdx.ID
				want string
			}{
				{name: "renders LICENSE of MIT at the baseline", give: spdx.MIT, want: "mit"},
				{name: "renders LICENSE and NOTICE of Apache-2.0", give: spdx.Apache20, want: "apache"},
			}
			for _, tt := range goldens {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					a := baselinetest.Answers()
					a.License = tt.give
					dir := baselinetest.New(t, new(language.Catalog), a, producer())
					baselinetest.Hygiene(t, dir)
					golden.MatchTree(t, tt.want, os.DirFS(dir), golden.ShouldUpdate())
				})
			}

			t.Run("returns ErrInvalid from New for BUSL-1.1 without its parameters", func(t *testing.T) {
				t.Parallel()
				root, err := os.OpenRoot(t.TempDir())
				assert.NoError(t, err, "OpenRoot")
				t.Cleanup(func() { _ = root.Close() })
				r, err := baseline.Open(root, new(language.Catalog), baselinetest.Version, producer())
				assert.NoError(t, err, "Open")
				a := baselinetest.Answers()
				a.License = spdx.BUSL11
				_, err = r.New(a, baseline.Options{})
				assert.ErrorIs(t, err, option.ErrInvalid, "New")
			})
		})

		t.Run("Data", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the texts of the answers for options of another type", func(t *testing.T) {
				t.Parallel()
				got, err := licensebaseline.Producer{}.Data(baselinetest.Answers(), nil, &workflow.Contribution{})
				assert.NoError(t, err, "Data")
				assert.NotNil(t, got, "the data")
			})

			t.Run("returns the error of a license without its parameters", func(t *testing.T) {
				t.Parallel()
				a := baselinetest.Answers()
				a.License = spdx.BUSL11
				_, err := licensebaseline.Producer{}.Data(a, nil, &workflow.Contribution{})
				assert.ErrorIs(t, err, option.ErrInvalid, "Data")
			})
		})

		t.Run("Files", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the LICENSE of each directory and the NOTICE of one under Apache-2.0", func(t *testing.T) {
				t.Parallel()
				a := baselinetest.Answers()
				c := section(a)
				c.Directories = []licenses.Directory{
					{Path: "enterprise", SPDX: spdx.BUSL11, Parameters: busl},
					{Path: "sdk/go", SPDX: spdx.Apache20},
				}
				got, err := licensebaseline.Producer{}.Files(a, c, &workflow.Contribution{})
				assert.NoError(t, err, "Files")
				h := licenses.Holder{Owner: a.Owner, Name: a.Name, Repository: string(a.Repository), Year: a.Year}
				enterprise, _, err := licenses.Text(&licenses.Config{SPDX: spdx.BUSL11, Parameters: busl}, h)
				assert.NoError(t, err, "Text of BUSL-1.1")
				sdk, notice, err := licenses.Text(&licenses.Config{SPDX: spdx.Apache20}, h)
				assert.NoError(t, err, "Text of Apache-2.0")
				assert.Equal(t, got, []language.File{
					{Path: "enterprise/LICENSE", Content: enterprise},
					{Path: "sdk/go/LICENSE", Content: sdk},
					{Path: "sdk/go/NOTICE", Content: notice},
				}, "the files")
			})

			t.Run("returns no file for options of another type", func(t *testing.T) {
				t.Parallel()
				got, err := licensebaseline.Producer{}.Files(baselinetest.Answers(), nil, &workflow.Contribution{})
				assert.NoError(t, err, "Files")
				assert.Empty(t, got, "the files")
			})

			t.Run("returns ErrInvalid for a directory under BUSL-1.1 without its parameters", func(t *testing.T) {
				t.Parallel()
				a := baselinetest.Answers()
				c := section(a)
				c.Directories = []licenses.Directory{{Path: "enterprise", SPDX: spdx.BUSL11}}
				_, err := licensebaseline.Producer{}.Files(a, c, &workflow.Contribution{})
				assert.ErrorIs(t, err, option.ErrInvalid, "Files")
			})

			t.Run("writes the files of a directory that .ergon.yaml lists, which the lock records", func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				root, err := os.OpenRoot(dir)
				assert.NoError(t, err, "OpenRoot")
				t.Cleanup(func() { _ = root.Close() })
				r, err := baseline.Open(root, new(language.Catalog), baselinetest.Version, producer())
				assert.NoError(t, err, "Open")
				a := baselinetest.Answers()
				_, err = r.New(a, baseline.Options{})
				assert.NoError(t, err, "New")
				config, err := os.ReadFile(filepath.Join(dir, language.Config))
				assert.NoError(t, err, "ReadFile of .ergon.yaml")
				assert.Contains(t, string(config), noDirectories, "the section license of ergon init new")
				listed := strings.Replace(string(config), noDirectories, sdkDirectory, 1)
				assert.NoError(t, os.WriteFile(filepath.Join(dir, language.Config), []byte(listed), 0o644), "WriteFile")
				_, err = r.Sync(nil, baseline.Options{})
				assert.NoError(t, err, "Sync")
				h := licenses.Holder{Owner: a.Owner, Name: a.Name, Repository: string(a.Repository), Year: a.Year}
				text, notice, err := licenses.Text(&licenses.Config{SPDX: spdx.Apache20}, h)
				assert.NoError(t, err, "Text of Apache-2.0")
				files.HasContent(t, filepath.Join(dir, "sdk", "LICENSE"), string(text), "the LICENSE of sdk")
				files.HasContent(t, filepath.Join(dir, "sdk", "NOTICE"), string(notice), "the NOTICE of sdk")
				findings, err := r.Check()
				assert.NoError(t, err, "Check")
				assert.Empty(t, findings, "the findings of Check")
			})
		})

		t.Run("Contribution", func(t *testing.T) {
			t.Parallel()

			t.Run("returns the job license, which runs ergon license check", func(t *testing.T) {
				t.Parallel()
				got := licensebaseline.Producer{}.Contribution(licensebaseline.Producer{}.Options())
				assert.Equal(t, got, workflow.Contribution{Jobs: []workflow.Job{
					{
						ID:          "license",
						Name:        "License",
						Text:        true,
						Timeout:     10,
						Permissions: map[string]string{"contents": "read"},
						Ergon:       true,
						Steps: []workflow.Step{
							{Name: "Check the license headers", Run: []string{"ergon license check"}},
						},
					},
				}}, "the contribution")
			})

			t.Run("returns a valid contribution", func(t *testing.T) {
				t.Parallel()
				got := licensebaseline.Producer{}.Contribution(licensebaseline.Producer{}.Options())
				assert.NoError(t, got.Validate(), "Validate of the contribution")
			})

			t.Run("returns the contribution of the baseline for options of another type", func(t *testing.T) {
				t.Parallel()
				want := licensebaseline.Producer{}.Contribution(licensebaseline.Producer{}.Options())
				assert.Equal(t, licensebaseline.Producer{}.Contribution(nil), want, "the contribution for nil options")
			})
		})
	})
}

// producer returns the producer of the license files as a base producer.
func producer() baseline.Producer {
	return baseline.Producer{Name: licensebaseline.Name, Producer: licensebaseline.Producer{}}
}

// section returns the section license at the baseline with the owner and the license of a, as
// ergon init writes it.
func section(a *language.Answers) *licenses.Config {
	c, _ := licensebaseline.Producer{}.Options().(*licenses.Config)
	c.Owner, c.SPDX = a.Owner, a.License
	return c
}
