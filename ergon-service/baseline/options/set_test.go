// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package options_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/service/baseline/options"
)

// changeDate is the key of the Change Date of BUSL-1.1 in .ergon.yaml.
const changeDate = "license.parameters.change-date"

func TestSet(t *testing.T) {
	t.Parallel()

	t.Run("Set", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the file for no settings", func(t *testing.T) {
			t.Parallel()
			file := []byte("# ours\ncommon:   {ergon: [ergon]}\n")
			got, err := options.Set(file, nil)
			assert.NoError(t, err, "Set")
			assert.Equal(t, string(got), string(file), "the file")
		})

		changed := []struct {
			name         string
			giveFile     string
			giveSettings map[string]string
			want         string
		}{
			{
				name:         "adds the mappings of a key that the file does not have",
				giveFile:     "",
				giveSettings: map[string]string{"license.parameters.licensed-work": "demo 1.0"},
				want:         "license:\n  parameters:\n    licensed-work: demo 1.0\n",
			},
			{
				name:         "replaces the value of a key that the file has",
				giveFile:     "# ours\nlicense:\n  # The owner.\n  owner: Example B.V.\n  parameters:\n    change-date: 2029\n",
				giveSettings: map[string]string{changeDate: "2030"},
				want:         "# ours\nlicense:\n  # The owner.\n  owner: Example B.V.\n  parameters:\n    change-date: \"2030\"\n",
			},
			{
				name:         "writes a value that YAML reads as a date as a string",
				giveFile:     "license: {}\n",
				giveSettings: map[string]string{changeDate: "2030-01-01"},
				want:         "license: {parameters: {change-date: \"2030-01-01\"}}\n",
			},
			{
				name:     "sets each key of the settings",
				giveFile: "",
				giveSettings: map[string]string{
					changeDate:                         "2030-01-01",
					"license.parameters.licensed-work": "demo 1.0",
				},
				want: "license:\n  parameters:\n    change-date: \"2030-01-01\"\n    licensed-work: demo 1.0\n",
			},
		}
		for _, tt := range changed {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := options.Set([]byte(tt.giveFile), tt.giveSettings)
				assert.NoError(t, err, "Set")
				assert.Equal(t, string(got), tt.want, "the file")
			})
		}

		invalid := []struct {
			name     string
			giveFile string
		}{
			{
				name:     "returns ErrInvalid for a key that crosses a value that is not a mapping",
				giveFile: "license: MIT\n",
			},
			{name: "returns ErrInvalid for a file that does not parse", giveFile: "license: [\n"},
			{name: "returns ErrInvalid for a file whose root is not a mapping", giveFile: "- license\n"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := options.Set([]byte(tt.giveFile), map[string]string{changeDate: "2030-01-01"})
				assert.ErrorIs(t, err, options.ErrInvalid, "Set")
			})
		}
	})
}
