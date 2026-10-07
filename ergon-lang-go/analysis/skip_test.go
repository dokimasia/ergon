// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/ergon/lang/go/analysis"
	goanalysis "golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// legacy is the package of the cases that breaks both rules.
const legacy = "legacy"

func TestSkip(t *testing.T) {
	t.Parallel()

	analyzers := []struct {
		name  string
		build func(analysis.Skip) *goanalysis.Analyzer
	}{
		{name: "ErrorPrefix", build: analysis.ErrorPrefix},
		{
			name:  "SkipExpiry",
			build: func(skip analysis.Skip) *goanalysis.Analyzer { return analysis.SkipExpiry(now, skip) },
		},
	}
	for _, a := range analyzers {
		t.Run(a.name, func(t *testing.T) {
			t.Parallel()

			t.Run("reports a package that skip does not skip", func(t *testing.T) {
				t.Parallel()
				r := assert.NewRecorder()
				analysistest.Run(r, analysistest.TestData(), a.build(none), legacy)
				failures := strings.Join(r.Messages(), "\n")
				assert.Contains(t, failures, "unexpected diagnostic", "the failures of the analysis")
			})

			t.Run("skips a package and its external tests that skip reports", func(t *testing.T) {
				t.Parallel()
				skip := func(path string) bool { return path == legacy }
				analysistest.Run(t, analysistest.TestData(), a.build(skip), legacy)
			})
		})
	}
}
