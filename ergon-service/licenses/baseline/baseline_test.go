// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package baseline_test

import (
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/option"
	"go.dokimi.dev/ergon/core/spdx"
	"go.dokimi.dev/ergon/core/workflow"
	"go.dokimi.dev/ergon/service/baseline"
	"go.dokimi.dev/ergon/service/baseline/baselinetest"
	licensebaseline "go.dokimi.dev/ergon/service/licenses/baseline"
)

// name pins the name of the producer of the license files.
const name = "license"

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
