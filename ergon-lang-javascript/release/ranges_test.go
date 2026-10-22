// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package release_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/ergon/core/language"
	"go.dokimi.dev/ergon/core/version"
	"go.dokimi.dev/ergon/lang/javascript/release"
)

// satisfaction is an expectation of testdata/satisfies.json: whether satisfies of node-semver 7.8.5
// admits the version under the range.
type satisfaction struct {
	GiveRange   string `json:"range"`
	GiveVersion string `json:"version"`
	Want        bool   `json:"want"`
}

func TestRanges(t *testing.T) {
	t.Parallel()

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()
		var tests []satisfaction
		data := files.Read(t, filepath.Join("testdata", "satisfies.json"))
		assert.NoError(t, json.Unmarshal([]byte(data), &tests), "the decoding of the expectations")
		assert.NotEmpty(t, tests, "the expectations")
		for _, tt := range tests {
			want, word := language.ResolutionExcluded, "Excluded"
			if tt.Want {
				want, word = language.ResolutionSelected, "Selected"
			}
			t.Run(fmt.Sprintf("returns %s for %s in %q", word, tt.GiveVersion, tt.GiveRange), func(t *testing.T) {
				t.Parallel()
				v, err := version.Parse(tt.GiveVersion)
				assert.NoError(t, err, "Parse")
				got, err := release.Versioner{}.Resolve(tt.GiveRange, v)
				assert.NoError(t, err, "Resolve")
				assert.Equal(t, got, want, "the resolution")
			})
		}
	})
}
